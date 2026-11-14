// engineconfig.go — check de ESTADO DO ENGINE-CONFIG do doctor. Com perfis
// essencial/completo + packs, o config do engine (~/.config/opencode etc.)
// é CACHE reconstrutível de 3 fontes (kit embarcado, packs SHA-pinados,
// zona do projeto) — EXCETO pelo estado do usuário que o kit não
// reconstrói: customizações reais em arquivos geridos, arquivos locais sem
// proveniência no manifest (knowledge/, plugins/, settings) e mudanças
// não-commitadas quando o config é um repo git. NADA avisava o que seria
// perdido — este check CLASSIFICA por arquivo (read-only — nunca escreve)
// contra o manifest de conteúdo que o install/materialize escreveu.
// Exceção declarada: <root>/.golmet/personal.json designa estado PESSOAL
// POR DESIGN (ex.: rules/client-profile.md e rules/operator-profile.md —
// perfis com dado de cliente/operador gitignored por design IMP-103/136;
// AGENTS.md gerado do template com deltas do operador) — não versionado de
// propósito, não entra no risco nem vira ruído de "não-versionado".
package doctor

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Golmet/golmet-cli/internal/core/content"
	"github.com/Golmet/golmet-cli/internal/core/pack"
)

// untrackedListCap — cap de cada lista de paths na seção/render (o resumo
// por diretório cobre o resto — não inundar terminal nem payload JSON).
const untrackedListCap = 20

// dirRootLabel — rótulo do grupo de arquivos soltos na raiz do config
// (sem diretório de 1º nível).
const dirRootLabel = "(raiz)"

// personalManifestRel — designação pessoal-por-design do usuário
// (<root>/.golmet/personal.json), irmã do manifest de conteúdo.
const personalManifestRel = ".golmet/personal.json"

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
// personal = sem proveniência MAS designado pessoal-por-design (isento —
// o usuário DECIDIU não versionar; o kit não reconstrói, não é risco);
// removed_local = layer local ausente em disco (dado perdido).
type EngineConfigSection struct {
	Root              string     `json:"root"`
	ManagedOK         int        `json:"managed_ok"`              // geridos iguais ao manifest (cache reconstrutível)
	GitVersioned      int        `json:"git_versioned,omitempty"` // sem proveniência de kit, mas versionados pelo git do config (reconstrutíveis — seguro)
	Customized        []string   `json:"customized,omitempty"`    // cap untrackedListCap
	CustomizedTotal   int        `json:"customized_total,omitempty"`
	UntrackedDirs     []DirCount `json:"untracked_dirs,omitempty"` // resumo por diretório de 1º nível
	Untracked         []string   `json:"untracked,omitempty"`      // cap untrackedListCap
	UntrackedTotal    int        `json:"untracked_total,omitempty"`
	Personal          []string   `json:"personal,omitempty"` // cap untrackedListCap
	PersonalTotal     int        `json:"personal_total,omitempty"`
	RemovedLocal      []string   `json:"removed_local,omitempty"` // cap untrackedListCap
	RemovedLocalTotal int        `json:"removed_local_total,omitempty"`
	Unreadable        []string   `json:"unreadable,omitempty"` // ilegíveis — não classificáveis (anomalia)
	GitRepo           bool       `json:"git_repo"`
	GitDirty          int        `json:"git_dirty,omitempty"` // mudanças não-commitadas (melhor-esforço)

	// State packs (arquétipo kind: state): o working copy em packs/<id>/ é
	// um repo Git PRÓPRIO do pack — arquivos versionados lá são seguros
	// (reconstrutíveis pelo git do pack, não pelo manifest do config);
	// apenas mudanças não-commitadas no working copy contam como risco.
	PackVersioned int      `json:"pack_versioned,omitempty"`  // arquivos em working copies de state packs (seguro)
	PackDirty     int      `json:"pack_dirty,omitempty"`      // mudanças não-commitadas em working copies de state packs
	PackDirtyDirs []string `json:"pack_dirty_dirs,omitempty"` // quais packs têm WIP (rel "packs/<id>")
}

// AtRisk devolve o total de estado do usuário que o kit não reconstrói —
// o que uma limpeza do config (ou um --force descuidado) perderia.
// Pessoal-por-design NÃO conta: o usuário decidiu não versionar (IMP-103/136)
// — o kit não reconstrói, mas não é risco nem ruído. Mudanças não-commitadas
// em working copies de state packs contam (o dado ainda não está em commit).
func (s *EngineConfigSection) AtRisk() int {
	return s.CustomizedTotal + s.UntrackedTotal + s.RemovedLocalTotal + s.PackDirty
}

// runEngineConfig executa o check contra a raiz do config do engine e
// registra o check + a seção estruturada no relatório. Root vazio nunca
// chega aqui (Run só chama com root não-vazio — compatibilidade total).
func (r *Report) runEngineConfig(root string) {
	if st, serr := os.Stat(root); serr != nil || !st.IsDir() {
		r.EngineConfig = &EngineConfigSection{Root: root}
		r.add("engine-config", OK, "raiz do config inexistente ("+root+") — nada gerido, nada a perder", "")
		return
	}
	sec, aerr := analyzeEngineConfig(root)
	r.EngineConfig = sec
	switch {
	case aerr != nil:
		r.add("engine-config", Warn, "manifest de conteúdo ilegível em "+root+" ("+aerr.Error()+")",
			"corrija o .golmet/manifest.json do config ou rode o update do kit para reescrevê-lo")
	case sec.AtRisk() == 0 && len(sec.Unreadable) == 0 && sec.ManagedOK == 0 && sec.PersonalTotal == 0:
		if sec.PackVersioned > 0 {
			r.add("engine-config", OK,
				fmt.Sprintf("%d arquivo(s) versionado(s) em working copy de state pack (seguro) — nada a perder", sec.PackVersioned), "")
			return
		}
		r.add("engine-config", OK, "config root vazio (sem arquivos nem manifest) — nada a perder", "")
	case sec.AtRisk() == 0 && len(sec.Unreadable) == 0 && sec.ManagedOK == 0:
		r.add("engine-config", OK,
			fmt.Sprintf("%d pessoal-por-design (isento) — nada gerido pelo kit, nada em risco", sec.PersonalTotal), "")
	case sec.AtRisk() == 0 && len(sec.Unreadable) == 0:
		detail := fmt.Sprintf("cache reconstrutível — %d arquivo(s) gerido(s) conforme(s) ao manifest", sec.ManagedOK)
		if sec.PersonalTotal > 0 {
			detail += fmt.Sprintf(" · %d pessoal-por-design (isento)", sec.PersonalTotal)
		}
		if sec.PackVersioned > 0 {
			detail += fmt.Sprintf(" · %d em working copy de state pack (seguro)", sec.PackVersioned)
		}
		r.add("engine-config", OK, detail, "")
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
	if s.GitVersioned > 0 {
		parts = append(parts, fmt.Sprintf("%d versionado(s) pelo git do config", s.GitVersioned))
	}
	if s.PersonalTotal > 0 {
		parts = append(parts, fmt.Sprintf("%d pessoal-por-design (isento)", s.PersonalTotal))
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
	if s.PackVersioned > 0 {
		parts = append(parts, fmt.Sprintf("%d em working copy de state pack (seguro)", s.PackVersioned))
	}
	if s.PackDirty > 0 {
		parts = append(parts, fmt.Sprintf("%d mudança(s) não-commitada(s) em state pack(s) (%s)",
			s.PackDirty, strings.Join(s.PackDirtyDirs, ", ")))
	}
	if s.GitRepo && s.GitDirty > 0 {
		parts = append(parts, fmt.Sprintf("%d mudança(s) não-commitada(s)", s.GitDirty))
	}
	if len(s.Unreadable) > 0 {
		parts = append(parts, fmt.Sprintf("%d arquivo(s) ilegível(is)", len(s.Unreadable)))
	}
	return strings.Join(parts, " — ")
}

// fixText — remediação estilo dos checks existentes. Config não-versionado
// com estado em risco ganha o versionamento como primeiro passo (o design:
// git init && git add -A && git commit OU mover para zona/pack). Pessoal
// por design NÃO gera fix — é isento por decisão do usuário.
func (s *EngineConfigSection) fixText() string {
	if s.PackDirty > 0 {
		return "commite ou publique o WIP do state pack — 'golmet pack publish <origem>' (commit + push) ou git commit no working copy; o kit nunca descarta acumulação"
	}
	if !s.GitRepo && s.AtRisk() > 0 {
		return fmt.Sprintf("config não versionado — %d arquivo(s) não-reconstrutíveis; versione com 'git init && git add -A && git commit' em %s, ou mova o conteúdo para a zona do workspace (.golmet/) ou para um pack", s.AtRisk(), s.Root)
	}
	if s.AtRisk() > 0 {
		return "revise e commite o estado local — o kit só rematerializa o que está no manifest; sobrescrever divergente exige --force explícito"
	}
	if len(s.Unreadable) > 0 {
		return "verifique as permissões dos arquivos ilegíveis em " + s.Root
	}
	return ""
}

// personalDesignations — designações pessoais-por-design carregadas do
// personal.json: match EXATO de arquivo OU PREFIXO de diretório (path
// terminado em "/", subárvore inteira). Comparação em slash-path.
type personalDesignations struct {
	exact    map[string]bool
	prefixes []string
}

// readPersonalPaths carrega as designações de <root>/.golmet/personal.json.
// Ausente → (nil, nil): sem designações = comportamento de classificação
// atual preservado. Ilegível/corrompido → erro (o caller registra a
// anomalia em Unreadable e segue SEM designações — nunca bloqueia).
func readPersonalPaths(root string) (*personalDesignations, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(personalManifestRel)))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var pf struct {
		Version int      `json:"version"` // 1 — lido e reservado (leitura tolerante)
		Paths   []string `json:"paths"`
	}
	if err := json.Unmarshal(data, &pf); err != nil {
		return nil, err
	}
	pd := &personalDesignations{exact: map[string]bool{}}
	for _, p := range pf.Paths {
		p = strings.TrimSpace(filepath.ToSlash(p))
		if p == "" {
			continue
		}
		if strings.HasSuffix(p, "/") {
			pd.prefixes = append(pd.prefixes, p)
		} else {
			pd.exact[p] = true
		}
	}
	return pd, nil
}

// personalMatch — o path (slash, relativo ao root) casa uma designação
// exata ou um prefixo de diretório? Nil-safe: sem designações → false.
func (pd *personalDesignations) match(rel string) bool {
	if pd == nil {
		return false
	}
	if pd.exact[rel] {
		return true
	}
	for _, prefix := range pd.prefixes {
		if strings.HasPrefix(rel, prefix) {
			return true
		}
	}
	return false
}

// analyzeEngineConfig orquestra a classificação (SRP por função): walk do
// disco, varredura de ausentes, agregação e estado git. Erro só para
// manifest ilegível — falhas por arquivo/pasta ficam na seção (ilegíveis).
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

	// designações pessoal-por-design: personal.json ausente = sem isenção
	// (comportamento atual); corrompido = anomalia registrada, segue sem
	// designações — nunca bloqueia a classificação.
	personal, perr := readPersonalPaths(root)
	if perr != nil {
		sec.Unreadable = append(sec.Unreadable, personalManifestRel)
	}

	var untracked []string
	gitRepo, gitDirty, gitTracked := gitState(root)
	sec.GitRepo = gitRepo
	sec.GitDirty = gitDirty

	if werr := classifyDisk(root, managed, personal, gitRepo, gitTracked, sec, &untracked); werr != nil {
		return sec, werr
	}
	sec.RemovedLocal = removedLocalEntries(root, managed)
	aggregateFindings(sec, untracked)
	return sec, nil
}

// classifyDisk percorre o config root (EXCLUINDO .git/, .golmet/ e o
// próprio manifest de conteúdo; subárvores packs/<id> de state packs são
// classificadas à parte — versionadas no repo do pack) e classifica cada
// arquivo regular contra o manifest: gerido-igual conta, gerido divergente
// é customização (a designação pessoal NÃO isenta gerido — divergência de
// conteúdo gerido nunca é encoberta), sem proveniência vai para untracked
// EXCETO quando casa uma designação pessoal (bucket isento), ilegível fica
// anotado.
func classifyDisk(root string, managed map[string]content.FileEntry, personal *personalDesignations, gitRepo bool, gitTracked map[string]bool, sec *EngineConfigSection, untracked *[]string) error {
	return filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == ".golmet" {
				return filepath.SkipDir
			}
			// state pack working copy: packs/<id> com pack.yaml kind:state —
			// o conteúdo é versionado no repo Git do PRÓPRIO pack (seguro,
			// não entra na classificação do config); só o WIP local conta
			// como risco. Subárvore inteira skipada.
			if rel, rerr := filepath.Rel(root, p); rerr == nil && isStatePackDir(filepath.ToSlash(rel)) {
				packDir := p
				versioned, dirty := scanStateWorkingCopy(packDir)
				sec.PackVersioned += versioned
				sec.PackDirty += dirty
				if dirty > 0 {
					sec.PackDirtyDirs = append(sec.PackDirtyDirs, filepath.ToSlash(rel))
				}
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
		// manifest e personal.json são a FONTE da classificação — nunca
		// classificados (defensivo: o default já vive em .golmet/, skipado)
		if rel == ".golmet/manifest.json" || rel == personalManifestRel {
			return nil
		}
		entry, isManaged := managed[rel]
		if !isManaged {
			if personal.match(rel) {
				sec.Personal = append(sec.Personal, rel)
			} else if gitRepo && gitTracked[rel] {
				sec.GitVersioned++ // versionado pelo git do config — reconstrutível, seguro
			} else {
				*untracked = append(*untracked, rel)
			}
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
}

// isStatePackDir — rel path é um diretório packs/<id> (um único segmento
// sob packs/)? É nesse nível que mora o working copy de um state pack.
func isStatePackDir(rel string) bool {
	i := strings.IndexByte(rel, '/')
	return i > 0 && rel[:i] == "packs" && !strings.Contains(rel[i+1:], "/")
}

// isStateWorkingCopy — dir contém pack.yaml com kind: state? Ilegível/sem
// spec → não é (o diretório cai na classificação padrão).
func isStateWorkingCopy(dir string) bool {
	data, err := os.ReadFile(filepath.Join(dir, "pack.yaml"))
	if err != nil {
		return false
	}
	spec, err := pack.ParseSpec(data)
	return err == nil && spec.Kind == pack.KindState
}

// scanStateWorkingCopy conta, num working copy de state pack, os arquivos
// VERSIONADOS no repo do pack (seguros — não AtRisk) e as mudanças não-
// commitadas (risco). Versionado = git ls-files (arquivo untracked é WIP,
// não versionado); git indisponível → best-effort total de arquivos. O walk
// exclui .git/; o status é melhor-esforço (nunca bloqueia o check).
func scanStateWorkingCopy(dir string) (versioned, dirty int) {
	total := 0
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // ilegível no working copy — contabiliza o que der
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() {
			total++
		}
		return nil
	})
	versioned = total
	if out, err := exec.Command("git", "-C", dir, "-c", "core.quotepath=false", "ls-files").Output(); err == nil {
		versioned = 0
		for _, line := range strings.Split(string(out), "\n") {
			if strings.TrimSpace(line) != "" {
				versioned++
			}
		}
	}
	if out, err := exec.Command("git", "-C", dir, "status", "--porcelain", "--untracked-files=all").Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.TrimSpace(line) != "" {
				dirty++
			}
		}
	}
	return versioned, dirty
}

// removedLocalEntries devolve os caminhos registrados com layer local que
// sumiram do disco — dado perdido (core/pack ausentes o kit rematerializa,
// nunca reportados).
func removedLocalEntries(root string, managed map[string]content.FileEntry) []string {
	var out []string
	for rel, entry := range managed {
		if _, serr := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); serr != nil {
			if os.IsNotExist(serr) && entry.Layer == "local" {
				out = append(out, rel)
			}
		}
	}
	return out
}

// aggregateFindings ordena as listas, aplica o cap de listagem e monta o
// resumo por diretório de 1º nível dos não-reconstrutíveis (pessoal-por-
// design não entra no resumo — é isento, não ruído).
func aggregateFindings(sec *EngineConfigSection, untracked []string) {
	sort.Strings(sec.Customized)
	sort.Strings(sec.Personal)
	sort.Strings(sec.RemovedLocal)
	sort.Strings(sec.Unreadable)
	sec.Customized, sec.CustomizedTotal = capList(sec.Customized)
	sec.Personal, sec.PersonalTotal = capList(sec.Personal)
	sec.RemovedLocal, sec.RemovedLocalTotal = capList(sec.RemovedLocal)
	sort.Strings(untracked)
	sec.UntrackedTotal = len(untracked)
	sec.UntrackedDirs = summarizeUntracked(untracked)
	if len(untracked) > untrackedListCap {
		untracked = untracked[:untrackedListCap]
	}
	sec.Untracked = untracked
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
	groups := map[string][]string{} // 1º componente → caminhos (arquivos soltos na raiz compartilham o grupo)
	for _, p := range paths {
		key := dirRootLabel
		if i := strings.IndexByte(p, '/'); i >= 0 {
			key = p[:i]
		}
		groups[key] = append(groups[key], p)
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
	first := strings.Split(paths[0], "/")
	dirs := first[:len(first)-1]
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
func gitState(root string) (repo bool, dirty int, tracked map[string]bool) {
	tracked = map[string]bool{}
	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		return false, 0, tracked
	}
	repo = true
	if out, err := exec.Command("git", "-C", root, "-c", "core.quotepath=false", "ls-files").Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if p := strings.TrimSpace(line); p != "" {
				tracked[filepath.ToSlash(p)] = true
			}
		}
	}
	out, err := exec.Command("git", "-C", root, "status", "--porcelain").Output()
	if err != nil {
		return repo, 0, tracked
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) != "" {
			dirty++
		}
	}
	return repo, dirty, tracked
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
	for _, p := range s.Personal {
		fmt.Fprintf(b, "     pessoal-por-design (isento): %s\n", p)
	}
	if s.PackVersioned > 0 {
		fmt.Fprintf(b, "     versionado no repo do pack (working copy): %d arquivo(s) — seguro\n", s.PackVersioned)
	}
	for _, d := range s.PackDirtyDirs {
		fmt.Fprintf(b, "     state pack com mudanças não-commitadas: %s\n", d)
	}
	if s.PersonalTotal > len(s.Personal) {
		fmt.Fprintf(b, "     … +%d pessoal-por-design\n", s.PersonalTotal-len(s.Personal))
	}
	if s.CustomizedTotal > len(s.Customized) {
		fmt.Fprintf(b, "     … +%d customização(ões)\n", s.CustomizedTotal-len(s.Customized))
	}
	if s.RemovedLocalTotal > len(s.RemovedLocal) {
		fmt.Fprintf(b, "     … +%d removido(s)\n", s.RemovedLocalTotal-len(s.RemovedLocal))
	}
}
