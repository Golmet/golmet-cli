package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Golmet/golmet-cli/internal/core/indexstate"
)

// runCmd executa a árvore de comandos com os args dados e devolve a saída.
func runCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := newRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

// inTempDir troca o cwd do teste para um diretório temporário (init cria
// no cwd — semântica do CLI).
func inTempDir(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(old) })
	return tmp
}

func scaffoldWs(t *testing.T) string {
	t.Helper()
	base := inTempDir(t)
	out, err := runCmd(t, "init", "MeuApp")
	if err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	return filepath.Join(base, "MeuApp")
}

func TestInitCmdScaffoldsAndValidateCreatesBaseline(t *testing.T) {
	ws := scaffoldWs(t)
	if _, err := os.Stat(filepath.Join(ws, "golmet.yaml")); err != nil {
		t.Fatalf("golmet.yaml não criado: %v", err)
	}
	// init cria o baseline; validate sem mudanças NÃO reescreve
	out, err := runCmd(t, "validate", ws, "--tolerant")
	if err != nil {
		t.Fatalf("validate: %v\n%s", err, out)
	}
	if strings.Contains(out, "índice atualizado") {
		t.Errorf("validate sobre baseline conforme não deveria reescrever:\n%s", out)
	}
	if _, err := indexstate.Parse(mustReadFile(t, filepath.Join(ws, "golmet.index.json"))); err != nil {
		t.Errorf("baseline ilegível: %v", err)
	}

	// segunda validate idempotente
	out2, err := runCmd(t, "validate", ws, "--tolerant")
	if err != nil {
		t.Fatalf("validate 2: %v\n%s", err, out2)
	}
	if strings.Contains(out2, "índice atualizado") {
		t.Errorf("validate idempotente não deveria reescrever o índice:\n%s", out2)
	}
}

func TestIndexCmdRecompose(t *testing.T) {
	ws := scaffoldWs(t)
	out, err := runCmd(t, "index", ws)
	if err != nil {
		t.Fatalf("index: %v\n%s", err, out)
	}
	if !strings.Contains(out, "golmet.index.json") {
		t.Errorf("index deveria recompor o índice:\n%s", out)
	}
}

func TestStatusCmdJSON(t *testing.T) {
	ws := scaffoldWs(t)
	// status é somente-leitura: sentinel no índice deve permanecer intacto
	sentinel := "{\"format\":1,\"generated_at\":\"sentinel\",\"workspace\":\"MeuApp\",\"entities\":{},\"counts\":{}}\n"
	os.WriteFile(filepath.Join(ws, "golmet.index.json"), []byte(sentinel), 0o644)

	out, err := runCmd(t, "status", ws, "--json")
	if err != nil {
		t.Fatalf("status: %v\n%s", err, out)
	}
	var rep map[string]any
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		t.Fatalf("status --json inválido: %v\n%s", err, out)
	}
	if rep["workspace"] != "MeuApp" {
		t.Errorf("workspace = %v", rep["workspace"])
	}
	if got := string(mustReadFile(t, filepath.Join(ws, "golmet.index.json"))); got != sentinel {
		t.Errorf("status alterou o golmet.index.json (NUNCA deve escrever):\n%s", got)
	}
}

func TestStatusCmdInvalidExits2(t *testing.T) {
	_, err := runCmd(t, "status", filepath.Join(t.TempDir(), "nada"))
	if err == nil {
		t.Errorf("status em workspace inexistente deveria falhar")
	}
	if code, ok := ExitCodeOf(err); !ok || code != 2 {
		t.Errorf("exit code = %d (ok=%v), want 2", code, ok)
	}
}

func TestDoctorCmdJSON(t *testing.T) {
	ws := scaffoldWs(t)
	out, err := runCmd(t, "doctor", ws, "--json")
	if err != nil {
		t.Fatalf("doctor: %v\n%s", err, out)
	}
	// --json devolve o relatório COMPLETO (checks + seção engine_config
	// quando o check roda) — mesma forma do payload do gm_doctor no MCP.
	var rep struct {
		Checks       []map[string]any `json:"checks"`
		EngineConfig *map[string]any  `json:"engine_config"`
	}
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		t.Fatalf("doctor --json inválido: %v\n%s", err, out)
	}
	items := map[string]bool{}
	for _, c := range rep.Checks {
		if s, ok := c["item"].(string); ok {
			items[s] = true
		}
	}
	for _, want := range []string{"golmet.yaml", "indice", "manifest", "capacidades", "kit.jev", "kit.graph", "git", "worklist-s2"} {
		if !items[want] {
			t.Errorf("check %q ausente do doctor --json", want)
		}
	}
	if rep.EngineConfig == nil {
		t.Errorf("engine_config ausente — o default --engine opencode sempre roda o check: %s", out)
	}
}

// --engine resolve a raiz do config para o check de estado; engine
// workspace-scoped (vscode-copilot) não roda o check com nota informada;
// engine desconhecido sai 2 com a lista de disponíveis.
func TestDoctorCmdEngineFlag(t *testing.T) {
	ws := scaffoldWs(t)

	out, err := runCmd(t, "doctor", ws, "--engine", "vscode-copilot", "--json")
	if err != nil {
		t.Fatalf("doctor --engine vscode-copilot: %v\n%s", err, out)
	}
	if !strings.Contains(out, "workspace-scoped") {
		t.Errorf("nota de workspace-scoped ausente:\n%s", out)
	}
	// o relatório não pode conter o check nem a seção — a nota é a única saída
	repJSON := out[strings.IndexByte(out, '{'):]
	var rep struct {
		Checks       []map[string]any `json:"checks"`
		EngineConfig *map[string]any  `json:"engine_config"`
	}
	if err := json.Unmarshal([]byte(repJSON), &rep); err != nil {
		t.Fatalf("doctor --json inválido: %v\n%s", err, out)
	}
	for _, c := range rep.Checks {
		if c["item"] == "engine-config" {
			t.Errorf("check de engine-config não deveria rodar para workspace-scoped: %s", out)
		}
	}
	if rep.EngineConfig != nil {
		t.Errorf("seção engine_config presente para workspace-scoped: %s", out)
	}

	out2, err := runCmd(t, "doctor", ws, "--engine", "nao-existe")
	if code, ok := ExitCodeOf(err); !ok || code != 2 {
		t.Errorf("exit code = %d (ok=%v), want 2", code, ok)
	}
	if !strings.Contains(out2, "não existe") {
		t.Errorf("erro sem a lista de engines disponíveis:\n%s", out2)
	}
}

func TestDoctorCmdInvalidExits2(t *testing.T) {
	_, err := runCmd(t, "doctor", filepath.Join(t.TempDir(), "nada"))
	if code, ok := ExitCodeOf(err); !ok || code != 2 {
		t.Errorf("exit code = %d (ok=%v), want 2", code, ok)
	}
}

func TestInitCmdOrgAndInto(t *testing.T) {
	base := inTempDir(t)
	if _, err := runCmd(t, "init", "MinhaOrg", "--org"); err != nil {
		t.Fatalf("init --org: %v", err)
	}
	org := filepath.Join(base, "MinhaOrg")
	cfg := string(mustReadFile(t, filepath.Join(org, "golmet.yaml")))
	if !strings.Contains(cfg, "container: true") {
		t.Errorf("container ausente:\n%s", cfg)
	}
	out, err := runCmd(t, "init", "MeuApp", "--into", org)
	if err != nil {
		t.Fatalf("init --into: %v\n%s", err, out)
	}
	if !strings.Contains(out, "container atualizado: products: [MeuApp]") {
		t.Errorf("saída do --into sem atualização do container:\n%s", out)
	}
	// exclusividade
	_, err = runCmd(t, "init", "X", "--org", "--into", org)
	if code, ok := ExitCodeOf(err); !ok || code != 2 {
		t.Errorf("--org + --into deveria sair 2, veio %d (ok=%v)", code, ok)
	}
}

func TestStubCommandsStillStub(t *testing.T) {
	for _, name := range []string{"eval"} {
		_, err := runCmd(t, name)
		if code, ok := ExitCodeOf(err); !ok || code != 2 {
			t.Errorf("%s deveria permanecer stub (exit 2), veio %d (ok=%v)", name, code, ok)
		}
	}
}

func TestUpdateCmdInstallCheckAndDoctorGreen(t *testing.T) {
	ws := scaffoldWs(t)

	// --check antes do install: não instalado, sem escrita
	out, err := runCmd(t, "update", ws, "--check", "--json")
	if err != nil {
		t.Fatalf("update --check: %v\n%s", err, out)
	}
	if !strings.Contains(out, `"not_installed": true`) {
		t.Errorf("--check sem install deveria reportar not_installed: %s", out)
	}
	// scaffold já cria a zona vazia — o assertion correto é: nenhum SKILL.md
	matches, _ := filepath.Glob(filepath.Join(ws, ".golmet", "skills", "*", "SKILL.md"))
	if len(matches) != 0 {
		t.Errorf("--check não deveria instalar skills: %v", matches)
	}

	// install: 16 skills + manifest
	out2, err := runCmd(t, "update", ws)
	if err != nil {
		t.Fatalf("update: %v\n%s", err, out2)
	}
	if !strings.Contains(out2, "16 skill(es) instalada(s)") {
		t.Errorf("saída do install inesperada:\n%s", out2)
	}
	if _, err := os.Stat(filepath.Join(ws, ".golmet", "skills", "gm-charter", "SKILL.md")); err != nil {
		t.Fatalf("gm-charter não instalada: %v", err)
	}

	// --check pós-install: conforme, sem escrita
	out3, err := runCmd(t, "update", ws, "--check", "--json")
	if err != nil {
		t.Fatalf("update --check 2: %v\n%s", err, out3)
	}
	if strings.Contains(out3, "not_installed") || strings.Contains(out3, "modificados") {
		t.Errorf("pós-install deveria estar conforme: %s", out3)
	}

	// doctor manifest verde pós-update
	out4, err := runCmd(t, "doctor", ws, "--json")
	if err != nil {
		t.Fatalf("doctor: %v\n%s", err, out4)
	}
	if !strings.Contains(out4, `"item": "manifest"`) || strings.Contains(out4, "subset BASE não instalado") {
		t.Errorf("doctor deveria marcar manifest ok pós-update:\n%s", out4)
	}
}

func TestNextCmdHumanAndJSON(t *testing.T) {
	ws := scaffoldWs(t)
	out, err := runCmd(t, "next", ws)
	if err != nil {
		t.Fatalf("next: %v\n%s", err, out)
	}
	if !strings.Contains(out, "golmet update") {
		t.Errorf("next deveria recomendar update:\n%s", out)
	}
	out2, err := runCmd(t, "next", ws, "--json")
	if err != nil {
		t.Fatalf("next --json: %v\n%s", err, out2)
	}
	if !strings.Contains(out2, `"recommendations"`) || !strings.Contains(out2, `"priority"`) {
		t.Errorf("next --json inválido:\n%s", out2)
	}
}

func TestToolsCmdList(t *testing.T) {
	out, err := runCmd(t, "tools", "list")
	if err != nil {
		t.Fatalf("tools list: %v\n%s", err, out)
	}
	if !strings.Contains(out, "codegraph") {
		t.Errorf("tools list deveria mencionar codegraph:\n%s", out)
	}
	// tool desconhecido → exit 2
	_, err = runCmd(t, "tools", "install", "outro")
	if code, ok := ExitCodeOf(err); !ok || code != 2 {
		t.Errorf("tools install de tool desconhecido deveria sair 2, veio %d (ok=%v)", code, ok)
	}
}

func TestAdoptCmdDryRunJSONAndExclusives(t *testing.T) {
	// workspace pré-FORMATO mínimo
	base := inTempDir(t)
	ws := filepath.Join(base, "ws")
	os.MkdirAll(ws, 0o755)
	os.WriteFile(filepath.Join(ws, "REQUIREMENTS.md"),
		[]byte("# R\n\n## RN-001 — X\n> estado: active\nsem evidência [src/a.cs:1]\n"), 0o644)

	// default = dry-run, saída human com seções
	out, err := runCmd(t, "adopt", ws)
	if err != nil {
		t.Fatalf("adopt: %v\n%s", err, out)
	}
	if !strings.Contains(out, "leitura pura") || !strings.Contains(out, "precisa de julgamento") {
		t.Errorf("saída human do dry-run inesperada:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(ws, "golmet.yaml")); !os.IsNotExist(err) {
		t.Errorf("adopt default não deveria escrever golmet.yaml")
	}

	// --json estruturado
	out2, err := runCmd(t, "adopt", ws, "--json")
	if err != nil {
		t.Fatalf("adopt --json: %v\n%s", err, out2)
	}
	var rep map[string]any
	if err := json.Unmarshal([]byte(out2), &rep); err != nil {
		t.Fatalf("adopt --json inválido: %v\n%s", err, out2)
	}
	if rep["mode"] != "dry-run" {
		t.Errorf("mode = %v", rep["mode"])
	}

	// exclusivas → exit 2
	_, err = runCmd(t, "adopt", ws, "--dry-run", "--apply")
	if code, ok := ExitCodeOf(err); !ok || code != 2 {
		t.Errorf("--dry-run + --apply deveria sair 2, veio %d (ok=%v)", code, ok)
	}
	// charter-status inválido → exit 2
	_, err = runCmd(t, "adopt", ws, "--apply", "--charter-status", "talvez")
	if code, ok := ExitCodeOf(err); !ok || code != 2 {
		t.Errorf("charter-status inválido deveria sair 2, veio %d (ok=%v)", code, ok)
	}
}

func TestAdoptCmdApplyEndToEnd(t *testing.T) {
	base := inTempDir(t)
	ws := filepath.Join(base, "ws")
	os.MkdirAll(filepath.Join(ws, "delivery"), 0o755)
	os.WriteFile(filepath.Join(ws, "REQUIREMENTS.md"),
		[]byte("# R\n\n## RN-001 — Pedidos\n> estado: draft\nok.\n"), 0o644)

	out, err := runCmd(t, "adopt", ws, "--apply", "--charter-status", "confirmed", "--keep-legacy-layouts")
	if err != nil {
		t.Fatalf("adopt --apply: %v\n%s", err, out)
	}
	if !strings.Contains(out, "adoção aplicada") || !strings.Contains(out, "validate tolerant") {
		t.Errorf("saída do apply incompleta:\n%s", out)
	}
	// golmet.yaml + charter confirmed + índice
	if got := string(mustReadFile(t, filepath.Join(ws, "golmet.yaml"))); got != "format: 1\n" {
		t.Errorf("golmet.yaml = %q", got)
	}
	if got := string(mustReadFile(t, filepath.Join(ws, "PROJECT_CHARTER.md"))); !strings.HasPrefix(got, "---\nstatus: confirmed\n---\n") {
		t.Errorf("charter deveria nascer confirmed:\n%s", got)
	}
	if _, err := indexstate.Parse(mustReadFile(t, filepath.Join(ws, "golmet.index.json"))); err != nil {
		t.Errorf("índice ilegível: %v", err)
	}
	// keep-legacy-layouts: nenhum movimento, prosa preservada
	if got := string(mustReadFile(t, filepath.Join(ws, "REQUIREMENTS.md"))); !strings.Contains(got, "RN-001") {
		t.Errorf("REQUIREMENTS.md foi alterado pelo apply")
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
