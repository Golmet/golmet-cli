package install

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// EngineConfirm na 1ª confirmação detecta os já-registrados (1 chamada —
// cache na struct) e aplica os defaults: já-registrados DESMARCADOS
// (golmet permanece marcado mesmo registrado — obrigatório).
func TestWizardRegisteredDefaults(t *testing.T) {
	home := newHome(t)
	m := NewWizardModel(home, ".")
	m.Engines = []EngineInfo{{ID: "opencode", Detected: true}}
	m.EngineCursor = 0

	calls := 0
	old := installedMCPsFn
	installedMCPsFn = func(home, workspace, engineID string) map[string]string {
		calls++
		if engineID != "opencode" {
			t.Errorf("detecção com engine errado: %q", engineID)
		}
		return map[string]string{
			"figma":    "connected",
			"postgres": "needs authentication",
			"golmet":   "connected", // até o obrigatório registrado
		}
	}
	t.Cleanup(func() { installedMCPsFn = old })

	m.EngineConfirm()
	if m.EngineChosen != "opencode" {
		t.Fatalf("engine não confirmado: %q", m.EngineChosen)
	}
	// defaults: já-registrados desmarcados, EXCETO golmet (obrigatório)
	if m.Checked["figma"] {
		t.Errorf("figma registrado deveria virar DESMARCADO por default")
	}
	if m.Checked["postgres"] {
		t.Errorf("postgres registrado deveria virar DESMARCADO por default")
	}
	if !m.Checked["golmet"] {
		t.Errorf("golmet obrigatório permanece marcado mesmo registrado")
	}
	if m.Registered["postgres"] != "needs authentication" {
		t.Errorf("status textual perdido: %q", m.Registered["postgres"])
	}
	// cache: re-confirmar o engine NÃO re-detecta (1 chamada de list por wizard)
	m.Step = StepEngine
	m.EngineConfirm()
	if calls != 1 {
		t.Errorf("detecção rodou %d vezes, want 1 (cache na struct)", calls)
	}
}

// toggle-all (`a`) ignora os já-registrados: só alterna os não-registrados.
func TestMCPToggleAllIgnoresRegistered(t *testing.T) {
	m := NewWizardModel(newHome(t), ".")
	m.Registered = map[string]string{"figma": "connected", "postgres": "connected"}

	m.MCPToggleAll() // marca todos os não-registrados
	for _, def := range m.MCPs {
		if _, registered := m.Registered[def.Name]; registered {
			if m.Checked[def.Name] {
				t.Errorf("%s registrado foi marcado pelo toggle-all (deveria ignorar)", def.Name)
			}
			continue
		}
		if def.Required && !m.Checked[def.Name] {
			t.Errorf("%s obrigatório desmarcado pelo toggle-all", def.Name)
		}
	}
	m.MCPToggleAll() // desmarca os não-registrados
	for _, def := range m.MCPs {
		if _, registered := m.Registered[def.Name]; registered {
			if m.Checked[def.Name] {
				t.Errorf("%s registrado alterado no 2º toggle-all", def.Name)
			}
		}
	}
	// marcar um já-registrado MANUALMENTE (espaço) é opt-in e o toggle-all
	// não o desmarca mais
	m.Checked["figma"] = true
	m.MCPToggleAll()
	if !m.Checked["figma"] {
		// toggle-all marca os desmarcados... figma é registered → ignorado,
		// continua como o usuário deixou
		t.Errorf("opt-in do usuário foi revertido pelo toggle-all")
	}
}

// InstalledMCPs por engine: opencode via CLI parseada (status textual
// real), codex via TOML, claude-code via JSON, indisponível/sem detecção
// → nil (nada quebra).
func TestInstalledMCPs(t *testing.T) {
	// opencode: fixture da CLI
	old := runEngineCmd
	runEngineCmd = func(name string, args []string) (string, error) {
		return "✓ figma  connected\n✗ sqlite  failed: exit 1\n", nil
	}
	t.Cleanup(func() { runEngineCmd = old })
	m := InstalledMCPs(newHome(t), newHome(t), "opencode")
	if m["figma"] != "connected" || m["sqlite"] != "failed: exit 1" {
		t.Errorf("opencode status = %v", m)
	}

	// list indisponível → nil (sem badges)
	runEngineCmd = func(name string, args []string) (string, error) {
		return "", context.DeadlineExceeded
	}
	if got := InstalledMCPs(newHome(t), newHome(t), "opencode"); got != nil {
		t.Errorf("list indisponível deveria devolver nil, veio %v", got)
	}

	// codex: TOML com seções mcp_servers
	home := newHome(t)
	os.MkdirAll(filepath.Join(home, ".codex"), 0o755)
	os.WriteFile(filepath.Join(home, ".codex", "config.toml"),
		[]byte("[mcp_servers.golmet]\ncommand = \"x\"\n\n[mcp_servers.playwright]\ncommand = \"y\"\n"), 0o644)
	m2 := InstalledMCPs(home, newHome(t), "codex")
	if m2["golmet"] != "registered" || m2["playwright"] != "registered" {
		t.Errorf("codex detection = %v", m2)
	}

	// claude-code: JSON com mcpServers
	home2 := newHome(t)
	os.MkdirAll(home2, 0o755)
	os.WriteFile(filepath.Join(home2, ".claude.json"),
		[]byte(`{"mcpServers": {"golmet": {"command": "x"}}}`), 0o644)
	m3 := InstalledMCPs(home2, newHome(t), "claude-code")
	if m3["golmet"] != "registered" {
		t.Errorf("claude-code detection = %v", m3)
	}

	// engine sem detecção barata → nil (lista sem badges)
	if got := InstalledMCPs(newHome(t), newHome(t), "cursor"); got != nil {
		t.Errorf("cursor deveria devolver nil, veio %v", got)
	}
}
