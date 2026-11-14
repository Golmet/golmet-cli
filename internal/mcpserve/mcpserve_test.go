package mcpserve

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	golmet "github.com/Golmet/golmet-cli"
	"github.com/Golmet/golmet-cli/internal/core/indexstate"
	"github.com/Golmet/golmet-cli/internal/core/scaffold"
	"github.com/Golmet/golmet-cli/internal/core/update"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// session conecta um client MCP de teste ao server via transporte
// in-memory (equivalente stdio do SDK).
func session(t *testing.T, ctx context.Context) *mcp.ClientSession {
	t.Helper()
	server := NewServer()
	ct, st := mcp.NewInMemoryTransports()
	if _, err := server.Connect(ctx, st, nil); err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "golmet-test", Version: "0"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

// callTool executa a tool e devolve o resultado + o texto do conteúdo.
func callTool(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) (*mcp.CallToolResult, string) {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("CallTool %s: %v", name, err)
	}
	var sb strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			sb.WriteString(tc.Text)
		}
	}
	return res, sb.String()
}

func scaffoldWS(t *testing.T) string {
	t.Helper()
	ws := filepath.Join(t.TempDir(), "ws")
	if _, err := scaffold.Run(filepath.Dir(ws), filepath.Base(ws), scaffold.Options{}); err != nil {
		t.Fatalf("scaffold: %v", err)
	}
	return ws
}

func TestHandshakeAndServerInfo(t *testing.T) {
	cs := session(t, context.Background())
	ir := cs.InitializeResult()
	if ir == nil || ir.ServerInfo == nil {
		t.Fatalf("InitializeResult incompleto: %+v", ir)
	}
	if ir.ServerInfo.Name != ServerName {
		t.Errorf("server name = %q, want %q", ir.ServerInfo.Name, ServerName)
	}
	if ir.ServerInfo.Version != indexstate.KitVersion {
		t.Errorf("server version = %q, want %q", ir.ServerInfo.Version, indexstate.KitVersion)
	}
	// tools anunciadas
	lt, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	names := map[string]bool{}
	for _, tl := range lt.Tools {
		names[tl.Name] = true
	}
	for _, want := range []string{"gm_validate", "gm_doctor", "gm_next", "gm_jev_classify", "gm_update", "gm_pack_update", "gm_pack_list", "gm_eval_reports"} {
		if !names[want] {
			t.Errorf("tool %s não anunciada (%v)", want, names)
		}
	}
}

func TestGmValidateViaMCP(t *testing.T) {
	ws := scaffoldWS(t)
	cs := session(t, context.Background())
	_, out := callTool(t, cs, "gm_validate", map[string]any{"workspace": ws})
	if !strings.Contains(out, `"ok":true`) {
		t.Errorf("workspace scaffoldado deveria validar ok: %s", out)
	}
	if !strings.Contains(out, `"counts"`) {
		t.Errorf("counts ausente do payload: %s", out)
	}

	// workspace com violação → ok false + erro §11
	bad := filepath.Join(t.TempDir(), "bad")
	os.MkdirAll(bad, 0o755)
	os.WriteFile(filepath.Join(bad, "golmet.yaml"), []byte("format: 1\n"), 0o644)
	os.WriteFile(filepath.Join(bad, "REQUIREMENTS.md"),
		[]byte("# R\n\n## RN-001 — X\n> estado: active\n"), 0o644)
	_, out2 := callTool(t, cs, "gm_validate", map[string]any{"workspace": bad})
	if !strings.Contains(out2, `"ok":false`) || !strings.Contains(out2, `"H4"`) {
		t.Errorf("violations deveria produzir ok=false com H4: %s", out2)
	}

	// workspace inexistente → tool-error (isError)
	res, _ := callTool(t, cs, "gm_validate", map[string]any{"workspace": filepath.Join(t.TempDir(), "nada")})
	if !res.IsError {
		t.Errorf("workspace inválido deveria virar tool-error")
	}
}

func TestGmDoctorAndNextViaMCP(t *testing.T) {
	ws := scaffoldWS(t)
	cs := session(t, context.Background())

	_, out := callTool(t, cs, "gm_doctor", map[string]any{"workspace": ws})
	if !strings.Contains(out, `"checks"`) || !strings.Contains(out, `"golmet.yaml"`) {
		t.Errorf("gm_doctor payload inesperado: %s", out)
	}

	_, out2 := callTool(t, cs, "gm_next", map[string]any{"workspace": ws})
	if !strings.Contains(out2, "golmet update") {
		t.Errorf("gm_next deveria recomendar update em workspace sem skills: %s", out2)
	}
	// com skills instaladas → nada pendente
	if _, err := update.Install(ws); err != nil {
		t.Fatalf("update: %v", err)
	}
	_, out3 := callTool(t, cs, "gm_next", map[string]any{"workspace": ws})
	if !strings.Contains(out3, "nada pendente") {
		t.Errorf("gm_next pós-update deveria estar em dia: %s", out3)
	}
}

func TestGmJevClassifyViaMCP(t *testing.T) {
	// sem key → degradação estruturada offline
	t.Setenv("TYPESAFE_API_KEY", "")
	cs := session(t, context.Background())
	_, out := callTool(t, cs, "gm_jev_classify", map[string]any{"text": "x", "questions": "{}"})
	if !strings.Contains(out, `"offline":true`) || !strings.Contains(out, "TYPESAFE_API_KEY") {
		t.Errorf("sem key deveria degradar estruturado: %s", out)
	}

	// com key + mock da API Typesafe
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"resposta":"confirmada","p":0.83}`))
	}))
	defer srv.Close()
	t.Setenv("TYPESAFE_API_KEY", "test-key")
	t.Setenv("TYPESAFE_API_URL", srv.URL)
	_, out2 := callTool(t, cs, "gm_jev_classify", map[string]any{"text": "corpo RN-001", "questions": `{"confirmada":"crit"}`})
	if !strings.Contains(out2, `"offline":false`) || !strings.Contains(out2, "confirmada") {
		t.Errorf("mock deveria responder online: %s", out2)
	}
}

func TestGmUpdateConfirmationAndDryRunViaMCP(t *testing.T) {
	home := t.TempDir()
	cs := session(t, context.Background())

	res, out := callTool(t, cs, "gm_update", map[string]any{
		"home": home, "engine": "opencode",
	})
	if !res.IsError {
		t.Fatalf("confirmação ausente deveria falhar claramente: error=%v out=%s", res.IsError, out)
	}
	res, out = callTool(t, cs, "gm_update", map[string]any{
		"home": home, "engine": "opencode", "confirm": false,
	})
	if !res.IsError || !strings.Contains(out, "confirm=true") {
		t.Fatalf("confirm=false deveria falhar claramente: error=%v out=%s", res.IsError, out)
	}
	res, out = callTool(t, cs, "gm_update", map[string]any{
		"home": home, "engine": "opencode", "confirm": true, "dry_run": true,
	})
	if res.IsError || !strings.Contains(out, `"dry_run":true`) || !strings.Contains(out, `"written"`) {
		t.Fatalf("dry-run inesperado: error=%v out=%s", res.IsError, out)
	}
	if _, err := os.Stat(filepath.Join(home, ".config")); !os.IsNotExist(err) {
		t.Fatalf("dry-run criou runtime: %v", err)
	}

	res, out = callTool(t, cs, "gm_update", map[string]any{
		"home": home, "engine": "opencode", "confirm": true,
	})
	if res.IsError || !strings.Contains(out, `"origin":"embedded"`) {
		t.Fatalf("FULL embarcado inesperado: error=%v out=%s", res.IsError, out)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "opencode", "skills", "gm-reverse", "SKILL.md")); err != nil {
		t.Fatalf("FULL não materializado: %v", err)
	}
}

// fail-closed: gm_update com engine sem skills nativas = erro no payload
// citando os compatíveis — sem degradação silenciosa para a zona.
func TestGmUpdateWeakEngineErrors(t *testing.T) {
	home := t.TempDir()
	cs := session(t, context.Background())
	res, out := callTool(t, cs, "gm_update", map[string]any{
		"home": home, "engine": "cursor", "confirm": true,
	})
	if !res.IsError {
		t.Fatalf("gm_update em engine sem skills nativas deveria falhar: out=%s", out)
	}
	if !strings.Contains(out, "perfil completo exige engine com skills nativas") ||
		!strings.Contains(out, "opencode") || !strings.Contains(out, "essencial") {
		t.Errorf("mensagem de erro incompleta: %s", out)
	}
	if _, err := os.Stat(filepath.Join(home, ".golmet")); !os.IsNotExist(err) {
		t.Errorf("gm_update inválido escreveu na zona: %v", err)
	}
}

func TestGmUpdateExternalCoreAndPackViaMCP(t *testing.T) {
	home := t.TempDir()
	coreDir := filepath.Join(t.TempDir(), "core")
	coreFS, err := fs.Sub(golmet.CoreDataFS(), "coredata")
	if err != nil {
		t.Fatal(err)
	}
	if err := fs.WalkDir(coreFS, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		target := filepath.Join(coreDir, filepath.FromSlash(path))
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := fs.ReadFile(coreFS, path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	}); err != nil {
		t.Fatal(err)
	}

	cs := session(t, context.Background())
	_, out := callTool(t, cs, "gm_update", map[string]any{
		"home": home, "engine": "opencode", "core": coreDir,
		"confirm": true, "dry_run": true,
	})
	if !strings.Contains(out, `"origin":"external"`) || !strings.Contains(out, `"source"`) {
		t.Fatalf("core externo dry-run inesperado: %s", out)
	}
	if _, err := os.Stat(filepath.Join(home, ".golmet")); !os.IsNotExist(err) {
		t.Fatalf("core externo dry-run criou cache/runtime: %v", err)
	}

	packDir := filepath.Join(t.TempDir(), "pack")
	if err := os.MkdirAll(filepath.Join(packDir, "skills", "gm-local"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packDir, "pack.yaml"), []byte("id: local-pack\nversion: 1.0.0\nname: Local pack\nprovides:\n  - skills/gm-local/SKILL.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packDir, "skills", "gm-local", "SKILL.md"), []byte("local pack\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, out = callTool(t, cs, "gm_pack_update", map[string]any{
		"home": home, "engine": "opencode", "source": packDir,
		"confirm": true, "dry_run": true,
	})
	if !strings.Contains(out, `"tool":"gm_pack_update"`) || !strings.Contains(out, `"dry_run":true`) {
		t.Fatalf("pack dry-run inesperado: %s", out)
	}
	if _, err := os.Stat(filepath.Join(home, ".config")); !os.IsNotExist(err) {
		t.Fatalf("pack dry-run criou runtime: %v", err)
	}

	_, out = callTool(t, cs, "gm_pack_update", map[string]any{
		"home": home, "engine": "opencode", "source": packDir, "confirm": true,
	})
	if !strings.Contains(out, `"id":"local-pack"`) {
		t.Fatalf("pack aplicado inesperadamente: %s", out)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "opencode", "skills", "gm-local", "SKILL.md")); err != nil {
		t.Fatalf("pack não materializado: %v", err)
	}
}

func TestGmPackListViaMCP(t *testing.T) {
	cs := session(t, context.Background())
	_, out := callTool(t, cs, "gm_pack_list", map[string]any{"home": t.TempDir()})
	if !strings.Contains(out, `"tool":"gm_pack_list"`) || !strings.Contains(out, `"packs":[]`) {
		t.Fatalf("lista de packs inesperada: %s", out)
	}
}

// gm_eval grava o relatório canônico na home injetada e devolve
// report_path existente; gm_eval_reports lista o relatório gravado com
// skill/fixture parseados. A home real nunca é tocada (home injetável).
func TestGmEvalReportAndReportsList(t *testing.T) {
	home := t.TempDir()
	base := t.TempDir()
	fx := filepath.Join(base, "fx", "gm-datamodel", "fixture-001")
	ws := filepath.Join(base, "ws")
	os.MkdirAll(fx, 0o755)
	os.MkdirAll(filepath.Join(ws, "specs"), 0o755)
	os.WriteFile(filepath.Join(fx, "criteria.yaml"),
		[]byte("checks:\n  - type: file_exists\n    path: \"specs/DATA_MODEL.md\"\n"), 0o644)
	os.WriteFile(filepath.Join(ws, "specs", "DATA_MODEL.md"), []byte("# DM\nPRODUTOR\n"), 0o644)

	cs := session(t, context.Background())
	res, out := callTool(t, cs, "gm_eval", map[string]any{
		"home": home, "fixture_dir": fx, "workspace": ws,
	})
	if res.IsError || !strings.Contains(out, `"report_path"`) {
		t.Fatalf("gm_eval sem report_path: error=%v out=%s", res.IsError, out)
	}
	var payload struct {
		Pass       int    `json:"pass"`
		ReportPath string `json:"report_path"`
	}
	if err := extractJSON(out, &payload); err != nil {
		t.Fatalf("payload gm_eval inválido: %v\n%s", err, out)
	}
	if payload.Pass != 1 {
		t.Errorf("pass = %d, want 1", payload.Pass)
	}
	if st, err := os.Stat(payload.ReportPath); err != nil || st.IsDir() {
		t.Fatalf("report_path inexistente: %v (%s)", err, payload.ReportPath)
	}
	if !strings.Contains(filepath.Base(payload.ReportPath), "eval-gm-datamodel-fixture-001-") {
		t.Errorf("nome fora do padrão: %s", payload.ReportPath)
	}

	// gm_eval_reports lista o relatório gravado
	_, out2 := callTool(t, cs, "gm_eval_reports", map[string]any{"home": home})
	if !strings.Contains(out2, `"tool":"gm_eval_reports"`) ||
		!strings.Contains(out2, `"skill":"gm-datamodel"`) ||
		!strings.Contains(out2, `"fixture":"fixture-001"`) ||
		!strings.Contains(out2, `"modified"`) {
		t.Fatalf("lista de relatórios inesperada: %s", out2)
	}
	var list struct {
		Reports []struct {
			File    string `json:"file"`
			Skill   string `json:"skill"`
			Fixture string `json:"fixture"`
		} `json:"reports"`
	}
	if err := extractJSON(out2, &list); err != nil {
		t.Fatalf("payload gm_eval_reports inválido: %v\n%s", err, out2)
	}
	if len(list.Reports) != 1 || list.Reports[0].File != payload.ReportPath {
		t.Errorf("relatório listado diverge: %+v (want %s)", list.Reports, payload.ReportPath)
	}

	// home sem relatórios → lista vazia (não erro)
	_, out3 := callTool(t, cs, "gm_eval_reports", map[string]any{"home": t.TempDir()})
	if !strings.Contains(out3, `"reports":[]`) {
		t.Errorf("home vazia deveria listar []: %s", out3)
	}
}

// extractJSON decodifica o primeiro valor JSON do conteúdo da tool.
func extractJSON(text string, v any) error {
	dec := json.NewDecoder(strings.NewReader(text))
	return dec.Decode(v)
}
