// mcp_detect.go — detecção de MCPs JÁ REGISTRADOS no engine escolhido,
// usada pelo wizard para anotar a lista de seleção com o badge "já
// registrado" e desmarcá-los por default (nada a fazer — remarcar é
// opt-in). Fontes, em cascata, por engine:
//
//	opencode:     1. `opencode mcp list` parseado (status textual REAL) —
//	                  MAS a sondagem é NÃO-DETERMINÍSTICA na disponibilidade
//	                  (exit 0 com "No MCP servers configured" minutos após
//	                  listar 13) → 2. FALLBACK: enumeração do config
//	                  (opencode.jsonc no disco — fonte autoritativa,
//	                  status "config")
//	codex:        TOML [mcp_servers.*] (config-only, determinístico)
//	claude-code:  JSON mcpServers (config-only, determinístico)
//	outros:       nil (sem badges — o pré-check do add protege contra re-add)
//
// 1 chamada por wizard (cache na struct do modelo). Nada aqui ESCREVE no
// config — leitura apenas.
package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
)

// installedMCPsFn injetável (testes do wizard); default = InstalledMCPs.
var installedMCPsFn = InstalledMCPs

// opencodeConfigPath — caminho do config jsonc do opencode (leitura
// apenas; o kit NUNCA escreve nele — tem comentários).
func opencodeConfigPath(home string) string {
	return filepath.Join(home, ".config", "opencode", "opencode.jsonc")
}

// golemMCPConfigBlockRe — um bloco de MCP no jsonc: nome como CHAVE de
// objeto em qualquer nível, seguido de um objeto SEM chaves internas
// (`[^{}]` — o objeto do MCP é flat no formato que o `mcp add` escreve:
// type/command/url primeiro) contendo "command" (stdio) ou "url" (remote).
// O `[^{}]` é o que impede a chave do NESTING ("mcp": {) de casar e
// engolir o bloco do primeiro filho; nome-como-VALOR (ex.: "golmet":
// "C:\\..." num contexto de string) também não casa. Tolerante a
// comentários jsonc no arquivo (não exige json válido).
var golemMCPConfigBlockRe = regexp.MustCompile(`"([A-Za-z0-9_-]+)"\s*:\s*\{[^{}]*"(command|url)"`)

// opencodeConfigMCPs ENUMERA os MCPs registrados no config do engine
// (fonte autoritativa: o que o `mcp add` de fato escreveu, sem sondagem no
// meio — o `mcp list` é não-determinístico na disponibilidade). Retorna
// {nome: "config"} para cada bloco encontrado. Arquivo ausente/sem MCPs →
// nil. LEITURA apenas.
func opencodeConfigMCPs(home string) map[string]string {
	data, err := os.ReadFile(opencodeConfigPath(home))
	if err != nil {
		return nil
	}
	m := map[string]string{}
	for _, match := range golemMCPConfigBlockRe.FindAllStringSubmatch(string(data), -1) {
		m[match[1]] = "config"
	}
	if len(m) == 0 {
		return nil
	}
	return m
}

// opencodeConfigHasMCP — o MCP está registrado no config do engine?
// Reuso da enumeração (configHasMCP = nome ∈ enumeração) — fonte
// autoritativa usada pelo pré-check/pós-verify do add.
func opencodeConfigHasMCP(home, name string) bool {
	_, ok := opencodeConfigMCPs(home)[name]
	return ok
}

// InstalledMCPs devolve os MCPs já registrados no engine (nome → status
// textual). Engines sem detecção barata → nil — a lista aparece sem badges
// e o pré-check do add continua protegendo contra re-add. O workspace só
// importa para o vscode-copilot (mcp.json é por workspace).
func InstalledMCPs(home, workspace, engineID string) map[string]string {
	switch engineID {
	case "opencode":
		// fonte 1: sondagem (status textual real) — só é confiável quando
		// RESPONDE com entradas; err OU vazio não provam ausência
		if entries, err := listOpencode(); err == nil && len(entries) > 0 {
			m := make(map[string]string, len(entries))
			for _, e := range entries {
				m[e.name] = e.status // connected/needs authentication/failed/...
			}
			return m
		}
		// fonte 2 (fallback): o config no disco — nunca falha por sondagem
		return opencodeConfigMCPs(home)
	case "codex":
		return detectFromTOML(filepath.Join(home, ".codex", "config.toml"))
	case "claude-code":
		return detectJSONNames(filepath.Join(home, ".claude.json"), "mcpServers")
	case "vscode-copilot":
		return detectJSONNames(filepath.Join(workspace, ".vscode", "mcp.json"), "servers")
	case "cursor":
		return detectJSONNames(filepath.Join(home, ".cursor", "mcp.json"), "mcpServers")
	case "gemini-cli":
		return detectJSONNames(filepath.Join(home, ".gemini", "settings.json"), "mcpServers")
	}
	return nil
}

// detectJSONNames — nomes registrados no objeto `key` de um JSON (stdlib):
// vscode (servers), cursor/gemini/claude (mcpServers) — todos mapeiam para
// o status genérico "registered". Arquivo ausente/objeto vazio → nil.
func detectJSONNames(path, key string) map[string]string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var doc map[string]json.RawMessage
	if json.Unmarshal(data, &doc) != nil {
		return nil
	}
	raw, ok := doc[key]
	if !ok {
		return nil
	}
	var servers map[string]json.RawMessage
	if json.Unmarshal(raw, &servers) != nil || len(servers) == 0 {
		return nil
	}
	m := map[string]string{}
	for name := range servers {
		m[name] = "registered"
	}
	return m
}

// detectFromTOML — seções [mcp_servers.<nome>] do config do codex (parse
// manual — sem dep de TOML; status genérico "registered").
func detectFromTOML(path string) map[string]string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	re := regexp.MustCompile(`\[mcp_servers\.([A-Za-z0-9_-]+)\]`)
	m := map[string]string{}
	for _, match := range re.FindAllStringSubmatch(string(data), -1) {
		m[match[1]] = "registered"
	}
	if len(m) == 0 {
		return nil
	}
	return m
}
