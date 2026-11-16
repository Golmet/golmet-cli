// state.go — arquétipo `kind: state`: o pack é um working copy Git
// bidirecional em <configRoot>/packs/<id> (clone completo, origin fixada).
// skills/ e agents/ do working copy são materializadas por overlay 3-way
// (mesmo ApplyOverlay do overlay pack — posse do pack substitui o render do
// core para o mesmo caminho; arquivos locais divergentes continuam
// preservados e o no-override do pack é respeitado). DADOS — knowledge/,
// accumulation/ e rules/ — NÃO são materializados: são lidos IN-PLACE do
// working copy e os leitores (gm-domain-research, pattern library, lazy
// index) apontam para packs/<id>/....
package pack

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Golmet/golmet-cli/internal/core/content"
	"github.com/Golmet/golmet-cli/internal/core/indexstate"
	"github.com/Golmet/golmet-cli/internal/core/source"
)

// probeSource lê o pack.yaml da fonte sem materializar nada: diretório
// local é lido direto; para URL Git há duas vias — (1) fast-path: se o cache
// overlay (<home>/.golmet/packs/<slug>) já conhece o pack como overlay, não
// há sondagem (fluxo original intocado); (2) clone raso temporário, apagado
// pelo cleanup. Devolve o spec e o diretório de onde ele foi lido.
func probeSource(sourceURL, cacheRoot string) (Spec, string, func(), error) {
	if st, err := os.Stat(sourceURL); err == nil && st.IsDir() {
		spec, serr := loadSpec(sourceURL)
		return spec, sourceURL, func() {}, serr
	}
	// fast-path overlay: pack já em cache conhecido como overlay → sem probe
	if spec, err := loadSpec(filepath.Join(cacheRoot, source.Slug(sourceURL))); err == nil && spec.Kind == KindOverlay {
		return spec, "", func() {}, nil
	}
	tmp, err := os.MkdirTemp("", "golmet-pack-probe-")
	if err != nil {
		return Spec{}, "", nil, err
	}
	cleanup := func() { os.RemoveAll(tmp) }
	if err := source.ProbeGitSource(sourceURL, tmp); err != nil {
		cleanup()
		return Spec{}, "", nil, err
	}
	spec, serr := loadSpec(tmp)
	if serr != nil {
		cleanup()
		return Spec{}, "", nil, serr
	}
	return spec, tmp, cleanup, nil
}

// addOrUpdateState resolve o working copy do state pack em
// <configRoot>/packs/<id>, sincroniza (commit WIP + fetch + rebase) e
// re-materializa. Working copy JÁ EXISTENTE com a mesma origem é ADOTADO
// (caminho de migração do operador) — nunca reclone nem sobrescreva.
// Dry-run não clona nem sincroniza: inspeciona a fonte/working copy e
// calcula o plano de materialização.
func addOrUpdateState(configRoot, sourceURL string, spec Spec, probeDir string, force, dryRun bool) (*Report, error) {
	if err := ValidateRequiresCore(spec.RequiresCore, indexstate.KitVersion); err != nil {
		return nil, fmt.Errorf("pack %s: %w", spec.ID, err)
	}
	target := filepath.Join(statePacksRoot(configRoot), spec.ID)
	var (
		state   *source.StateSyncReport
		matRoot string
		cloned  bool
		err     error
	)
	switch {
	case dirExists(target):
		matRoot = target
		if !sameDirPath(probeDir, target) {
			if err := ensureStateWorkingCopy(target, sourceURL); err != nil {
				return nil, err
			}
		}
		state, err = source.SyncStateWithOptions(target, source.SyncStateOptions{DryRun: dryRun})
	case dryRun:
		matRoot = probeDir // plano sobre a fonte; nada é clonado nem sincronizado
		state, err = source.SyncStateWithOptions(probeDir, source.SyncStateOptions{DryRun: true})
		if err == nil {
			state.Dir = target // o plano aponta para o destino real
		}
	default:
		if err := source.CloneDir(sourceURL, target); err != nil {
			return nil, fmt.Errorf("pack state %s: %w", spec.ID, err)
		}
		cloned = true
		matRoot = target
		state, err = source.SyncStateWithOptions(target, source.SyncStateOptions{})
	}
	if err != nil {
		return nil, fmt.Errorf("pack state %s: %w", spec.ID, err)
	}
	return finishState(configRoot, sourceURL, spec, target, matRoot, state, cloned, force, dryRun)
}

// finishState carrega o spec fresco do working copy, materializa e monta o
// Report unificado (Sync para compatibilidade + StateSync estruturado).
func finishState(configRoot, sourceURL string, spec Spec, target, matRoot string, state *source.StateSyncReport, cloned, force, dryRun bool) (*Report, error) {
	if matRoot == target {
		if fresh, err := loadSpec(target); err == nil {
			spec = fresh
		}
	}
	files, err := stateOverlayFiles(matRoot)
	if err != nil {
		return nil, fmt.Errorf("pack state %s: %w", spec.ID, err)
	}
	rep, err := content.ApplyOverlay(configRoot, files, content.OverlayOptions{
		Layer:      "pack:" + spec.ID,
		Source:     source.Redact(sourceURL) + "@" + state.SHA,
		NoOverride: spec.NoOverride, Force: force, DryRun: dryRun,
	})
	if err != nil {
		return nil, err
	}
	return &Report{
		Spec: spec,
		Sync: &source.SyncReport{
			Source:  source.Redact(sourceURL),
			Path:    target,
			SHA:     state.SHA,
			Cloned:  cloned,
			Updated: state.Rebased,
		},
		StateSync: state,
		Written:   rep.Written,
		Preserved: rep.Preserved,
		Conflicts: rep.Conflicts,
	}, nil
}

// ensureStateWorkingCopy valida que target é um working copy Git de state
// pack apontando para a MESMA origem — o caminho de adoção nunca reclone nem
// sobrescreve; origem divergente é erro com remediação explícita.
func ensureStateWorkingCopy(target, sourceURL string) error {
	if _, err := os.Lstat(filepath.Join(target, ".git")); err != nil {
		return fmt.Errorf("%s já existe mas não é um working copy Git — remova-o ou use outro id antes de instalar o pack state", target)
	}
	origin, err := source.Origin(target)
	if err != nil {
		return fmt.Errorf("working copy %s sem remote origin — configure com 'git remote set-url origin <url>' ou remova o diretório", target)
	}
	if !sameOrigin(origin, sourceURL) {
		return fmt.Errorf("origem divergente: %s aponta para %q, mas a fonte informada foi %q — alinhe com 'git remote set-url origin' ou remova o working copy (o kit nunca reclone nem descarta acumulação)", target, origin, source.Redact(sourceURL))
	}
	return nil
}

// stateOverlayFiles coleta o conteúdo materializável do working copy:
// APENAS skills/** e agents/** (overlay 3-way — mecânica do engine). Dados
// — knowledge/, accumulation/ e rules/ — ficam NO working copy e são lidos
// IN-PLACE pelos consumers (lazy index aponta packs/<id>/rules/packs/<id>.md).
func stateOverlayFiles(root string) ([]content.OverlayFile, error) {
	var files []content.OverlayFile
	for _, top := range []string{"skills", "agents"} {
		dir := filepath.Join(root, top)
		if _, err := os.Stat(dir); err != nil {
			continue
		}
		werr := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == ".git" {
					return filepath.SkipDir
				}
				return nil
			}
			if !d.Type().IsRegular() {
				return nil // symlinks/junctions nunca são materializados
			}
			rel, rerr := filepath.Rel(root, p)
			if rerr != nil {
				return rerr
			}
			rel = filepath.ToSlash(rel)
			data, rerr := readProvidedFile(root, rel)
			if rerr != nil {
				return rerr
			}
			files = append(files, content.OverlayFile{Rel: rel, Data: data})
			return nil
		})
		if werr != nil {
			return nil, werr
		}
	}
	return files, nil
}

// statePacksRoot — raiz dos working copies de state packs no config do
// engine (<configRoot>/packs; os leitores de conhecimento apontam para cá).
func statePacksRoot(configRoot string) string {
	return filepath.Join(configRoot, "packs")
}

// IsStateWorkingCopy — dir é um working copy de state pack (pack.yaml com
// kind: state). Erro de leitura/parse → não é (comportamento padrão).
func IsStateWorkingCopy(dir string) bool {
	spec, err := loadSpec(dir)
	return err == nil && spec.Kind == KindState
}

// ResolveStateDir localiza o working copy de um state pack para operações
// sem engine (publish): sourceURL local com .git e kind state → ele mesmo;
// senão varre as raízes packsRoots (<config>/packs de cada engine) casando
// a origem. Primeiro match na ordem das raízes.
func ResolveStateDir(sourceURL string, packsRoots []string) (string, error) {
	if st, err := os.Stat(sourceURL); err == nil && st.IsDir() {
		if _, gerr := os.Lstat(filepath.Join(sourceURL, ".git")); gerr == nil {
			if IsStateWorkingCopy(sourceURL) {
				// Repo CENTRAL da org (sem origin próprio): publique a partir
				// do working copy instalado cuja origin aponta para ele.
				if _, oerr := source.Origin(sourceURL); oerr != nil {
					if dir, ferr := workingCopyByOrigin(sourceURL, packsRoots); ferr == nil {
						return dir, nil
					}
				}
				return sourceURL, nil
			}
			return "", fmt.Errorf("%s é um working copy Git mas não é um state pack (pack.yaml sem kind: state)", sourceURL)
		}
	}
	for _, root := range packsRoots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			dir := filepath.Join(root, e.Name())
			if _, gerr := os.Lstat(filepath.Join(dir, ".git")); gerr != nil {
				continue
			}
			if !IsStateWorkingCopy(dir) {
				continue
			}
			if origin, oerr := source.Origin(dir); oerr == nil && sameOrigin(origin, sourceURL) {
				return dir, nil
			}
		}
	}
	return "", fmt.Errorf("nenhum working copy de state pack encontrado para a origem %s — rode 'golmet pack add <fonte> --engine <id>' ou passe o caminho do working copy", source.Redact(sourceURL))
}

// PublishReport é o resultado do publish de um state pack (o "subir" do
// colaborador): commit do WIP + push origin/<branch>.
type PublishReport struct {
	Pack      string `json:"pack"`
	Dir       string `json:"dir"`
	Branch    string `json:"branch"`
	SHA       string `json:"sha,omitempty"`
	Committed bool   `json:"committed,omitempty"`
	Pushed    bool   `json:"pushed"`
}

// PublishState resolve o working copy pela origem (ou aceita o caminho dele)
// e publica: commit do WIP ("publish: <message>") + push.
func PublishState(sourceURL, message string, packsRoots []string) (*PublishReport, error) {
	dir, err := ResolveStateDir(sourceURL, packsRoots)
	if err != nil {
		return nil, err
	}
	spec, serr := loadSpec(dir)
	if serr != nil {
		return nil, serr
	}
	pub, err := source.Publish(dir, message)
	if err != nil {
		return nil, err
	}
	return &PublishReport{
		Pack: spec.ID, Dir: pub.Dir, Branch: pub.Branch, SHA: pub.SHA,
		Committed: pub.Committed, Pushed: pub.Pushed,
	}, nil
}

// workingCopyByOrigin devolve o working copy de state pack cujo origin
// remoto aponta para o caminho informado (o repo central da organização).
func workingCopyByOrigin(central string, packsRoots []string) (string, error) {
	for _, root := range packsRoots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range entries {
			dir := filepath.Join(root, e.Name())
			if !IsStateWorkingCopy(dir) {
				continue
			}
			if origin, oerr := source.Origin(dir); oerr == nil && sameOrigin(origin, central) {
				return dir, nil
			}
		}
	}
	return "", fmt.Errorf("nenhum working copy com origin %s", central)
}

// ListState lista os working copies de state packs sob as raízes informadas
// (<config>/packs de cada engine) — working copies não são cache somente-
// leitura, então não aparecem no List do cache overlay.
func ListState(packsRoots []string) ([]Summary, error) {
	var out []Summary
	seen := map[string]bool{}
	for _, root := range packsRoots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			dir := filepath.Join(root, e.Name())
			spec, serr := loadSpec(dir)
			if serr != nil || spec.Kind != KindState {
				continue
			}
			if seen[spec.ID] {
				continue
			}
			seen[spec.ID] = true
			out = append(out, Summary{Spec: spec, Slug: e.Name(), Path: dir})
		}
	}
	return out, nil
}

// sameOrigin compara origens tolerando sufixo .git e barra final.
func sameOrigin(a, b string) bool {
	norm := func(s string) string {
		s = strings.TrimSpace(s)
		s = strings.TrimSuffix(s, "/")
		return strings.TrimSuffix(s, ".git")
	}
	return norm(a) == norm(b)
}

// sameDirPath compara caminhos de diretório (case-insensitive no Windows).
func sameDirPath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	clean := func(p string) string {
		if abs, err := filepath.Abs(p); err == nil {
			p = abs
		}
		return filepath.Clean(p)
	}
	a, b = clean(a), clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// dirExists — caminho existe e é diretório.
func dirExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}
