// engine_opencode.go — adapter do OpenCode. Config jsonc COM comentários:
// o registro de MCP vai SEMPRE pela CLI do engine (`opencode mcp add
// --global`), nunca patch direto no jsonc. CLI ausente → instruções
// manuais impressas (nunca improvisar patch). LER o jsonc para
// verificação é seguro — e é a autoridade final: `opencode mcp list`
// sonda conexões e emite snapshots NÃO-DETERMINÍSTICOS (status pending,
// nomes ausentes do output resolvido); o config registra o que o `mcp
// add` de fato escreveu.
//
// Pipeline de registro (cascata):
//  1. pré-check: `mcp list` mostra o nome (qualquer status) → skip;
//     senão config (opencodeConfigHasMCP) já tem o bloco → skip;
//     senão add
//  2. `mcp add` (stdio ou remote --url)
//  3. pós-verify: list (fonte 1 — health notes por status, inclusive
//     pending) → config (fonte 2 — autoritativa) → erro manual só quando
//     AMBAS não têm o nome (list indisponível ≠ ausente — registered com
//     aviso declarado)
package install

import (
	"fmt"
	"path/filepath"
	"strings"
)

type engineOpenCode struct{}

func (engineOpenCode) ID() string { return "opencode" }

func (engineOpenCode) Detect(home, workspace string) bool {
	return fileExists(filepath.Join(home, ".config", "opencode", "opencode.jsonc"))
}

func (engineOpenCode) SkillsDir(home string) string {
	return filepath.Join(home, ".config", "opencode", "skills")
}

func (engineOpenCode) Description() string {
	return "Skills em ~/.config/opencode/skills + MCP via CLI do engine (stdio e remote); primeira classe"
}

func (engineOpenCode) RegisterMCP(home string, entry map[string]any, def MCPDef, force bool) (string, string, error) {
	if _, err := lookPathFn("opencode"); err != nil {
		return OutcomeManual, manualOpenCode(def), nil
	}

	// 1. pré-check em cascata (sem --force): list mostra o nome (QUALQUER
	//    status — connected, needs authentication, failed, disabled,
	//    pending) → skip; senão o config já tem o bloco → skip (o re-add
	//    duplicava entradas no config do operador)
	if !force {
		if entries, lerr := listOpencode(); lerr == nil {
			for _, e := range entries {
				if e.name == def.Name {
					return OutcomeUnchanged, "já registrado no engine (status: " + e.status + ")", nil
				}
			}
		}
		if opencodeConfigHasMCP(home, def.Name) {
			return OutcomeUnchanged, "já registrado no config do engine (opencode.jsonc)", nil
		}
	}

	// 2. add
	var args []string
	if def.Type == MCPTypeRemote {
		// `opencode mcp add <nome> --global --url <url>` (formato V2)
		args = []string{"mcp", "add", def.Name, "--global", "--url", def.URL}
	} else {
		// `opencode mcp add <nome> --global -- <cmd> [args...]`
		args = []string{"mcp", "add", def.Name, "--global", "--", def.Command}
		args = append(args, def.Args...)
	}
	if _, err := runEngineCmd("opencode", args); err != nil {
		return OutcomeError, "", fmt.Errorf("opencode mcp add falhou: %w", err)
	}

	// 3. pós-verify em cascata: list (fonte 1, não-determinística) →
	//    config (fonte 2, autoritativa) → erro só quando AMBAS não têm
	//    o nome. List indisponível ≠ ausente (sondagem não resolvida —
	//    não é prova de falha).
	entries2, lerr2 := listOpencode()
	if lerr2 == nil {
		status := opencodeStatusOf(entries2, def.Name)
		if status != "" {
			if note := opencodeHealthNote(status); note != "" {
				return OutcomeRegistered, "registrado via CLI do engine — " + note, nil
			}
			return OutcomeRegistered, "registrado via CLI do engine (status: " + status + ")", nil
		}
	}
	if opencodeConfigHasMCP(home, def.Name) {
		return OutcomeRegistered, "registrado via CLI do engine — confirmado no config do engine (opencode.jsonc); health via 'opencode mcp list'", nil
	}
	if lerr2 != nil {
		// nenhuma fonte confirmou, mas a indisponibilidade do list não é
		// prova de falha — registered com aviso declarado
		return OutcomeRegistered, "registrado via CLI do engine — aviso: não foi possível confirmar via 'opencode mcp list' (indisponível)", nil
	}
	return OutcomeError, "", fmt.Errorf(
		"opencode mcp add saiu 0 mas o MCP não aparece em 'opencode mcp list' nem no config do engine (opencode.jsonc) — confirme com 'opencode mcp list' ou registre manualmente (mcpServers.%s)", def.Name)
}

func manualOpenCode(def MCPDef) string {
	if def.Type == MCPTypeRemote {
		return fmt.Sprintf("rode manualmente: opencode mcp add %s --global --url %s (opencode CLI não encontrada no PATH — o jsonc tem comentários, o kit não faz patch nele)", def.Name, def.URL)
	}
	return fmt.Sprintf("rode manualmente: opencode mcp add %s --global -- %s %s (opencode CLI não encontrada no PATH — o jsonc tem comentários, o kit não faz patch nele)", def.Name, def.Command, strings.Join(def.Args, " "))
}

// listOpencode roda `opencode mcp list` e parseia as entradas. Err != nil
// = a CLI/lista está indisponível (caller decide — indisponibilidade não
// é prova de falha do registro).
func listOpencode() ([]opencodeListEntry, error) {
	out, err := opencodeMCPList()
	if err != nil {
		return nil, err
	}
	return parseOpencodeList(out), nil
}

// opencodeStatusOf devolve o status textual do MCP numa lista parseada
// ("" = ausente).
func opencodeStatusOf(entries []opencodeListEntry, name string) string {
	for _, e := range entries {
		if e.name == name {
			return e.status
		}
	}
	return ""
}

// opencodeMCPList roda `opencode mcp list` (injetável via runEngineCmd).
func opencodeMCPList() (string, error) {
	return runEngineCmd("opencode", []string{"mcp", "list"})
}

// glifo de status do `opencode mcp list` (primeiro rune da linha).
const opencodeGlyphs = "✓⚠✗○"

// opencodeListEntry é uma linha parseada do `opencode mcp list`.
type opencodeListEntry struct {
	glyph  string // ✓ ⚠ ✗ ○
	name   string
	status string // connected · pending · needs authentication · failed: <motivo> · disabled
}

// opencodeEmptyListText — o engine responde exit 0 com ESTE texto quando
// a sondagem não resolve (não-determinismo documentado no header deste
// arquivo: pode listar 13 e, minutos depois, "não ter nada"). O parser o
// trata como lista VAZIA — nunca vira entrada falsa ("No" / "MCP servers
// configured") e os callers caem no fallback do config, que é a fonte
// autoritativa.
const opencodeEmptyListText = "no mcp servers configured"

// parseOpencodeList extrai (glifo, nome, status) do output REAL: cada
// linha é `<glifo> <nome> <status...>` (glifos ✓ ⚠ ✗ ○). Cabeçalhos e
// linhas sem glifo conhecido são ignorados. Também aceita o formato
// simples `<nome> [status]` (mocks de teste) para não acoplar o parser
// ao enfeite visual do terminal.
func parseOpencodeList(output string) []opencodeListEntry {
	var out []opencodeListEntry
	for _, raw := range strings.Split(output, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.EqualFold(line, opencodeEmptyListText) {
			continue
		}
		line = stripGlyph(line)
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		name := fields[0]
		status := strings.TrimSpace(strings.Join(fields[1:], " "))
		glyph := ""
		if r := []rune(line); len(r) > 0 && strings.ContainsRune(opencodeGlyphs, r[0]) {
			glyph = string(r[0])
		}
		out = append(out, opencodeListEntry{glyph: glyph, name: name, status: status})
	}
	return out
}

// stripGlyph remove o glifo de status (primeiro rune) da linha quando ele
// é um dos glifos conhecidos — o nome nunca começa com esses runes.
func stripGlyph(line string) string {
	r := []rune(line)
	if len(r) == 0 {
		return line
	}
	if strings.ContainsRune(opencodeGlyphs, r[0]) {
		return strings.TrimSpace(string(r[1:]))
	}
	return line
}

// opencodeHealthNote traduz o status da lista para a nota do relatório do
// install. O install REGISTRA — a saúde da conexão é do doctor; "failed"
// pode ser causa externa ao install (ex.: processo morto por kill de
// travamento de arquivo, caso real do operador); "pending" é snapshot
// parcial da sondagem (não-determinismo do `mcp list`).
func opencodeHealthNote(status string) string {
	switch {
	case status == "connected":
		return ""
	case status == "pending":
		return "registrado — sondagem em curso no engine (status: pending)"
	case status == "needs authentication":
		return "registrado — autentique no engine (status: needs authentication)"
	case strings.HasPrefix(status, "failed"):
		return "registrado, mas o engine não conseguiu conectar — verifique o binário ou reinicie o engine (status: " + status + ")"
	case status == "disabled":
		return "registrado, mas desabilitado no engine — habilite no config do engine (status: disabled)"
	}
	return ""
}

// opencodeConfigHasMCP/opencodeConfigMCPs vivem em mcp_detect.go (mesmo
// package) — a enumeração do config é compartilhada entre o pré-check/
// pós-verify do add e os badges do wizard (DRY).
