// engineconfig.go — check de ESTADO DO ENGINE-CONFIG do doctor. Com perfis
// essencial/completo + packs, o config do engine (~/.config/opencode etc.)
// é CACHE reconstrutível de 3 fontes (kit embarcado, packs SHA-pinados,
// zona do projeto) — EXCETO pelo estado do usuário que o kit não
// reconstrói: customizações reais em arquivos geridos, arquivos locais sem
// proveniência no manifest (knowledge/, plugins/, settings) e mudanças
// não-commitadas quando o config é um repo git. NADA avisava o que seria
// perdido — este check CLASSIFICA por arquivo (read-only — nunca escreve)
// contra o manifest de conteúdo que o install/materialize escreveu.
package doctor

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Golmet/golmet-cli/internal/core/content"
)

// untrackedListCap — cap de cada lista de paths na seção/render (o resumo
// por diretório cobre o resto — não inundar terminal nem payload JSON).
const untrackedListCap = 20

// dirRootLabel — rótulo do grupo de arquivos soltos na raiz do config
// (sem diretório de 1º nível).
const dirRootLabel = "(raiz)"

// DirCount resume arquivos não-reconstrutíveis por diretório (o prefixo de
// diretório mais profundo compartilhado por todos os arquivos do grupo —
// ex.: plugins/ e knowledge/banco-dados-distribuidos/).
type DirCount struct {
	Dir   string `json:"dir"`
	Count int    `json:"count"`
}

// EngineConfigSection é o estado classificado do config do engine — seção
// estruturada (JSON do doctor) e insumo do render textual. Kinds:
// customized = gerido divergente do manifest (dado REAL do usuário);
// untracked = sem proveniência no manifest (não-reconstrutível pelo kit);
// removed_local = layer local ausente em disco (dado perdido).
type EngineConfigSection struct {
	Root              string     `json:"root"`
	ManagedOK         int        `json:"managed_ok"`           // geridos iguais ao manifest (cache reconstrutível)
	Customized        []string   `json:"customized,omitempty"` // cap untrackedListCap
	CustomizedTotal   int        `json:"customized_total,omitempty"`
	UntrackedDirs     []DirCount `json:"untracked_dirs,omitempty"` // resumo por diretório de 1º nível
	Untracked         []string   `json:"untracked,omitempty"`      // cap untrackedListCap
	UntrackedTotal    int        `json:"untracked_total,omitempty"`
	RemovedLocal      []string   `json:"removed_local,omitempty"` // cap untrackedListCap
	RemovedLocalTotal int        `json:"removed_local_total,omitempty"`
	Unreadable        []string   `json:"unreadable,omitempty"` // ilegíveis — não classificáveis (anomalia)
	GitRepo           bool       `json:"git_repo"`
	GitDirty          int        `json:"git_dirty,omitempty"` // mudanças não-commitadas (melhor-esforço)
}

// AtRisk devolve o total de estado do usuário que o kit não reconstrói —
// o que uma limpeza do config (ou um --force descuidado) perderia.
func (s *EngineConfigSection) AtRisk() int {
	return s.CustomizedTotal + s.UntrackedTotal + s.RemovedLocalTotal
}

// runEngineConfig executa o check contra a raiz do config do engine e
// registra o check + a seção estruturada no relatório. Root vazio nunca
// chega aqui (Run só chama com root não-vazio — compatibilidade total).
func (r *Report) runEngineConfig(root string) {
	sec, aerr := analyzeEngineConfig(root)
	r.EngineConfig = sec
	switch {
	case aerr != nil:
		r.add("engine-config", Warn, "manifest de conteúdo ilegível em "+root+" ("+aerr.Error()+")",
			"corrija o .golmet/manifest.json do config ou rode o update do kit para reescrevê-lo")
	case sec.ManagedOK == 0 && sec.AtRisk() == 0 && len(sec.Unreadable) == 0:
		r.add("engine-config", OK, "raiz do config inexistente ("+root+") — nada gerido, nada a perder", "")
	case sec.AtRisk() == 0 && len(sec.Unreadable) == 0:
		r.add("engine-config", OK,
			fmt.Sprintf("cache reconstrutível — %d arquivo(s) gerido(s) conforme(s) ao manifest", sec.ManagedOK), "")
	default:
		r.add("engine-config", Warn, sec.summaryLine(), sec.fixText())
	}
}

// summaryLine — uma linha com o estado classificado (estilo dos checks
// existentes: contagens + agregados, nunca a lista inteira).
func (s *EngineConfigSection) summaryLine() string {
	var parts []string
	if s.ManagedOK > 0 {
		parts = append(parts, fmt.Sprintf("%d gerido(s) conforme(s)", s.ManagedOK))
	}
	if s.CustomizedTotal > 0 {
		parts = append(parts, fmt.Sprintf("%d customização(ões) local(is)", s.CustomizedTotal))
	}
	if s.UntrackedTotal > 0 {
		var dirs []string
		for _, d := range s.UntrackedDirs {
			if d.Dir == dirRootLabel {
				dirs = append(dirs, fmt.Sprintf("%s %d", dirRootLabel, d.Count))
			} else {
				dirs = append(dirs, fmt.Sprintf("%s/ %d", d.Dir, d.Count))
			}
		}
		parts = append(parts, fmt.Sprintf("%d arquivo(s) não-reconstrutível(is) pelo kit (%s)",
			s.UntrackedTotal, strings.Join(dirs, ", ")))
	}
	if s.RemovedLocalTotal > 0 {
		parts = append(parts, fmt.Sprintf("%d local(is) removido(s) do disco", s.RemovedLocalTotal))
	}
	if s.GitRepo && s.GitDirty > 0 {
		parts = append(parts, fmt.Sprintf("%d mudança(s) não-commitada(s)", s.GitDirty))
	}
	return strings.Join(parts, " — ")
}

// fixText — remediação estilo dos checks existentes. Config não-versionado
// com estado em risco ganha o versionamento como primeiro passo (o design:
// git init && git add -A && git commit OU mover para zona/pack).
func (s *EngineConfigSection) fixText() string {
	if !s.GitRepo && s.AtRisk() > 0 {
		return fmt.Sprintf("config não versionado — %d arquivo(s) não-reconstrutíveis; versione com 'git init && git add -A && git commit' em %s, ou mova o conteúdo para a zona do workspace (.golmet/) ou para um pack", s.AtRisk(), s.Root)
	}
	if s.AtRisk() > 0 {
		return "revise e commite o estado local — o kit só rematerializa o que está no manifest; sobrescrever divergente exige --force explícito"
	}
	return ""
}

// analyzeEngineConfig classifica cada arquivo do config root (walk
// recursivo, EXCLUINDO .git/, .golmet/ e o próprio manifest de conteúdo):
//   - gerido + hash bate (EOL/BOM normalizado) → cache reconstrutível (conta);
//   - gerido + divergente → customização local (dado real — reporta path);
//   - sem proveniência no manifest → não-reconstrutível (resumo por
//     diretório de 1º nível + lista com cap);
//   - layer local ausente em disco → local removido (dado perdido).
//     Layers core/pack ausentes o kit rematerializa — nunca reportados.
//     Erro só para manifest ilegível — walk ilegível retorna na seção.
func analyzeEngineConfig(root string) (*EngineConfigSection, error) {
	sec := &EngineConfigSection{Root: root}
	if st, serr := os.Stat(root); serr != nil || !st.IsDir() {
		return sec, nil
	}
	manifest, merr := content.ReadContentManifest(root)
	if merr != nil {
		return sec, merr
	}
	managed := map[string]content.FileEntry{}
	if manifest != nil {
		for _, e := range manifest.Files {
			if e.File != "" {
				managed[e.File] = e
			}
		}
	}

	var untracked []string
	werr := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == ".golmet" {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil // symlink/junction/socket — o kit não rematerializa não-regulares
		}
		rel, rerr := filepath.Rel(root, p)
		if rerr != nil {
			return rerr
		}
		rel = filepath.ToSlash(rel)
		// o próprio manifest é a fonte da classificação — nunca classificado
		// (defensivo: o default já vive em .golmet/, skipado acima)
		if rel == ".golmet/manifest.json" {
			return nil
		}
		entry, isManaged := managed[rel]
		if !isManaged {
			untracked = append(untracked, rel)
			return nil
		}
		data, readErr := os.ReadFile(p)
		if readErr != nil {
			sec.Unreadable = append(sec.Unreadable, rel)
			return nil
		}
		if managedMatches(data, entry.SHA256) {
			sec.ManagedOK++
		} else {
			sec.Customized = append(sec.Customized, rel)
		}
		return nil
	})
	if werr != nil {
		return sec, werr
	}

	// ausentes em disco: só layer local é perda (core/pack o kit rematerializa)
	for rel, entry := range managed {
		if _, serr := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); serr != nil {
			if os.IsNotExist(serr) && entry.Layer == "local" {
				sec.RemovedLocal = append(sec.RemovedLocal, rel)
			}
		}
	}

	// agregados + caps
	sort.Strings(sec.Customized)
	sort.Strings(sec.RemovedLocal)
	sort.Strings(sec.Unreadable)
	sec.Customized, sec.CustomizedTotal = capList(sec.Customized)
	sec.RemovedLocal, sec.RemovedLocalTotal = capList(sec.RemovedLocal)
	sort.Strings(untracked)
	sec.UntrackedTotal = len(untracked)
	sec.UntrackedDirs = summarizeUntracked(untracked)
	if len(untracked) > untrackedListCap {
		untracked = untracked[:untrackedListCap]
	}
	sec.Untracked = untracked
	sec.GitRepo, sec.GitDirty = gitState(root)
	return sec, nil
}

// capList aplica o cap de listagem e devolve o total original.
func capList(list []string) (capped []string, total int) {
	total = len(list)
	if total > untrackedListCap {
		list = list[:untrackedListCap]
	}
	return list, total
}

// managedMatches compara o disco com o hash do manifest com a MESMA
// tolerância do install: byte-a-byte OU após normalizar resquícios de
// editor (EOL/BOM — content.Normalize) — nada de falso "customização".
func managedMatches(disk []byte, manifestSHA string) bool {
	if sha256Hex(disk) == manifestSHA {
		return true
	}
	return sha256Hex(content.Normalize(disk)) == manifestSHA
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// summarizeUntracked agrupa caminhos por diretório de 1º nível com o
// prefixo de diretório mais profundo compartilhado pelo grupo: plugins/ com
// 3 soltos vira "plugins"; knowledge/ com um único subdir de 12 arquivos
// vira "knowledge/banco-dados-distribuidos" (o caso real da máquina).
func summarizeUntracked(paths []string) []DirCount {
	groups := map[string][]string{} // 1º componente → caminhos
	for _, p := range paths {
		first := p
		if i := strings.IndexByte(p, '/'); i >= 0 {
			first = p[:i]
		}
		groups[first] = append(groups[first], p)
	}
	out := make([]DirCount, 0, len(groups))
	for _, members := range groups {
		dir := commonDirPrefix(members)
		if dir == "" {
			dir = dirRootLabel
		}
		out = append(out, DirCount{Dir: dir, Count: len(members)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Dir < out[j].Dir })
	return out
}

// commonDirPrefix devolve o diretório mais profundo compartilhado por todos
// os caminhos ("" = arquivos soltos na raiz). Componentes de arquivo não
// participam do prefixo.
func commonDirPrefix(paths []string) string {
	dirs := strings.Split(paths[0], "/")[:len(strings.Split(paths[0], "/"))-1]
	for _, p := range paths[1:] {
		parts := strings.Split(p, "/")
		other := parts[:len(parts)-1]
		n := 0
		for n < len(dirs) && n < len(other) && dirs[n] == other[n] {
			n++
		}
		dirs = dirs[:n]
		if len(dirs) == 0 {
			return ""
		}
	}
	return strings.Join(dirs, "/")
}

// gitState — estado git do config root, melhor-esforço: sem .git → repo
// false; git ausente/status falha → skip silencioso (repo true, dirty 0 —
// a contagem é diagnóstico, nunca bloqueia o check).
func gitState(root string) (repo bool, dirty int) {
	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		return false, 0
	}
	repo = true
	out, err := exec.Command("git", "-C", root, "status", "--porcelain").Output()
	if err != nil {
		return repo, 0
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) != "" {
			dirty++
		}
	}
	return repo, dirty
}

// renderLines — bloco textual do Human() logo abaixo da linha do check:
// paths de customização/remoção um por linha, resumo por diretório +
// primeiras N paths não-reconstrutíveis (cap — o resto no resumo).
func (s *EngineConfigSection) renderLines(b *strings.Builder) {
	for _, p := range s.Customized {
		fmt.Fprintf(b, "     customização local: %s\n", p)
	}
	if len(s.UntrackedDirs) > 0 {
		var parts []string
		for _, d := range s.UntrackedDirs {
			if d.Dir == dirRootLabel {
				parts = append(parts, fmt.Sprintf("%s: %d arquivo(s)", dirRootLabel, d.Count))
			} else {
				parts = append(parts, fmt.Sprintf("%s/: %d arquivo(s)", d.Dir, d.Count))
			}
		}
		fmt.Fprintf(b, "     não-reconstrutível pelo kit: %s\n", strings.Join(parts, ", "))
	}
	for _, p := range s.Untracked {
		fmt.Fprintf(b, "       %s\n", p)
	}
	if s.UntrackedTotal > len(s.Untracked) {
		fmt.Fprintf(b, "       … +%d arquivo(s)\n", s.UntrackedTotal-len(s.Untracked))
	}
	for _, p := range s.RemovedLocal {
		fmt.Fprintf(b, "     local removido (dado perdido): %s\n", p)
	}
	if s.CustomizedTotal > len(s.Customized) {
		fmt.Fprintf(b, "     … +%d customização(ões)\n", s.CustomizedTotal-len(s.Customized))
	}
	if s.RemovedLocalTotal > len(s.RemovedLocal) {
		fmt.Fprintf(b, "     … +%d removido(s)\n", s.RemovedLocalTotal-len(s.RemovedLocal))
	}
}
