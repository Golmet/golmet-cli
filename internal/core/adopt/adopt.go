// Package adopt implementa `golmet adopt` — o papel do KIT no handshake
// de 7 passos da skill gm-adopt: (passo 1) dry-run com relatório de
// conformidade em leitura PURA, classificado em conforme / julgamento
// (gate humano) / kit-resolve (apply); (passo 4) apply = transform
// determinístico das escolhas DECLARADAS — cria golmet.yaml + índice +
// scaffold mínimo, repara o front matter do charter; (passo 5) validate
// tolerant interno com resumo no relatório (erros NÃO falham a adoção —
// são a worklist do passo 6). O kit NUNCA edita prosa além do prefixo de
// front matter do charter e NUNCA escolhe hash/estado sozinho — re-pin e
// desambiguação são julgamento do agente (passos 2/6).
package adopt

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Golmet/golmet-cli/internal/adapters/fsutil"
	"github.com/Golmet/golmet-cli/internal/adapters/git"
	"github.com/Golmet/golmet-cli/internal/core"
	"github.com/Golmet/golmet-cli/internal/core/entities"
	"github.com/Golmet/golmet-cli/internal/core/evidence"
	"github.com/Golmet/golmet-cli/internal/core/frontmatter"
	"github.com/Golmet/golmet-cli/internal/core/scaffold"
	"github.com/Golmet/golmet-cli/internal/core/scan"
)

// Seções do relatório (passo 1 da skill: conforme / julgamento /
// kit-resolve; worklist = validate pós-apply, passo 6).
const (
	SectionConforme   = "conforme"
	SectionJulgment   = "julgamento"
	SectionKitResolve = "kit_resolve"
	SectionWorklist   = "worklist"
)

// Finding é um item do relatório com remediação §11-style.
type Finding struct {
	Item   string `json:"item"`
	Detail string `json:"detail,omitempty"`
	Fix    string `json:"fix,omitempty"`
	Rule   string `json:"rule,omitempty"`
}

// ValidateSummary é o resumo do validate tolerant interno (passo 5).
type ValidateSummary struct {
	Errors   []core.GolmetError `json:"errors"`
	Warnings []core.GolmetError `json:"warnings"`
	Counts   map[string]int     `json:"counts"`
}

// Report é o resultado completo do adopt (dry-run ou apply).
type Report struct {
	Workspace         string           `json:"workspace"`
	Mode              string           `json:"mode"` // dry-run | apply
	CharterStatus     string           `json:"charter_status"`
	KeepLegacyLayouts bool             `json:"keep_legacy_layouts"`
	Conforme          []Finding        `json:"conforme,omitempty"`
	Julgamento        []Finding        `json:"julgamento,omitempty"`
	KitResolve        []Finding        `json:"kit_resolve,omitempty"`
	Worklist          []Finding        `json:"worklist,omitempty"`
	Applied           []string         `json:"applied,omitempty"`
	Skipped           []string         `json:"skipped,omitempty"`
	Preserved         []string         `json:"preserved,omitempty"`     // layouts mantidos por escolha (DR)
	IndexWritten      bool             `json:"index_written,omitempty"` // índice (re)escrito pelo kit neste run
	Validate          *ValidateSummary `json:"validate,omitempty"`
}

// Options controla o comando. Apply=false = dry-run (safe default).
type Options struct {
	Apply             bool
	CharterStatus     string // draft (default) | confirmed (§7.1)
	KeepLegacyLayouts bool
}

// ValidateOptions valida as opções (erro = uso inválido, exit 2).
func (o *Options) Validate() error {
	if o.CharterStatus == "" {
		o.CharterStatus = "draft"
	}
	if o.CharterStatus != "draft" && o.CharterStatus != "confirmed" {
		return fmt.Errorf("charter-status %q inválido — use draft ou confirmed (§7.1)", o.CharterStatus)
	}
	return nil
}

// inventário pré-formato — espelha o parse do scan sem EXIGIR golmet.yaml
// (o caso típico do adopt é um workspace que ainda não é FORMAT).
type inventory struct {
	root     string
	files    []fsutil.FileDoc
	charter  *entities.Charter
	rns      []entities.Section
	pends    []entities.Section
	crs      []entities.Section
	adrs     []entities.Section
	cands    []entities.Section
	features []entities.Feature
	specs    []entities.Spec
	evidence []evidence.Finding
	hasYaml  bool
}

func loadInventory(root string) (*inventory, error) {
	files, err := fsutil.Walk(root)
	if err != nil {
		return nil, fmt.Errorf("falha ao caminhar o workspace: %w", err)
	}
	inv := &inventory{root: root, files: files, hasYaml: fileExists(filepath.Join(root, "golmet.yaml"))}
	byRel := map[string]*fsutil.FileDoc{}
	for i := range files {
		byRel[files[i].Rel] = &files[i]
	}
	if d := byRel["PROJECT_CHARTER.md"]; d != nil {
		inv.charter = entities.ParseCharter(d, root)
	}
	parse := func(rel string, want func(id string, lvl int) bool) []entities.Section {
		d := byRel[rel]
		if d == nil {
			return nil
		}
		return entities.ParseSections(d.Rel, d.Plain, d.Lines, want)
	}
	inv.rns = parse("REQUIREMENTS.md", func(id string, lvl int) bool {
		return lvl == 2 && strings.HasPrefix(id, "RN-") && !entities.IsTaskID(id)
	})
	inv.pends = parse("REQUIREMENTS.md", func(id string, lvl int) bool {
		return lvl == 2 && strings.HasPrefix(id, "PEND-")
	})
	inv.crs = parse("CHANGE_REQUESTS.md", func(id string, lvl int) bool {
		return lvl == 2 && strings.HasPrefix(id, "CR-")
	})
	inv.adrs = parse("DECISIONS.md", func(id string, lvl int) bool {
		return lvl == 2 && strings.HasPrefix(id, "ADR-")
	})
	inv.cands = parse("reverse/CANDIDATES.md", func(id string, lvl int) bool {
		return lvl == 3 && strings.HasPrefix(id, "CAND-")
	})
	for i := range files {
		d := &files[i]
		if entities.IsFeatureFile(d.Rel) {
			inv.features = append(inv.features, *entities.ParseFeature(d))
		}
		if strings.HasPrefix(d.Rel, "analysis/") {
			if sp := entities.ParseSpec(d); sp.ID != "" {
				inv.specs = append(inv.specs, *sp)
			}
		}
		inv.evidence = append(inv.evidence, evidence.Scan(d.Rel, d.Plain)...)
	}
	return inv, nil
}

// Run executa o adopt: dry-run (leitura pura) ou apply (transform
// determinístico). Erro = uso/config (CLI exit 2); findings de validate
// NÃO são erro.
func Run(dir string, opt Options) (*Report, error) {
	if err := opt.Validate(); err != nil {
		return nil, err
	}
	root, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("workspace inválido: %w", err)
	}
	if st, serr := os.Stat(root); serr != nil || !st.IsDir() {
		return nil, fmt.Errorf("workspace não encontrado: %s", root)
	}
	inv, err := loadInventory(root)
	if err != nil {
		return nil, err
	}

	mode := "dry-run"
	if opt.Apply {
		mode = "apply"
	}
	rep := &Report{
		Workspace:         filepath.Base(root),
		Mode:              mode,
		CharterStatus:     opt.CharterStatus,
		KeepLegacyLayouts: opt.KeepLegacyLayouts,
	}

	rep.classify(inv, opt)

	if !opt.Apply {
		return rep, nil
	}

	// passo 4 — transform determinístico das escolhas DECLARADAS
	// 1. reparo mecânico do front matter do charter (nunca toca o corpo)
	if action, err := repairCharterFrontMatter(root, opt.CharterStatus); err != nil {
		return nil, err
	} else if action != "" {
		rep.Applied = append(rep.Applied, action)
	}
	// 2. scaffold mínimo / ensure idempotente do §2 (reuse integral do init)
	srep, err := scaffold.Run(root, "", scaffold.Options{CharterStatus: opt.CharterStatus})
	if err != nil {
		return nil, fmt.Errorf("scaffold da adoção falhou: %w", err)
	}
	for _, f := range srep.Files {
		if f.Skipped {
			rep.Skipped = append(rep.Skipped, f.Path)
			continue
		}
		rep.Applied = append(rep.Applied, f.Path)
	}
	// 3. layouts legados: normalização física não existe no v0 — o validate
	//    aceita ambas as formas; sem a flag fica pendente de decisão (gate)
	if opt.KeepLegacyLayouts {
		rep.Preserved = append(rep.Preserved, legacyLayoutNames(inv)...)
	}

	// passos 5+6 — validate tolerant interno + índice (D2: o kit é o único
	// escritor; ForceIndex recompõe ignorando divergências)
	res, serr := scan.Run(root, scan.Options{Tolerant: true, WriteIndex: true, ForceIndex: true})
	if serr != nil {
		return nil, fmt.Errorf("validate pós-apply falhou (config): %w", serr)
	}
	vs := &ValidateSummary{
		Errors:   res.Errors,
		Warnings: res.Warnings,
		Counts:   res.Counts,
	}
	for _, e := range res.Errors {
		rep.Worklist = append(rep.Worklist, Finding{
			Item:   e.Entity + " " + e.ID + " em " + e.File + ":" + fmt.Sprint(e.Line),
			Detail: e.Message,
			Fix:    e.Fix,
			Rule:   e.Rule,
		})
	}
	for _, w := range res.Warnings {
		if w.Rule == "S2" || w.Rule == "H4" || w.Rule == "H7" {
			rep.Worklist = append(rep.Worklist, Finding{
				Item:   w.Entity + " " + w.ID + " em " + w.File + ":" + fmt.Sprint(w.Line),
				Detail: w.Message,
				Fix:    w.Fix,
				Rule:   w.Rule,
			})
		}
	}
	rep.IndexWritten = res.IndexWritten
	rep.Validate = vs
	rep.Applied = append(rep.Applied, "golmet.index.json (recomposto pelo kit)")
	return rep, nil
}

// classify produz o relatório de conformidade do passo 1 — leitura pura.
func (rep *Report) classify(inv *inventory, opt Options) {
	// charter (§7.1)
	switch {
	case inv.charter == nil:
		rep.KitResolve = append(rep.KitResolve, Finding{
			Item:   "criar PROJECT_CHARTER.md",
			Detail: "charter ausente — o apply cria o scaffold mínimo com status " + opt.CharterStatus,
			Fix:    "rode o apply (charter nasce draft; confirmed exige gate §7.1)",
		})
	case inv.charter.FMError != nil:
		rep.Julgamento = append(rep.Julgamento, Finding{
			Item:   "front matter do charter ilegível",
			Detail: inv.charter.FMError.Error(),
			Fix:    "corrija o front matter conforme o subset §6.1 (julgamento — o kit não altera prosa ilegível)",
			Rule:   "H5",
		})
	case !inv.charter.HasFM:
		rep.KitResolve = append(rep.KitResolve, Finding{
			Item:   "prefixar front matter do charter",
			Detail: "sem bloco --- … --- — o apply prefixa 'status: " + opt.CharterStatus + "' sem tocar o corpo",
		})
	case valueOf(inv.charter.FM, "status") == "":
		rep.KitResolve = append(rep.KitResolve, Finding{
			Item:   "adicionar status ao front matter do charter",
			Detail: "front matter presente sem status — o apply insere 'status: " + opt.CharterStatus + "'",
		})
	default:
		st := valueOf(inv.charter.FM, "status")
		if st == "draft" || st == "confirmed" {
			rep.Conforme = append(rep.Conforme, Finding{
				Item:   "charter com status: " + st,
				Detail: mapBool(st == "draft", "transição para confirmed é gate humano (§7.1: §1 com repos+HEAD congelado)", "§1 deve registrar cada repo com HEAD congelado"),
			})
		} else {
			rep.Julgamento = append(rep.Julgamento, Finding{
				Item: "status do charter fora do vocabulário: " + st,
				Fix:  "use draft|confirmed (§7.1) — julgamento, o kit não altera status existente",
				Rule: "H6",
			})
		}
	}

	// estrutura FORMAT §2 faltante — kit resolve no apply
	for _, m := range missingSection2(inv.root) {
		rep.KitResolve = append(rep.KitResolve, Finding{
			Item:   "criar " + m,
			Detail: "estrutura §2 ausente — o apply cria (existente não-vazio é preservado)",
		})
	}

	// evidências sem hash — re-pin é JULGAMENTO do agente (kit nunca escolhe hash)
	for _, f := range inv.evidence {
		if f.Kind != evidence.KindLegacy {
			continue
		}
		rep.Julgamento = append(rep.Julgamento, Finding{
			Item:   fmt.Sprintf("evidência sem hash [%s:%d] em %s:%d", f.Ev.Path, f.Ev.Line, f.File, f.Line),
			Detail: "re-pin é julgamento do agente — qual linha sustenta a RN?",
			Fix:    "substitua por [caminho:linha @hash] com o hash atual do repo",
			Rule:   "S4",
		})
	}

	// RNs active sem evidência parseável — H4 na mira
	for _, rn := range inv.rns {
		if entities.StateOf(rn, entities.Default("RN")) != "active" {
			continue
		}
		has := false
		for _, f := range evidence.Scan(rn.File, rn.Lines) {
			if f.Kind == evidence.KindCanonical {
				has = true
				break
			}
		}
		if has {
			continue
		}
		rep.Julgamento = append(rep.Julgamento, Finding{
			Item:   rn.ID + " active sem evidência parseável (" + rn.File + ":" + fmt.Sprint(rn.Line) + ")",
			Detail: "decida: re-pin da evidência ou rebaixar para draft (gate)",
			Fix:    "adicione [caminho:linha @hash] ao corpo da seção",
			Rule:   "H4",
		})
	}

	// IDs duplicados — desambiguação é do agente (H2)
	dups := duplicateIDs(inv)
	for _, d := range dups {
		rep.Julgamento = append(rep.Julgamento, Finding{
			Item:   "ID duplicado: " + d.id + " (" + strings.Join(d.files, ", ") + ")",
			Detail: "IDs são únicos e imutáveis — separe os conteúdos e cite a proveniência",
			Fix:    "desambigue manualmente antes do apply",
			Rule:   "H2",
		})
	}

	// evidência com hash fora do charter §1 (H7 na mira) — requer git + repos
	rep.classifyEvidenceRepos(inv)

	// variantes legadas de layout
	for _, l := range legacyLayouts(inv.root, inv.files) {
		if opt.KeepLegacyLayouts {
			rep.Conforme = append(rep.Conforme, Finding{
				Item:   "layout legado mantido por escolha: " + l.desc,
				Detail: l.fix,
			})
		} else {
			rep.Julgamento = append(rep.Julgamento, Finding{
				Item:   "variante legada de layout: " + l.desc,
				Detail: "o validate aceita ambas as formas e recomenda a normalização",
				Fix:    l.fix + " — ou declare --keep-legacy-layouts",
			})
		}
	}
}

// classifyEvidenceRepos verifica H7 na mira: hash que não resolve em
// exatamente 1 repo do charter §1 (requer git; sem git, adiada com nota).
func (rep *Report) classifyEvidenceRepos(inv *inventory) {
	g := git.New()
	if !g.Available() {
		if len(inv.evidence) > 0 {
			rep.Worklist = append(rep.Worklist, Finding{
				Item:   "verificação H7 adiada",
				Detail: "git indisponível no dry-run — hashes não foram resolvidos contra os repos",
			})
		}
		return
	}
	if inv.charter == nil || len(inv.charter.Repos) == 0 {
		return // sem charter/repos, H7 não é aplicável ainda (finding de charter cobre)
	}
	for _, f := range inv.evidence {
		if f.Kind != evidence.KindCanonical {
			continue
		}
		n := 0
		seen := map[string]bool{}
		for _, rp := range inv.charter.Repos {
			if seen[rp.Abs] {
				continue
			}
			seen[rp.Abs] = true
			if g.HasObject(rp.Abs, f.Ev.Hash) {
				n++
			}
		}
		if n == 1 {
			continue
		}
		detail := "hash não resolve em nenhum repo do charter §1 — corrija o charter (gate) ou a evidência"
		if n > 1 {
			detail = "hash ambíguo — resolve em mais de um repo do charter §1"
		}
		rep.Julgamento = append(rep.Julgamento, Finding{
			Item:   fmt.Sprintf("evidência [%s:%d @%s] em %s:%d", f.Ev.Path, f.Ev.Line, f.Ev.Hash, f.File, f.Line),
			Detail: detail,
			Fix:    "registre o repo no charter §1 ou re-pine o hash correto",
			Rule:   "H7",
		})
	}
}

// legacyLayout detecta as variantes de layout pré-FORMAT (heurística do
// dry-run — cada uma com sugestão de normalização).
type legacyLayout struct {
	desc, fix string
}

func legacyLayouts(root string, files []fsutil.FileDoc) []legacyLayout {
	var out []legacyLayout
	seen := map[string]bool{}
	add := func(desc, fix string) {
		if !seen[desc] {
			seen[desc] = true
			out = append(out, legacyLayout{desc: desc, fix: fix})
		}
	}
	// 1. feature em pasta própria: delivery/<grupo>/<RN-NNN …>/ com ≥2
	//    arquivos cujo nome começa com RN-NNN (análise + feature juntas)
	perDir := map[string][]string{}
	for _, d := range files {
		if !strings.HasPrefix(d.Rel, "delivery/") {
			continue
		}
		base := filepath.Base(d.Rel)
		if !rxFileRN.MatchString(base) {
			continue
		}
		dir := path.Dir(d.Rel) // separador '/' — convenção de rel do kit
		perDir[dir] = append(perDir[dir], base)
	}
	var dirs []string
	for dir := range perDir {
		if dir == "delivery" {
			continue // flat na raiz — variante distinta (item 4)
		}
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	for _, dir := range dirs {
		fs2 := perDir[dir]
		if len(fs2) >= 2 {
			add(fmt.Sprintf("pasta própria %s com %d arquivo(s) RN-* (feature dentro de pasta — variante §2)",
				dir, len(fs2)),
				"mova para "+dir+"/RN-NNN — <título> — Feature.md")
		}
	}
	// 2. estrutura docs/projects/
	if dirExists(filepath.Join(root, "docs", "projects")) {
		add("estrutura docs/projects/ (layout pré-FORMAT)",
			"mover os docs para o layout §2 (charter na raiz do workspace, analysis/, delivery/, specs/)")
	}
	// 3. TASKS*.md / TASKS.md em qualquer lugar
	for _, d := range files {
		if rxTasks.MatchString(filepath.Base(d.Rel)) {
			add("arquivo de tasks legado: "+d.Rel,
				"converter para delivery/<grupo>/RN-NNN — <título> — Feature.md (tasks como seções RN-NNN-M)")
		}
	}
	// 4. RN-*.md flat na raiz do delivery (sem grupo)
	for _, d := range files {
		if filepath.Dir(d.Rel) == "delivery" && rxFileRN.MatchString(filepath.Base(d.Rel)) {
			add("RN-* na raiz do delivery (sem grupo): "+d.Rel,
				"mova para delivery/<grupo>/"+filepath.Base(d.Rel))
		}
	}
	return out
}

// legacyLayoutNames resume as variantes detectadas (para o Preserved do apply).
func legacyLayoutNames(inv *inventory) []string {
	var out []string
	for _, l := range legacyLayouts(inv.root, inv.files) {
		out = append(out, l.desc)
	}
	return out
}

// duplicateIDs lista IDs repetidos em qualquer namespace (H2 na mira) —
// desambiguação é do agente; o kit só reporta.
type dupID struct {
	id    string
	files []string
}

func duplicateIDs(inv *inventory) []dupID {
	type occurrence struct {
		file string
	}
	byID := map[string][]occurrence{}
	register := func(secs []entities.Section) {
		for _, s := range secs {
			byID[s.ID] = append(byID[s.ID], occurrence{file: s.File})
		}
	}
	register(inv.rns)
	register(inv.pends)
	register(inv.crs)
	register(inv.adrs)
	register(inv.cands)
	var out []dupID
	for id, occs := range byID {
		if len(occs) < 2 {
			continue
		}
		seen := map[string]bool{}
		var fs2 []string
		for _, o := range occs {
			if !seen[o.file] {
				seen[o.file] = true
				fs2 = append(fs2, o.file)
			}
		}
		sort.Strings(fs2)
		out = append(out, dupID{id: id, files: fs2})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].id < out[j].id })
	return out
}

// repairCharterFrontMatter aplica o transform mecânico do item 3: prefixa
// o bloco --- … --- ou insere a chave status — NUNCA altera o corpo.
// Retorna a ação aplicada ("" = nada a fazer).
func repairCharterFrontMatter(root, status string) (string, error) {
	path := filepath.Join(root, "PROJECT_CHARTER.md")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // scaffold cria
		}
		return "", fmt.Errorf("charter ilegível: %w", err)
	}
	doc, has, perr := frontmatter.Extract(string(data))
	if perr != nil {
		// prosa ilegível não é mexida pelo kit (julgamento — reportada no
		// classify); aqui apenas não reparamos
		return "", nil
	}
	eol := "\n"
	if strings.Contains(string(data), "\r\n") {
		eol = "\r\n"
	}
	if !has {
		prefixed := "---" + eol + "status: " + status + eol + "---" + eol + string(data)
		if err := os.WriteFile(path, []byte(prefixed), 0o644); err != nil {
			return "", fmt.Errorf("falha ao prefixar o front matter do charter: %w", err)
		}
		return "PROJECT_CHARTER.md (front matter prefixado: status " + status + ")", nil
	}
	if _, ok := doc.Fields["status"]; ok {
		return "", nil // status presente — nada a fazer (transição é gate §7.1)
	}
	lines := strings.Split(string(data), eol)
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", nil // defensivo — Extract já validou
	}
	out := []string{lines[0], "status: " + status}
	out = append(out, lines[1:]...)
	if err := os.WriteFile(path, []byte(strings.Join(out, eol)), 0o644); err != nil {
		return "", fmt.Errorf("falha ao inserir status no charter: %w", err)
	}
	return "PROJECT_CHARTER.md (status inserido no front matter: " + status + ")", nil
}

// missingSection2 devolve os itens do layout §2 ausentes no workspace.
func missingSection2(root string) []string {
	var out []string
	if !fileExists(filepath.Join(root, "golmet.yaml")) {
		out = append(out, "golmet.yaml")
	}
	if !dirExists(filepath.Join(root, "analysis")) {
		out = append(out, "analysis/")
	}
	if !dirExists(filepath.Join(root, "delivery")) {
		out = append(out, "delivery/")
	}
	if !fileExists(filepath.Join(root, "reverse", "CANDIDATES.md")) {
		out = append(out, "reverse/CANDIDATES.md")
	}
	if !dirExists(filepath.Join(root, "specs")) {
		out = append(out, "specs/")
	}
	if !dirExists(filepath.Join(root, ".golmet")) {
		out = append(out, ".golmet/")
	}
	return out
}

// Human renderiza o relatório no padrão do kit.
func (r *Report) Human() string {
	var b strings.Builder
	mode := r.Mode
	if mode == "dry-run" {
		mode += " (leitura pura — zero escrita)"
	}
	fmt.Fprintf(&b, "workspace: %s — adoção do FORMAT v1 [%s]\n", r.Workspace, mode)

	section := func(title string, items []Finding) {
		if len(items) == 0 {
			return
		}
		fmt.Fprintf(&b, "\n%s (%d):\n", title, len(items))
		for _, f := range items {
			fmt.Fprintf(&b, "  - %s\n", f.Item)
			if f.Detail != "" {
				fmt.Fprintf(&b, "      %s\n", f.Detail)
			}
			if f.Fix != "" {
				fmt.Fprintf(&b, "      fix: %s\n", f.Fix)
			}
			if f.Rule != "" {
				fmt.Fprintf(&b, "      regra: %s\n", f.Rule)
			}
		}
	}
	section("conforme", r.Conforme)
	section("precisa de julgamento (gate humano — o kit nunca decide)", r.Julgamento)
	section("kit resolve no apply", r.KitResolve)
	section("worklist do validate tolerant (passo 6)", r.Worklist)

	if r.Mode == "apply" {
		fmt.Fprintf(&b, "\nadoção aplicada — %d escrito(s), %d preservado(s) (existente não-vazio)\n",
			len(r.Applied), len(r.Skipped))
		for _, p := range r.Preserved {
			fmt.Fprintf(&b, "  layout legado mantido por escolha: %s\n", p)
		}
		if r.Validate != nil {
			fmt.Fprintf(&b, "validate tolerant: %d erro(s), %d aviso(s) — worklist acima; strict entra no CI só no gate de conclusão (passo 7)\n",
				len(r.Validate.Errors), len(r.Validate.Warnings))
		}
	} else if len(r.KitResolve) > 0 {
		fmt.Fprintf(&b, "\npróximo passo: revise os gates e rode 'golmet adopt --apply --charter-status %s'\n", r.CharterStatus)
	}
	return b.String()
}

// ---- helpers ----

func valueOf(fm map[string]frontmatter.Value, key string) string {
	if fm == nil {
		return ""
	}
	return fm[key].String()
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func dirExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

func mapBool(cond bool, ifTrue, ifFalse string) string {
	if cond {
		return ifTrue
	}
	return ifFalse
}

var (
	rxFileRN = regexp.MustCompile(`^RN-\d{1,4}`)
	rxTasks  = regexp.MustCompile(`^TASKS.*\.md$`)
)
