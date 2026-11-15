// Package doctor implementa o `golmet doctor` — checklist de saúde do
// workspace e do ambiente (FORMATO §9 soft: S2/S3/S5, §10 capacidades de
// kit, §2 zona do kit). Consulta: NUNCA escreve; exit 0 salvo workspace
// inválido (CLI sai 2).
package doctor

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/Golmet/golmet-cli/internal/adapters/fsutil"
	"github.com/Golmet/golmet-cli/internal/core/container"
	"github.com/Golmet/golmet-cli/internal/core/frontmatter"
	"github.com/Golmet/golmet-cli/internal/core/indexstate"
	"github.com/Golmet/golmet-cli/internal/core/rules"
	"github.com/Golmet/golmet-cli/internal/core/scan"
	"github.com/Golmet/golmet-cli/internal/core/tools"
)

// Status de um check: ok | warn | fail.
type Status string

const (
	OK   Status = "ok"
	Warn Status = "warn"
	Fail Status = "fail"
)

// Check é um item do checklist com remediação.
type Check struct {
	Item   string `json:"item"`
	Status Status `json:"status"`
	Detail string `json:"detail"`
	Fix    string `json:"fix,omitempty"`
}

// Options controla o doctor. O smoke do kit.jev é POR DEFAULT quando a
// key existe (health = o Jev FUNCIONA, não só "key presente"); NoNetwork
// pula o smoke (CI/offline); Network é alias de compatibilidade --network
// (o smoke já é default — sem efeito extra). EngineConfigRoot é a raiz do
// config do engine (ex.: ~/.config/opencode) para o check de estado
// (cache reconstrutível vs dado do usuário) — vazio = check não roda
// (compatibilidade total com chamadas workspace-only).
type Options struct {
	Network          bool   // alias de compatibilidade (--network) — sem efeito extra
	NoNetwork        bool   // --no-network: pula o smoke do kit.jev (CI/offline)
	EngineConfigRoot string // raiz do config do engine; vazio = check não roda
}

// Report agrega o checklist.
type Report struct {
	Workspace string `json:"workspace"`
	// Contexto do relatório (M21): o doctor é contexto-aware — o modo
	// detectado orienta quais checks se aplicam (ver ContextEngineConfig
	// e irmãs).
	Context      string               `json:"context"`
	Version      string               `json:"version"` // versão do kit servindo — diagnóstico instantâneo de binário velho (MCP/CLI --json)
	Checks       []Check              `json:"checks"`
	OKs          int                  `json:"ok"`
	Warns        int                  `json:"warns"`
	Fails        int                  `json:"fails"`
	EngineConfig *EngineConfigSection `json:"engine_config,omitempty"` // presente só quando o check rodou (EngineConfigRoot)
}

func (r *Report) add(item string, st Status, detail, fix string) {
	r.Checks = append(r.Checks, Check{Item: item, Status: st, Detail: detail, Fix: fix})
	switch st {
	case OK:
		r.OKs++
	case Warn:
		r.Warns++
	case Fail:
		r.Fails++
	}
}

// Contextos do doctor (M21): workspace-produto (checks S1-S5 + índice +
// manifest + capacidades + máquina), container de organização (checklist
// de hierarquia §2.3 + máquina) e engine-config (o dir dado É o config
// root do engine — estado INTEGRAL do config como corpo principal + só os
// checks de máquina; o config não é workspace, os checks de produto não se
// aplicam).
const (
	ContextEngineConfig     = "engine-config"
	ContextContainer        = "container"
	ContextWorkspaceProduct = "workspace-produto"
)

// Run executa o checklist. Erro de retorno = workspace inválido (CLI 2);
// os checks já coletados voltam no Report para exibição.
func Run(dir string, opt Options) (*Report, error) {
	rep := &Report{Version: indexstate.KitVersion, Context: ContextWorkspaceProduct}
	root, err := filepath.Abs(dir)
	if err != nil {
		rep.add("golmet.yaml", Fail, "caminho inválido: "+err.Error(), "informe um caminho válido")
		return rep, fmt.Errorf("workspace inválido: %w", err)
	}
	rep.Workspace = filepath.Base(root)
	if st, serr := os.Stat(root); serr != nil || !st.IsDir() {
		// dir inexistente pode ser o config root do engine apontado antes
		// do primeiro install — o contexto engine-config trata com mensagem
		// própria ("raiz inexistente — nada gerido, nada a perder")
		if opt.EngineConfigRoot != "" && samePath(root, opt.EngineConfigRoot) {
			return rep.runEngineConfigContext(root, opt)
		}
		rep.add("golmet.yaml", Fail, "workspace não encontrado: "+root, "informe um workspace existente")
		return rep, fmt.Errorf("workspace não encontrado: %s", root)
	}

	// 1. golmet.yaml — presente + parseável + format suportado
	cfgPath := filepath.Join(root, "golmet.yaml")
	cfgData, err := os.ReadFile(cfgPath)
	if err != nil {
		// M21 — contexto engine-config: o dir dado É o config root do
		// engine (sem golmet.yaml por natureza) — relatório COMPLETO do
		// estado do config; os checks de workspace-produto não se aplicam
		// (o config não é workspace — a falha inválida "0 ok, 1 falha"
		// some).
		if opt.EngineConfigRoot != "" && samePath(root, opt.EngineConfigRoot) {
			return rep.runEngineConfigContext(root, opt)
		}
		rep.add("golmet.yaml", Fail, "ausente em "+root, "rode 'golmet init'")
		return rep, fmt.Errorf("golmet.yaml ausente em %s — workspace não inicializado (rode 'golmet init')", root)
	}
	fields, perr := frontmatter.ParseBlock(string(cfgData))
	if perr != nil {
		rep.add("golmet.yaml", Fail, "ilegível: "+perr.Error(), "corrija conforme o subset §6.1")
		return rep, fmt.Errorf("golmet.yaml ilegível: %v", perr)
	}
	fv, ok := fields["format"]
	if !ok || fv.Str == "" {
		rep.add("golmet.yaml", Fail, "sem format", "adicione 'format: 1'")
		return rep, fmt.Errorf("golmet.yaml sem format (DEVE ser 1)")
	}
	if fv.Str != "1" {
		rep.add("golmet.yaml", Fail, "format "+fv.Str+" não suportado", "este kit suporta format: 1 (§12)")
		return rep, fmt.Errorf("formato %s não suportado — este kit suporta format: 1 (§12)", fv.Str)
	}
	rep.add("golmet.yaml", OK, "format: 1", "")

	// 1b. container de organização (FORMATO §2): o golmet.yaml declara
	// container: true — a raiz é um CONTAINER, não um workspace-produto.
	// Roda o checklist de hierarquia próprio + ambiente de máquina e
	// ENCERRA: os checks de workspace-produto (índice, scan de entidades,
	// zona .golmet, capacidades vendored) não se aplicam ao container —
	// ele não roda o pipeline de RNs.
	isContainer := fields["container"].Str == "true"
	gitOK := exec.Command("git", "--version").Run() == nil
	if isContainer {
		rep.Context = ContextContainer
		rep.runContainerChecks(root, container.Products(fields))
		rep.addMachineChecks(opt, gitOK, nil)
		return rep, nil
	}
	rep.Context = ContextWorkspaceProduct

	// 2. índice — presente/parseável/idade vs prosa (§8)
	idxPath := filepath.Join(root, "golmet.index.json")
	idxOK := false
	var generatedAt time.Time
	if data, rerr := os.ReadFile(idxPath); rerr != nil {
		rep.add("indice", Warn, "golmet.index.json ausente", "rode 'golmet validate' — cria o baseline")
	} else if idx, perr := parseIndexLoose(data); perr != nil {
		rep.add("indice", Warn, "golmet.index.json ilegível ("+perr.Error()+")", "rode 'golmet index' — recomposição")
	} else {
		idxOK = true
		if t, terr := time.Parse(time.RFC3339, idx.GeneratedAt); terr == nil {
			generatedAt = t
		}
		rep.add("indice", OK, "última validação boa: "+idx.GeneratedAt, "")
	}
	// idade: prosa (.md) modificada após a última validação boa (2s de
	// tolerância para precisão de filesystem)
	if idxOK && !generatedAt.IsZero() {
		newest, nerr := fsutil.NewestMDMod(root)
		if nerr == nil && newest.Sub(generatedAt) > 2*time.Second {
			rep.add("indice-idade", Warn,
				"prosa modificada após a última validação boa ("+generatedAt.Format(time.RFC3339)+")",
				"rode 'golmet validate' para reconciliar o índice")
		} else {
			rep.add("indice-idade", OK, "índice em dia com a prosa", "")
		}
	}

	// scan único para S2/S3/S5 (read-only — WriteIndex false)
	res, rerr := scan.Run(dir, scan.Options{Tolerant: true})

	// 3. worklist S2 — hash de evidência ≠ HEAD (requer git + repos)
	repos := 0
	if res != nil && res.Inventory != nil && res.Inventory.Charter != nil {
		repos = len(res.Inventory.Charter.Repos)
	}
	switch {
	case rerr != nil:
		// golmet.yaml ok mas scan falhou — improvável; não bloqueia os demais
		rep.add("worklist-s2", Warn, "scan falhou: "+rerr.Error(), "rode 'golmet validate' para o diagnóstico")
	case !gitOK:
		rep.add("worklist-s2", Warn, "git indisponível — S2 não verificável", "instale o git no PATH")
	case repos == 0:
		rep.add("worklist-s2", OK, "nenhum repo no charter §1 — S2 não aplicável", "")
	default:
		var stale []string
		for _, w := range res.Warnings {
			if w.Rule == "S2" {
				stale = append(stale, w.ID)
			}
		}
		if len(stale) == 0 {
			rep.add("worklist-s2", OK, "evidência em dia com o HEAD dos repos", "")
		} else {
			sort.Strings(stale)
			rep.add("worklist-s2", Warn,
				fmt.Sprintf("%d evidência(s) stale: %s", len(stale), strings.Join(stale, ", ")),
				"refresh dos hashes para o HEAD atual (worklist S2)")
		}
	}

	// 4. zona do kit — manifest vs skills presentes (DR-008)
	manifestSkills, merr := readManifest(root)
	skillDirs := readSkillDirs(root)
	switch {
	case merr != nil && os.IsNotExist(merr):
		rep.add("manifest", Warn, ".golmet/manifest.json ausente", "rode 'golmet update' — reinstala a zona do kit")
	case merr != nil:
		rep.add("manifest", Warn, "manifest ilegível ("+merr.Error()+")", "rode 'golmet update' — reescreve preservando modificados")
	default:
		inManifest := map[string]bool{}
		for _, s := range manifestSkills {
			inManifest[s] = true
		}
		onDisk := map[string]bool{}
		for _, s := range skillDirs {
			onDisk[s] = true
		}
		var missing, extra []string
		for _, s := range manifestSkills {
			if !onDisk[s] {
				missing = append(missing, s)
			}
		}
		for _, s := range skillDirs {
			if !inManifest[s] {
				extra = append(extra, s)
			}
		}
		if len(missing) == 0 && len(extra) == 0 {
			if len(manifestSkills) == 0 {
				rep.add("manifest", OK, "zona do kit vazia (subset BASE não instalado — válido)", "rode 'golmet update' para instalar o BASE")
			} else {
				rep.add("manifest", OK, fmt.Sprintf("%d skill(es) conforme(s) ao manifest", len(manifestSkills)), "")
			}
		} else {
			detail := ""
			if len(missing) > 0 {
				detail += "no manifest mas ausente em disco: " + strings.Join(missing, ", ")
			}
			if len(extra) > 0 {
				if detail != "" {
					detail += "; "
				}
				detail += "em disco mas ausente do manifest: " + strings.Join(extra, ", ")
			}
			rep.add("manifest", Warn, detail, "rode 'golmet update' — reescreve a zona preservando modificados (DR-008)")
		}
	}

	// 5. requires das skills vs capacidades detectadas (S3/S5) — apresentação
	// AGREGADA por capacidade: skills com degradação DECLARADA para a
	// capacidade ausente viram 1 linha por capacidade (a degradação já
	// aceita é o caso esperado em contexto CLI); skills SEM degradação
	// declarada mantêm aviso individual. Semântica S3 inalterada — só a
	// apresentação do doctor.
	skills := rules.ReadSkills(root)
	caps := rules.DetectEngine(root)
	type capMissing struct {
		skills   []string        // nomes ordenados
		strategy map[string]bool // estratégias de degradação declaradas
	}
	aggregated := map[string]*capMissing{}
	var individual []struct {
		skill   string
		missing []string
	}
	capsOK := true
	for _, sk := range skills {
		if sk.RequiresErr {
			rep.add("capacidades", Warn, sk.Name+": requires ilegível", "declara 'requires: [cap, ...]' no front matter")
			capsOK = false
			continue
		}
		var missing []string
		var degraded []string
		for _, r := range sk.Requires {
			if strings.HasPrefix(r, "kit.") {
				continue // capacidades de kit têm check próprio (kit.jev/kit.graph)
			}
			if !rules.ValidCapability(r) || !caps[r] {
				missing = append(missing, r)
				if sk.DegradeFor(r) != "" {
					degraded = append(degraded, r)
				}
			}
		}
		if len(missing) == 0 {
			continue
		}
		// capacidades COM degradação declarada → agregação por capacidade
		for _, cap := range degraded {
			cm := aggregated[cap]
			if cm == nil {
				cm = &capMissing{strategy: map[string]bool{}}
				aggregated[cap] = cm
			}
			cm.skills = append(cm.skills, sk.Name)
			cm.strategy[sk.DegradeFor(cap)] = true
		}
		// capacidades SEM degradação declarada → aviso individual (por skill)
		var undegraded []string
		for _, m := range missing {
			isDegraded := false
			for _, d := range degraded {
				if d == m {
					isDegraded = true
					break
				}
			}
			if !isDegraded {
				undegraded = append(undegraded, m)
			}
		}
		if len(undegraded) > 0 {
			individual = append(individual, struct {
				skill   string
				missing []string
			}{skill: sk.Name, missing: undegraded})
		}
		capsOK = false
	}
	// linhas agregadas — 1 por capacidade, ordenada; nomes truncados em 6
	var capsSorted []string
	for cap := range aggregated {
		capsSorted = append(capsSorted, cap)
	}
	sort.Strings(capsSorted)
	for _, cap := range capsSorted {
		cm := aggregated[cap]
		sort.Strings(cm.skills)
		names := cm.skills
		shown := names
		extra := 0
		if len(names) > 6 {
			shown = names[:6]
			extra = len(names) - 6
		}
		strategies := make([]string, 0, len(cm.strategy))
		for s := range cm.strategy {
			strategies = append(strategies, s)
		}
		sort.Strings(strategies)
		detail := fmt.Sprintf("%d skill(es) usam %s — ausente no engine detectado (degradação declarada: %s)",
			len(names), cap, strings.Join(strategies, ", "))
		summary := strings.Join(shown, ", ")
		if extra > 0 {
			summary += fmt.Sprintf(", … +%d", extra)
		}
		detail += " — skills: " + summary
		fix := fmt.Sprintf("em contexto CLI as skills degradam conforme declarado; para %s nativo, use um engine com a capacidade (§10)", cap)
		rep.add("capacidades", Warn, detail, fix)
	}
	// avisos individuais — skill sem degradação declarada para o que falta
	for _, ind := range individual {
		rep.add("capacidades", Warn,
			ind.skill+": exige "+strings.Join(ind.missing, ", ")+" — ausente no engine detectado (sem degradação declarada)",
			"declare 'degrades' no front matter ou use um engine FULL (§10)")
	}
	if capsOK {
		detail := "engine BASE (file+shell"
		if caps["mcp"] {
			detail += "+mcp"
		}
		detail += ")"
		if len(skills) == 0 {
			detail += " — nenhuma skill vendored"
		}
		rep.add("capacidades", OK, detail, "")
	}

	// 6-9. ambiente de máquina (kit.jev, kit.graph, git, engine-config) —
	// comum a workspace-produto e container.
	rep.addMachineChecks(opt, gitOK, skills)

	return rep, nil
}

// índiceLoose parseia o índice tolerando campos a mais (leitura doctor).
type looseIndex struct {
	GeneratedAt string `json:"generated_at"`
}

func parseIndexLoose(data []byte) (*looseIndex, error) {
	var idx looseIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, err
	}
	return &idx, nil
}

func readManifest(root string) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(root, ".golmet", "manifest.json"))
	if err != nil {
		return nil, err
	}
	var m struct {
		Skills []struct {
			Name string `json:"name"`
		} `json:"skills"`
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	var out []string
	for _, s := range m.Skills {
		if s.Name != "" {
			out = append(out, s.Name)
		}
	}
	return out, nil
}

func readSkillDirs(root string) []string {
	entries, err := os.ReadDir(filepath.Join(root, ".golmet", "skills"))
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

// detectCodeGraph delega a detecção ao pacote tools (mesmo destino que o
// instalador `golmet tools install codegraph` usa — fonte única).
func detectCodeGraph() bool {
	return tools.DetectCodegraph()
}

// samePath compara dois caminhos com tolerância do SO (Windows é case-
// insensitive; o config root resolvido do engine e o arg do usuário podem
// diferir em case/barras).
func samePath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// Human renderiza o checklist ✅/⚠️/❌ (CLI sem --json). A primeira linha
// carrega a versão do kit e o CONTEXTO detectado — diagnóstico instantâneo
// de binário velho servindo MCP (o processo do servidor sobrevive a `go
// install` até reiniciar o cliente) e de modo errado de uso.
func (r *Report) Human() string {
	var b strings.Builder
	ctx := r.Context
	if ctx == "" {
		ctx = ContextWorkspaceProduct
	}
	fmt.Fprintf(&b, "workspace: %s — golmet %s — contexto: %s\n", r.Workspace, indexstate.KitVersion, ctx)
	mark := map[Status]string{OK: "✅", Warn: "⚠️ ", Fail: "❌"}
	for _, c := range r.Checks {
		fmt.Fprintf(&b, "%s %-14s %s\n", mark[c.Status], c.Item, c.Detail)
		// findings do engine-config: paths/resumo logo abaixo da linha do
		// check (o fix comum já vem na linha seguinte, estilo existente).
		// No contexto engine-config o check É o corpo principal — cabeçalho
		// do relatório completo + camadas dos geridos (diferença do modo
		// compacto do workspace-produto).
		if c.Item == "engine-config" && r.EngineConfig != nil {
			if ctx == ContextEngineConfig {
				fmt.Fprintln(&b, "     relatório completo do engine-config:")
			}
			r.EngineConfig.renderLines(&b, ctx == ContextEngineConfig)
		}
		// fix: só faz sentido quando há o que corrigir — status OK não carrega hint.
		if c.Fix != "" && c.Status != OK {
			fmt.Fprintf(&b, "     fix: %s\n", c.Fix)
		}
	}
	fmt.Fprintf(&b, "\n%d ok, %d aviso(s), %d falha(s)\n", r.OKs, r.Warns, r.Fails)
	return b.String()
}
