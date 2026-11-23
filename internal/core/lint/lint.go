// Package lint implementa `golmet lint` — varredura estrutural
// determinística do kit/config/packs: referências mortas (L1), órfãos (L3),
// manifest stale (L4), nomes fora do padrão (L5), componentes fantasmas
// (L6), estrutura de pastas canônica (L7), conformidade estrutural
// definitivo ↔ template (L8) e schema do registry de stack-templates
// materializado no config (L9). Read-only — nunca muta. A IA
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
	"strconv"
	"strings"
	"unicode/utf8"

	golmet "github.com/Golmet/golmet-cli"
	"github.com/Golmet/golmet-cli/internal/core/container"
	"github.com/Golmet/golmet-cli/internal/core/content"
	"github.com/Golmet/golmet-cli/internal/core/doctor"
	"github.com/Golmet/golmet-cli/internal/core/pack"
	"gopkg.in/yaml.v3"
)

// Finding é um apontamento do lint: severidade, arquivo (+linha quando
// aplicável), descrição e remediação sugerida (PT-BR).
type Finding struct {
	Check       string `json:"check"`          // L1 | L3 | L4 | L5 | L6 | L7 | L7-size | L10 | L8 | L9
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
	Container  bool           `json:"container,omitempty"` // golmet.yaml com container: true (FORMAT §2)
	Findings   []Finding      `json:"findings"`
	Errors     int            `json:"errors"`
	Warnings   int            `json:"warnings"`
	ByCheck    map[string]int `json:"by_check"`
}

// ---- padrões canônicos DECLARADOS (contrato da estrutura — fonte única
// em código; documentado no FORMAT §9.1) ----

// packStateRootDirs — dirs canônicos na raiz de um state pack (working
// copy): pack.yaml + qualquer subconjunto destes (+ meta .git/README/LICENSE).
// `mcp` é a convenção do server MCP próprio do pack (`golmet pack mcp`):
// mcp/server.js (server stdio Node) com registry dinâmico em mcp/tools/.
var packStateRootDirs = []string{"skills", "agents", "rules", "domains", "accumulation", "mcp"}

// mcpOnly — único subdir canônico de mcp/ (mesmo padrão do accumulationOnly
// para accumulation/). Arquivos diretos em mcp/ são canônicos (server.js).
var mcpOnly = []string{"tools"}

// accumulationOnly — único conteúdo canônico de accumulation/.
var accumulationOnly = []string{"patterns", "metrics", "nuances"}

// patternTypes — taxonomia EN canônica dos tipos de pattern (constante
// COMPARTILHADA: a mesma lista é citada pelo source do gm-pattern-extractor —
// manter coerente). Era PT (processo|arquitetura|codigo|ui|teste|dados) até
// a uniformização; regressão PT = finding L10-B. Tipo fora do conjunto =
// estrutura não canônica.
var patternTypes = []string{"process", "architecture", "code", "ui", "test", "data"}

// specAggregates — agregados técnicos do workspace de produto que aceitam
// QUEBRA por micro-contexto (FORMAT §7.7.1): `specs/<dir>/` (formato
// quebrado — INDEX.md obrigatório + `<escopo>.md` auto-contidos) ou o
// unitário `specs/<file>.md` (continua válido). AS DUAS formas são
// canônicas simultaneamente; a implantação começa no DATA_MODEL.
var specAggregates = []struct {
	Dir  string // formato quebrado — specs/<dir>/
	File string // formato unitário — specs/<file>
}{
	{"data_model", "DATA_MODEL.md"},
	{"api_contracts", "API_CONTRACTS.md"},
	{"ui_spec", "UI_SPEC.md"},
}

// DataModelSplitLines — limiar (em linhas) do aviso de quebra do agregado
// unitário specs/DATA_MODEL.md: acima disso o lint emite warning
// (categoria L7-size) recomendando o split por micro-contexto. Constante
// COMPARTILHADA com o gm-datamodel (a skill cita o mesmo número no
// procedimento de Split) e declarada no FORMAT §7.7.1 — mudar aqui =
// mudar lá.
const DataModelSplitLines = 800

// aggregateSizeWarns — avisos PRECOCES de tamanho (warning L7-size) para os
// próximos candidatos a quebra por micro-contexto: agregados que crescem
// mas AINDA NÃO TÊM gramática quebrada declarada no FORMAT (só o
// data_model tem — §7.7.1). Acima do limiar de aviso, o warning recomenda
// PLANEJAR a quebra com o gm-doc — a gramática de cada artefato será
// declarada quando o volume justificar (decisão do operador). Limiar
// próprio por natureza do conteúdo. Paths em ordem de precedência (1º
// existente vale: caminho canônico primeiro, raiz legado como fallback).
var aggregateSizeWarns = []struct {
	Paths []string // onde o agregado vive no workspace
	Lines int      // limiar de aviso
}{
	{[]string{"DECISIONS.md"}, 800},                               // ADRs auto-contidos; quebrado decisions/ será planejado quando disparar
	{[]string{"specs/ERROR_CATALOG.md", "ERROR_CATALOG.md"}, 600}, // specs/ canônico, raiz legado
	{[]string{"support/KNOWN_ERRORS.md", "KNOWN_ERRORS.md"}, 400}, // support/ canônico, raiz legado
	{[]string{"GLOSSARY.md"}, 300},                                // glossários por público (DEV/BUSINESS/USER) ficam fora — o agregado é o GLOSSARY.md
}

// configRootDirs — dirs canônicos de 1º nível do config do engine.
// knowledge/ é legado reconhecido. Unknown dir → warning.
var configRootDirs = []string{"agents", "skills", "profile", "packs", "plugins", "knowledge", ".golmet"}

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
	"gm-coverage-audit":    "avaliada e não criada — cobertura é do gm-dev-qa/playbook da stack",
	"gm-a11y":              "exemplo ilustrativo do FORMAT §6",
	"gm-a11y-reviewer":     "exemplo ilustrativo do FORMAT §6",
	"gm-xxx":               "exemplo ilustrativo (gm-registry-render)",
	"gm-yyy":               "exemplo ilustrativo (gm-registry-render)",
	"gm-zzz":               "exemplo ilustrativo (gm-registry-render)",
	"gm-orchestrator":      "exemplo ilustrativo (gm-registry-render)",
	"gm-s1":                "referência de seção S1 do FORMAT, não componente",
	"gm-docs":              "exemplo de nome de REPO_DOCS no gm-init (`minha-org/gm-docs`) — não é componente",
	"gm-x":                 "exemplo de id no front matter do FORMAT §10",
}

// refAllowlist — paths citados sabidamente mortos, com motivo (pendências
// declaradas). Declarativa e versionada — o L1 não apina o que está aqui.
var refAllowlist = map[string]string{
	"AGENTS.template.md": "pendência declarada da wave do template AGENTS — o render ainda não existe no kit",
}

// ---- L8 — estrutura mínima definitivo ↔ template (declarativa; política
// anti-ruído do FORMAT §9.1: evolução NÃO flaga — o mínimo é o que não
// muda entre versões do kit; template ganhar seção nova não obriga
// definitivos antigos; só a PERDA/ausência da estrutura mínima flaga) ----

// l8MarkerBlocks — pares de blocos marcadores que o AGENTS.md definitivo
// PRECISA ter abertos e fechados.
var l8MarkerBlocks = []string{"azure-devops", "org-pack", "projeto-padrao", "gm-pack-accumulation", "knowledge-routing"}

// l8AGENTSMinSections — seções canônicas fundamentais (substring — sufixos
// podem evoluir).
var l8AGENTSMinSections = []string{"Tools MCP do kit", "Índice de Referências"}

// l8ClientProfileMin — seções/campos IMP-103/105/111 mínimos (calibrado ao
// definitivo: o Tracker migrou de estrutura na org — não é mínimo).
var l8ClientProfileMin = []string{"Identidade e infraestrutura", "Packs", "org_github", "ghcr_registry", "dns_domain"}

// l8OperatorProfileMin — seções/campos IMP-136 mínimos.
var l8OperatorProfileMin = []string{"Identidade", "Identidade git e contas", "git user.name", "git user.email", "username"}

// l8Pairs — (template no embed, definitivo no config, campos mínimos).
var l8Pairs = []struct {
	Template string
	Final    string
	Min      []string
}{
	{"templates/AGENTS.template.md", "AGENTS.md", nil}, // blocos + seções (caso especial)
	{"rules/client-profile.template.md", "profile/client-profile.md", l8ClientProfileMin},
	{"rules/operator-profile.template.md", "profile/operator-profile.md", l8OperatorProfileMin},
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
	ghostRe    = regexp.MustCompile(`gm-[a-z0-9]+(-[a-z0-9]+)*`)                            // sem trailing hyphen — não captura prefixo truncado (gm-stack- de "gm-stack-*")
	reOrgID    = regexp.MustCompile(`(?m)^\s*- (?:\{\s*)?id:\s*([A-Za-z0-9][A-Za-z0-9-]*)`) // - id: x (multiline) e - { id: x, ... } (inline dict)
	kebabRe    = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
)

// knownExt — extensões que caracterizam um path citado verificável (ref de
// DIR sem extensão não é verificada — heurística anti-falso-positivo
// documentada no FORMAT §9.1).
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

	// container de organização (FORMAT §2): golmet.yaml com container: true
	// — a raiz é um CONTAINER, não um workspace-produto: os checks ganham a
	// visão de hierarquia (produtos declarados, docs/, meetings/, migrations/)
	// e o L1 varre os *.md do container.
	var contProducts []string
	if fields, isCont, cerr := container.Config(ws); cerr == nil && isCont {
		rep.Container = true
		contProducts = container.Products(fields)
	}

	// config root do engine (quando existir na máquina)
	configRoot := ""
	if home != "" {
		if root, applies := engineConfigRoot(home, engine); applies && dirExists(root) {
			configRoot = root
		}
	}
	rep.ConfigRoot = configRoot

	validGm := registryIDs(ws, kitMode)
	// fontes de extensão da org (norma component-lifecycle: componentes
	// novos nascem no pack): o modo config/packs resolve gm-* contra
	// kit ∪ org — registry.yaml do config root + componentes declarados
	// nos packs de state ativos. Resolver só contra o registry do binário
	// gerava falso positivo estrutural: todo componente de org citado num
	// AGENTS.md era fantasma (caso real gm-mkt, 2026-11-20).
	if configRoot != "" {
		validGm = append(validGm, orgComponentNames(configRoot)...)
	}

	// ---- alvos de scan ----
	var kitMDs []string // rel ao ws
	if kitMode {
		kitMDs = append(kitMDs, mdFilesUnder(ws, "coredata/sources")...)
		kitMDs = append(kitMDs, mdFilesUnder(ws, "coredata/rules")...)
		kitMDs = append(kitMDs, mdFilesUnder(ws, "docs")...)
		kitMDs = append(kitMDs, mdFilesUnder(ws, "")...)
	}

	// ---- L1 - referências mortas (error) ----
	if kitMode {
		ids := validGm
		for _, rel := range kitMDs {
			// o doc de migração DESCREVE o modelo antigo por natureza
			// (paths mortos, componentes extintos citados como passado) —
			// L1/L6 não se aplicam a ele (exceção declarada no FORMAT §9.1)
			if strings.HasPrefix(rel, "docs/migration") {
				continue
			}
			fpath := filepath.Join(ws, filepath.FromSlash(rel))
			roots := []string{filepath.Dir(fpath), ws}
			if configRoot != "" {
				roots = append(roots, configRoot)
			}
			checkMDRefs(rep, rel, fpath, roots, func(cand string) (bool, bool) {
				// resolução canônica kit (M13): o destino no config é
				// profile/ — profile/x → coredata/rules/x (a FONTE
				// canônica: context7, stack-templates); perfil
				// (client/operator) não tem fonte canônica → runtime do
				// config do operador → SKIP. rules/x (prosa antiga) →
				// coredata/rules/x (compat). skills/<id>/SKILL.md e
				// agents/<id>.md → id no registry.
				if strings.HasPrefix(cand, "profile/") {
					base := strings.TrimPrefix(cand, "profile/")
					if fileExists(filepath.Join(ws, "coredata", "rules", filepath.FromSlash(base))) {
						return true, false
					}
					if containsStr([]string{"client-profile.md", "operator-profile.md"}, base) {
						return false, true // perfil = pessoal-por-design
					}
					return false, true // runtime do config; sem config o kit não opina
				}
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
			}, nil)
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

	// ---- L7 — agregados de spec do workspace (unitário | quebrado, §7.7.1) ----
	lintL7SpecAggregates(rep, ws)

	// ---- L7/L1 — container de organização (FORMAT §2) ----
	if rep.Container {
		lintL7Container(rep, ws, container.Inspect(ws, contProducts))
		lintL1Container(rep, ws, configRoot)
	}

	// ---- L8 — conformidade estrutural definitivo ↔ template (warning) ----
	if configRoot != "" {
		lintL8(rep, configRoot)
	}

	// ---- L9 — schema do registry de stack-templates no config (error) ----
	if configRoot != "" {
		lintL9(rep, configRoot)
	}

	// ---- L10 - estado interno dos state packs (M20) ----
	if configRoot != "" {
		lintL10PackState(rep, configRoot)
	}

	// ---- L11 - diagramas Mermaid (warning) ----
	lintL11Mermaid(rep, ws, kitMDs, kitMode, configRoot)

	// ---- L12 - encoding e texto quebrado (error; BOM = warning) ----
	lintL12Encoding(rep, ws, kitMode, configRoot)

	// ---- L13 - âncoras de idioma (warning) ----
	lintL13Anchors(rep, ws, kitMode, configRoot)

	// ---- L14 - front matter YAML parseável (error) ----
	lintL14FrontMatter(rep, ws, kitMode, configRoot)

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
	rels = append(rels, mdFilesUnder(configRoot, "profile")...)
	for _, rel := range rels {
		fpath := filepath.Join(configRoot, filepath.FromSlash(rel))
		checkMDRefs(rep, rel, fpath, []string{filepath.Dir(fpath), configRoot}, nil, nil)
	}
}

// checkMDRefs extrai candidatos (backticks + links md) linha a linha
// (respeitando fences marcados como exemplo) e verifica existência.
// extra = resoluções adicionais do contexto (ex.: mapping canônico do kit);
// segundo retorno true = SKIP (runtime que o contexto não pode verificar).
// storeExempt = decisão por candidato sobre a isenção de store de
// workspace de produto (docs/, delivery/, specs/…): nil = TODOS isentos
// (kit/runtime — stores citados por design); container decide (refs
// ancoradas no container — docs/, meetings/, migrations/, canônicos — são
// arquivos PRÓPRIOS e verificáveis; as demais stores permanecem isentas).
func checkMDRefs(rep *Report, rel, fpath string, roots []string, extra func(string) (bool, bool), storeExempt func(string) bool) {
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
			if isModuleStoreRef(cand) && (storeExempt == nil || storeExempt(cand)) {
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
// L1 não opina (heurística declarativa documentada no FORMAT §9.1).
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
				// a taxonomia do 1º nível (tipo EN) é vigiada pelo L10 (M20) —
				// aqui fica só o naming do slug
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

type allowUse struct {
	count int
	file  string
	line  int
}

func lintL6(rep *Report, ws string, kitMode bool, configRoot string, kitMDs []string, validGm []string) {
	// A allowlist NÃO é silêncio perpétuo: cada nome exceção que AINDA
	// aparece no conteúdo é superficiado no relatório (L6-allowlist) para
	// revisão periódica. Uma entrada esquecida aqui foi o que deixou uma
	// referência a skill extinta atravessar três releases do kit.
	allowUsed := map[string]*allowUse{}
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
					if allowUsed[m] == nil {
						allowUsed[m] = &allowUse{file: rel, line: i + 1}
					}
					allowUsed[m].count++
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
	// targets do L6: kitMDs (compartilhado com o L1) + conteúdo gerido que o
	// L1 NÃO varre — descrições do registry e goldens citam paths de config
	// que não existem no workspace (false positivo de L1), mas menção a
	// componente extinto aqui é exatamente o que o L6 deve pegar
	l6targets := append([]string{}, kitMDs...)
	if kitMode {
		l6targets = append(l6targets, "coredata/registry.yaml")
		l6targets = append(l6targets, mdFilesDeep(ws, "skills")...)
		l6targets = append(l6targets, mdFilesDeep(ws, filepath.Join("internal", "core", "content", "testdata"))...)
	}
	// severidade por contexto: contrato vivo = error (sources, rules, skills
	// BASE, registry, goldens de render); histórico = warning; docs/migration
	// descreve o modelo antigo por natureza — skip
	if kitMode {
		for _, rel := range l6targets {
			// o doc de migração descreve o modelo antigo por natureza
			// (componentes extintos citados como passado) — skip (L6)
			if strings.HasPrefix(rel, "docs/migration") {
				continue
			}
			sev := "warning"
			switch {
			case strings.HasPrefix(rel, "coredata/sources/"),
				strings.HasPrefix(rel, "coredata/rules/"),
				strings.HasPrefix(rel, "skills/"),
				strings.HasPrefix(rel, "internal/core/content/testdata/"), // rels do walker são ToSlash — literal /, não filepath.Join
				rel == "coredata/registry.yaml":
				sev = "error"
			}
			scan(rel, filepath.Join(ws, filepath.FromSlash(rel)), sev)
		}
	}
	// config: AGENTS.md + AGENTS.template.md + profile/*.md = warning (runtime do operador)
	if configRoot != "" {
		if fileExists(filepath.Join(configRoot, "AGENTS.md")) {
			scan("AGENTS.md", filepath.Join(configRoot, "AGENTS.md"), "warning")
		}
		if fileExists(filepath.Join(configRoot, "AGENTS.template.md")) {
			scan("AGENTS.template.md", filepath.Join(configRoot, "AGENTS.template.md"), "warning")
		}
		for _, rel := range mdFilesUnder(configRoot, "profile") {
			scan(rel, filepath.Join(configRoot, filepath.FromSlash(rel)), "warning")
		}
	}
	// superficia exceções em uso — agregado por nome, 1 finding por nome
	names := make([]string, 0, len(allowUsed))
	for n := range allowUsed {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		h := allowUsed[n]
		rep.Findings = append(rep.Findings, Finding{
			Check:       "L6-allowlist",
			Severity:    "warning",
			File:        h.file,
			Line:        h.line,
			Description: fmt.Sprintf("ghostAllowlist em uso: `%s` — %d ocorrência(s) silenciada(s) nesta varredura; revisar se a exceção ainda se aplica", n, h.count),
			Fix:         "remova a entrada se o nome deixou de ser citado, ou mantenha com o motivo atualizado no mapa ghostAllowlist (referência de IMP)",
		})
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
				if e.IsDir() && !containsStr([]string{"sources", "rules", "templates"}, e.Name()) {
					structErr("coredata/"+e.Name(), "dir não canônico em coredata/ (canônico: sources/, rules/, templates/, registry.yaml)",
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
				if e.IsDir() && !containsStr([]string{"agents", "skills", "profile"}, e.Name()) {
					structErr("internal/core/content/testdata/golden-opencode/"+e.Name(),
						"dir não canônico no golden (espelha o render: agents/, skills/, profile/)",
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
			if mentries, err := os.ReadDir(filepath.Join(packDir, "mcp")); err == nil {
				for _, m := range mentries {
					if m.IsDir() && !containsStr(mcpOnly, m.Name()) {
						structErr("packs/"+packID+"/mcp/"+m.Name(),
							"dir não canônico em mcp/ (canônico: "+strings.Join(mcpOnly, ", ")+" — mcp/server.js é o server do pack)",
							"mova o conteúdo para mcp/tools/ ou declare a exceção na allowlist do lint")
					}
				}
			}
		}
	}
	// CONFIG — 1º nível: unknown dir = warning (padrão declarado); template
	// depositado no config = warning específico (M12: template é etapa de
	// construção — vive no kit; obtenha via `golmet template`)
	if entries, err := os.ReadDir(configRoot); err == nil {
		for _, e := range entries {
			if e.IsDir() && !containsStr(configRootDirs, e.Name()) {
				rep.Findings = append(rep.Findings, Finding{
					Check: "L7", Severity: "warning", File: e.Name(),
					Description: "dir não reconhecido no 1º nível do config do engine (padrão declarado: " + strings.Join(configRootDirs, ", ") + ")",
					Fix:         "mova o conteúdo para o lugar canônico, designe pessoal-por-design, ou amplie configRootDirs declarativa no lint",
				})
			}
			if !e.IsDir() && e.Name() == "AGENTS.template.md" {
				rep.Findings = append(rep.Findings, Finding{
					Check: "L7", Severity: "warning", File: e.Name(),
					Description: "template depositado no config — template é etapa de construção: vive no KIT, o config carrega só os definitivos",
					Fix:         "apague o arquivo e use `golmet template agents` para emitir o esqueleto quando precisar",
				})
			}
		}
	}
}

// ---- L7 — agregados de spec do workspace (unitário | quebrado) ----

// lintL7SpecAggregates — L7 sobre os agregados técnicos do WORKSPACE de
// produto (specs/, FORMAT §7.7.1): reconhece AS DUAS formas canônicas —
// unitário (`specs/DATA_MODEL.md`) e quebrado (`specs/data_model/` com
// INDEX.md OBRIGATÓRIO, o mestre: visão global + índice dos escopos) —,
// avisa quando o unitário passa do limiar de quebra (DataModelSplitLines)
// e emite o aviso PRECOCE (L7-size) para os agregados candidatos sem
// gramática quebrada declarada (aggregateSizeWarns). Roda em qualquer
// workspace (kit e produto); sem specs/ é silencioso.
func lintL7SpecAggregates(rep *Report, ws string) {
	for _, agg := range specAggregates {
		dir := filepath.Join(ws, "specs", filepath.FromSlash(agg.Dir))
		if dirExists(dir) && !fileExists(filepath.Join(dir, "INDEX.md")) {
			rep.Findings = append(rep.Findings, Finding{
				Check: "L7", Severity: "error", File: "specs/" + agg.Dir,
				Description: "agregado quebrado sem INDEX.md — no formato quebrado o mestre é o INDEX (visão global, ER agregado e índice dos escopos; o leitor começa por ele)",
				Fix:         "crie o INDEX — veja gm-datamodel Split (procedimento de quebra por micro-contexto)",
			})
		}
	}
	// DATA_MODEL — gramática quebrada JÁ declarada (FORMAT §7.7.1): o
	// aviso aponta o Split do gm-datamodel
	if n, ok := lineCount(filepath.Join(ws, "specs", "DATA_MODEL.md")); ok && n > DataModelSplitLines {
		rep.Findings = append(rep.Findings, Finding{
			Check: "L7-size", Severity: "warning", File: "specs/DATA_MODEL.md",
			Description: fmt.Sprintf("agregado unitário com %d linhas (limiar de quebra: %d) — contexto inteiro carregado a cada RN deixa de pagar o preço", n, DataModelSplitLines),
			Fix:         "considere quebrar por micro-contexto (specs/data_model/ com INDEX.md + escopos auto-contidos) — veja gm-datamodel Split",
		})
	}
	// Candidatos PRECOCES — sem gramática quebrada declarada ainda: o aviso
	// só recomenda PLANEJAR a quebra (gramática será definida no FORMAT
	// quando o volume justificar). 1º path existente do agregado vale.
	for _, a := range aggregateSizeWarns {
		for _, p := range a.Paths {
			n, ok := lineCount(filepath.Join(ws, filepath.FromSlash(p)))
			if !ok {
				continue
			}
			if n > a.Lines {
				rep.Findings = append(rep.Findings, Finding{
					Check: "L7-size", Severity: "warning", File: p,
					Description: fmt.Sprintf("agregado candidato à quebra com %d linhas (limiar de aviso: %d) — crescendo sem gramática quebrada declarada", n, a.Lines),
					Fix:         "considere planejar a quebra por micro-contexto deste agregado com o gm-doc — a gramática quebrada deste artefato ainda não está declarada no FORMAT (será definida quando o volume justificar)",
				})
			}
			break
		}
	}
}

// lineCount — nº de linhas do arquivo (split em \n com newline final
// normalizado — mesma contagem de `wc -l`); segundo retorno false =
// arquivo ausente/ilegível.
func lineCount(p string) (int, bool) {
	data, err := os.ReadFile(p)
	if err != nil {
		return 0, false
	}
	text := strings.TrimSuffix(string(data), "\n")
	if text == "" {
		return 0, true
	}
	return len(strings.Split(text, "\n")), true
}

// ---- L8 — conformidade estrutural definitivo ↔ template ----

// lintL8 valida o MÍNIMO estrutural declarado (l8*) do definitivo contra o
// template do embed — NUNCA conteúdo (o definitivo evolui e ganha seções;
// seção extra = OK, nunca finding; template ganhar seção nova não obriga
// definitivos antigos). Definitivo ausente (máquina nova antes do setup) =
// skip silencioso. Template ausente no embed = erro de ambiente (finding).
func lintL8(rep *Report, configRoot string) {
	for _, p := range l8Pairs {
		if _, terr := fs.ReadFile(golmet.CoreDataFS(), "coredata/"+p.Template); terr != nil {
			rep.Findings = append(rep.Findings, Finding{
				Check: "L8", Severity: "error", File: p.Template,
				Description: "erro de ambiente: template ausente no embed do kit (" + p.Template + ")",
				Fix:         "restaure o template no coredata do kit — o L8 não pode validar sem a referência",
			})
			continue
		}
		finalPath := filepath.Join(configRoot, filepath.FromSlash(p.Final))
		data, rerr := os.ReadFile(finalPath)
		if rerr != nil {
			continue // definitivo ausente (máquina nova antes do setup) — skip silencioso
		}
		final := string(data)
		syncFix := "definitivo distanciou-se da estrutura do template — sincronize a seção (o conteúdo pode divergir, a estrutura não); re-emita o esqueleto com `golmet template` se a perda foi acidental"
		if p.Min == nil {
			// caso especial AGENTS.md: blocos marcadores abertos E fechados —
			// a casa usa DUAS convenções: token único repetido
			// (`<!-- x -->` ... `<!-- x -->`) ou par com barra (`<!-- /x -->`).
			// Estrutura OK = ≥2 delimitadores do bloco (aberto e fechado).
			for _, b := range l8MarkerBlocks {
				tokens := strings.Count(final, "<!-- "+b+" -->") +
					strings.Count(final, "<!-- /"+b+" -->")
				if tokens < 2 {
					rep.Findings = append(rep.Findings, Finding{
						Check: "L8", Severity: "warning", File: p.Final,
						Description: "bloco marcador ausente ou não fechado no definitivo: " + b + " (" + fmt.Sprint(tokens) + " delimitador(es) de 2)",
						Fix:         syncFix,
					})
				}
			}
			for _, s := range l8AGENTSMinSections {
				if !strings.Contains(final, s) {
					rep.Findings = append(rep.Findings, Finding{
						Check: "L8", Severity: "warning", File: p.Final,
						Description: "seção canônica ausente no definitivo: \"" + s + "\"",
						Fix:         syncFix,
					})
				}
			}
			continue
		}
		for _, s := range p.Min {
			if !strings.Contains(final, s) {
				rep.Findings = append(rep.Findings, Finding{
					Check: "L8", Severity: "warning", File: p.Final,
					Description: "seção/campo canônico ausente no definitivo: \"" + s + "\"",
					Fix:         syncFix,
				})
			}
		}
	}
}

// ---- L9 — schema do registry de stack-templates (config do engine) ----

// lintL9 valida o schema mínimo dos registries de boilerplates no config
// do engine (M14 determinismo do bootstrap; M15 pack-first):
// (1) o kit-canônico materializado (`profile/stack-templates.yaml`) e
// (2) os registries da org IN-PLACE nos packs (`packs/*/rules/stack-templates.yaml`
// — fonte primária do gm-stack-boilerplate), ambos com o MESMO schema:
// (a) YAML parseável; (b) `boilerplate_org` presente (placeholder `<...>`
// aceito); (c) cada template com `detect` não-vazio; (d) chave
// `boilerplate` presente (null aceito — stack sem boilerplate ainda);
// (e) `description` presente quando `boilerplate` é não-null.
// Severidade error. Arquivo ausente = skip silencioso (máquina nova antes
// do install; pack sem registry é estado legítimo). Check SEPARADO do L7
// de propósito: L7 é estrutura de DIRETÓRIOS declarativa; L9 é schema de
// CONTEÚDO de arquivos específicos — separados, o by_check do gm_lint
// diagnostica com precisão (contrato documentado no FORMAT §9.1 e §2.2).
func lintL9(rep *Report, configRoot string) {
	// (1) registry kit-canônico materializado no config.
	const rel = "profile/stack-templates.yaml"
	if data, err := os.ReadFile(filepath.Join(configRoot, filepath.FromSlash(rel))); err == nil {
		l9CheckRegistry(rep, rel, data,
			"restaure o registry a partir do kit-canônico (`coredata/rules/stack-templates.yaml` — materializado pelo install FULL) ou corrija o schema local",
			"corrija a sintaxe YAML do registry de boilerplates (fonte kit-canônica: coredata/rules/stack-templates.yaml)")
	}
	// (2) registries de pack (registry da org in-place — M15 pack-first):
	// mesmo schema do kit; a remediação aponta o working copy do pack.
	packsDir := filepath.Join(configRoot, "packs")
	entries, err := os.ReadDir(packsDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		rel := "packs/" + e.Name() + "/rules/stack-templates.yaml"
		data, err := os.ReadFile(filepath.Join(configRoot, filepath.FromSlash(rel)))
		if err != nil {
			continue // pack sem registry = skip silencioso (estado legítimo)
		}
		l9CheckRegistry(rep, rel, data,
			"corrija o schema no registry do pack (working copy `packs/"+e.Name()+"/` — repo da org) e publique (`golmet pack publish`)",
			"corrija a sintaxe YAML do registry de boilerplates do pack (working copy: packs/"+e.Name()+"/rules/stack-templates.yaml)")
	}
}

// l9CheckRegistry valida UM registry de boilerplates (kit materializado em
// profile/ ou org in-place no pack — mesmo schema). fixRoot/fixParse
// personalizam a remediação pela origem do arquivo.
func l9CheckRegistry(rep *Report, rel string, data []byte, fixRoot, fixParse string) {
	flag := func(line int, desc, fix string) {
		rep.Findings = append(rep.Findings, Finding{
			Check: "L9", Severity: "error", File: rel, Line: line,
			Description: desc, Fix: fix,
		})
	}
	var root yaml.Node
	if uerr := yaml.Unmarshal(data, &root); uerr != nil {
		flag(0, "stack-templates.yaml não é YAML parseável: "+uerr.Error(), fixParse)
		return
	}
	if root.Kind == 0 || len(root.Content) == 0 {
		flag(0, "stack-templates.yaml está vazio (sem mapeamento raiz)", fixRoot)
		return
	}
	doc := root.Content[0]
	if doc.Kind != yaml.MappingNode {
		flag(doc.Line, "stack-templates.yaml não tem mapeamento raiz (esperado: `boilerplate_org` + `templates`)", fixRoot)
		return
	}
	// mappingValue devolve o value node da chave name (nil se ausente) —
	// em mapping YAML, Content é [k1, v1, k2, v2, ...].
	mappingValue := func(n *yaml.Node, name string) *yaml.Node {
		if n == nil || n.Kind != yaml.MappingNode {
			return nil
		}
		for i := 0; i+1 < len(n.Content); i += 2 {
			if n.Content[i].Value == name {
				return n.Content[i+1]
			}
		}
		return nil
	}
	isEmptyScalar := func(n *yaml.Node) bool {
		return n == nil || n.Tag == "!!null" || strings.TrimSpace(n.Value) == ""
	}
	// (b) boilerplate_org presente — placeholder `<...>` é aceito (IMP-105:
	// a org real vem do client-profile, o registry é organização-neutro).
	if org := mappingValue(doc, "boilerplate_org"); isEmptyScalar(org) {
		line := 0
		if org != nil {
			line = org.Line
		}
		flag(line, "campo `boilerplate_org` ausente ou vazio no registry de stack-templates",
			"declare `boilerplate_org` (org donosa dos boilerplates — resolve a partir do client-profile, IMP-105; placeholder `<...>` é aceito)")
	}
	// templates: mapeamento obrigatório — um template por stack.
	templates := mappingValue(doc, "templates")
	if templates == nil || templates.Tag == "!!null" {
		line := 0
		if templates != nil {
			line = templates.Line
		}
		flag(line, "seção `templates` ausente ou vazia no registry de stack-templates",
			"declare a seção `templates:` com um template por stack (detect + boilerplate) — "+fixRoot)
		return
	}
	if templates.Kind != yaml.MappingNode {
		flag(templates.Line, "seção `templates` não é um mapeamento (esperado: nome da stack → campos do template)", fixRoot)
		return
	}
	for i := 0; i+1 < len(templates.Content); i += 2 {
		stack := templates.Content[i].Value
		line := templates.Content[i].Line
		t := templates.Content[i+1]
		if t.Kind != yaml.MappingNode {
			flag(t.Line, "template `"+stack+"` não é um mapeamento de campos (esperado: detect, boilerplate, …)", fixRoot)
			continue
		}
		// (c) detect: sequência não-vazia de padrões não-vazios.
		detect := mappingValue(t, "detect")
		detectOK := detect != nil && detect.Kind == yaml.SequenceNode && len(detect.Content) > 0
		if detectOK {
			for _, pat := range detect.Content {
				if isEmptyScalar(pat) {
					detectOK = false
					break
				}
			}
		}
		if !detectOK {
			flag(line, "template `"+stack+"` sem `detect` não-vazio (array de padrões de arquivo para detectar a stack)",
				"declare `detect: [<glob>, …]` (ex.: [\"*.csproj\", \"*.sln\"]) no template `"+stack+"`")
		}
		// (d) chave `boilerplate` presente — null aceito (placeholder de
		// stack cujo boilerplate ainda não existe na org).
		bp := mappingValue(t, "boilerplate")
		if bp == nil {
			flag(line, "template `"+stack+"` sem a chave `boilerplate`",
				"declare `boilerplate: <boilerplate_org>/<repo>` ou `boilerplate: null` (stack sem boilerplate criado na org ainda) no template `"+stack+"`")
			continue
		}
		// (e) boilerplate definido ⇒ description presente.
		if !isEmptyScalar(bp) {
			if desc := mappingValue(t, "description"); isEmptyScalar(desc) {
				flag(line, "template `"+stack+"` com `boilerplate` definido exige `description`",
					"declare a `description` do boilerplate no template `"+stack+"` (o que o boilerplate traz)")
			}
		}
	}
}

// ---- helpers ----

// registryIDs — ids do registry (workspace em kit mode; senão o embarcado —
// o lint do config/packs valida fantasmas contra o registry do binário).
// ---- L11 - diagramas Mermaid (warning) ----

// mermaidTypes - tipos de diagrama conhecidos (primeira linha significativa
// do bloco ```mermaid``` - lista declarada; fora dela = finding).
var mermaidTypes = []string{
	"flowchart", "graph", "sequenceDiagram", "erDiagram", "classDiagram",
	"stateDiagram", "stateDiagram-v2", "gantt", "pie", "mindmap", "journey",
	"timeline", "gitGraph", "quadrantChart", "requirementDiagram",
	"sankey-beta", "xychart-beta",
	"C4Context", "C4Container", "C4Component", "C4Dynamic", "C4Deployment",
}

// lintL11Mermaid valida blocos ```mermaid ... ``` dos *.md dos alvos com
// heurística determinística SEM parser/Node (o kit é standalone): tipo de
// diagrama válido na 1ª linha, delimitadores [](){} balanceados por linha
// (fora de aspas), aspas em pares, seta com origem E destino, subgraph/end
// pareados, fence fechado. Severidade warning: diagrama quebrado não
// bloqueia fluxo - é qualidade de render (o mostrador exibe o código cru).
// Anti-falso-positivo (política da casa): a heurística pega os ERROS
// COMUNS - construção exótica válida passa (ex.: sequenceDiagram ->>
// não tem pontas validadas); falso-positivo é pior que falso-negativo.
// Limitação documentada no FORMAT §9.1.
func lintL11Mermaid(rep *Report, ws string, kitMDs []string, kitMode bool, configRoot string) {
	type lintMD struct {
		rel, abs string
	}
	var targets []lintMD
	add := func(rel, abs string) { targets = append(targets, lintMD{rel, abs}) }
	if kitMode {
		for _, rel := range kitMDs {
			add(rel, filepath.Join(ws, filepath.FromSlash(rel)))
		}
	}
	if configRoot != "" {
		add("AGENTS.md", filepath.Join(configRoot, "AGENTS.md"))
		for _, p := range mdFilesUnder(configRoot, "profile") {
			add(p, filepath.Join(configRoot, filepath.FromSlash(p)))
		}
		if packsRoot := filepath.Join(configRoot, "packs"); dirExists(packsRoot) {
			if entries, err := os.ReadDir(packsRoot); err == nil {
				for _, e := range entries {
					if !e.IsDir() {
						continue
					}
					for _, p := range mdFilesDeep(packsRoot, e.Name(), packWalkSkipDirs...) {
						add(p, filepath.Join(packsRoot, filepath.FromSlash(p)))
					}
				}
			}
		}
	}
	if !kitMode {
		// workspace-produto: artefatos gerados (specs/ etc.) carregam diagramas
		_ = filepath.WalkDir(ws, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				switch d.Name() {
				case ".git", "node_modules", ".golmet":
					return fs.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(p, ".md") {
				rel, _ := filepath.Rel(ws, p)
				add(filepath.ToSlash(rel), p)
			}
			return nil
		})
	}
	for _, t := range targets {
		data, rerr := os.ReadFile(t.abs)
		if rerr != nil {
			continue
		}
		lintMermaidFile(rep, t.rel, string(data))
	}
}

// lintMermaidFile extrai os blocos ```mermaid ... ``` do conteúdo e valida
// cada um (Line dos findings = absoluta no arquivo, 1-based).
func lintMermaidFile(rep *Report, rel string, content string) {
	lines := strings.Split(content, "\n")
	for i := 0; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "```mermaid" {
			continue
		}
		startLine := i + 1
		var body []string
		closed := false
		for j := i + 1; j < len(lines); j++ {
			if strings.TrimSpace(lines[j]) == "```" {
				closed = true
				i = j
				break
			}
			body = append(body, lines[j])
		}
		if !closed {
			rep.Findings = append(rep.Findings, Finding{
				Check: "L11", Severity: "warning", File: rel, Line: startLine,
				Description: "bloco ```mermaid``` aberto sem fechamento (fence de fechamento ausente)",
				Fix:         "feche o bloco com ``` - o render exibe o código cru quando o Mermaid está quebrado",
			})
			return
		}
		lintMermaidBlock(rep, rel, startLine, body)
	}
}

// lintMermaidBlock aplica as heurísticas a um bloco (linhas do corpo, sem
// os fences). Findings apontam a linha absoluta do problema.
func lintMermaidBlock(rep *Report, rel string, startLine int, body []string) {
	find := func(line int, desc string) {
		rep.Findings = append(rep.Findings, Finding{
			Check: "L11", Severity: "warning", File: rel, Line: line,
			Description: desc,
			Fix:         "corrija a sintaxe do diagrama - o render exibe o código cru quando o Mermaid está quebrado (tipo válido na 1ª linha, delimitadores balanceados, setas com origem e destino, subgraph/end pareados)",
		})
	}
	if len(body) == 0 {
		find(startLine, "bloco ```mermaid``` vazio")
		return
	}
	// (1) tipo de diagrama válido na primeira linha significativa
	first := ""
	for _, l := range body {
		t := strings.TrimSpace(l)
		if t == "" || strings.HasPrefix(t, "%%") {
			continue
		}
		first = t
		break
	}
	firstWord := strings.Fields(first)[0]
	if !containsStr(mermaidTypes, firstWord) {
		find(startLine, "primeira linha do bloco mermaid não inicia com um tipo de diagrama conhecido: \""+
			firstWord+"\" (esperado ex.: flowchart, sequenceDiagram, erDiagram, stateDiagram, gantt, pie, mindmap, C4*)")
		return // sem tipo válido, as demais heurísticas são ruído
	}
	// (2..5) corpo: aspas, setas, subgraph/end — os delimitadores são
	// agregados POR BLOCO (o padrão ER de atributos abre `{` numa linha e
	// fecha em outra: julgar por linha quebra sintaxe válida — falso-
	// positivo real pego na 1ª execução sobre o gm-datamodel)
	labelRe := regexp.MustCompile(`\|[^|\n]*\|`)
	openerRe := regexp.MustCompile(`^(subgraph|alt|opt|loop|par|critical|break|rect)(\s|$)`)
	openers, ends := 0, 0
	balance := map[string]int{} // por tipo: aberturas - fechamentos no bloco
	for j, raw := range body {
		line := startLine + 1 + j
		t := strings.TrimSpace(raw)
		if t == "" || strings.HasPrefix(t, "%%") {
			continue
		}
		if openerRe.MatchString(t) {
			openers++
		}
		if t == "end" {
			ends++
		}
		// aspas em pares (número par de " fora de comentário)
		if n := strings.Count(t, `"`); n%2 == 1 {
			find(line, "aspas desbalanceadas na linha do diagrama ("+fmt.Sprint(n)+" aspas)")
			continue
		}
		// delimitadores: conteúdo dentro de aspas removido (pode conter
		// [ ] ( ) legítimos em texto), agregado no saldo do bloco.
		// erDiagram: os tokens de CARDINALIDADE (`}o--||`, `|o--o{`...)
		// usam chaves/pipe como sintaxe consagrada - removidos antes de
		// contar (falso-positivo real pego na 1ª execução sobre o
		// gm-datamodel)
		if firstWord == "erDiagram" {
			cardRe := regexp.MustCompile(`(?:\}o|\}|\|o|\|\||o\||o\{|\|\{)\s*--\s*(?:\}o|\}|\|o|\|\||o\||o\{|\|\{)`)
			t = cardRe.ReplaceAllString(t, "--")
		}
		noStr := regexp.MustCompile(`"[^"]*"`).ReplaceAllString(t, "")
		for _, pair := range []struct {
			open, close rune
			name        string
		}{
			{'[', ']', "colchetes"}, {'(', ')', "parênteses"}, {'{', '}', "chaves"},
		} {
			balance[pair.name] += strings.Count(noStr, string(pair.open)) -
				strings.Count(noStr, string(pair.close))
		}
		// seta com origem E destino (labels |...| removidos; sequenceDiagram
		// ->> e construções exóticas NÃO validadas - anti-falso-positivo)
		for _, arrow := range []string{"-->", "-.->", "==>"} {
			idx := strings.Index(t, arrow)
			if idx < 0 {
				continue
			}
			clean := labelRe.ReplaceAllString(t, "")
			idx2 := strings.Index(clean, arrow)
			if idx2 < 0 {
				continue // a seta estava dentro de um label
			}
			before := strings.TrimSpace(clean[:idx2])
			after := ""
			for _, a := range []string{"-->", "-.->", "==>"} {
				if li := strings.LastIndex(clean, a); li >= 0 {
					after = strings.TrimSpace(clean[li+len(a):])
					break
				}
			}
			if before == "" || after == "" {
				find(line, "seta sem origem ou sem destino no diagrama (antes: \""+
					before+"\" / depois: \""+after+"\")")
			}
			break // uma validação de seta por linha basta
		}
	}
	// delimitadores: o SALDO do bloco inteiro (abertura e fechamento podem
	// estar em linhas diferentes — padrão ER de atributos)
	var dangle []string
	for _, name := range []string{"colchetes", "parênteses", "chaves"} {
		if balance[name] != 0 {
			dangle = append(dangle, fmt.Sprintf("%s (%+d)", name, balance[name]))
		}
	}
	if len(dangle) > 0 {
		find(startLine+len(body), "delimitadores desbalanceados no diagrama: "+
			strings.Join(dangle, ", "))
	}
	// openers (subgraph no flowchart; alt/opt/loop/par/critical/break/rect
	// no sequenceDiagram) vs end — o par universal do Mermaid
	if openers != ends {
		find(startLine+len(body), "aberturas/end desbalanceados no diagrama ("+
			fmt.Sprint(openers)+" opener / "+fmt.Sprint(ends)+" end)")
	}
}

// ---- L12 - encoding e texto quebrado (error; BOM = warning) ----

// mojibakeRe - bigramas de DUPLA CODIFICAÇÃO: o UTF-8 foi lido como
// Latin-1/Windows-1252 e regravado — os acentos PT viram "Ã©", "Ã£",
// "Ã§"... (o texto é UTF-8 válido, mas ilegível).
var mojibakeRe = regexp.MustCompile(`Ã[\x{0080}-\x{00FF}]`)

// lintL12Encoding valida o ENCODING dos arquivos de conteúdo — texto
// quebrado é invisível na estrutura e só quebra no render:
// (1) UTF-8 inválido (bytes Latin-1/Windows-1252 salvos crus) = error;
// (2) replacement character U+FFFD = error (o conteúdo JÁ está corrompido
// no arquivo); (3) dupla codificação (bigramas "Ã©/Ã£/Ã§...") = error;
// (4) BOM UTF-16 em .md/.yaml = error; (5) BOM UTF-8 = warning (IMP-096:
// sem BOM). Alvos: os arquivos de texto de conteúdo (*.md + *.yaml) —
// evals ficam fora (fixtures podem conter conteúdo proposital).
func lintL12Encoding(rep *Report, ws string, kitMode bool, configRoot string) {
	type lintFile struct{ rel, abs string }
	var targets []lintFile
	add := func(rel, abs string) { targets = append(targets, lintFile{rel, abs}) }
	skipDir := func(name string) bool {
		return name == ".git" || name == "node_modules" || name == ".golmet"
	}
	if kitMode {
		// docs vivos do kit — evals fora (fixtures com conteúdo proposital)
		for _, sub := range []string{"coredata/sources", "coredata/rules", "coredata/templates", "docs"} {
			for _, p := range mdFilesUnder(ws, sub) {
				add(p, filepath.Join(ws, filepath.FromSlash(p)))
			}
		}
		if out, err := filepath.Glob(filepath.Join(ws, "*.md")); err == nil {
			for _, p := range out {
				rel, _ := filepath.Rel(ws, p)
				add(filepath.ToSlash(rel), p)
			}
		}
	}
	if configRoot != "" {
		add("AGENTS.md", filepath.Join(configRoot, "AGENTS.md"))
		if packsRoot := filepath.Join(configRoot, "packs"); dirExists(packsRoot) {
			if entries, err := os.ReadDir(packsRoot); err == nil {
				for _, e := range entries {
					if !e.IsDir() {
						continue
					}
					for _, p := range mdFilesDeep(packsRoot, e.Name(), packWalkSkipDirs...) {
						add(p, filepath.Join(packsRoot, filepath.FromSlash(p)))
					}
				}
			}
		}
	}
	if !kitMode {
		_ = filepath.WalkDir(ws, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if skipDir(d.Name()) {
					return fs.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(p, ".md") || strings.HasSuffix(p, ".yaml") {
				rel, _ := filepath.Rel(ws, p)
				add(filepath.ToSlash(rel), p)
			}
			return nil
		})
	}
	// profile/ do config + rules/ de packs também carregam .yaml/.md (o walk
	// por pack acima cobre os packs; o profile do config abaixo)
	if configRoot != "" {
		for _, p := range mdFilesUnder(configRoot, "profile") {
			add(p, filepath.Join(configRoot, filepath.FromSlash(p)))
		}
	}

	for _, t := range targets {
		data, rerr := os.ReadFile(t.abs)
		if rerr != nil {
			continue
		}
		l12CheckFile(rep, t.rel, data)
	}
}

// l12CheckFile aplica as heurísticas de encoding a um arquivo (dados crus).
func l12CheckFile(rep *Report, rel string, data []byte) {
	l12Find := func(sev, desc, fix string) {
		rep.Findings = append(rep.Findings, Finding{
			Check: "L12", Severity: sev, File: rel, Line: 0,
			Description: desc, Fix: fix,
		})
	}
	utf16Fix := "regravar o arquivo como UTF-8 (sem BOM) — o conteúdo UTF-16 não renderiza nos mostradores"
	if len(data) >= 2 && ((data[0] == 0xFF && data[1] == 0xFE) || (data[0] == 0xFE && data[1] == 0xFF)) {
		l12Find("error", "arquivo em UTF-16 (BOM UTF-16 detectado) — o conteúdo não é UTF-8", utf16Fix)
		return
	}
	bom := len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF
	body := data
	if bom {
		body = data[3:]
	}
	if !utf8.Valid(body) {
		l12Find("error", "arquivo NÃO é UTF-8 válido — bytes de Latin-1/Windows-1252 salvos crus (o render decodifica errado)",
			"regravar como UTF-8 (sem BOM) — converter o encoding de origem para UTF-8 antes de gravar")
		return // sem UTF-8 válido, as heurísticas de texto são ruído
	}
	if bom {
		l12Find("warning", "BOM UTF-8 presente (o padrão do kit é UTF-8 SEM BOM — IMP-096)",
			"regravar o arquivo sem BOM")
	}
	s := string(body)
	if strings.Contains(s, "\uFFFD") {
		l12Find("error", "replacement character (U+FFFD) presente — o conteúdo JÁ está corrompido no arquivo (a decodificação de origem perdeu bytes)",
			"recuperar o texto da fonte original e regravar como UTF-8 — o replacement não é reversível no arquivo atual")
	}
	if m := mojibakeRe.FindString(s); m != "" {
		l12Find("error", "dupla codificação detectada (ex.: \""+m+"\" — UTF-8 lido como Latin-1/Windows-1252 e regravado: o acento virou dois caracteres ilegíveis)",
			"regravar o arquivo com a decodificação correta (converter de Latin-1 para UTF-8 — o texto original é recuperável")
	}
}

// ---- L13 - âncoras de idioma (warning) ----

// lintL13Anchors valida as âncoras de navegação dos docs PT-BR: um link de
// um *.pt-br.md deve apontar o PAR PT do destino quando o par existe no
// disco (o achado real: o README.pt-br navegava para os docs EN — o link
// resolvia no L1, mas ensinava o idioma errado). Só docs PT são validados:
// o EN (base) pode linkar o par PT legitimamente (badge de alternância,
// cross-links de corpo). Severidade warning: a navegação FUNCIONA — é
// idioma, não estrutura. Anti-falso-positivo: o par PT DEVE existir no
// disco (senão o link EN é o único destino possível); âncoras (#seção)
// mantêm o fragmento e a remediação avisa para conferir a âncora no idioma.
func lintL13Anchors(rep *Report, ws string, kitMode bool, configRoot string) {
	var targets []string
	add := func(abs string) { targets = append(targets, abs) }
	skipDir := func(name string) bool {
		return name == ".git" || name == "node_modules" || name == ".golmet" ||
			name == "evals" || name == "testdata" || name == "coredata"
	}
	if kitMode {
		_ = filepath.WalkDir(ws, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if skipDir(d.Name()) {
					return fs.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(p, ".pt-br.md") {
				add(p)
			}
			return nil
		})
	}
	if configRoot != "" {
		if packsRoot := filepath.Join(configRoot, "packs"); dirExists(packsRoot) {
			_ = filepath.WalkDir(packsRoot, func(p string, d fs.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if !d.IsDir() && strings.HasSuffix(p, ".pt-br.md") {
					add(p)
				}
				return nil
			})
		}
	}
	if !kitMode {
		_ = filepath.WalkDir(ws, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if skipDir(d.Name()) {
					return fs.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(p, ".pt-br.md") {
				add(p)
			}
			return nil
		})
	}

	linkRe := regexp.MustCompile(`\]\(([^\)\s]+)(#[^\)\s]*)?\)`)
	for _, f := range targets {
		data, rerr := os.ReadFile(f)
		if rerr != nil {
			continue
		}
		dir := filepath.Dir(f)
		lines := strings.Split(string(data), "\n")
		for li, raw := range lines {
			for _, m := range linkRe.FindAllStringSubmatch(raw, -1) {
				path := m[1]
				frag := m[2]
				if path == "" || strings.Contains(path, "://") || path == "#" {
					continue
				}
				if strings.HasSuffix(path, ".pt-br.md") {
					continue // já aponta o par PT
				}
				abs := filepath.Clean(filepath.Join(dir, filepath.FromSlash(path)))
				if !fileExists(abs) {
					continue // ref morta é do L1
				}
				absPt := filepath.FromSlash(strings.TrimSuffix(path, filepath.Ext(path)) + ".pt-br.md")
				absPt = filepath.Clean(filepath.Join(dir, absPt))
				if !fileExists(absPt) {
					continue // o par PT não existe - o EN é o único destino
				}
				fix := "troque o link para o par PT-BR: `" + absPt + "`"
				if frag != "" {
					fix += " — e confira a âncora `#...` no idioma (a âncora da seção muda com o idioma)"
				}
				rep.Findings = append(rep.Findings, Finding{
					Check: "L13", Severity: "warning", File: f,
					Line:        li + 1,
					Description: "âncora de doc PT-BR aponta o destino em EN — e o par PT-BR existe no disco (\"" + path + "\")",
					Fix:         fix,
				})
			}
		}
	}
}

// ---- L14 - front matter YAML parseável (error) ----

// yamlLineRe extrai a linha do erro de parse reportada pelo yaml.v3
// ("yaml: line N: ...") — quando presente, vira o Line do finding.
var yamlLineRe = regexp.MustCompile(`yaml: line (\d+):`)

// lintL14FrontMatter valida o FRONT MATTER YAML dos *.md de conteúdo: um
// bloco delimitado por `---` na primeira linha deve parsear como YAML
// genérico (sem schema de componente — skills/agents têm suas regras
// próprias de front matter). O achado real (IMP-013): `titulo: X: Y` sem
// aspas passou por todos os checks e só quebrou no consumidor (editor) —
// o lint torna isso audível. Arquivo sem front matter (primeira linha ≠
// ---) = skip silencioso; bloco vazio entre delimitadores = parse válido.
// Alvos: os mesmos *.md do L12 (mesmo walker: kit/config/packs; evals e
// testdata fora no walk de runtime — fixtures podem conter conteúdo
// proposital).
func lintL14FrontMatter(rep *Report, ws string, kitMode bool, configRoot string) {
	type lintFile struct{ rel, abs string }
	var targets []lintFile
	add := func(rel, abs string) { targets = append(targets, lintFile{rel, abs}) }
	skipDir := func(name string) bool {
		return name == ".git" || name == "node_modules" || name == ".golmet" ||
			name == "evals" || name == "testdata"
	}
	if kitMode {
		// docs vivos do kit — evals fora (fixtures com conteúdo proposital)
		for _, sub := range []string{"coredata/sources", "coredata/rules", "coredata/templates", "docs"} {
			for _, p := range mdFilesUnder(ws, sub) {
				add(p, filepath.Join(ws, filepath.FromSlash(p)))
			}
		}
		if out, err := filepath.Glob(filepath.Join(ws, "*.md")); err == nil {
			for _, p := range out {
				rel, _ := filepath.Rel(ws, p)
				add(filepath.ToSlash(rel), p)
			}
		}
	}
	if configRoot != "" {
		add("AGENTS.md", filepath.Join(configRoot, "AGENTS.md"))
		if packsRoot := filepath.Join(configRoot, "packs"); dirExists(packsRoot) {
			if entries, err := os.ReadDir(packsRoot); err == nil {
				for _, e := range entries {
					if !e.IsDir() {
						continue
					}
					for _, p := range mdFilesDeep(packsRoot, e.Name(), packWalkSkipDirs...) {
						add(p, filepath.Join(packsRoot, filepath.FromSlash(p)))
					}
				}
			}
		}
		for _, p := range mdFilesUnder(configRoot, "profile") {
			add(p, filepath.Join(configRoot, filepath.FromSlash(p)))
		}
	}
	if !kitMode {
		_ = filepath.WalkDir(ws, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if skipDir(d.Name()) {
					return fs.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(p, ".md") {
				rel, _ := filepath.Rel(ws, p)
				add(filepath.ToSlash(rel), p)
			}
			return nil
		})
	}

	for _, t := range targets {
		data, rerr := os.ReadFile(t.abs)
		if rerr != nil {
			continue
		}
		l14CheckFile(rep, t.rel, data)
	}
}

// l14CheckFile extrai o front matter de um arquivo (dados crus): primeira
// linha `---` até a linha de fechamento `---` (a primeira após a abertura),
// parseado como YAML genérico. O `---` solto no corpo (sem abertura na
// primeira linha) é horizontal rule markdown — não é front matter. CRLF é
// tolerado na comparação dos delimitadores (configs de engine em Windows).
func l14CheckFile(rep *Report, rel string, data []byte) {
	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 || strings.TrimSuffix(lines[0], "\r") != "---" {
		return // sem front matter — skip silencioso, não é erro
	}
	closeIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSuffix(lines[i], "\r") == "---" {
			closeIdx = i
			break
		}
	}
	if closeIdx < 0 {
		return // abertura sem fechamento — não é bloco de front matter
	}
	block := strings.Join(lines[1:closeIdx], "\n")
	if strings.TrimSpace(block) == "" {
		return // bloco vazio = parse válido, sem finding
	}
	var node yaml.Node
	if uerr := yaml.Unmarshal([]byte(block), &node); uerr != nil {
		line := 0
		if m := yamlLineRe.FindStringSubmatch(uerr.Error()); m != nil {
			if n, perr := strconv.Atoi(m[1]); perr == nil {
				line = n + 1 // a linha do parser é relativa ao bloco — a abertura ocupa a linha 1 do arquivo
			}
		}
		rep.Findings = append(rep.Findings, Finding{
			Check: "L14", Severity: "error", File: rel, Line: line,
			Description: "front matter YAML inválido: " + uerr.Error(),
			Fix:         "corrija o YAML do front matter — valores com dois-pontos-e-espaço devem ser quotados com aspas duplas",
		})
	}
}

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

// orgComponentNames — componentes declarados pelas fontes de extensão da
// org no config root. Fontes: <configRoot>/registry.yaml (mesmo formato do
// registry do binário) e packs de state ativos (agents/*.md, skills/<dir>/
// com SKILL.md). Complementa registryIDs: o L6/L1 do modo config/packs
// resolve gm-* contra kit ∪ org, não só contra o registro público.
func orgComponentNames(configRoot string) []string {
	if configRoot == "" {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	add := func(n string) {
		if n == "" || seen[n] {
			return
		}
		seen[n] = true
		out = append(out, n)
	}
	// registry.yaml do config root — dois formatos em uso: multiline
	// (`- id: x`) e inline dict (`- { id: x, ... }`). O content.ParseRegistry
	// só suporta o multiline; regex local cobre os dois até o parser central
	// suportar inline dicts (extensão futura, PR separado).
	if regData, rerr := os.ReadFile(filepath.Join(configRoot, "registry.yaml")); rerr == nil {
		for _, m := range reOrgID.FindAllStringSubmatch(string(regData), -1) {
			add(m[1])
		}
	}
	entries, err := os.ReadDir(filepath.Join(configRoot, "packs"))
	if err != nil {
		return out
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		packDir := filepath.Join(configRoot, "packs", e.Name())
		if !isStatePack(packDir) {
			continue
		}
		if agentMDs, gerr := filepath.Glob(filepath.Join(packDir, "agents", "*.md")); gerr == nil {
			for _, f := range agentMDs {
				add(strings.TrimSuffix(filepath.Base(f), ".md"))
			}
		}
		if skillDirs, serr := os.ReadDir(filepath.Join(packDir, "skills")); serr == nil {
			for _, sd := range skillDirs {
				if sd.IsDir() && fileExists(filepath.Join(packDir, "skills", sd.Name(), "SKILL.md")) {
					add(sd.Name())
				}
			}
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

// packWalkSkipDirs — skips do walker kit-mode de packs (mdFilesDeep):
// a invariante "Evals ficam fora — fixtures podem conter conteúdo
// proposital" vale também DENTRO dos packs; node_modules/.golmet por
// paridade com o walker não-kit.
var packWalkSkipDirs = []string{"node_modules", ".golmet", "evals", "testdata"}

// mdFilesDeep - .md recursivo (mdFilesUnder é só 1 nível); usado para os
// escopos aninhados do L6 (skills/**, goldens de conteúdo) e para os
// walkers kit-mode de packs (L11/L12/L14, com packWalkSkipDirs). Nota de
// modelo de confiança (operação local, arquivos do próprio operador):
// leitura integral SEM teto de tamanho por design, e WalkDir não segue
// symlink de dir porém SEGUE junctions Windows como dirs — monitorar com
// packs de terceiros.
func mdFilesDeep(root, sub string, skipDirs ...string) []string {
	skip := map[string]bool{".git": true}
	for _, s := range skipDirs {
		skip[s] = true
	}
	dir := root
	if sub != "" {
		dir = filepath.Join(root, filepath.FromSlash(sub))
	}
	var out []string
	filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // dir ausente/ilegível = sem alvos, não é falha
		}
		if d.IsDir() {
			if skip[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		if rel, rerr := filepath.Rel(root, p); rerr == nil {
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	return out
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
