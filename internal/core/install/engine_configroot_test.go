package install

import (
	"path/filepath"
	"testing"
)

// EngineConfigRootFor resolve a raiz do config por engine — fonte do check
// de estado do doctor (CLI --engine, gm_doctor MCP e doctor final do
// install passam pela MESMA resolução, sem duplicar caminhos).
func TestEngineConfigRootFor(t *testing.T) {
	home := t.TempDir()

	// engines com skills nativas: dir pai do SkillsDir
	if root, ok := EngineConfigRootFor(home, "opencode"); !ok || root != filepath.Join(home, ".config", "opencode") {
		t.Errorf("opencode = %q (%v), want %q", root, ok, filepath.Join(home, ".config", "opencode"))
	}
	if root, ok := EngineConfigRootFor(home, "claude-code"); !ok || root != filepath.Join(home, ".claude") {
		t.Errorf("claude-code = %q (%v), want %q", root, ok, filepath.Join(home, ".claude"))
	}

	// engines sem skills: raiz canônica de config
	if root, ok := EngineConfigRootFor(home, "cursor"); !ok || root != filepath.Join(home, ".cursor") {
		t.Errorf("cursor = %q (%v), want %q", root, ok, filepath.Join(home, ".cursor"))
	}
	if root, ok := EngineConfigRootFor(home, "gemini-cli"); !ok || root != filepath.Join(home, ".gemini") {
		t.Errorf("gemini-cli = %q (%v), want %q", root, ok, filepath.Join(home, ".gemini"))
	}
	if root, ok := EngineConfigRootFor(home, "codex"); !ok || root != filepath.Join(home, ".codex") {
		t.Errorf("codex = %q (%v), want %q", root, ok, filepath.Join(home, ".codex"))
	}

	// copilot-cli respeita $COPILOT_HOME (mesma regra do adapter)
	t.Setenv("COPILOT_HOME", filepath.Join(home, "copilot-custom"))
	if root, ok := EngineConfigRootFor(home, "copilot-cli"); !ok || root != filepath.Join(home, "copilot-custom") {
		t.Errorf("copilot-cli com COPILOT_HOME = %q (%v), want %q", root, ok, filepath.Join(home, "copilot-custom"))
	}
	t.Setenv("COPILOT_HOME", "")
	if root, ok := EngineConfigRootFor(home, "copilot-cli"); !ok || root != filepath.Join(home, ".copilot") {
		t.Errorf("copilot-cli default = %q (%v), want %q", root, ok, filepath.Join(home, ".copilot"))
	}

	// vscode-copilot é workspace-scoped (.vscode/ do workspace) — check não se aplica
	if root, ok := EngineConfigRootFor(home, "vscode-copilot"); ok || root != "" {
		t.Errorf("vscode-copilot = %q (%v), want (\"\", false)", root, ok)
	}

	// engine desconhecido → false (caller não seta EngineConfigRoot)
	if root, ok := EngineConfigRootFor(home, "nao-existe"); ok || root != "" {
		t.Errorf("engine desconhecido = %q (%v), want (\"\", false)", root, ok)
	}
}
