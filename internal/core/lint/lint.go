// Package lint implementa `golmet lint` — varredura estrutural
// determinística do kit/config/packs: referências mortas (L1), órfãos (L3),
// manifest stale (L4), nomes fora do padrão (L5), componentes fantasmas
// (L6) e estrutura de pastas canônica (L7). Read-only — nunca muta. A IA
// corrige o que o lint aponta: o lint carrega o estado todo, o julgamento
// fica com o agent/operador. Sem check de vocabulário legado (L2 ficou
// fora por decisão do operador).
package lint

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	golmet "github.com/Golmet/golmet-cli"
	"github.com/Golmet/golmet-cli/internal/core/content"
	"github.com/Golmet/golmet-cli/internal/core/doctor"
	"github.com/Golmet/golmet-cli/internal/core/pack"
)

// Finding é um apontamento do lint: severidade, arquivo (+linha quando
// aplicável), descrição e remediação sugerida (PT-BR).
type Finding struct {
	Check       string `json:"check"`          // L1 | L3 | L4 | L5 | L6 | L7
	Severity    string `json:"severity"`       // error | warning
	File        string `json:"file,omitempty"` // rel ao alvo do check (slash)
	Line        int    `json:"line,omitempty"` // 0 = arquivo inteiro
	Description string `json:"description"`
	Fix         string `json:"fix"` // remediação sugerida (PT-BR)
}

// Report do lint — o CLI imprime Human(); o MCP devolve o struct.
type Report struct {
	Workspace  string         `json:"workspace"`
	ConfigRoot string         `json:"config_root,omitempty"`
	KitMode    bool           `json:"kit_mode"`
	Findings   []Finding      `json:"findings"`
	Errors     int            `json:"errors"`
	Warnings   int            `json:"warnings"`
	ByCheck    map[string]int `json:"by_check"`
}

// ---- padrões canônicos DECLARADOS (contrato da estrutura — fonte única
// em código; documentado no FORMATO §9.1) ----

// packStateRootDirs — dirs canônicos na raiz de um state pack (working
// copy): pack.yaml + qualquer subconjunto destes (+ meta .git/README/LICENSE).
var packStateRootDirs = []string{"skills", "agents", "rules", "domains", "accumulation"}

// accumulationOnly — único conteúdo canônico de accumulation/.
var accumulationOnly = []string{"patterns", "metrics", "nuances"}

// patternTypes — tipos canônicos de PATTERN.md (tabela do INDEX da pattern
// library). Tipo fora do conjunto = estrutura não canônica.
var patternTypes = []string{"process", "architecture", "code", "ui", "test", "data"}

// configRootDirs — dirs canônicos de 1º nível do config do engine.
// knowledge/ é legado reconhecido. Unknown dir → warning.
var configRootDirs = []string{"agents", "skills", "rules", "packs", "plugins", "knowledge", ".golmet"}

// orphanAllowlist — prefixes de path (slash) sabidamente SEM dono no config
// do engine (órfãos-legítimos: artefatos de runtime, plugins de terceiros,
// settings raiz já cobertos pelo design). Declarativa e versionada aqui —
// ampliar com julgamento, nunca silenciosamente.
var orphanAllowlist = []string{"plugins/", "quality/health", ".github/", "cli.json", "service.json", "opencode.jsonc", "AGENTS.md", ".gitignore"}

// ghostAllowlist — nomes gm-* legados/removidos/exemplificativos que ainda
// constam em prosa, com o motivo. Declarativa e versionada aqui — ampliar
// com julgamento, nunca silenciosamente.
var ghostAllowlist = map[string]string{
	"gm-pack-accumulation": "bloco <!-- gm-pack-accumulation --> do AGENTS.md (marcador de acúmulo) — não é componente",
	"gm-scan":              "renomeada — a varredura hoje é `golmet validate` / gm_validate (histórico de prosa)",
	"gm-stack-react":       "playbook React não criado — o frontend da casa é Angular (gm-stack-angular)",
	"gm-pia-connect-setup": "skill opcional de pack (IMP-105) — só existe quando o pack a fornece",
	"gm-coverage-audit":    "avaliada e não criada — cobertura é do gm-dev-qa/playbook da stack",
	"gm-a11y":              "exemplo ilustrativo do FORMATO §6",
	"gm-a11y-reviewer":     "exemplo ilustrativo do FORMATO §6",
	"gm-xxx":               "exemplo ilustrativo (gm-registry-render)",
	"gm-yyy":               "exemplo ilustrativo (gm-registry-render)",
	"gm-zzz":               "exemplo ilustrativo (gm-registry-render)",
	"gm-orchestrator":      "exemplo ilustrativo (gm-registry-render)",
	"gm-s1":                "referência de seção S1 do FORMATO, não componente",
	"gm-docs":              "exemplo de nome de REPO_DOCS no gm-init (`minha-org/gm-docs`) — não é componente",
	"gm-x":                 "exemplo de id no front matter do FORMATO §10",
}

// refAllowlist — paths citados sabidamente mortos, com motivo (pendências
// declaradas). Declarativa e versionada — o L1 não apina o que está aqui.
var refAllowlist = map[string]string{
	"AGENTS.template.md": "pendência declarada da wave do template AGENTS — o render ainda não existe no kit",
}

// moduleStoreNames — nomes de artefato de WORKSPACE DE PRODUTO citados por
// design nas skills (o destino existe no workspace do operador, não no
// kit) — o L1 de kit não verifica. Declarativa e versionada.
var moduleStoreNames = map[string]bool{
	"REQUIREMENTS.md": true, "TASKS.md": true, "DECISIONS.md": true,
	"PROJECT_CHARTER.md": true, "ARCHITECTURE.md": true, "DATA_MODEL.md": true,
	"ACCEPTANCE_CRITERIA.md": true, "API_CONTRACTS.md": true, "ERROR_CATALOG.md": true,
	"UI_SPEC.md": true, "SCOPE_AGREEMENT.md": true, "CLIENT_REQUIREMENTS.md": true,
	"CHANGE_REQUESTS.md": true, "TRACEABILITY.md": true, "PROPAGATION_STATUS.md": true,
	"VALIDATION_REPORT.md": true, "KNOWN_ERRORS.md": true, "HANDOVER_LOG.md": true,
	"RETROSPECTIVE.md": true, "METRICS.md": true, "INDEX.md": true, "PATTERN.md": true,
	"KNOWLEDGE.md": true, "DESIGN_BRIEF.md": true, "COMPLIANCE.md": true,
	"TEST_STRATEGY.md": true, "DEFINITION_OF_DONE.md": true, "WORKFLOW_IMPROVEMENTS.md": true,
	"MIGRATION.md": true, "CANDIDATES.md": true, "REVERSE_STATE.md": true,
	"COVERAGE.md": true, "PARITY_CHECKLIST.md": true, "MIGRATION_PLAN.md": true,
	"DATA_MAP.md": true, "DB_DRIFT.md": true, "DB_PERFORMANCE.md": true,
	"golmet.index.json": true, "FIGMA_PROMPTS.md": true,
	// stores de workspace/pack/release citadas como contrato por design
	"golmet.yaml": true, "pack.yaml": true, "criteria.yaml": true,
	".mcp.json": true, "checksums.txt": true, "CONSTITUTION.md": true,
	"opencode.json": true, "deploy.yml": true, "deploy.sh": true,
	"daemon.json": true, "credentials.json": true, "config.yml": true,
	"SKILL.md": true, "pack-lock.json": true,
}

// moduleStoreDirs — prefixos de path de workspace de produto/pack citados
// por design (o destino é o workspace do operador ou o working copy do
// pack) — o L1 de kit não verifica. rules/ NÃO está aqui: no config é dir
// real e checável (caso motivador do lint). Declarativa e versionada.
var moduleStoreDirs = []string{
	"delivery/", "specs/", "analysis/", "reverse/", "quality/", "glossaries/",
	"meetings/", "knowledge/", "support/", "docs/legacy/", "accumulation/",
	"domains/", "docs/howto/", "migration/", "docs/", "evals/", "scripts/",
	"guide/", "ci/", ".opencode/", "config/", "howto/", ".vscode/", ".golmet/",
}

// reportPrefixes — famílias de relatório com timestamp no nome (glob por
// prefixo citado em prosa — nunca existem literalmente).
var reportPrefixes = []string{
	"CLIENT_REPORT_", "EVAL_REPORT_", "PERF_REPORT_", "SECURITY_AUDIT_",
	"VALIDATION_REPORT_", "DRIFT_REPORT_", "CLIENT_REPORT",
}

var (
	mdLinkRe   = regexp.MustCompile(`\[[^\]]*\]\(([^)\s]+)\)`)
	backtickRe = regexp.MustCompile("`([^`\n]+)`")
	ghostRe    = regexp.MustCompile(`gm-[a-z0-9]+(-[a-z0-9]+)*`) // sem trailing hyphen — não captura prefixo truncado (gm-stack- de "gm-stack-*")
	kebabRe    = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
)

// knownExt — extensões que caracterizam um path citado verificável (ref de
// DIR sem extensão não é verificada — heurística anti-falso-positivo
// documentada no FORMATO §9.1).
var knownExt = map[string]bool{
	".md": true, ".json": true, ".yaml": true, ".yml": true, ".go": true,
	".sh": true, ".ps1": true, ".mjs": true, ".txt": true, ".toml": true,
	".template": true, ".cmd": true,
}

// Options do lint. Workspace vazio = "."; EngineID vazio = opencode;
// Home vazio = os.UserHomeDir.
type Options struct {
	Workspace string
	EngineID  string
	Home      string
}

// Run executa a varredura. Erro só para workspace inexistente — findings
// são o resultado normal (exit 2 no CLI).
func Run(opt Options) (*Report, error) {
	ws := opt.Workspace
	if ws == "" {
		ws = "."
	}
	if st, serr := os.Stat(ws); serr != nil || !st.IsDir() {
		return nil, fmt.Errorf("workspace %q não existe", ws)
	}
	engine := opt.EngineID
	if engine == "" {
		engine = "opencode"
	}
	home := opt.Home
	if home == "" {
		home = userHome()
	}
	rep := &Report{Workspace: filepath.ToSlash(ws), Findings: []Finding{}, ByCheck: map[string]int{}}

	// kit mode: workspace tem coredata/registry.yaml
	kitMode := fileExists(filepath.Join(ws, "coredata", "registry.yaml"))
	rep.KitMode = kitMode

	// config root do engine (quando existir na máquina)
	configRoot := ""
	if home != "" {
		if root, applies := engineConfigRoot(home, engine); applies && dirExists(root) {
			configRoot = root
		}
	}
	rep.ConfigRoot = configRoot

	validGm := registryIDs(ws, kitMode)

	// ---- alvos de scan ----
	var kitMDs []string // rel ao ws
	if kitMode {
		kitMDs = append(kitMDs, mdFilesUnder(ws, "coredata/sources")...)
		kitMDs = append(kitMDs, mdFilesUnder(ws, "coredata/rules")...)
		kitMDs = append(kitMDs, mdFilesUnder(ws, "docs")...)
		kitMDs = append(kitMDs, mdFilesUnder(ws, "")...)
	}

	// ---- L1 — referências mortas (error) ----
	if kitMode {
		ids := validGm
		for _, rel := range kitMDs {
			fpath := filepath.Join(ws, filepath.FromSlash(rel))
			roots := []string{filepath.Dir(fpath), ws}
			if configRoot != "" {
				roots = append(roots, configRoot)
			}
			checkMDRefs(rep, rel, fpath, roots, func(cand string) (bool, bool) {
				// resolução canônica kit: rules/x → coredata/rules/x;
				// skills/<id>/SKILL.md e agents/<id>.md → id no registry.
				// rules/<x> NÃO canônico = runtime do config do operador
				// (client/operator-profile, templates) — verificável só com
				// config presente; sem ele o L1 de kit NÃO opina (skip).
				if strings.HasPrefix(cand, "rules/") {
					if fileExists(filepath.Join(ws, "coredata", filepath.FromSlash(cand))) {
						return true, false
					}
					return false, true
				}
				if strings.HasPrefix(cand, "skills/") {
					parts := strings.Split(cand, "/")
					if len(parts) >= 2 && containsStr(ids, parts[1]) {
						return true, false
					}
					return false, false
				}
				if strings.HasPrefix(cand, "agents/") && strings.HasSuffix(cand, ".md") {
					id := strings.TrimSuffix(strings.TrimPrefix(cand, "agents/"), ".md")
					return containsStr(ids, id), false
				}
				return false, false
			})
		}
	}
	// L1 runtime view: AGENTS.md + rules/*.md do config + *.md de packs
	if configRoot != "" {
		lintL1Runtime(rep, configRoot)
	}

	// ---- L4 — manifest vs disco (error) ----
	if configRoot != "" {
		lintL4(rep, configRoot)
	}

	// ---- L3 — órfãos (warning) ----
	if configRoot != "" {
		lintL3(rep, configRoot)
	}

	// ---- L5 — nomes fora do padrão (warning) ----
	lintL5(rep, ws, kitMode, configRoot)

	// ---- L6 — componentes fantasmas ----
	lintL6(rep, ws, kitMode, configRoot, kitMDs, validGm)

	// ---- L7 — estrutura canônica ----
	lintL7(rep, ws, kitMode, configRoot)

	// ---- resumo ----
	for _, f := range rep.Findings {
		if f.Severity == "error" {
			rep.Errors++
		} else {
			rep.Warnings++
		}
		rep.ByCheck[f.Check]++
	}
	sortFindings(rep.Findings)
	return rep, nil
}

// ---- L1 — referências mortas ----

// lintL1Runtime varre AGENTS.md + rules/*.md do config (runtime view):
// candidatos resolvem contra [dir do arquivo, config root]. *.md de packs
// NÃO são varridos v0.1 — docs de negócio do pack citam paths de PRODUTO
// por design (ex.: <repo>/deploy.yml), falso-positivo estrutural; refs
// internas do pack são domínio do L5/L7 (naming/estrutura).
func lintL1Runtime(rep *Report, configRoot string) {
	var rels []string
	if fileExists(filepath.Join(configRoot, "AGENTS.md")) {
		rels = append(rels, "AGENTS.md")
	}
	if fileExists(filepath.Join(configRoot, "AGENTS.template.md")) {
		rels = append(rels, "AGENTS.template.md")
	}
	rels = append(rels, mdFilesUnder(configRoot, "rules")...)
	for _, rel := range rels {
		fpath := filepath.Join(configRoot, filepath.FromSlash(rel))
		checkMDRefs(rep, rel, fpath, []string{filepath.Dir(fpath), configRoot}, nil)
	}
}

// checkMDRefs extrai candidatos (backticks + links md) linha a linha
// (respeitando fences marcados como exemplo) e verifica existência.
// extra = resoluções adicionais do contexto (ex.: mapping canônico do kit);
// segundo retorno true = SKIP (runtime que o contexto não pode verificar).
func checkMDRefs(rep *Report, rel, fpath string, roots []string, extra func(string) (bool, bool)) {
	data, err := os.ReadFile(fpath)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	inSkip := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			info := strings.ToLower(strings.TrimPrefix(trimmed, "```"))
			if inSkip {
				inSkip = false
			} else if strings.Contains(info, "exemplo") || strings.Contains(info, "example") {
				inSkip = true // heurística: bloco marcado como exemplo não é verificado
			}
			continue
		}
		if inSkip {
			continue
		}
		for _, cand := range extractCandidates(line) {
			// heurística declarativa: candidates SEM "/" (nome solto) não são
			// verificados quando não resolvem — a família é aberta (marcadores
			// de repo de produto: angular.json, Cargo.toml, LEGACY_CONTEXT.md);
			// refs COM localização (slash) são verificáveis.
			if resolveCand(cand, roots) {
				continue
			}
			if !strings.Contains(cand, "/") {
				continue
			}
			if extra != nil {
				if resolved, skip := extra(cand); resolved || skip {
					continue
				}
			}
			if isModuleStoreRef(cand) {
				continue // runtime de workspace de produto / pendência declarada
			}
			if reason, ok := refAllowlist[cand]; ok {
				_ = reason
				continue
			}
			rep.Findings = append(rep.Findings, Finding{
				Check: "L1", Severity: "error", File: rel, Line: i + 1,
				Description: "referência morta: `" + cand + "` não existe em nenhuma raiz conhecida",
				Fix:         "corrija o path citado, crie o arquivo referido ou remova a referência",
			})
		}
	}
}

// isModuleStoreRef — o candidate aponta para artefato de WORKSPACE DE
// PRODUTO citado por design (store de módulo/dir de store/relatório com
// timestamp)? O destino existe no workspace do operador, não no kit — o
// L1 não opina (heurística declarativa documentada no FORMATO §9.1).
func isModuleStoreRef(cand string) bool {
	base := filepath.Base(cand)
	if moduleStoreNames[base] {
		return true
	}
	for _, p := range reportPrefixes {
		if strings.HasPrefix(base, p) {
			return true
		}
	}
	for _, d := range moduleStoreDirs {
		if strings.HasPrefix(cand, d) {
			return true
		}
	}
	return false
}

// extractCandidates — backticks + links markdown de UMA linha, filtrados
// pelas heurísticas anti-falso-positivo (URL, âncora, placeholder,
// wildcard, braces, ref de dir sem extensão, precisa de extensão conhecida).
func extractCandidates(line string) []string {
	var cands []string
	seen := map[string]bool{}
	add := func(t string) {
		t = strings.TrimSpace(t)
		t = strings.TrimPrefix(t, "./") // rel ao projeto — mesma semântica de scripts/
		if t == "" || seen[t] {
			return
		}
		seen[t] = true
		low := strings.ToLower(t)
		if strings.HasPrefix(low, "http://") || strings.HasPrefix(low, "https://") ||
			strings.HasPrefix(low, "mailto:") || strings.HasPrefix(t, "#") {
			return
		}
		if strings.ContainsAny(t, " *{}<>%()") {
			return // espaços, wildcard, braces, placeholder, format, mal-formado
		}
		if strings.HasPrefix(t, "~/") || strings.HasPrefix(t, "$") {
			return // runtime de máquina (home do operador / env var)
		}
		if i := strings.Index(t, "#"); i > 0 {
			t = t[:i] // strip âncora
		}
		t = strings.TrimSuffix(t, ".")
		if t == "" || strings.HasSuffix(t, "/") {
			return // ref de DIR sem extensão — não verificada (heurística)
		}
		if !knownExt[strings.ToLower(filepath.Ext(t))] {
			return
		}
		cands = append(cands, t)
	}
	for _, m := range backtickRe.FindAllStringSubmatch(line, -1) {
		add(m[1])
	}
	for _, m := range mdLinkRe.FindAllStringSubmatch(line, -1) {
		add(m[1])
	}
	return cands
}

func resolveCand(cand string, roots []string) bool {
	for _, r := range roots {
		if fileExists(filepath.Join(r, filepath.FromSlash(cand))) {
			return true
		}
	}
	return false
}

// ---- L4 — manifest vs disco ----

func lintL4(rep *Report, configRoot string) {
	manifest, err := content.ReadContentManifest(configRoot)
	if err != nil {
		rep.Findings = append(rep.Findings, Finding{
			Check: "L4", Severity: "warning", File: ".golmet/manifest.json",
			Description: "manifest de conteúdo ilegível: " + err.Error(),
			Fix:         "corrija o JSON do manifest ou rode o update do kit para reescrevê-lo",
		})
		return
	}
	if manifest == nil {
		return
	}
	for _, e := range manifest.Files {
		if e.File == "" {
			continue
		}
		if !fileExists(filepath.Join(configRoot, filepath.FromSlash(e.File))) {
			rep.Findings = append(rep.Findings, Finding{
				Check: "L4", Severity: "error", File: e.File,
				Description: "entrada stale no manifest de conteúdo (layer " + e.Layer + ") — o arquivo não existe no disco",
				Fix:         "remova a entrada stale (o manifest é reescrito pelo próximo materialize) ou re-materialize o arquivo se ele deveria existir",
			})
		}
	}
}

// ---- L3 — órfãos ----

func lintL3(rep *Report, configRoot string) {
	manifest, err := content.ReadContentManifest(configRoot)
	if err != nil || manifest == nil {
		return // sem manifest = config sem gestão do kit — classificação de órfão não se aplica (ilegível já virou finding no L4)
	}
	managed := map[string]bool{}
	for _, e := range manifest.Files {
		managed[e.File] = true
	}
	personal, perr := doctor.ReadPersonalPaths(configRoot)
	if perr != nil {
		personal = nil
	}
	agentsContent := readIfExists(filepath.Join(configRoot, "AGENTS.md"))
	walkConfig(configRoot, func(abs, rel string) {
		if managed[rel] {
			return
		}
		if personal != nil && personal.Match(rel) {
			return
		}
		for _, p := range orphanAllowlist {
			if strings.HasPrefix(rel, p) {
				return
			}
		}
		if strings.Contains(agentsContent, rel) {
			return // citado pelo AGENTS.md — tem dono por referência
		}
		rep.Findings = append(rep.Findings, Finding{
			Check: "L3", Severity: "warning", File: rel,
			Description: "arquivo órfão no config do engine — sem proveniência no manifest, sem designação pessoal, fora de state pack, fora da allowlist e não citado pelo AGENTS.md",
			Fix:         "mova para o lugar canônico (pack/workspace), designe pessoal-por-design (.golmet/personal.json), apague, ou amplie a orphanAllowlist declarativa do lint se for órfão-legítimo",
		})
	})
}

// walkConfig percorre TODOS os arquivos regulares do config root (exclui
// .git/.golmet, o dir packs/ — proveniência própria do pack, classificada
// pelos checks de pack — e subárvores de state packs) chamando fn(abs, rel).
func walkConfig(root string, fn func(abs, rel string)) {
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == ".golmet" {
				return filepath.SkipDir
			}
			if rel, rerr := filepath.Rel(root, p); rerr == nil && rel == "packs" {
				return filepath.SkipDir // proveniência própria do pack (L1/L5/L7 de pack cobrem)
			}
			if isStatePack(p) {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel, rerr := filepath.Rel(root, p)
		if rerr != nil {
			return nil
		}
		fn(p, filepath.ToSlash(rel))
		return nil
	})
}

// ---- L5 — nomes fora do padrão ----

func lintL5(rep *Report, ws string, kitMode bool, configRoot string) {
	flag := func(file, name, fix string) {
		rep.Findings = append(rep.Findings, Finding{
			Check: "L5", Severity: "warning", File: file,
			Description: "nome fora do padrão: " + name,
			Fix:         fix,
		})
	}
	kebabFix := "renomeie para kebab-case ASCII (a-z0-9-) e atualize as referências cruzadas"
	// config 1º nível — dirs estruturais (nomes com ponto/ocultos isentos)
	if configRoot != "" {
		if entries, err := os.ReadDir(configRoot); err == nil {
			for _, e := range entries {
				n := e.Name()
				if !e.IsDir() || n == ".golmet" || strings.Contains(n, ".") || kebabOK(n) {
					continue
				}
				flag(n, n, kebabFix)
			}
		}
		lintPackNames(rep, configRoot)
	}
	if !kitMode {
		return
	}
	// kit: coredata/sources flat — base sans extensão em kebab
	if entries, err := os.ReadDir(filepath.Join(ws, "coredata", "sources")); err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			base := strings.TrimSuffix(e.Name(), ".agent.md")
			base = strings.TrimSuffix(base, ".md")
			if !kebabOK(base) {
				flag("coredata/sources/"+e.Name(), e.Name(), kebabFix)
			}
		}
	}
	// kit: skills/ raiz — dirs em kebab
	if entries, err := os.ReadDir(filepath.Join(ws, "skills")); err == nil {
		for _, e := range entries {
			if e.IsDir() && !kebabOK(e.Name()) {
				flag("skills/"+e.Name(), e.Name(), kebabFix)
			}
		}
	}
}

// kebabOK — kebab-case ASCII (a-z0-9, hífens internos).
func kebabOK(name string) bool { return kebabRe.MatchString(name) }

// lintPackNames varre working copies de state packs: rules/*.md (tema),
// patterns tipo+slug, nuances tema — kebab ASCII.
func lintPackNames(rep *Report, configRoot string) {
	packsDir := filepath.Join(configRoot, "packs")
	entries, err := os.ReadDir(packsDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() || !isStatePack(filepath.Join(packsDir, e.Name())) {
			continue
		}
		packID := e.Name()
		packDir := filepath.Join(packsDir, packID)
		for _, rel := range mdFilesUnder(packDir, "rules") {
			base := strings.TrimSuffix(filepath.Base(rel), ".md")
			if !kebabOK(base) {
				rep.Findings = append(rep.Findings, Finding{
					Check: "L5", Severity: "warning", File: "packs/" + packID + "/" + rel,
					Description: "nome de norma fora do padrão kebab-case ASCII: " + base,
					Fix:         "renomeie o arquivo da norma (uma norma por arquivo, tema kebab-case ASCII) e atualize o lazy index",
				})
			}
		}
		patDir := filepath.Join(packDir, "accumulation", "patterns")
		if types, err := os.ReadDir(patDir); err == nil {
			for _, t := range types {
				if !t.IsDir() {
					continue
				}
				if !containsStr(patternTypes, t.Name()) {
					rep.Findings = append(rep.Findings, Finding{
						Check: "L5", Severity: "warning", File: "packs/" + packID + "/accumulation/patterns/" + t.Name(),
						Description: "tipo de pattern fora do conjunto canônico (process|architecture|code|ui|test|data): " + t.Name(),
						Fix:         "renomeie o dir do tipo para um dos canônicos, mova os padrões e atualize o INDEX.md",
					})
				}
				if slugs, err := os.ReadDir(filepath.Join(patDir, t.Name())); err == nil {
					for _, s := range slugs {
						if s.IsDir() && !kebabOK(s.Name()) {
							rep.Findings = append(rep.Findings, Finding{
								Check: "L5", Severity: "warning", File: "packs/" + packID + "/accumulation/patterns/" + t.Name() + "/" + s.Name(),
								Description: "slug de pattern fora do padrão kebab-case ASCII: " + s.Name(),
								Fix:         "renomeie o dir do slug (kebab-case ASCII) e atualize o INDEX.md",
							})
						}
					}
				}
			}
		}
		if nuances, err := os.ReadDir(filepath.Join(packDir, "accumulation", "nuances")); err == nil {
			for _, n := range nuances {
				if n.IsDir() || !strings.HasSuffix(n.Name(), ".md") {
					continue
				}
				theme := strings.TrimSuffix(n.Name(), "-nuances.md")
				if !kebabOK(theme) {
					rep.Findings = append(rep.Findings, Finding{
						Check: "L5", Severity: "warning", File: "packs/" + packID + "/accumulation/nuances/" + n.Name(),
						Description: "tema de nuances fora do padrão kebab-case ASCII: " + theme,
						Fix:         "renomeie o arquivo do tema para <tema>-nuances.md em kebab-case ASCII",
					})
				}
			}
		}
	}
}

// ---- L6 — componentes fantasmas ----

func lintL6(rep *Report, ws string, kitMode bool, configRoot string, kitMDs []string, validGm []string) {
	scan := func(rel, fpath, sev string) {
		data, err := os.ReadFile(fpath)
		if err != nil {
			return
		}
		for i, line := range strings.Split(string(data), "\n") {
			for _, loc := range ghostRe.FindAllStringIndex(line, -1) {
				m := line[loc[0]:loc[1]]
				if containsStr(validGm, m) {
					continue
				}
				// guard de família: match seguido de wildcard/brace/hífen é
				// PREFIXO citado em prosa (gm-stack-*), não nome completo
				if next := loc[1]; next < len(line) {
					switch line[next] {
					case '*', '{', '-', '/':
						continue
					}
				}
				if _, ok := ghostAllowlist[m]; ok {
					continue
				}
				rep.Findings = append(rep.Findings, Finding{
					Check: "L6", Severity: sev, File: rel, Line: i + 1,
					Description: "componente fantasma: `" + m + "` não existe no registry.yaml",
					Fix:         "corrija o nome do componente, crie o componente, ou amplie a ghostAllowlist declarativa do lint com o motivo (nomes legados/removidos)",
				})
			}
		}
	}
	// kit: sources/rules = error (contrato vivo); docs/raiz = warning (histórico)
	if kitMode {
		for _, rel := range kitMDs {
			sev := "warning"
			if strings.HasPrefix(rel, "coredata/sources/") || strings.HasPrefix(rel, "coredata/rules/") {
				sev = "error"
			}
			scan(rel, filepath.Join(ws, filepath.FromSlash(rel)), sev)
		}
	}
	// config: AGENTS.md + AGENTS.template.md + rules/*.md = warning (runtime do operador)
	if configRoot != "" {
		if fileExists(filepath.Join(configRoot, "AGENTS.md")) {
			scan("AGENTS.md", filepath.Join(configRoot, "AGENTS.md"), "warning")
		}
		if fileExists(filepath.Join(configRoot, "AGENTS.template.md")) {
			scan("AGENTS.template.md", filepath.Join(configRoot, "AGENTS.template.md"), "warning")
		}
		for _, rel := range mdFilesUnder(configRoot, "rules") {
			scan(rel, filepath.Join(configRoot, filepath.FromSlash(rel)), "warning")
		}
	}
}

// ---- L7 — estrutura canônica ----

func lintL7(rep *Report, ws string, kitMode bool, configRoot string) {
	structErr := func(file, desc, fix string) {
		rep.Findings = append(rep.Findings, Finding{Check: "L7", Severity: "error", File: file, Description: desc, Fix: fix})
	}
	if kitMode {
		if entries, err := os.ReadDir(filepath.Join(ws, "coredata")); err == nil {
			for _, e := range entries {
				if e.IsDir() && !containsStr([]string{"sources", "rules"}, e.Name()) {
					structErr("coredata/"+e.Name(), "dir não canônico em coredata/ (canônico: sources/, rules/, registry.yaml)",
						"mova o conteúdo para o lugar canônico ou declare a exceção no lint (L7)")
				}
				if !e.IsDir() && e.Name() != "registry.yaml" {
					structErr("coredata/"+e.Name(), "arquivo não canônico em coredata/ (canônico: registry.yaml)",
						"mova o arquivo para o lugar canônico ou declare a exceção no lint (L7)")
				}
			}
		}
		if entries, err := os.ReadDir(filepath.Join(ws, "coredata", "sources")); err == nil {
			for _, e := range entries {
				if e.IsDir() {
					structErr("coredata/sources/"+e.Name(), "subdir em coredata/sources/ — a fonte é FLAT (*.md e *.agent.md)",
						"movimente o conteúdo para o nível flat ou declare a exceção no lint (L7)")
				}
			}
		}
		if entries, err := os.ReadDir(filepath.Join(ws, "skills")); err == nil {
			n := 0
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				n++
				if !fileExists(filepath.Join(ws, "skills", e.Name(), "SKILL.md")) {
					structErr("skills/"+e.Name(), "dir de skill BASE sem SKILL.md",
						"crie o SKILL.md ou remova o dir (zona BASE = skills/gm-*/SKILL.md, 16 skills)")
				}
			}
			if n != 16 {
				structErr("skills/", fmt.Sprintf("zona BASE com %d skills — o embed espera 16", n),
					"restaure/remova skills da zona BASE até 16 (o subset embutido é o contrato do perfil essencial)")
			}
		}
		golden := filepath.Join(ws, "internal", "core", "content", "testdata", "golden-opencode")
		if entries, err := os.ReadDir(golden); err == nil {
			for _, e := range entries {
				if e.IsDir() && !containsStr([]string{"agents", "skills", "rules"}, e.Name()) {
					structErr("internal/core/content/testdata/golden-opencode/"+e.Name(),
						"dir não canônico no golden (espelha o render: agents/, skills/, rules/)",
						"regenere o golden (GOLMET_UPDATE_GOLDEN=1) ou mova o dir")
				}
			}
		}
	}
	if configRoot == "" {
		return
	}
	// PACK STATE — raiz + accumulation canônicos
	packsDir := filepath.Join(configRoot, "packs")
	if entries, err := os.ReadDir(packsDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() || !isStatePack(filepath.Join(packsDir, e.Name())) {
				continue
			}
			packID := e.Name()
			packDir := filepath.Join(packsDir, packID)
			if rentries, err := os.ReadDir(packDir); err == nil {
				for _, r := range rentries {
					canonical := containsStr(packStateRootDirs, r.Name()) ||
						r.Name() == "pack.yaml" || r.Name() == ".git" ||
						r.Name() == "README.md" || r.Name() == "LICENSE.md" || r.Name() == "LICENSE"
					if r.IsDir() && !canonical {
						structErr("packs/"+packID+"/"+r.Name(),
							"dir não canônico na raiz do state pack (canônico: pack.yaml + "+strings.Join(packStateRootDirs, ", ")+")",
							"mova o conteúdo para o lugar canônico ou declare a exceção na allowlist do lint")
					}
				}
			}
			if aentries, err := os.ReadDir(filepath.Join(packDir, "accumulation")); err == nil {
				for _, a := range aentries {
					if a.IsDir() && !containsStr(accumulationOnly, a.Name()) {
						structErr("packs/"+packID+"/accumulation/"+a.Name(),
							"dir não canônico em accumulation/ (canônico: "+strings.Join(accumulationOnly, ", ")+")",
							"mova o conteúdo (patterns → gm-pattern-extractor, metrics → gm-metrics-collector, nuances → gm-nuance-collector) ou declare a exceção na allowlist do lint")
					}
				}
			}
		}
	}
	// CONFIG — 1º nível: unknown dir = warning (padrão declarado)
	if entries, err := os.ReadDir(configRoot); err == nil {
		for _, e := range entries {
			if e.IsDir() && !containsStr(configRootDirs, e.Name()) {
				rep.Findings = append(rep.Findings, Finding{
					Check: "L7", Severity: "warning", File: e.Name(),
					Description: "dir não reconhecido no 1º nível do config do engine (padrão declarado: " + strings.Join(configRootDirs, ", ") + ")",
					Fix:         "mova o conteúdo para o lugar canônico, designe pessoal-por-design, ou amplie configRootDirs declarativa no lint",
				})
			}
		}
	}
}

// ---- helpers ----

// registryIDs — ids do registry (workspace em kit mode; senão o embarcado —
// o lint do config/packs valida fantasmas contra o registry do binário).
func registryIDs(ws string, kitMode bool) []string {
	var data []byte
	var err error
	if kitMode {
		data, err = os.ReadFile(filepath.Join(ws, "coredata", "registry.yaml"))
	}
	if !kitMode || err != nil {
		if data, err = fs.ReadFile(golmet.CoreDataFS(), "coredata/registry.yaml"); err != nil {
			return nil
		}
	}
	reg := content.ParseRegistry(string(data))
	seen := map[string]bool{}
	var out []string
	for _, it := range reg.Items {
		id := it.Str("id")
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

// isStatePack — dir contém pack.yaml com kind: state (mesma semântica do
// doctor — parse via pack.ParseSpec).
func isStatePack(dir string) bool {
	data, err := os.ReadFile(filepath.Join(dir, "pack.yaml"))
	if err != nil {
		return false
	}
	spec, err := pack.ParseSpec(data)
	return err == nil && spec.Kind == pack.KindState
}

// mdFilesUnder — *.md diretos sob root/sub (1 nível; sub vazio = diretos do root).
func mdFilesUnder(root, sub string) []string {
	dir := root
	if sub != "" {
		dir = filepath.Join(root, filepath.FromSlash(sub))
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if sub == "" {
			out = append(out, e.Name())
		} else {
			out = append(out, sub+"/"+e.Name())
		}
	}
	return out
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func readIfExists(p string) string {
	data, _ := os.ReadFile(p)
	return string(data)
}

func containsStr(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

func userHome() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return h
}

// engineConfigRoot — resolve a raiz do config do engine (mesma regra
// declarada do kit: engines com config global; vscode-copilot é
// workspace-scoped — check não se aplica).
func engineConfigRoot(home, engineID string) (string, bool) {
	switch engineID {
	case "opencode":
		return filepath.Join(home, ".config", "opencode"), true
	case "claude-code":
		return filepath.Join(home, ".claude"), true
	case "cursor":
		return filepath.Join(home, ".cursor"), true
	case "gemini-cli":
		return filepath.Join(home, ".gemini"), true
	case "codex":
		return filepath.Join(home, ".codex"), true
	}
	return "", false
}

func sortFindings(fs []Finding) {
	sort.Slice(fs, func(i, j int) bool {
		if fs[i].Check != fs[j].Check {
			return fs[i].Check < fs[j].Check
		}
		if fs[i].File != fs[j].File {
			return fs[i].File < fs[j].File
		}
		return fs[i].Line < fs[j].Line
	})
}

// Human — render textual PT-BR (o JSON é o próprio Report).
func (r *Report) Human() string {
	var b strings.Builder
	fmt.Fprintf(&b, "golmet lint — %s", r.Workspace)
	if r.ConfigRoot != "" {
		fmt.Fprintf(&b, " [config %s]", r.ConfigRoot)
	}
	if r.KitMode {
		b.WriteString(" [kit]")
	}
	b.WriteString("\n")
	for _, f := range r.Findings {
		icon := "❌"
		if f.Severity == "warning" {
			icon = "⚠️ "
		}
		loc := f.File
		if f.Line > 0 {
			loc = fmt.Sprintf("%s:%d", f.File, f.Line)
		}
		fmt.Fprintf(&b, "%s %s %s — %s\n", icon, f.Check, loc, f.Description)
		if f.Fix != "" {
			fmt.Fprintf(&b, "     fix: %s\n", f.Fix)
		}
	}
	fmt.Fprintf(&b, "\n%d error(s), %d warning(s) — lint concluído\n", r.Errors, r.Warnings)
	return b.String()
}
