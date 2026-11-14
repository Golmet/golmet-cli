// Package doctor implementa o `golmet doctor` — checklist de saúde do
// workspace e do ambiente (FORMATO §9 soft: S2/S3/S5, §10 capacidades de
// kit, §2 zona do kit). Consulta: NUNCA escreve; exit 0 salvo workspace
// inválido (CLI sai 2).
package doctor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Golmet/golmet-cli/internal/adapters/fsutil"
	"github.com/Golmet/golmet-cli/internal/core/frontmatter"
	"github.com/Golmet/golmet-cli/internal/core/indexstate"
	"github.com/Golmet/golmet-cli/internal/core/jev"
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
// (o smoke já é default — sem efeito extra).
type Options struct {
	Network   bool // alias de compatibilidade (--network) — sem efeito extra
	NoNetwork bool // --no-network: pula o smoke do kit.jev (CI/offline)
}

// Report agrega o checklist.
type Report struct {
	Workspace string  `json:"workspace"`
	Version   string  `json:"version"` // versão do kit servindo — diagnóstico instantâneo de binário velho (MCP/CLI --json)
	Checks    []Check `json:"checks"`
	OKs       int     `json:"ok"`
	Warns     int     `json:"warns"`
	Fails     int     `json:"fails"`
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

// Run executa o checklist. Erro de retorno = workspace inválido (CLI 2);
// os checks já coletados voltam no Report para exibição.
func Run(dir string, opt Options) (*Report, error) {
	rep := &Report{Version: indexstate.KitVersion}
	root, err := filepath.Abs(dir)
	if err != nil {
		rep.add("golmet.yaml", Fail, "caminho inválido: "+err.Error(), "informe um caminho válido")
		return rep, fmt.Errorf("workspace inválido: %w", err)
	}
	rep.Workspace = filepath.Base(root)
	if st, serr := os.Stat(root); serr != nil || !st.IsDir() {
		rep.add("golmet.yaml", Fail, "workspace não encontrado: "+root, "informe um workspace existente")
		return rep, fmt.Errorf("workspace não encontrado: %s", root)
	}

	// 1. golmet.yaml — presente + parseável + format suportado
	cfgPath := filepath.Join(root, "golmet.yaml")
	cfgData, err := os.ReadFile(cfgPath)
	if err != nil {
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
	gitOK := exec.Command("git", "--version").Run() == nil
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

	// 6. kit.jev — TYPESAFE_API_KEY: key presente → smoke POR DEFAULT (1
	// POST pequeno — o aprendizado REAL da sessão: "key presente" enganou
	// o operador por semanas com o endpoint errado; o doctor checa se o Jev
	// FUNCIONA, não só se a key existe). Resultado é ⚠️ NUNCA ❌ (health é
	// soft). --no-network pula o smoke (CI/offline); --network é alias de
	// compatibilidade (o smoke já é default — sem efeito extra).
	jevRequired := false
	for _, sk := range skills {
		if sk.RequiresKit("jev") {
			jevRequired = true
		}
	}
	key := os.Getenv("TYPESAFE_API_KEY")
	switch {
	case key == "" && jevRequired:
		rep.add("kit.jev", Warn, "TYPESAFE_API_KEY ausente — skill requer kit.jev", "configure o runtime Typesafe/Jev (DR-015)")
	case key == "":
		rep.add("kit.jev", OK, "offline (não requerido por nenhuma skill)", "")
	case opt.NoNetwork:
		rep.add("kit.jev", OK, "TYPESAFE_API_KEY presente (smoke não executado — --no-network)", "verifique a key em console.typesafe.ai/keys")
	default:
		switch s, smoke := jev.SmokeHealth(context.Background()); s {
		case jev.SmokeOK:
			rep.add("kit.jev", OK, smoke+" — key presente e runtime operacional", "")
		case jev.SmokeRejected:
			rep.add("kit.jev", Warn, "key presente mas REJEITADA pelo runtime — "+smoke, "verifique a key em console.typesafe.ai/keys e o endpoint")
		default:
			rep.add("kit.jev", Warn, "key presente mas o runtime não respondeu — verifique o endpoint ("+smoke+")", "verifique a key em console.typesafe.ai/keys")
		}
	}

	// 7. kit.graph — binário DeusData em ~/.golmet/tools/codegraph/ (bloco 3 instala)
	graphRequired := false
	for _, sk := range skills {
		if sk.RequiresKit("graph") {
			graphRequired = true
		}
	}
	graphInstalled := detectCodeGraph()
	switch {
	case !graphInstalled && graphRequired:
		rep.add("kit.graph", Warn, "binário DeusData ausente em ~/.golmet/tools/codegraph/ — skill requer kit.graph", "instale o grafo de código do kit (DR-016)")
	case !graphInstalled:
		rep.add("kit.graph", OK, "não instalado (não requerido por nenhuma skill)", "")
	default:
		rep.add("kit.graph", OK, "binário detectado em ~/.golmet/tools/codegraph/", "")
	}

	// 8. git disponível
	if gitOK {
		rep.add("git", OK, "git disponível", "")
	} else {
		rep.add("git", Warn, "git indisponível — H7/S2 degradam", "instale o git no PATH")
	}

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

// Human renderiza o checklist ✅/⚠️/❌ (CLI sem --json). A primeira linha
// carrega a versão do kit — diagnóstico instantâneo de binário velho servindo
// MCP (o processo do servidor sobrevive a `go install` até reiniciar o cliente).
func (r *Report) Human() string {
	var b strings.Builder
	fmt.Fprintf(&b, "workspace: %s — golmet %s\n", r.Workspace, indexstate.KitVersion)
	mark := map[Status]string{OK: "✅", Warn: "⚠️ ", Fail: "❌"}
	for _, c := range r.Checks {
		fmt.Fprintf(&b, "%s %-14s %s\n", mark[c.Status], c.Item, c.Detail)
		// fix: só faz sentido quando há o que corrigir — status OK não carrega hint.
		if c.Fix != "" && c.Status != OK {
			fmt.Fprintf(&b, "     fix: %s\n", c.Fix)
		}
	}
	fmt.Fprintf(&b, "\n%d ok, %d aviso(s), %d falha(s)\n", r.OKs, r.Warns, r.Fails)
	return b.String()
}
