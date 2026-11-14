// engine_copilot.go — adapter do Copilot CLI (GitHub Copilot CLI). Config
// em $COPILOT_HOME/mcp-config.json (default ~/.copilot) com top-level
// "mcpServers" — MESMA forma do Claude (NÃO a "servers" do VS Code).
// Fontes: docs.github.com/copilot/how-tos/copilot-cli/add-mcp-servers +
// `copilot mcp add --help` verificado em máquina real (2026-11):
//
//	stdio:  copilot mcp add <nome> -- <command> [args...] [--env K=V...]
//	remote: copilot mcp add --transport http <nome> <url>
//	post-verify: copilot mcp get <nome> (exit 0 não garante — mesma lição
//	do opencode/claude).
//
// CLI-first quando `copilot` está no PATH (env vai por --env, valor
// literal); fallback patch JSON direto preservando o resto do arquivo.
package install

import (
	"fmt"
	"path/filepath"
	"strings"
)

type engineCopilot struct{}

func (engineCopilot) ID() string { return "copilot-cli" }

// copilotDir resolve a raiz do config do Copilot CLI: env COPILOT_HOME
// quando setada (comportamento oficial do CLI), senão ~/.copilot.
func copilotDir(home string) string {
	if v := osGetenv("COPILOT_HOME"); v != "" {
		return v
	}
	return filepath.Join(home, ".copilot")
}

func (engineCopilot) Detect(home, workspace string) bool {
	_, cliPresent := lookPathFn("copilot")
	return dirExistsP(copilotDir(home)) || cliPresent == nil
}

func (engineCopilot) SkillsDir(home string) string {
	return "" // sem sistema de skills — skills ficam na zona .golmet/ do workspace
}

func (engineCopilot) Description() string {
	return "MCP em $COPILOT_HOME/mcp-config.json (stdio e remote; CLI-first com fallback patch) — sem sistema de skills"
}

func (engineCopilot) RegisterMCP(home string, entry map[string]any, def MCPDef, force bool) (string, string, error) {
	if _, err := lookPathFn("copilot"); err == nil {
		return registerViaCopilotCLI(def)
	}
	// fallback: patch JSON direto no mcp-config.json (mcpServers top-level)
	return patchJSON(filepath.Join(copilotDir(home), "mcp-config.json"),
		"mcpServers", def.Name, copilotEntry(entry, def), force)
}

// registerViaCopilotCLI — CLI-first com env por --env (valor literal) e
// verificação pós-add via `copilot mcp get <nome>`.
func registerViaCopilotCLI(def MCPDef) (string, string, error) {
	var args []string
	if def.Type == MCPTypeRemote {
		args = []string{"mcp", "add", "--transport", "http", def.Name, def.URL}
	} else {
		args = []string{"mcp", "add", def.Name, "--", def.Command}
		args = append(args, def.Args...)
		for _, k := range def.EnvRequired {
			args = append(args, "--env", k+"="+osGetenv(k)) // valor literal
		}
	}
	if _, err := runEngineCmd("copilot", args); err != nil {
		return OutcomeError, "", fmt.Errorf("copilot mcp add falhou: %w", err)
	}
	// verificação pós-add: `copilot mcp get <nome>` deve exit 0
	if _, gerr := runEngineCmd("copilot", []string{"mcp", "get", def.Name}); gerr != nil {
		return OutcomeError, "", fmt.Errorf(
			"copilot mcp add saiu 0 mas 'copilot mcp get %s' falhou — registre manualmente em $COPILOT_HOME/mcp-config.json (mcpServers.%s)", def.Name, def.Name)
	}
	return OutcomeRegistered, "registrado via CLI do engine: copilot " + strings.Join(args, " "), nil
}

// copilotEntry adapta o entry central para o formato do mcp-config.json
// do Copilot CLI: stdio = {"type":"local","command","args","env"};
// remote = {"type":"http","url"} (docs.github.com copilot-cli).
func copilotEntry(entry map[string]any, def MCPDef) map[string]any {
	if def.Type == MCPTypeRemote {
		return map[string]any{"type": "http", "url": def.URL}
	}
	e := map[string]any{"type": "local"}
	for k, v := range entry {
		e[k] = v
	}
	return e
}
