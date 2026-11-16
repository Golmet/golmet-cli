// Package mcpserve implementa o `golmet mcp` — servidor MCP (stdio) que
// expõe o core do kit como tools para engines (DR-018). name golmet,
// versão do kit. Cada tool responde JSON estruturado; falhas de workspace
// viram tool-error; degradações (ex.: kit.jev offline) são payload
// declarado — nunca silenciosas.
package mcpserve

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Golmet/golmet-cli/internal/core"
	"github.com/Golmet/golmet-cli/internal/core/doctor"
	"github.com/Golmet/golmet-cli/internal/core/eval"
	"github.com/Golmet/golmet-cli/internal/core/indexstate"
	coreinstall "github.com/Golmet/golmet-cli/internal/core/install"
	"github.com/Golmet/golmet-cli/internal/core/jev"
	"github.com/Golmet/golmet-cli/internal/core/lint"
	"github.com/Golmet/golmet-cli/internal/core/next"
	"github.com/Golmet/golmet-cli/internal/core/pack"
	"github.com/Golmet/golmet-cli/internal/core/scan"
	"github.com/Golmet/golmet-cli/internal/core/source"
	"github.com/Golmet/golmet-cli/internal/core/status"
	"github.com/Golmet/golmet-cli/internal/core/upgrade"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ServerName identifica o servidor MCP.
const ServerName = "golmet"

// NewServer monta o servidor com as tools registradas (exportado para
// os testes via transporte in-memory).
func NewServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    ServerName,
		Version: indexstate.KitVersion,
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "gm_validate",
		Description: "Valida o workspace contra o FORMAT v1 (H1–H10/S1–S5) — JSON com ok, erros/avisos §11 e counts do inventário",
	}, gmValidate)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "gm_doctor",
		Description: "Checklist de saúde do workspace e ambiente (S2/S3/S5, zona do kit, kit.jev/kit.graph) + estado do engine-config (customizações/não-reconstrutíveis/pessoal-por-design — param engine, default opencode) — sem smoke de rede",
	}, gmDoctor)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "gm_next",
		Description: "Recomendação determinística da próxima ação sobre o workspace (prioridades p1..p9)",
	}, gmNext)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "gm_jev_classify",
		Description: "Classificação calibrada via runtime Typesafe/Jev (kit.jev); sem key, devolve degradação estruturada offline",
	}, gmJevClassify)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "gm_status",
		Description: "Dashboard do workspace: RNs, PENDs, CRs, features, specs e seu estado (read-only)",
	}, gmStatus)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "gm_index",
		Description: "Recompõe o golmet.index.json a partir da prosa (D2: escritor único é o kit)",
	}, gmIndex)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "gm_eval",
		Description: "Roda fixture(s) de eval e grava o relatório markdown em ~/.golmet/eval-reports — devolve checks + report_path. O relatório humano fica listável via gm_eval_reports (USE ESTA em vez de procurar EVAL_REPORT no filesystem)",
	}, gmEval)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "gm_eval_reports",
		Description: "Resultado/relatório dos evals mais recentes do kit (o mais recente primeiro) — USE ESTA TOOL quando pedirem 'resultado do eval' ANTES de explorar o filesystem; devolve skill, fixture, data e o file do relatório humano (leia com file tools)",
	}, gmEvalReports)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "gm_update",
		Description: "Atualiza o workflow FULL embarcado no engine; exige confirm=true e suporta dry_run",
	}, gmUpdate)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "gm_pack_update",
		Description: "Adiciona ou atualiza um pack no engine (arquétipos overlay e state — state preserva acumulação local com commit automático + rebase); exige confirm=true e suporta dry_run",
	}, gmPackUpdate)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "gm_pack_list",
		Description: "Lista packs já sincronizados (overlay no cache; state packs aparecem com kind e caminho do working copy) sem escrever",
	}, gmPackList)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "gm_pack_publish",
		Description: "Sobe a maturidade acumulada de um state pack para o repo da organização: commit do WIP no working copy + push origin <branch> (manual por design — o operador decide). Sem confirm=true devolve o PREVIEW da subida proposta (working copy, WIP, commit, push alvo) e nada aplica; exige confirm=true e suporta message",
	}, gmPackPublish)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "gm_upgrade",
		Description: "Self-update do binário golmet a partir da última release do GitHub — check=true read-only ({current, latest, has_update}); aplicar exige confirm=true (SHA-256 obrigatório do checksums.txt; rollback manual via <bin>.old)",
	}, gmUpgrade)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "gm_lint",
		Description: "Varredura estrutural determinística (L1 refs mortas, L3 órfãos, L4 manifest stale, L5 nomes fora do padrão, L6 componentes fantasmas, L7 estrutura canônica, L8 conformidade estrutural definitivo↔template, L9 schema do registry de stack-templates, L10 estado interno dos state packs, L11 diagramas Mermaid, L12 encoding e texto quebrado, L13 âncoras de idioma) sobre o kit/config/packs — read-only, sem confirm; findings com severidade + remediação PT-BR",
	}, gmLint)

	return server
}

// Run serve o protocolo via stdio — processo long-running; nil no
// fechamento limpo.
func Run(ctx context.Context) error {
	return NewServer().Run(ctx, &mcp.StdioTransport{})
}

// ---- params ----

type wsParams struct {
	Workspace string `json:"workspace" jsonschema:"caminho do workspace (default: .)"`
}

// doctorParams — gm_doctor. Engine (default opencode) resolve a raiz do
// config do engine para o check de estado (cache reconstrutível vs dado do
// usuário + designações pessoal-por-design); home é injetável p/ testes
// (default UserHomeDir).
type doctorParams struct {
	Workspace string `json:"workspace" jsonschema:"caminho do workspace (default: .)"`
	Engine    string `json:"engine,omitempty" jsonschema:"engine para o check de engine-config: opencode | claude-code | cursor | gemini-cli | codex | copilot-cli (default: opencode; vscode-copilot é workspace-scoped — check não se aplica)"`
	Home      string `json:"home,omitempty" jsonschema:"home do usuário (default: UserHomeDir)"`
}

type classifyParams struct {
	Text      string `json:"text" jsonschema:"texto a classificar"`
	Questions string `json:"questions" jsonschema:"critérios TypeSafe (JSON string, formato questions do System One)"`
}

type updateParams struct {
	Home      string `json:"home,omitempty" jsonschema:"home do usuário (default: UserHomeDir)"`
	Workspace string `json:"workspace,omitempty" jsonschema:"workspace do kit (contexto da materialização em zona interna; default: .)"`
	Engine    string `json:"engine" jsonschema:"engine: opencode | claude-code (engines sem skills nativas recebem o FULL na zona .golmet/ do workspace)"`
	Force     bool   `json:"force,omitempty"`
	DryRun    bool   `json:"dry_run,omitempty"`
	Confirm   bool   `json:"confirm" jsonschema:"deve ser true para autorizar a operação"`
}

type packUpdateParams struct {
	Home    string `json:"home,omitempty" jsonschema:"home do usuário (default: UserHomeDir)"`
	Engine  string `json:"engine" jsonschema:"engine obrigatório: opencode | claude-code"`
	Source  string `json:"source" jsonschema:"fonte Git ou diretório local do pack"`
	Force   bool   `json:"force,omitempty"`
	DryRun  bool   `json:"dry_run,omitempty"`
	Confirm bool   `json:"confirm" jsonschema:"deve ser true para autorizar a operação"`
}

type packPublishParams struct {
	Message string `json:"message,omitempty" jsonschema:"mensagem do commit (default: maturidade via gm_pack_publish — o commit fica publish: <message>)"`
	Confirm bool   `json:"confirm,omitempty" jsonschema:"deve ser true para aplicar a subida; sem confirm, devolve o PREVIEW da subida proposta"`
	Pack    string `json:"pack,omitempty" jsonschema:"id do state pack ou caminho do working copy; opcional quando a máquina tem um único state pack"`
	Home    string `json:"home,omitempty" jsonschema:"home do usuário (default: UserHomeDir)"`
}

type packListParams struct {
	Home string `json:"home,omitempty" jsonschema:"home do usuário (default: UserHomeDir)"`
}

// upgradeParams — gm_upgrade. check=true read-only ({current, latest,
// has_update}); aplicar exige confirm=true (padrão gm_update — sem
// confirm e sem check é erro). Home é reservado: o upgrade opera sobre o
// binário corrente do processo (os.Executable), não sobre a home.
type upgradeParams struct {
	Confirm bool   `json:"confirm,omitempty" jsonschema:"deve ser true para autorizar a substituição do binário"`
	Check   bool   `json:"check,omitempty" jsonschema:"read-only: reporta versão local vs latest sem baixar nada"`
	Home    string `json:"home,omitempty" jsonschema:"reservado — o upgrade opera sobre o binário corrente do processo"`
}

// lintParams — gm_lint. Read-only (sem confirm — lint não muta); padrões
// iguais aos do CLI: workspace "." e engine opencode; home injetável.
type lintParams struct {
	Workspace string `json:"workspace,omitempty" jsonschema:"dir alvo da varredura (default: .; kit repo para os checks L1/L5/L6/L7 de kit)"`
	Engine    string `json:"engine,omitempty" jsonschema:"engine para resolver o config root (default: opencode)"`
	Home      string `json:"home,omitempty" jsonschema:"home do usuário (default: UserHomeDir)"`
}

// ---- handlers ----

type validateOut struct {
	OK       bool               `json:"ok"`
	Context  string             `json:"context,omitempty"` // "container" | "workspace-produto" (§9 — M21)
	Errors   []core.GolmetError `json:"errors"`
	Warnings []core.GolmetError `json:"warnings"`
	Counts   map[string]int     `json:"counts"`
}

func gmValidate(ctx context.Context, req *mcp.CallToolRequest, in wsParams) (*mcp.CallToolResult, validateOut, error) {
	ws := defaultWS(in.Workspace)
	res, err := scan.Run(ws, scan.Options{})
	if err != nil {
		return nil, validateOut{}, err
	}
	out := validateOut{
		OK:       len(res.Errors) == 0,
		Context:  res.Context,
		Errors:   res.Errors,
		Warnings: res.Warnings,
		Counts:   res.Counts,
	}
	if out.Errors == nil {
		out.Errors = []core.GolmetError{}
	}
	if out.Warnings == nil {
		out.Warnings = []core.GolmetError{}
	}
	return nil, out, nil
}

func gmDoctor(ctx context.Context, req *mcp.CallToolRequest, in doctorParams) (*mcp.CallToolResult, any, error) {
	ws := defaultWS(in.Workspace)
	opt := doctor.Options{}
	engine := in.Engine
	if engine == "" {
		engine = "opencode"
	}
	if home, herr := mcpHome(in.Home); herr == nil {
		if root, applies := coreinstall.EngineConfigRootFor(home, engine); applies {
			opt.EngineConfigRoot = root
		}
	} // home indisponível → check não roda (payload sem a seção engine_config)
	rep, err := doctor.Run(ws, opt)
	if err != nil {
		// workspace inválido: o checklist já coletado volta como payload
		if rep != nil {
			return nil, rep, nil
		}
		return nil, nil, err
	}
	return nil, rep, nil
}

func gmNext(ctx context.Context, req *mcp.CallToolRequest, in wsParams) (*mcp.CallToolResult, any, error) {
	ws := defaultWS(in.Workspace)
	rep, err := next.Run(ws)
	if err != nil {
		return nil, nil, err
	}
	return nil, rep, nil
}

type classifyOut struct {
	Offline bool   `json:"offline"`
	Reason  string `json:"reason,omitempty"`
	Status  int    `json:"status,omitempty"`
	Result  any    `json:"result,omitempty"` // resposta decodificada do runtime Jev
}

func gmJevClassify(ctx context.Context, req *mcp.CallToolRequest, in classifyParams) (*mcp.CallToolResult, classifyOut, error) {
	resp := jev.Classify(ctx, in.Text, in.Questions)
	out := classifyOut{
		Offline: resp.Offline,
		Reason:  resp.Reason,
		Status:  resp.Status,
	}
	if len(resp.Raw) > 0 {
		if err := json.Unmarshal(resp.Raw, &out.Result); err != nil {
			out.Result = string(resp.Raw) // resposta não-JSON: declarada como string
		}
	}
	return nil, out, nil
}

func defaultWS(ws string) string {
	if ws == "" {
		return "."
	}
	return ws
}

// ---- gm_status ----

type statusParams struct {
	Workspace string `json:"workspace" jsonschema:"caminho do workspace (default: .)"`
}

func gmStatus(ctx context.Context, req *mcp.CallToolRequest, in statusParams) (*mcp.CallToolResult, any, error) {
	ws := defaultWS(in.Workspace)
	rep, err := status.Run(ws, status.Options{})
	if err != nil {
		if rep != nil {
			return nil, rep, nil
		}
		return nil, nil, err
	}
	return nil, rep, nil
}

// ---- gm_index ----

type indexParams struct {
	Workspace string `json:"workspace" jsonschema:"caminho do workspace (default: .)"`
}

func gmIndex(ctx context.Context, req *mcp.CallToolRequest, in indexParams) (*mcp.CallToolResult, any, error) {
	ws := defaultWS(in.Workspace)
	res, err := scan.Run(ws, scan.Options{WriteIndex: true, ForceIndex: true})
	if err != nil {
		return nil, nil, err
	}
	return nil, map[string]any{
		"index_written": res.IndexWritten,
		"counts":        res.Counts,
		"errors":        len(res.Errors),
	}, nil
}

// ---- gm_eval / gm_eval_reports ----

type evalParams struct {
	Home       string `json:"home,omitempty" jsonschema:"home do usuário (default: UserHomeDir)"`
	FixtureDir string `json:"fixture_dir" jsonschema:"caminho da fixture (criteria.yaml + input/)"`
	Workspace  string `json:"workspace" jsonschema:"caminho do workspace com os artefatos produzidos"`
}

type evalOut struct {
	Pass       int                `json:"pass"`
	Fail       int                `json:"fail"`
	Warn       int                `json:"warn"`
	Skip       int                `json:"skip"`
	Checks     []eval.CheckResult `json:"checks"`
	ReportPath string             `json:"report_path,omitempty"` // relatório gravado em ~/.golmet/eval-reports
}

func gmEval(ctx context.Context, req *mcp.CallToolRequest, in evalParams) (*mcp.CallToolResult, evalOut, error) {
	res, err := eval.RunFixture(in.FixtureDir, in.Workspace)
	if err != nil {
		return nil, evalOut{}, err
	}
	home, herr := mcpHome(in.Home)
	if herr != nil {
		return nil, evalOut{}, herr
	}
	skill, fixture := eval.SkillFixture(in.FixtureDir)
	reportPath, werr := eval.WriteReport(skill, fixture, in.Workspace, res, eval.ReportDirDefault(home))
	if werr != nil {
		return nil, evalOut{}, werr
	}
	out := evalOut{
		Pass:   res.Passed,
		Fail:   res.Failed,
		Warn:   res.Warned,
		Skip:   res.Skipped,
		Checks: res.Checks,
		// o runtime nunca gerencia a pasta: recebe o path e lê o arquivo
		// quando quiser o relatório humano
		ReportPath: reportPath,
	}
	return nil, out, nil
}

type evalReportsParams struct {
	Home string `json:"home,omitempty" jsonschema:"home do usuário (default: UserHomeDir)"`
}

type evalReportsOut struct {
	Tool    string             `json:"tool"`
	Reports []eval.ReportEntry `json:"reports"`
}

func gmEvalReports(ctx context.Context, req *mcp.CallToolRequest, in evalReportsParams) (*mcp.CallToolResult, evalReportsOut, error) {
	home, err := mcpHome(in.Home)
	if err != nil {
		return nil, evalReportsOut{}, err
	}
	reports, err := eval.ListReports(eval.ReportDirDefault(home))
	if err != nil {
		return nil, evalReportsOut{}, err
	}
	return nil, evalReportsOut{Tool: "gm_eval_reports", Reports: reports}, nil
}

type mutationOut struct {
	Tool         string     `json:"tool"`
	DryRun       bool       `json:"dry_run"`
	Confirmed    bool       `json:"confirmed"`
	Engine       string     `json:"engine"`
	Origin       string     `json:"origin"`
	Zone         string     `json:"zone,omitempty"` // destino do FULL: engine-config | workspace-zone
	Source       string     `json:"source,omitempty"`
	SHA          string     `json:"sha,omitempty"`
	Pack         *pack.Spec `json:"pack,omitempty"`
	Written      []string   `json:"written"`
	Preserved    []string   `json:"preserved"`
	Conflicts    []string   `json:"conflicts"`
	CacheCloned  bool       `json:"cache_cloned,omitempty"`
	CacheUpdated bool       `json:"cache_updated,omitempty"`
	CacheWarning string     `json:"cache_warning,omitempty"`
}

type packListOut struct {
	Tool  string         `json:"tool"`
	Home  string         `json:"home"`
	Packs []pack.Summary `json:"packs"`
}

func gmUpdate(ctx context.Context, req *mcp.CallToolRequest, in updateParams) (*mcp.CallToolResult, mutationOut, error) {
	if !in.Confirm {
		return nil, mutationOut{}, fmt.Errorf("gm_update exige confirm=true; nenhuma alteração foi aplicada")
	}
	home, err := mcpHome(in.Home)
	if err != nil {
		return nil, mutationOut{}, err
	}
	if in.Engine == "" {
		return nil, mutationOut{}, fmt.Errorf("gm_update exige engine (o perfil completo exige skills nativas — compatíveis: opencode, claude-code)")
	}
	// fail-closed: o perfil completo exige engine com skills nativas — o
	// tool não degrada silenciosamente para a zona do workspace
	if !coreinstall.EngineSupportsCompleto(in.Engine, home) {
		return nil, mutationOut{}, fmt.Errorf("o perfil completo exige engine com skills nativas — compatíveis: %s; para %s, use o perfil essencial",
			strings.Join(coreinstall.CompatibleEngineIDs(home), ", "), in.Engine)
	}
	out := mutationOut{
		Tool: "gm_update", DryRun: in.DryRun, Confirmed: in.Confirm,
		Engine: in.Engine, Origin: "embedded", Written: []string{}, Preserved: []string{}, Conflicts: []string{},
	}
	rep, place, err := coreinstall.MaterializeFullWithOptions(home, in.Engine, defaultWS(in.Workspace), in.Force, in.DryRun)
	if err != nil {
		return nil, mutationOut{}, err
	}
	out.Written, out.Preserved, out.Conflicts = nonNilList(rep.Written), nonNilList(rep.Preserved), nonNilList(rep.Conflicts)
	if place != nil {
		out.Zone = place.Zone
	}
	return nil, out, nil
}

func gmPackUpdate(ctx context.Context, req *mcp.CallToolRequest, in packUpdateParams) (*mcp.CallToolResult, mutationOut, error) {
	if !in.Confirm {
		return nil, mutationOut{}, fmt.Errorf("gm_pack_update exige confirm=true; nenhuma alteração foi aplicada")
	}
	if in.Source == "" {
		return nil, mutationOut{}, fmt.Errorf("gm_pack_update exige source")
	}
	home, err := mcpHome(in.Home)
	if err != nil {
		return nil, mutationOut{}, err
	}
	if in.Engine == "" {
		return nil, mutationOut{}, fmt.Errorf("gm_pack_update exige engine (opencode ou claude-code)")
	}
	adapter, ok := coreinstall.LookupEngine(in.Engine)
	if !ok || adapter.SkillsDir(home) == "" {
		return nil, mutationOut{}, fmt.Errorf("engine %q não possui suporte de conteúdo", in.Engine)
	}
	rep, err := pack.AddWithOptions(home, filepath.Dir(adapter.SkillsDir(home)), in.Engine, in.Source, in.Force, in.DryRun)
	if err != nil {
		return nil, mutationOut{}, err
	}
	out := mutationOut{
		Tool: "gm_pack_update", DryRun: in.DryRun, Confirmed: in.Confirm,
		Engine: in.Engine, Origin: "pack", Pack: &rep.Spec,
		Source: rep.Sync.Source, SHA: rep.Sync.SHA,
		Written: nonNilList(rep.Written), Preserved: nonNilList(rep.Preserved), Conflicts: nonNilList(rep.Conflicts),
		CacheCloned: rep.Sync.Cloned, CacheUpdated: rep.Sync.Updated,
		CacheWarning: rep.Sync.CacheWarning,
	}
	return nil, out, nil
}

func gmPackList(ctx context.Context, req *mcp.CallToolRequest, in packListParams) (*mcp.CallToolResult, packListOut, error) {
	home, err := mcpHome(in.Home)
	if err != nil {
		return nil, packListOut{}, err
	}
	packs, err := pack.List(home)
	if err != nil {
		return nil, packListOut{}, err
	}
	return nil, packListOut{Tool: "gm_pack_list", Home: home, Packs: packs}, nil
}

// ---- gm_pack_publish (tool 14 — M32) ----

// packPublishOut é o retorno do publish via MCP (o mesmo shape do
// PublishReport do core + o message efetivo e os arquivos do WIP).
type packPublishOut struct {
	Tool      string   `json:"tool"`
	Pack      string   `json:"pack"`
	Branch    string   `json:"branch"`
	Committed bool     `json:"committed,omitempty"`
	Pushed    bool     `json:"pushed"`
	SHA       string   `json:"sha,omitempty"`
	Files     []string `json:"files,omitempty"`
	Message   string   `json:"message"`
	Dir       string   `json:"dir"`
}

// packRootsForEnginesMCP — as raízes packs/ de cada engine com config
// global (a mesma semântica do packRootsForEngines do CLI — fonte única
// impossível: cli importa mcpserve).
func packRootsForEnginesMCP(home string) []string {
	var roots []string
	seen := map[string]bool{}
	for _, id := range coreinstall.Engines() {
		root, ok := coreinstall.EngineConfigRootFor(home, id)
		if !ok {
			continue
		}
		packsRoot := filepath.Join(root, "packs")
		if !seen[packsRoot] {
			seen[packsRoot] = true
			roots = append(roots, packsRoot)
		}
	}
	return roots
}

// resolveStatePack — o param pack (id ou caminho do working copy):
// caminho existente com pack.yaml kind state → ele mesmo; id → match no
// ListState; vazio → 1 state pack na máquina = direto, N = erro listando os
// id's, 0 = erro claro.
func resolveStatePack(packParam string, roots []string) (dir string, err error) {
	if packParam == "" {
		packs, lerr := pack.ListState(roots)
		if lerr != nil {
			return "", lerr
		}
		switch len(packs) {
		case 1:
			return packs[0].Path, nil
		case 0:
			return "", fmt.Errorf("nenhum state pack instalado nesta máquina — rode 'golmet pack add <fonte> --engine <id>' ou informe o caminho do working copy")
		default:
			ids := make([]string, 0, len(packs))
			for _, p := range packs {
				ids = append(ids, p.Spec.ID)
			}
			return "", fmt.Errorf("%d state packs instalados — informe o param pack (id's: %s)", len(packs), strings.Join(ids, ", "))
		}
	}
	// caminho do working copy
	if st, serr := os.Stat(packParam); serr == nil && st.IsDir() {
		if _, gerr := os.Lstat(filepath.Join(packParam, ".git")); gerr == nil {
			if dir, rerr := pack.ResolveStateDir(packParam, roots); rerr == nil {
				return dir, nil
			}
		}
	}
	// id: match no ListState
	packs, lerr := pack.ListState(roots)
	if lerr != nil {
		return "", lerr
	}
	for _, p := range packs {
		if strings.EqualFold(p.Spec.ID, packParam) {
			return p.Path, nil
		}
	}
	ids := make([]string, 0, len(packs))
	for _, p := range packs {
		ids = append(ids, p.Spec.ID)
	}
	return "", fmt.Errorf("state pack %q não encontrado — id's instalados: %s", packParam, strings.Join(ids, ", "))
}

func gmPackPublish(ctx context.Context, req *mcp.CallToolRequest, in packPublishParams) (*mcp.CallToolResult, packPublishOut, error) {
	home, err := mcpHome(in.Home)
	if err != nil {
		return nil, packPublishOut{}, err
	}
	roots := packRootsForEnginesMCP(home)
	dir, err := resolveStatePack(in.Pack, roots)
	if err != nil {
		return nil, packPublishOut{}, err
	}
	message := in.Message
	if strings.TrimSpace(message) == "" {
		message = "maturidade via gm_pack_publish"
	}
	branch, berr := source.CurrentBranch(dir)
	if berr != nil {
		return nil, packPublishOut{}, berr
	}
	origin, oerr := source.Origin(dir)
	if oerr != nil {
		return nil, packPublishOut{}, oerr
	}
	wip, werr := source.WIPFiles(dir)
	if werr != nil {
		return nil, packPublishOut{}, werr
	}

	if !in.Confirm {
		// o preview da subida proposta — a IA mostra ANTES de pedir o confirm
		var b strings.Builder
		b.WriteString("gm_pack_publish exige confirm=true; nada foi publicado.\n")
		b.WriteString("subida proposta:\n")
		fmt.Fprintf(&b, "  working copy: %s\n", dir)
		fmt.Fprintf(&b, "  branch:       %s\n", branch)
		fmt.Fprintf(&b, "  origin:       %s\n", source.Redact(origin))
		fmt.Fprintf(&b, "  commit:       publish: %s\n", message)
		fmt.Fprintf(&b, "  push:         origin %s\n", branch)
		if len(wip) > 0 {
			fmt.Fprintf(&b, "  WIP (%d arquivo(s)):\n", len(wip))
			for _, f := range wip {
				fmt.Fprintf(&b, "    %s\n", f)
			}
		} else {
			b.WriteString("  nada a publicar — o working copy está limpo (o publish republica o estado commitado)\n")
		}
		return nil, packPublishOut{}, fmt.Errorf("%s", b.String())
	}

	rep, perr := pack.PublishState(dir, message, roots)
	if perr != nil {
		return nil, packPublishOut{}, perr
	}
	return nil, packPublishOut{
		Tool: "gm_pack_publish", Pack: rep.Pack, Branch: rep.Branch,
		Committed: rep.Committed, Pushed: rep.Pushed, SHA: rep.SHA,
		Files: wip, Message: message, Dir: rep.Dir,
	}, nil
}

// ---- gm_upgrade ----

type upgradeOut struct {
	Tool       string              `json:"tool"`
	Check      upgrade.CheckResult `json:"check"`
	Applied    bool                `json:"applied"`
	BinPath    string              `json:"bin_path,omitempty"`
	BackupPath string              `json:"backup_path,omitempty"`
}

func gmUpgrade(ctx context.Context, req *mcp.CallToolRequest, in upgradeParams) (*mcp.CallToolResult, upgradeOut, error) {
	if !in.Confirm && !in.Check {
		return nil, upgradeOut{}, fmt.Errorf("gm_upgrade exige confirm=true; nenhuma alteração foi aplicada")
	}
	res, err := upgrade.Run(ctx, upgrade.Options{CheckOnly: in.Check}, upgrade.BinaryVersion)
	if err != nil {
		return nil, upgradeOut{}, err
	}
	out := upgradeOut{
		Tool:       "gm_upgrade",
		Check:      res.Check,
		Applied:    res.Applied,
		BinPath:    res.BinPath,
		BackupPath: res.BackupPath,
	}
	return nil, out, nil
}

// gmLint — varredura estrutural determinística (read-only, sem confirm).
// Workspace inválido vira tool-error; findings são payload normal (o CLI
// interpreta exit 2 — na tool, ok=false com contagens).
func gmLint(ctx context.Context, req *mcp.CallToolRequest, in lintParams) (*mcp.CallToolResult, lintOut, error) {
	engine := in.Engine
	if engine == "" {
		engine = "opencode"
	}
	home := ""
	if in.Home != "" {
		home = in.Home
	} else if h, herr := mcpHome(""); herr == nil {
		home = h
	}
	rep, err := lint.Run(lint.Options{Workspace: in.Workspace, EngineID: engine, Home: home})
	if err != nil {
		return nil, lintOut{}, err
	}
	out := lintOut{
		Workspace:  rep.Workspace,
		ConfigRoot: rep.ConfigRoot,
		KitMode:    rep.KitMode,
		OK:         rep.Errors == 0 && rep.Warnings == 0,
		Errors:     rep.Errors,
		Warnings:   rep.Warnings,
		ByCheck:    rep.ByCheck,
		Findings:   rep.Findings,
	}
	return nil, out, nil
}

// lintOut — payload do gm_lint (contagens + findings).
type lintOut struct {
	Workspace  string         `json:"workspace"`
	ConfigRoot string         `json:"config_root,omitempty"`
	KitMode    bool           `json:"kit_mode"`
	OK         bool           `json:"ok"`
	Errors     int            `json:"errors"`
	Warnings   int            `json:"warnings"`
	ByCheck    map[string]int `json:"by_check"`
	Findings   []lint.Finding `json:"findings"`
}

func mcpHome(home string) (string, error) {
	if home != "" {
		return home, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("sem UserHomeDir: %w", err)
	}
	return home, nil
}

func nonNilList(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
