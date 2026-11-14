// engines_wizard.go — adapters dos engines de config JSON simples + TOML
// do codex (patch direto com encoding/json, um arquivo por engine) e o
// registry dos engines.
package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// runEngineCmd é injetável (testes NUNCA executam os CLIs reais dos
// engines). Devolve (stdout+stderr, err).
var runEngineCmd = func(name string, args []string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// lookPathFn é injetável (testes NUNCA executam os CLIs reais dos
// engines — o operador pode ter opencode/claude no PATH).
var lookPathFn = exec.LookPath

// osGetenv wrapper (simetria com lookPathFn para injeção em testes).
func osGetenv(k string) string { return os.Getenv(k) }

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func dirExistsP(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

// registry — ordem de exibição no wizard (opencode primeiro: é o engine
// de referência do subset BASE).
func init() {
	registerEngine(engineOpenCode{})
	registerEngine(engineClaude{})
	registerEngine(engineCopilot{})
	registerEngine(engineVSCode{})
	registerEngine(engineCursor{})
	registerEngine(engineGemini{})
	registerEngine(engineCodex{})
}

// ---- vscode-copilot: .vscode/mcp.json no WORKSPACE (não home) ----

type engineVSCode struct{}

func (engineVSCode) ID() string { return "vscode-copilot" }

func (engineVSCode) Detect(home, workspace string) bool {
	return dirExistsP(filepath.Join(workspace, ".vscode")) // .vscode é do WORKSPACE, não da home
}

func (engineVSCode) SkillsDir(home string) string {
	return "" // sem sistema de skills — o install imprime nota
}

func (engineVSCode) Description() string {
	return "MCP em .vscode/mcp.json do workspace (só stdio) — sem sistema de skills"
}

func (engineVSCode) RegisterMCP(home string, entry map[string]any, def MCPDef, force bool) (string, string, error) {
	return OutcomeError, "", fmt.Errorf("vscode-copilot: registro requer workspace — use golmet install --workspace <ws> (.vscode/mcp.json)")
}

// RegisterMCPWorkspace é o caminho preferido do vscode-copilot: o destino
// correto é o WORKSPACE (.vscode/mcp.json), nunca a home — registerMCP o
// prefere quando o adapter implementa o hook.
func (engineVSCode) RegisterMCPWorkspace(workspace string, entry map[string]any, def MCPDef, force bool) (string, string, error) {
	if def.Type == MCPTypeRemote {
		return OutcomeUnsupported, "", fmt.Errorf("vscode-copilot: remote não verificado no v0.1 — registre manualmente em .vscode/mcp.json (url: %s)", def.URL)
	}
	e := map[string]any{"type": "stdio"}
	for k, v := range entry {
		e[k] = v
	}
	// .vscode/mcp.json vive na RAIZ DO WORKSPACE (igual ao Detect)
	return patchJSON(filepath.Join(workspace, ".vscode", "mcp.json"), "servers", def.Name, e, force)
}

// ---- cursor: ~/.cursor/mcp.json ----

type engineCursor struct{}

func (engineCursor) ID() string { return "cursor" }

func (engineCursor) Detect(home, workspace string) bool {
	return fileExists(filepath.Join(home, ".cursor", "mcp.json")) ||
		dirExistsP(filepath.Join(home, ".cursor"))
}

func (engineCursor) SkillsDir(home string) string { return "" }

func (engineCursor) Description() string {
	return "MCP em ~/.cursor/mcp.json (stdio e remote por url) — sem sistema de skills"
}

func (engineCursor) RegisterMCP(home string, entry map[string]any, def MCPDef, force bool) (string, string, error) {
	if def.Type == MCPTypeRemote {
		// Cursor: remote por "url" no bloco do server
		return patchJSON(filepath.Join(home, ".cursor", "mcp.json"), "mcpServers",
			def.Name, map[string]any{"url": def.URL}, force)
	}
	return patchJSON(filepath.Join(home, ".cursor", "mcp.json"), "mcpServers", def.Name, entry, force)
}

// ---- gemini-cli: ~/.gemini/settings.json ----

type engineGemini struct{}

func (engineGemini) ID() string { return "gemini-cli" }

func (engineGemini) Detect(home, workspace string) bool {
	return fileExists(filepath.Join(home, ".gemini", "settings.json")) ||
		dirExistsP(filepath.Join(home, ".gemini"))
}

func (engineGemini) SkillsDir(home string) string { return "" }

func (engineGemini) Description() string {
	return "MCP em ~/.gemini/settings.json (só stdio) — sem sistema de skills"
}

func (engineGemini) RegisterMCP(home string, entry map[string]any, def MCPDef, force bool) (string, string, error) {
	if def.Type == MCPTypeRemote {
		return OutcomeUnsupported, "", fmt.Errorf("gemini-cli: remote não suportado no v0.1 — registre manualmente em ~/.gemini/settings.json (url: %s)", def.URL)
	}
	return patchJSON(filepath.Join(home, ".gemini", "settings.json"), "mcpServers", def.Name, entry, force)
}

// ---- codex: ~/.codex/config.toml (append com guarda de seção) ----

type engineCodex struct{}

func (engineCodex) ID() string { return "codex" }

func (engineCodex) Detect(home, workspace string) bool {
	return fileExists(filepath.Join(home, ".codex", "config.toml")) ||
		dirExistsP(filepath.Join(home, ".codex"))
}

func (engineCodex) SkillsDir(home string) string { return "" }

func (engineCodex) Description() string {
	return "MCP em ~/.codex/config.toml (só stdio, TOML) — sem sistema de skills"
}

func (engineCodex) RegisterMCP(home string, entry map[string]any, def MCPDef, force bool) (string, string, error) {
	if def.Type == MCPTypeRemote {
		return OutcomeUnsupported, "", fmt.Errorf("codex: remote não suportado no v0.1 — registre manualmente em ~/.codex/config.toml (url: %s)", def.URL)
	}
	section := "mcp_servers." + def.Name
	argsTOML := make([]string, len(def.Args))
	for i, a := range def.Args {
		argsTOML[i] = fmt.Sprintf("%q", a)
	}
	lines := []string{
		"command = " + fmt.Sprintf("%q", def.Command),
		"args = [" + strings.Join(argsTOML, ", ") + "]",
	}
	return appendTOMLSection(filepath.Join(home, ".codex", "config.toml"), section, lines)
}
