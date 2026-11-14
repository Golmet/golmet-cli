// engine_claude.go — adapter do Claude Code. MCP: CLI `claude mcp add`
// quando existe; fallback patch JSON direto em ~/.claude.json
// ({"mcpServers": {nome: {command, args, env}}} — preserva o resto).
// O ENTRY vem centralizado do registerMCP (command/args/env com valor
// literal já resolvidos) — o adapter só escolhe o transporte.
package install

import (
	"fmt"
	"path/filepath"
	"strings"
)

type engineClaude struct{}

func (engineClaude) ID() string { return "claude-code" }

func (engineClaude) Detect(home, workspace string) bool {
	return fileExists(filepath.Join(home, ".claude.json")) ||
		dirExistsP(filepath.Join(home, ".claude"))
}

func (engineClaude) SkillsDir(home string) string {
	return filepath.Join(home, ".claude", "skills")
}

func (engineClaude) Description() string {
	return "Skills em ~/.claude/skills + MCP em ~/.claude.json (stdio e remote; env literal no patch)"
}

func (engineClaude) RegisterMCP(home string, entry map[string]any, def MCPDef, force bool) (string, string, error) {
	if _, err := lookPathFn("claude"); err == nil {
		var args []string
		if def.Type == MCPTypeRemote {
			// `claude mcp add -s user --transport http <nome> <url>`
			args = []string{"mcp", "add", "-s", "user", "--transport", "http", def.Name, def.URL}
		} else {
			// `claude mcp add -s user <nome> -- <cmd> [args...]`
			args = []string{"mcp", "add", "-s", "user", def.Name, "--", def.Command}
			args = append(args, def.Args...)
		}
		if _, err := runEngineCmd("claude", args); err != nil {
			return OutcomeError, "", fmt.Errorf("claude mcp add falhou: %w", err)
		}
		// verificação pós-add (exit 0 não garante o bloco — mesma lição do
		// opencode); `claude mcp list` deve listar o nome
		listOut, lerr := runEngineCmd("claude", []string{"mcp", "list"})
		if lerr == nil && !strings.Contains(listOut, def.Name) {
			return OutcomeError, "", fmt.Errorf(
				"claude mcp add saiu 0 mas o MCP não aparece em 'claude mcp list' — registre manualmente no ~/.claude.json (mcpServers.%s)", def.Name)
		}
		if lerr != nil {
			return OutcomeRegistered, "registrado via CLI do engine (claude mcp list indisponível para confirmar): " + strings.Join(args, " "), nil
		}
		return OutcomeRegistered, "registrado via CLI do engine: claude " + strings.Join(args, " "), nil
	}
	// fallback: patch JSON direto preservando o resto do arquivo
	return patchJSON(filepath.Join(home, ".claude.json"), "mcpServers", def.Name, entry, force)
}
