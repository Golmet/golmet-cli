package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// virtualHome aponta os.UserHomeDir para um diretório temporário do teste
// (Windows lê USERPROFILE; Unix lê HOME — os.UserHomeDir não faz cache).
func virtualHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	} else {
		t.Setenv("HOME", home)
	}
	return home
}

func TestInstallCmdFlagsEndToEnd(t *testing.T) {
	home := virtualHome(t)
	ws := inTempDir(t) // cwd do doctor final

	// engine codex: TOML append direto no HOME virtual (sem CLI externa)
	out, err := runCmd(t, "install", "--engine", "codex", "--profile", "base",
		"--mcp", "golmet,playwright", "--workspace", ws, "--yes")
	if err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	// codex não tem sistema de skills — nota impressa, skills ficam no workspace
	if !strings.Contains(out, "não tem sistema de skills") {
		t.Errorf("nota do engine sem skills ausente:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(home, ".codex", "config.toml")); err != nil {
		t.Fatalf("config.toml não criado: %v", err)
	}
	content := string(mustReadFile(t, filepath.Join(home, ".codex", "config.toml")))
	if !strings.Contains(content, "[mcp_servers.golmet]") || !strings.Contains(content, "[mcp_servers.playwright]") {
		t.Errorf("seções MCP ausentes:\n%s", content)
	}
	// command do golmet = exe corrente resolvido (nunca @self)
	if strings.Contains(content, "@self") {
		t.Errorf("placeholder @self vazou para o config do engine:\n%s", content)
	}
	// workspace do doctor sem golmet.yaml → doctor reporta no relatório mas não falha
	if !strings.Contains(out, "install concluído") {
		t.Errorf("saída sem o fechamento do install:\n%s", out)
	}
}

func TestInstallCmdListEnginesAndMCPs(t *testing.T) {
	out, err := runCmd(t, "install", "--list-engines")
	if err != nil {
		t.Fatalf("list-engines: %v\n%s", err, out)
	}
	for _, id := range []string{"opencode", "claude-code", "vscode-copilot", "cursor", "gemini-cli", "codex"} {
		if !strings.Contains(out, id) {
			t.Errorf("engine %s ausente da listagem:\n%s", id, out)
		}
	}
	out2, err := runCmd(t, "install", "--list-mcps")
	if err != nil {
		t.Fatalf("list-mcps: %v\n%s", err, out2)
	}
	for _, name := range []string{"golmet", "codegraph", "figma", "playwright", "postgres", "hostinger"} {
		if !strings.Contains(out2, name) {
			t.Errorf("mcp %s ausente da listagem:\n%s", name, out2)
		}
	}
}

func TestInstallCmdPartialExit1(t *testing.T) {
	virtualHome(t)
	t.Setenv("FIGMA_API_KEY", "") // env ausente → pendente → exit 1
	ws := inTempDir(t)
	out, err := runCmd(t, "install", "--engine", "claude-code", "--mcp", "golmet,figma", "--workspace", ws, "--yes")
	if err == nil {
		t.Fatalf("install com env ausente deveria sair 1:\n%s", out)
	}
	if code, ok := ExitCodeOf(err); !ok || code != 1 {
		t.Errorf("exit = %d (ok=%v), want 1", code, ok)
	}
	if !strings.Contains(out, "FIGMA_API_KEY") || !strings.Contains(out, "fix:") {
		t.Errorf("relatório deveria citar a env pendente com fix:\n%s", out)
	}
}

// --profile full é alias válido de completo (a camada de 125 componentes
// materializa); valores desconhecidos são rejeitados com exit 2 pelo
// install.Run, listando os perfis válidos.
func TestInstallCmdProfileUnknownExit2(t *testing.T) {
	virtualHome(t)
	out, err := runCmd(t, "install", "--engine", "claude-code", "--profile", "ultra", "--yes")
	if code, ok := ExitCodeOf(err); !ok || code != 2 {
		t.Errorf("perfil desconhecido deveria sair 2, veio %d (ok=%v)", code, ok)
	}
	if !strings.Contains(out, "perfil") || !strings.Contains(out, "essencial") || !strings.Contains(out, "completo") {
		t.Errorf("erro deveria citar o perfil rejeitado e os válidos:\n%s", out)
	}
}

func TestInstallCmdJSON(t *testing.T) {
	virtualHome(t)
	ws := inTempDir(t)
	out, err := runCmd(t, "install", "--engine", "codex", "--mcp", "golmet", "--workspace", ws, "--yes", "--json")
	if err != nil {
		t.Fatalf("install --json: %v\n%s", err, out)
	}
	var rep map[string]any
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		t.Fatalf("json inválido: %v\n%s", err, out)
	}
	for _, key := range []string{"engine", "profile", "version", "mcps"} {
		if _, ok := rep[key]; !ok {
			t.Errorf("chave %q ausente do JSON", key)
		}
	}
}

func catJSONFile(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json inválido em %s: %v", path, err)
	}
	return m
}
