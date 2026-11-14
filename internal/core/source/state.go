// state.go — working copy bidirecional de state packs (arquétipo `kind:
// state`). Diferente do cache somente-leitura do overlay (Sync), o state
// pack é um clone Git COMPLETO que o usuário acumula: o sync commita o WIP
// local (o dado NUNCA se perde), faz fetch + rebase sobre origin e FALHA
// com remediação clara em conflito. Push NÃO faz parte do sync — subir é
// decisão do operador (Publish / `golmet pack publish`).
package source

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// StateSyncReport descreve o resultado do sync de um working copy de state
// pack (subconjunto estruturado do relatório do CLI/MCP).
type StateSyncReport struct {
	Dir        string `json:"dir"`
	Branch     string `json:"branch"`
	SHA        string `json:"sha,omitempty"`
	AutoCommit bool   `json:"auto_commit,omitempty"` // houve WIP → commitado antes do rebase
	Committed  int    `json:"committed,omitempty"`   // entradas no status por ocasião do commit automático
	Rebased    bool   `json:"rebased,omitempty"`     // fetch + rebase sobre origin aplicados
	NoRemote   bool   `json:"no_remote,omitempty"`   // sem origin — sync local apenas
	DryRun     bool   `json:"dry_run,omitempty"`
}

// SyncStateOptions controla efeitos colaterais do sync de state packs.
type SyncStateOptions struct {
	DryRun bool // só inspeciona (branch + WIP pendente); nunca commita/rebasa
}

// SyncState sincroniza o working copy em dir (wrapper compatível).
func SyncState(dir string) (*StateSyncReport, error) {
	return SyncStateWithOptions(dir, SyncStateOptions{})
}

// SyncStateWithOptions sincroniza um working copy de state pack:
//
//  1. mudanças não-commitadas (qualquer arquivo exceto .git/) → git add -A
//     + commit "sync: acumulação local <UTC ts>" — o dado do usuário NUNCA
//     se perde;
//  2. git fetch origin + git rebase origin/<branch-atual> (branch = a
//     checked out, normalmente main);
//  3. conflito de rebase = FALHA com remediação clara — o kit nunca
//     descarta acumulação; o WIP já está commitado, nada se perde.
//
// Push NÃO faz parte do sync (manual, decisão do operador — Publish).
func SyncStateWithOptions(dir string, opt SyncStateOptions) (*StateSyncReport, error) {
	rep := &StateSyncReport{Dir: dir}
	if err := requireGitRepo(dir); err != nil {
		return nil, err
	}
	branch, err := CurrentBranch(dir)
	if err != nil {
		return nil, err
	}
	rep.Branch = branch
	status, err := run(dir, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	rep.Committed = countLines(status)
	if opt.DryRun {
		rep.DryRun = true
		sha, herr := head(dir)
		if herr != nil {
			return nil, herr
		}
		rep.SHA = sha
		return rep, nil
	}
	if rep.Committed > 0 {
		if _, err := run(dir, "add", "-A"); err != nil {
			return nil, err
		}
		msg := "sync: acumulação local " + time.Now().UTC().Format("2006-01-02T15:04:05Z")
		if _, err := commitWIP(dir, msg); err != nil {
			return nil, err
		}
		rep.AutoCommit = true
	}
	if _, err := run(dir, "remote", "get-url", "origin"); err == nil {
		if _, err := run(dir, "fetch", "origin"); err != nil {
			return nil, err
		}
		if _, err := run(dir, "rebase", "origin/"+branch); err != nil {
			if rebaseInProgress(dir) {
				return nil, fmt.Errorf("rebase com conflito no working copy do state pack (%s): resolva no git (git rebase --continue/--abort) e rode de novo — o kit nunca descarta acumulação", dir)
			}
			return nil, err
		}
		rep.Rebased = true
	} else {
		rep.NoRemote = true
	}
	sha, err := head(dir)
	if err != nil {
		return nil, err
	}
	rep.SHA = sha
	return rep, nil
}

// PublishReport descreve o resultado do publish de um state pack.
type PublishReport struct {
	Dir       string `json:"dir"`
	Branch    string `json:"branch"`
	SHA       string `json:"sha,omitempty"`
	Committed bool   `json:"committed,omitempty"` // houve WIP → commitado no publish
	Pushed    bool   `json:"pushed"`
}

// Publish "sobe" o working copy: commita o WIP ("publish: <message>") e faz
// git push origin <branch-atual>. Sem WIP, apenas publica o estado commitado
// (idempotente).
func Publish(dir, message string) (*PublishReport, error) {
	rep := &PublishReport{Dir: dir}
	if err := requireGitRepo(dir); err != nil {
		return nil, err
	}
	branch, err := CurrentBranch(dir)
	if err != nil {
		return nil, err
	}
	rep.Branch = branch
	if strings.TrimSpace(message) == "" {
		message = "acumulação local"
	}
	status, err := run(dir, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	if countLines(status) > 0 {
		if _, err := run(dir, "add", "-A"); err != nil {
			return nil, err
		}
		if _, err := commitWIP(dir, "publish: "+message); err != nil {
			return nil, err
		}
		rep.Committed = true
	}
	if _, err := run(dir, "push", "origin", branch); err != nil {
		return nil, err
	}
	rep.Pushed = true
	sha, err := head(dir)
	if err != nil {
		return nil, err
	}
	rep.SHA = sha
	return rep, nil
}

// CurrentBranch devolve a branch checked out (abbrev-ref HEAD).
func CurrentBranch(dir string) (string, error) {
	out, err := run(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" || branch == "HEAD" {
		return "", fmt.Errorf("working copy em detached HEAD — volte para uma branch antes de sincronizar: %s", dir)
	}
	return branch, nil
}

// Origin devolve a URL do remote origin (normalizada — sem espaços).
func Origin(dir string) (string, error) {
	out, err := run(dir, "remote", "get-url", "origin")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// CloneDir clona remote em target (clone COMPLETO — um state pack é working
// copy que rebasa e publica; clone raso quebraria o rebase).
func CloneDir(remote, target string) error {
	parent := filepath.Dir(target)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	_, err := run(parent, "clone", "--", remote, target)
	return err
}

// ProbeGitSource faz um clone raso de remote em dir (dir não pode existir).
// Serve para ler o pack.yaml de uma fonte remota sem materializar o cache —
// o caller apaga dir no fim da sondagem.
func ProbeGitSource(remote, dir string) error {
	parent := filepath.Dir(dir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	_, err := run(parent, "clone", "--depth", "1", "--", remote, dir)
	return err
}

// requireGitRepo garante que dir é um working copy Git (.git presente —
// diretório ou arquivo de worktree).
func requireGitRepo(dir string) error {
	if _, err := os.Lstat(filepath.Join(dir, ".git")); err != nil {
		return fmt.Errorf("state pack requer um working copy Git (com .git): %s", dir)
	}
	return nil
}

// commitWIP cria um commit; sem user.email configurado (local+global),
// usa a identidade do kit — a acumulação nunca falha por falta de config git.
func commitWIP(dir, message string) ([]byte, error) {
	args := []string{"commit", "-m", message}
	if out, err := run(dir, "config", "user.email"); err != nil || strings.TrimSpace(string(out)) == "" {
		args = append([]string{"-c", "user.name=golmet-kit", "-c", "user.email=kit@golmet.dev"}, args...)
	}
	return run(dir, args...)
}

// rebaseInProgress — há um rebase parado no working copy? (conflito).
func rebaseInProgress(dir string) bool {
	gitDir := filepath.Join(dir, ".git")
	for _, marker := range []string{"rebase-merge", "rebase-apply"} {
		if _, err := os.Stat(filepath.Join(gitDir, marker)); err == nil {
			return true
		}
	}
	return false
}

// countLines conta entradas não-vazias de uma saída --porcelain.
func countLines(out []byte) int {
	n := 0
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) != "" {
			n++
		}
	}
	return n
}
