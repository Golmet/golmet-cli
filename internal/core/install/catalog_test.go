package install

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---- catálogo: as 12 entradas, nomes/tipos/ordem ----

func TestCatalogTwelveEntries(t *testing.T) {
	cat := Catalog()
	if len(cat) != 12 {
		t.Fatalf("catálogo com %d entradas, want 12", len(cat))
	}
	byName := map[string]MCPDef{}
	for _, def := range cat {
		byName[def.Name] = def
	}
	// kit primeiro
	if cat[0].Name != "golmet" || !cat[0].Required {
		t.Errorf("cat[0] = %s, want golmet obrigatório", cat[0].Name)
	}
	if cat[1].Name != "codegraph" || !cat[1].Recommended {
		t.Errorf("cat[1] = %s, want codegraph recomendado", cat[1].Name)
	}
	// novas entradas: nome/tipo/fonte
	want := map[string]struct {
		mcpType string
		command string
		url     string
	}{
		"azure-devops": {"stdio", "npx", ""},
		"github":       {"remote", "", "https://api.githubcopilot.com/mcp/"},
		"supabase":     {"remote", "", "https://mcp.supabase.com/mcp"},
		"cloudflare":   {"remote", "", "https://mcp.cloudflare.com/mcp"},
		"aws":          {"stdio", "uvx", ""},
		"sqlite":       {"stdio", "@sqlite", ""},
	}
	for name, w := range want {
		def, ok := byName[name]
		if !ok {
			t.Errorf("entrada %s ausente do catálogo", name)
			continue
		}
		if def.Type != w.mcpType {
			t.Errorf("%s type = %s, want %s", name, def.Type, w.mcpType)
		}
		if def.Command != w.command {
			t.Errorf("%s command = %q, want %q", name, def.Command, w.command)
		}
		if def.URL != w.url {
			t.Errorf("%s url = %q, want %q", name, def.URL, w.url)
		}
	}
	// azure-devops: param ORG + env ADO_PAT_B64 + args com o token
	az := byName["azure-devops"]
	if az.Param != "ORG" {
		t.Errorf("azure Param = %q, want ORG", az.Param)
	}
	joined := strings.Join(az.Args, " ")
	if !strings.Contains(joined, "<ORG>") || !strings.Contains(joined, "--authentication pat") {
		t.Errorf("args do azure sem token/env wire: %s", joined)
	}
	if !hasEnv(az.EnvRequired, "ADO_PAT_B64") {
		t.Errorf("azure env_required = %v, want ADO_PAT_B64", az.EnvRequired)
	}
	// aws: env AWS_PROFILE
	aws := byName["aws"]
	if !hasEnv(aws.EnvRequired, "AWS_PROFILE") {
		t.Errorf("aws env_required = %v, want AWS_PROFILE", aws.EnvRequired)
	}
	// sqlite: sem env, sem npx
	sql := byName["sqlite"]
	if len(sql.EnvRequired) != 0 {
		t.Errorf("sqlite env = %v, want vazio", sql.EnvRequired)
	}
	if strings.Contains(sql.Command, "npx") {
		t.Errorf("sqlite não é npx (binário rusq local): %s", sql.Command)
	}
	// Names: kit primeiro (golmet, codegraph), serviços alfabético
	names := Names()
	if names[0] != "golmet" || names[1] != "codegraph" {
		t.Errorf("Names[0..1] = %v, %v", names[0], names[1])
	}
	services := names[2:]
	sorted := append([]string(nil), services...)
	sortStrings(sorted)
	for i := range services {
		if services[i] != sorted[i] {
			t.Errorf("serviços fora de ordem alfabética: %v", names)
			break
		}
	}
}

func hasEnv(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

func sortStrings(xs []string) {
	for i := 1; i < len(xs); i++ {
		for j := i; j > 0 && xs[j] < xs[j-1]; j-- {
			xs[j], xs[j-1] = xs[j-1], xs[j]
		}
	}
}

// ---- parse da seleção com param (sintaxe nome=valor) ----

func TestParseMCPSelectionWithParam(t *testing.T) {
	out, err := ParseMCPSelection([]string{"azure-devops=MinhaOrg", "playwright"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(out) != 3 { // golmet entra por ser obrigatório
		t.Fatalf("out = %+v", out)
	}
	if out[0].Name != "golmet" {
		t.Errorf("golmet deveria ser prependido: %+v", out)
	}
	var az *SelectedMCP
	for i := range out {
		if out[i].Name == "azure-devops" {
			az = &out[i]
		}
	}
	if az == nil || az.ParamValue != "MinhaOrg" {
		t.Errorf("param do azure não capturado: %+v", out)
	}

	// azure sem =valor → erro claro citando a sintaxe
	if _, err := ParseMCPSelection([]string{"azure-devops"}); err == nil ||
		!strings.Contains(err.Error(), "exige o parâmetro") {
		t.Errorf("param ausente deveria falhar com sintaxe, veio %v", err)
	}
	// = em mcp sem Param → erro
	if _, err := ParseMCPSelection([]string{"playwright=x"}); err == nil {
		t.Errorf("param em mcp sem Param deveria falhar")
	}
	// desconhecido → erro com catálogo
	if _, err := ParseMCPSelection([]string{"warp"}); err == nil ||
		!strings.Contains(err.Error(), "não existe no catálogo") {
		t.Errorf("desconhecido deveria falhar com catálogo")
	}
}

// ---- azure-devops: param presente → args corretos no wire ----

func TestAzureDevopsWireWithParam(t *testing.T) {
	noEngineCLI(t)
	home := newHome(t)
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	t.Setenv("ADO_PAT_B64", "pat-base64-fake")

	res, err := Run(Options{
		EngineID: "claude-code",
		MCPs:     []string{"golmet", "azure-devops=MinhaOrg"},
		Home:     home,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Partial {
		t.Errorf("install com param não deveria ser parcial: %+v", res.Report.MCPs)
	}
	got := entryOf(t, filepath.Join(home, ".claude.json"), "mcpServers", "azure-devops")
	// args com a ORG interpolada no token e o wire de autenticação
	argsJSON, _ := json.Marshal(got["args"])
	want := `["-y","@azure-devops/mcp","MinhaOrg","--authentication","pat"]`
	if string(argsJSON) != want {
		t.Errorf("args = %s, want %s", argsJSON, want)
	}
	envMap, _ := got["env"].(map[string]any)
	if envMap["ADO_PAT_B64"] != "pat-base64-fake" {
		t.Errorf("env literal ausente: %+v", got)
	}
}

// ---- azure-devops: param ausente → pendente, não registra ----

func TestAzureDevopsWithoutParamPending(t *testing.T) {
	noEngineCLI(t)
	home := newHome(t)
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	t.Setenv("ADO_PAT_B64", "pat-base64-fake")

	// wizard envia "nome=" com valor vazio → Run classifica como PENDÊNCIA
	res, err := Run(Options{EngineID: "claude-code", MCPs: []string{"azure-devops="}, Home: home})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !res.Partial {
		t.Errorf("param ausente deveria marcar Partial")
	}
	if len(res.Report.Pending) != 1 || res.Report.Pending[0].Name != "azure-devops" {
		t.Fatalf("pending = %+v", res.Report.Pending)
	}
	if !strings.Contains(res.Report.Pending[0].Fix, "azure-devops=MinhaOrg") {
		t.Errorf("fix deveria citar a sintaxe nome=valor: %+v", res.Report.Pending[0])
	}
	// config quebrada nunca: azure NÃO está no config
	cfg := catJSON(t, filepath.Join(home, ".claude.json"))
	if mcp, ok := cfg["mcpServers"].(map[string]any); ok {
		if _, ok := mcp["azure-devops"]; ok {
			t.Errorf("azure registrado sem param (config quebrada)")
		}
	}

	// flag SEM "=" → erro de uso com a sintaxe (ParseMCPSelection)
	if _, err := ParseMCPSelection([]string{"azure-devops"}); err == nil ||
		!strings.Contains(err.Error(), "sintaxe: --mcp azure-devops=<ORG>") {
		t.Errorf("flag sem = deveria falhar com a sintaxe, veio %v", err)
	}
}

// ---- aws: env AWS_PROFILE ausente → pendente ----

func TestAWSEnvMissingPending(t *testing.T) {
	noEngineCLI(t)
	t.Setenv("AWS_PROFILE", "")
	home := newHome(t)
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	res, err := Run(Options{EngineID: "claude-code", MCPs: []string{"aws"}, Home: home})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Report.Pending) != 1 || res.Report.Pending[0].Name != "aws" {
		t.Fatalf("pending = %+v", res.Report.Pending)
	}
	if !strings.Contains(res.Report.Pending[0].Detail, "AWS_PROFILE") {
		t.Errorf("detail deveria citar AWS_PROFILE: %+v", res.Report.Pending[0])
	}
}

// ---- sqlite: binário presente → path; ausente → pendente com fix ----

func TestSQLiteBinaryPresentAndMissing(t *testing.T) {
	noEngineCLI(t)
	lad := t.TempDir() // LOCALAPPDATA virtual

	// ausente → auto-install falha (fn fake) → pendente com fix citando rusq/mcp-sqlite
	home := newHome(t)
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	res, err := Run(Options{
		EngineID: "claude-code", MCPs: []string{"sqlite"},
		Home: home, LocalAppData: lad,
		InstallSqliteFn: func(ctx context.Context) error {
			return context.DeadlineExceeded // download indisponível nos testes
		},
	})
	if err != nil {
		t.Fatalf("Run (ausente): %v", err)
	}
	if len(res.Report.Pending) != 1 || res.Report.Pending[0].Name != "sqlite" {
		t.Fatalf("pending = %+v", res.Report.Pending)
	}
	if !strings.Contains(res.Report.Pending[0].Fix, "rusq/mcp-sqlite") {
		t.Errorf("fix deveria citar as releases do rusq/mcp-sqlite: %+v", res.Report.Pending[0])
	}

	// presente → registra o path absoluto
	binDir := filepath.Join(lad, "Programs", "mcp-sqlite")
	os.MkdirAll(binDir, 0o755)
	binName := "mcp-sqlite.exe"
	if !strings.Contains(filepath.Base(binDir+string(os.PathSeparator)+binName), ".exe") {
		binName = "mcp-sqlite"
	}
	os.WriteFile(filepath.Join(binDir, binName), []byte("fake"), 0o755)

	res2, err := Run(Options{
		EngineID: "claude-code", MCPs: []string{"sqlite"},
		Home: home, LocalAppData: lad,
	})
	if err != nil {
		t.Fatalf("Run (presente): %v", err)
	}
	got := entryOf(t, filepath.Join(home, ".claude.json"), "mcpServers", "sqlite")
	if !strings.Contains(got["command"].(string), "mcp-sqlite") {
		t.Errorf("command do sqlite = %v", got["command"])
	}
	if res2.Partial {
		t.Errorf("sqlite com binário não deveria ser parcial: %+v", res2.Report.MCPs)
	}
}

// ---- remotes novos na matriz de engines ----

func TestNewRemotesEngineMatrix(t *testing.T) {
	home := newHome(t)
	os.MkdirAll(filepath.Join(home, ".cursor"), 0o755)
	os.MkdirAll(filepath.Join(home, ".gemini"), 0o755)

	// cursor: remote por url (github/supabase/cloudflare)
	cur := engineCursor{}
	for _, name := range []string{"github", "supabase", "cloudflare"} {
		def, _ := Lookup(name)
		o, _, err := cur.RegisterMCP(home, nil, def, false)
		if err != nil || o != OutcomeRegistered {
			t.Errorf("cursor %s = %s (%v)", name, o, err)
		}
		got := entryOf(t, filepath.Join(home, ".cursor", "mcp.json"), "mcpServers", name)
		if got["url"] != def.URL {
			t.Errorf("cursor %s url = %v", name, got["url"])
		}
	}
	// gemini/codex/vscode: remote sem suporte → unsupported com instrução
	for _, e := range []EngineAdapter{engineGemini{}, engineCodex{}, engineVSCode{}} {
		ws := newHome(t) // root do config p/ adapters workspace-aware
		for _, name := range []string{"github", "supabase", "cloudflare"} {
			def, _ := Lookup(name)
			var o string
			var detail string
			if wr, ok := e.(WorkspaceMCPRegistrar); ok {
				o, detail, _ = wr.RegisterMCPWorkspace(ws, nil, def, false)
			} else {
				o, detail, _ = e.RegisterMCP(home, nil, def, false)
			}
			if o != OutcomeUnsupported {
				t.Errorf("%s/%s remote = %s, want unsupported (%s)", e.ID(), name, o, detail)
			}
		}
	}
}

// ---- list-mcps (human) mostra as novas entradas ----

func TestCatalogHumanListCoversNewEntries(t *testing.T) {
	var names []string
	for _, def := range Catalog() {
		names = append(names, def.Name)
	}
	for _, want := range []string{
		"azure-devops", "github", "supabase", "cloudflare", "aws", "sqlite",
	} {
		found := false
		for _, n := range names {
			if n == want {
				found = true
			}
		}
		if !found {
			t.Errorf("entrada %s não listável", want)
		}
	}
}
