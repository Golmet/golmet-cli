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
	"github.com/Golmet/golmet-cli/internal/core/next"
	"github.com/Golmet/golmet-cli/internal/core/pack"
	"github.com/Golmet/golmet-cli/internal/core/scan"
	"github.com/Golmet/golmet-cli/internal/core/status"

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
		Description: "Valida o workspace contra o FORMATO v1 (H1–H10/S1–S5) — JSON com ok, erros/avisos §11 e counts do inventário",
	}, gmValidate)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "gm_doctor",
		Description: "Checklist de saúde do workspace e ambiente (S2/S3/S5, zona do kit, kit.jev/kit.graph) — sem smoke de rede",
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
		Description: "Checker determinístico de evals — pontua criteria.yaml de fixture contra o workspace (DR-017); grava o relatório markdown em ~/.golmet/eval-reports e devolve report_path",
	}, gmEval)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "gm_eval_reports",
		Description: "Lista os relatórios de eval gravados em ~/.golmet/eval-reports (mais recentes primeiro, cap 50) — leia o file com file tools para o relatório humano",
	}, gmEvalReports)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "gm_update",
		Description: "Atualiza o workflow FULL no engine: embarcado ou core externo; exige confirm=true e suporta dry_run",
	}, gmUpdate)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "gm_pack_update",
		Description: "Adiciona ou atualiza um pack no engine; exige confirm=true e suporta dry_run",
	}, gmPackUpdate)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "gm_pack_list",
		Description: "Lista packs já sincronizados no cache local sem escrever",
	}, gmPackList)

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

type classifyParams struct {
	Text      string `json:"text" jsonschema:"texto a classificar"`
	Questions string `json:"questions" jsonschema:"critérios TypeSafe (JSON string, formato questions do System One)"`
}

type updateParams struct {
	Home      string `json:"home,omitempty" jsonschema:"home do usuário (default: UserHomeDir)"`
	Workspace string `json:"workspace,omitempty" jsonschema:"workspace do kit (contexto da materialização em zona interna; default: .)"`
	Engine    string `json:"engine" jsonschema:"engine: opencode | claude-code (engines sem skills nativas recebem o FULL na zona .golmet/ do workspace)"`
	Core      string `json:"core,omitempty" jsonschema:"fonte Git ou diretório local do core; vazio = FULL embarcado"`
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

type packListParams struct {
	Home string `json:"home,omitempty" jsonschema:"home do usuário (default: UserHomeDir)"`
}

// ---- handlers ----

type validateOut struct {
	OK       bool               `json:"ok"`
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

func gmDoctor(ctx context.Context, req *mcp.CallToolRequest, in wsParams) (*mcp.CallToolResult, any, error) {
	ws := defaultWS(in.Workspace)
	rep, err := doctor.Run(ws, doctor.Options{})
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
		Engine: in.Engine, Written: []string{}, Preserved: []string{}, Conflicts: []string{},
	}
	if in.Core == "" {
		out.Origin = "embedded"
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
	rep, sync, err := coreinstall.MaterializeExternalCoreWithOptions(home, in.Engine, in.Core, in.Force, in.DryRun)
	if err != nil {
		return nil, mutationOut{}, err
	}
	out.Origin, out.Source, out.SHA = "external", sync.Source, sync.SHA
	out.CacheCloned, out.CacheUpdated = sync.Cloned, sync.Updated
	out.CacheWarning = sync.CacheWarning
	out.Written, out.Preserved, out.Conflicts = nonNilList(rep.Written), nonNilList(rep.Preserved), nonNilList(rep.Conflicts)
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
