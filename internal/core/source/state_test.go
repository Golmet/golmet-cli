package source

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ---- fixtures git (herméticos: remote bare local + clones com identidade
// fixa; pulam quando git indisponível) ----

// hermeticGitEnv isola o git do ambiente da máquina: config global/system
// próprios do teste (autocrlf=false — o global do Windows reescreve EOL no
// checkout/rebase; identity fixa; defaultBranch=main).
func hermeticGitEnv(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	cfg := "[init]\n\tdefaultBranch = main\n[core]\n\tautocrlf = false\n[user]\n\tname = Test\n\temail = test@example.com\n"
	global := filepath.Join(dir, "gitconfig-global")
	if err := os.WriteFile(global, []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	system := filepath.Join(dir, "gitconfig-system")
	if err := os.WriteFile(system, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	t.Setenv("GIT_CONFIG_SYSTEM", system)
}

type gitTest struct {
	t   *testing.T
	dir string
}

func newGitTest(t *testing.T, dir string) *gitTest {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git indisponível")
	}
	return &gitTest{t: t, dir: dir}
}

func (g *gitTest) run(args ...string) string {
	g.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		g.t.Fatalf("git %v: %v (%s)", args, err, out)
	}
	return string(out)
}

func (g *gitTest) commitAll(msg string) {
	g.t.Helper()
	g.run("-c", "user.name=Test", "-c", "user.email=test@example.com", "add", "-A")
	g.run("-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-m", msg)
}

func (g *gitTest) write(rel, content string) {
	g.t.Helper()
	p := filepath.Join(g.dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		g.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		g.t.Fatal(err)
	}
}

func (g *gitTest) read(rel string) string {
	g.t.Helper()
	data, err := os.ReadFile(filepath.Join(g.dir, filepath.FromSlash(rel)))
	if err != nil {
		g.t.Fatalf("arquivo esperado ausente: %s: %v", rel, err)
	}
	return string(data)
}

// newStatePackRemote cria <root>/origin.git (bare) com um pack state semeado
// e devolve (remoteURL, seed) — remoteURL é file:/// (bare repo É um
// diretório: sem o scheme o kit o trataria como fonte local), seed é um
// clone comum para avançar o remote.
func newStatePackRemote(t *testing.T, root string) (remoteURL, seed string) {
	t.Helper()
	hermeticGitEnv(t)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	newGitTest(t, root) // garante git disponível
	remote := filepath.Join(root, "origin.git")
	seed = filepath.Join(root, "seed")
	g := newGitTest(t, root)
	g.run("init", "--bare", "-b", "main", remote)
	g.run("clone", remote, seed)
	s := newGitTest(t, seed)
	s.write("pack.yaml", "id: state-pack\nversion: 1.0.0\nname: State pack\nkind: state\n")
	s.write("skills/gm-state/SKILL.md", "skill original\n")
	s.commitAll("seed do pack state")
	s.run("branch", "-M", "main")
	s.run("push", "-u", "origin", "main")
	return "file:///" + filepath.ToSlash(remote), seed
}

// ---- SyncState ----

func TestSyncStateCommitsWIPAndRebases(t *testing.T) {
	root := t.TempDir()
	remoteURL, _ := newStatePackRemote(t, root)
	work := filepath.Join(root, "work")
	g := newGitTest(t, root)
	g.run("clone", remoteURL, work)
	w := newGitTest(t, work)
	w.write("skills/gm-state/SKILL.md", "acumulação local\n")
	rep, err := SyncState(work)
	if err != nil {
		t.Fatalf("SyncState: %v", err)
	}
	if !rep.AutoCommit {
		t.Errorf("AutoCommit = false, want true (WIP deve ser commitado — dado nunca se perde)")
	}
	if rep.Committed != 1 {
		t.Errorf("Committed = %d, want 1", rep.Committed)
	}
	if !rep.Rebased {
		t.Errorf("Rebased = false, want true (fetch + rebase executados)")
	}
	if rep.Branch != "main" {
		t.Errorf("Branch = %q, want main", rep.Branch)
	}
	if rep.SHA == "" {
		t.Error("SHA vazio no relatório do sync")
	}
	// o dado do usuário está no working copy E commitado
	if got := w.read("skills/gm-state/SKILL.md"); got != "acumulação local\n" {
		t.Errorf("conteúdo acumulado perdido: %q", got)
	}
	if subject := g.run("-C", work, "log", "-1", "--format=%s"); !strings.Contains(subject, "sync: acumulação local") {
		t.Errorf("commit automático ausente: %q", subject)
	}
	// working copy limpo após o sync
	if status := g.run("-C", work, "status", "--porcelain"); strings.TrimSpace(status) != "" {
		t.Errorf("working copy sujo após o sync: %q", status)
	}
}

func TestSyncStateConflictFailsWithRemediationAndPreservesData(t *testing.T) {
	root := t.TempDir()
	remoteURL, seed := newStatePackRemote(t, root)
	work := filepath.Join(root, "work")
	g := newGitTest(t, root)
	g.run("clone", remoteURL, work)
	w := newGitTest(t, work)

	// o remote avança mudando a MESMA linha da skill
	s := newGitTest(t, seed)
	s.write("skills/gm-state/SKILL.md", "versão do remoto\n")
	s.commitAll("mudança remota")
	s.run("push", "origin", "main")

	// WIP local conflitante: mesmo arquivo, mesma linha
	w.write("skills/gm-state/SKILL.md", "versão local\n")

	_, err := SyncState(work)
	if err == nil || !strings.Contains(err.Error(), "resolva no git") || !strings.Contains(err.Error(), "nunca descarta") {
		t.Fatalf("conflito de rebase deveria falhar com remediação clara: %v", err)
	}
	// rebase parado no working copy (a remediação sugerida é aplicável)
	if !rebaseInProgress(work) {
		t.Fatal("rebase deveria estar em progresso após o conflito")
	}
	// o dado do usuário foi PRESERVADO no commit do WIP (ORIG_HEAD aponta
	// para o pré-rebase = commit do sync)
	subject := g.run("-C", work, "log", "-1", "--format=%s", "ORIG_HEAD")
	if !strings.Contains(subject, "sync: acumulação local") {
		t.Errorf("WIP não foi commitado antes do rebase (dado em risco): %q", subject)
	}
	// remediação sugerida funciona: abort restaura o estado com o WIP commitado
	g.run("-C", work, "rebase", "--abort")
	if got := w.read("skills/gm-state/SKILL.md"); got != "versão local\n" {
		t.Errorf("acumulação local perdida após abort: %q", got)
	}
}

func TestSyncStateWithoutRemoteIsLocalOnly(t *testing.T) {
	root := t.TempDir()
	hermeticGitEnv(t)
	g := newGitTest(t, root)
	g.run("init", "-b", "main", root)
	w := newGitTest(t, root)
	w.write("base.md", "base commitada\n")
	w.commitAll("base") // branch com commit — rev-parse HEAD funciona
	w.write("k.md", "dado\n")
	rep, err := SyncState(root)
	if err != nil {
		t.Fatalf("SyncState sem origin deveria funcionar apenas local: %v", err)
	}
	if !rep.NoRemote || rep.Rebased {
		t.Errorf("NoRemote = %v, Rebased = %v; want true, false", rep.NoRemote, rep.Rebased)
	}
	if !rep.AutoCommit {
		t.Error("AutoCommit = false, want true (WIP foi commitado)")
	}
}

func TestSyncStateRequiresGitRepo(t *testing.T) {
	dir := t.TempDir() // sem .git
	if _, err := SyncState(dir); err == nil || !strings.Contains(err.Error(), "working copy Git") {
		t.Fatalf("sync em diretório sem .git deveria falhar claro: %v", err)
	}
}

func TestSyncStateDryRunOnlyInspects(t *testing.T) {
	root := t.TempDir()
	remoteURL, _ := newStatePackRemote(t, root)

	work := filepath.Join(root, "work")
	g := newGitTest(t, root)
	g.run("clone", remoteURL, work)
	w := newGitTest(t, work)
	w.write("skills/gm-state/SKILL.md", "rascunho\n")

	rep, err := SyncStateWithOptions(work, SyncStateOptions{DryRun: true})
	if err != nil {
		t.Fatalf("SyncState dry-run: %v", err)
	}
	if !rep.DryRun || rep.AutoCommit || rep.Rebased {
		t.Errorf("dry-run mutou estado: %+v", rep)
	}
	if rep.Committed != 1 {
		t.Errorf("Committed = %d, want 1 (WIP pendente reportado)", rep.Committed)
	}
	// nada foi commitado: WIP continua pendente
	status := g.run("-C", work, "status", "--porcelain")
	if !strings.Contains(status, "SKILL.md") {
		t.Errorf("dry-run não deveria commitar o WIP: %q", status)
	}
}

// ---- Publish ----

func TestPublishCommitsWIPAndPushes(t *testing.T) {
	root := t.TempDir()
	remoteURL, _ := newStatePackRemote(t, root)
	work := filepath.Join(root, "work")
	g := newGitTest(t, root)
	g.run("clone", remoteURL, work)
	w := newGitTest(t, work)
	w.write("accumulation/patterns/novo.md", "padrão acumulado\n")

	rep, err := Publish(work, "publicando acumulação")
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if !rep.Committed || !rep.Pushed {
		t.Errorf("Committed = %v, Pushed = %v; want true, true", rep.Committed, rep.Pushed)
	}
	if subject := g.run("-C", work, "log", "-1", "--format=%s"); !strings.Contains(subject, "publish: publicando acumulação") {
		t.Errorf("mensagem do publish ausente: %q", subject)
	}
	// o remote recebeu: um clone fresco já traz o conteúdo publicado
	fresh := filepath.Join(root, "fresh")
	g.run("clone", remoteURL, fresh)
	f := newGitTest(t, fresh)
	if got := f.read("accumulation/patterns/novo.md"); got != "padrão acumulado\n" {
		t.Errorf("push não chegou ao remote: %q", got)
	}
}

func TestPublishWithoutWIPStillPushes(t *testing.T) {
	root := t.TempDir()
	remoteURL, _ := newStatePackRemote(t, root)
	work := filepath.Join(root, "work")
	g := newGitTest(t, root)
	g.run("clone", remoteURL, work)
	rep, err := Publish(work, "")
	if err != nil {
		t.Fatalf("Publish sem WIP: %v", err)
	}
	if rep.Committed {
		t.Error("Committed = true, want false (não havia WIP)")
	}
	if !rep.Pushed {
		t.Error("Pushed = false, want true (estado commitado é publicado)")
	}
}

func TestPublishRequiresGitRepo(t *testing.T) {
	if _, err := Publish(t.TempDir(), "x"); err == nil || !strings.Contains(err.Error(), "working copy Git") {
		t.Fatalf("publish em diretório sem .git deveria falhar claro: %v", err)
	}
}
