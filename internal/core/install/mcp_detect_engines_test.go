package install

import (
	"os"
	"path/filepath"
	"testing"
)

// vscode-copilot: mcp.json por WORKSPACE com chave "servers".
func TestInstalledMCPsVSCodeWorkspace(t *testing.T) {
	home := newHome(t)
	ws := newHome(t)
	os.MkdirAll(filepath.Join(ws, ".vscode"), 0o755)
	os.WriteFile(filepath.Join(ws, ".vscode", "mcp.json"),
		[]byte(`{"servers": {"figma": {"command": "npx"}, "sqlite": {"command": "x"}}}`), 0o644)

	m := InstalledMCPs(home, ws, "vscode-copilot")
	if m["figma"] != "registered" || m["sqlite"] != "registered" {
		t.Errorf("vscode detection = %v", m)
	}
	if len(m) != 2 {
		t.Errorf("vscode detection com extras: %v", m)
	}
}

// cursor: mcpServers em ~/.cursor/mcp.json.
func TestInstalledMCPsCursor(t *testing.T) {
	home := newHome(t)
	os.MkdirAll(filepath.Join(home, ".cursor"), 0o755)
	os.WriteFile(filepath.Join(home, ".cursor", "mcp.json"),
		[]byte(`{"mcpServers": {"golmet": {"command": "x"}}}`), 0o644)

	m := InstalledMCPs(home, newHome(t), "cursor")
	if m["golmet"] != "registered" {
		t.Errorf("cursor detection = %v", m)
	}
}

// gemini-cli: mcpServers DENTRO do settings.json (não top-level).
func TestInstalledMCPsGemini(t *testing.T) {
	home := newHome(t)
	os.MkdirAll(filepath.Join(home, ".gemini"), 0o755)
	os.WriteFile(filepath.Join(home, ".gemini", "settings.json"),
		[]byte(`{"theme": "dark", "mcpServers": {"codegraph": {"command": "x"}}}`), 0o644)

	m := InstalledMCPs(home, newHome(t), "gemini-cli")
	if m["codegraph"] != "registered" {
		t.Errorf("gemini detection = %v", m)
	}
	if len(m) != 1 {
		t.Errorf("gemini detection com extras: %v", m)
	}
}

// arquivo ausente → nil (sem badges — nada quebra).
func TestInstalledMCPsMissingConfigs(t *testing.T) {
	home := newHome(t)
	ws := newHome(t)
	for _, engine := range []string{"vscode-copilot", "cursor", "gemini-cli"} {
		if m := InstalledMCPs(home, ws, engine); m != nil {
			t.Errorf("%s sem config deveria devolver nil, veio %v", engine, m)
		}
	}
}
