// mcp_pack.go — server MCP próprio de state packs (kind: state). Um pack
// pode trazer um server MCP stdio Node na convenção
// <configRoot>/packs/<id>/mcp/server.js (registry dinâmico de
// <packRoot>/mcp/tools/*.js). O registro no engine segue SEMPRE o pipeline
// existente dos adapters (opencode: pré-check em cascata →
// `opencode mcp add --global` → pós-verify — NUNCA patch direto no jsonc,
// que tem comentários), com o nome "pack-<id>" (id do pack.yaml —
// desambigua multi-pack). Status é LEITURA apenas; sync é idempotente
// (duplicado → unchanged, no-override do pipeline) e --force repassa o
// re-add do pipeline.
package install

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Golmet/golmet-cli/internal/core/pack"
)

// nodeMinMajor — versão mínima do Node para o server.js do pack (a
// convenção do server usa stdlib ESM presente a partir do Node 20).
const nodeMinMajor = 20

// Estados possíveis do server MCP de um state pack (ordem = precedência
// quando mais de um problema existe — o registro vem primeiro: é o que o
// comando sync resolve; condições de runtime do node ficam nas colunas).
const (
	PackMCPStateOK            = "ok"
	PackMCPStateServerAusente = "server-ausente"
	PackMCPStateNaoRegistro   = "nao-registrado"
	PackMCPStateNodeAusente   = "node-ausente"
	PackMCPStateNodeAntigo    = "node-antigo"
)

// PackMCPStatus é o estado do server MCP de um state pack sincronizado
// (leitura apenas — nada aqui registra nem escreve no engine).
type PackMCPStatus struct {
	Pack        string `json:"pack"`
	Engine      string `json:"engine"`
	ServerName  string `json:"server_name"`            // pack-<id> no engine
	ServerPath  string `json:"server_path"`            // convenção <packRoot>/mcp/server.js
	HasServer   bool   `json:"has_server"`             // server.js existe no working copy
	Registered  bool   `json:"registered"`             // registrado no engine?
	RegStatus   string `json:"reg_status,omitempty"`   // status textual do engine (connected/config/registered/…)
	NodePresent bool   `json:"node_present"`           // node no PATH?
	NodeRecent  bool   `json:"node_recent"`            // node >= nodeMinMajor?
	NodeVersion string `json:"node_version,omitempty"` // saída de `node --version`
	State       string `json:"state"`                  // ok | server-ausente | nao-registrado | node-ausente | node-antigo
}

// PackMCPSyncResult é o resultado do registro do server MCP de um state
// pack — outcome + health note do pipeline do adapter.
type PackMCPSyncResult struct {
	Pack       string `json:"pack"`
	Engine     string `json:"engine"`
	ServerName string `json:"server_name"`
	Status     string `json:"status"` // registered | unchanged | manual | error | … (outcomes do pipeline)
	Detail     string `json:"detail,omitempty"`
}

// packMCPServerName — nome do server do pack no engine: pack-<id>.
func packMCPServerName(id string) string { return "pack-" + id }

// packStateServer é um state pack sincronizado com o estado do server
// MCP da convenção (hasServer = mcp/server.js existe na raiz do clone).
type packStateServer struct {
	spec       pack.Spec
	dir        string
	engineID   string
	serverPath string
	hasServer  bool
}

// scanStatePackServers varre os roots packs/ de cada engine com config
// global (mesma varredura do `pack list` — working copies de state packs)
// e anota quem tem server MCP na convenção <packRoot>/mcp/server.js. Raiz
// ilegível não derruba a varredura dos demais engines.
func scanStatePackServers(home string) ([]packStateServer, error) {
	var out []packStateServer
	seen := map[string]bool{}
	for _, engineID := range Engines() {
		root, ok := EngineConfigRootFor(home, engineID)
		if !ok {
			continue // workspace-scoped (vscode-copilot) e desconhecidos: sem raiz global
		}
		sums, err := pack.ListState([]string{filepath.Join(root, "packs")})
		if err != nil {
			continue
		}
		for _, s := range sums {
			if seen[s.Path] {
				continue
			}
			seen[s.Path] = true
			serverPath := filepath.Join(s.Path, "mcp", "server.js")
			out = append(out, packStateServer{
				spec: s.Spec, dir: s.Path, engineID: engineID,
				serverPath: serverPath, hasServer: fileExists(serverPath),
			})
		}
	}
	return out, nil
}

// PackMCPStatuses devolve o estado read-only do server MCP de CADA state
// pack sincronizado nos roots dos engines (com e sem mcp/server.js — sem
// server o estado é "server-ausente" e o engine/node não são sondados).
// Detecção de registro: InstalledMCPs (mesma cascata do wizard — opencode:
// `mcp list` com fallback ao config; demais: enumeração do config),
// memoizada por engine (1 sondagem por engine, não por pack).
func PackMCPStatuses(home string) ([]PackMCPStatus, error) {
	packs, err := scanStatePackServers(home)
	if err != nil {
		return nil, err
	}
	registeredCache := map[string]map[string]string{}
	out := make([]PackMCPStatus, 0, len(packs))
	for _, p := range packs {
		st := PackMCPStatus{
			Pack:       p.spec.ID,
			Engine:     p.engineID,
			ServerName: packMCPServerName(p.spec.ID),
			ServerPath: p.serverPath,
			HasServer:  p.hasServer,
		}
		if !p.hasServer {
			st.State = PackMCPStateServerAusente
			out = append(out, st)
			continue
		}
		registered, ok := registeredCache[p.engineID]
		if !ok {
			registered = InstalledMCPs(home, "", p.engineID)
			registeredCache[p.engineID] = registered
		}
		if status, has := registered[st.ServerName]; has {
			st.Registered = true
			st.RegStatus = status
		}
		st.NodeVersion, st.NodePresent, st.NodeRecent = nodeRuntimeReport()
		switch {
		case !st.Registered:
			st.State = PackMCPStateNaoRegistro
		case !st.NodePresent:
			st.State = PackMCPStateNodeAusente
		case !st.NodeRecent:
			st.State = PackMCPStateNodeAntigo
		default:
			st.State = PackMCPStateOK
		}
		out = append(out, st)
	}
	return out, nil
}

// PackMCPSync registra o server MCP de cada state pack elegível (com
// mcp/server.js) no engine dono do working copy, pelo pipeline existente
// do adapter (opencode: pré-check → `opencode mcp add --global` →
// pós-verify; NUNCA patch no jsonc). Idempotente: duplicado → unchanged.
// --force repassa o re-add ao pipeline. Erro de um pack não interrompe os
// demais (relatório completo; caller decide o exit por outcome).
func PackMCPSync(home string, force bool) ([]PackMCPSyncResult, error) {
	packs, err := scanStatePackServers(home)
	if err != nil {
		return nil, err
	}
	out := make([]PackMCPSyncResult, 0, len(packs))
	for _, p := range packs {
		if !p.hasServer {
			continue // elegibilidade: só packs com server na convenção
		}
		res := PackMCPSyncResult{
			Pack:       p.spec.ID,
			Engine:     p.engineID,
			ServerName: packMCPServerName(p.spec.ID),
		}
		adapter, ok := LookupEngine(p.engineID)
		if !ok {
			res.Status = OutcomeError
			res.Detail = "engine sem adapter: " + p.engineID
			out = append(out, res)
			continue
		}
		def := MCPDef{
			Name:    res.ServerName,
			Type:    MCPTypeStdio,
			Command: "node",
			Args:    []string{p.serverPath},
		}
		// entrada no formato dos engines de patch (claude/cursor/gemini);
		// o opencode ignora `entry` e monta o add pela CLI a partir do def.
		entry := map[string]any{
			"command": def.Command,
			"args":    def.Args,
		}
		outcome, detail, rerr := adapter.RegisterMCP(home, entry, def, force)
		res.Status, res.Detail = outcome, detail
		if rerr != nil {
			res.Status = OutcomeError
			res.Detail = rerr.Error()
		}
		out = append(out, res)
	}
	return out, nil
}

// nodeRuntimeReport — presença do node no PATH e versão. lookPathFn e
// runEngineCmd são os MESMOS pontos de injeção do pipeline de MCP — os
// testes nunca executam binários reais.
func nodeRuntimeReport() (version string, present, recent bool) {
	if _, err := lookPathFn("node"); err != nil {
		return "", false, false
	}
	out, err := runEngineCmd("node", []string{"--version"})
	if err != nil {
		return "", false, false
	}
	version = strings.TrimSpace(out)
	return version, true, parseNodeMajor(version) >= nodeMinMajor
}

// parseNodeMajor — "v22.14.0"/"22.14.0"/"22" → major; ilegível → 0.
func parseNodeMajor(raw string) int {
	raw = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "v"))
	major, _, _ := strings.Cut(raw, ".")
	n, err := strconv.Atoi(major)
	if err != nil {
		return 0
	}
	return n
}
