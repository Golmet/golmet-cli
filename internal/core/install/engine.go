// engine.go — contratos dos adapters de engine + registry + helpers de
// escrita idempotente (patch JSON preservando o resto do arquivo, append
// TOML com guarda de seção). Um arquivo por engine (engine_*.go).
package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Golmet/golmet-cli/internal/core/content"
	"github.com/Golmet/golmet-cli/internal/core/update"
)

// Outcome do registro de um MCP.
const (
	OutcomeRegistered  = "registered" // bloco novo escrito
	OutcomeUnchanged   = "unchanged"  // bloco já existente com MESMO conteúdo
	OutcomeDiverged    = "diverged"   // bloco existente com conteúdo DIFERENTE — pulado sem --force
	OutcomeManual      = "manual"     // CLI do engine ausente — instruções impressas (opencode: nunca patch no jsonc)
	OutcomeError       = "error"
	OutcomeUnsupported = "unsupported" // engine não suporta (ex.: remote no codex)
	OutcomeSkipped     = "skipped"     // env ausente etc. — não registra config quebrada
)

// EngineAdapter é o contrato mínimo por engine. `home` é injetado
// (testável — os.UserHomeDir é o default no Options); `workspace` é o
// workspace golmet corrente (vscode-copilot configura .vscode/ nele).
type EngineAdapter interface {
	ID() string
	Detect(home, workspace string) bool // config do engine existe?
	SkillsDir(home string) string
	RegisterMCP(home string, entry map[string]any, def MCPDef, force bool) (string, string, error) // outcome, detail, err
}

// WorkspaceMCPRegistrar é o hook opcional para adapters que registram MCP
// no WORKSPACE (ex.: vscode-copilot em .vscode/mcp.json). Quando o adapter
// implementa, registerMCP prefere este caminho passando opt.Workspace.
type WorkspaceMCPRegistrar interface {
	RegisterMCPWorkspace(workspace string, entry map[string]any, def MCPDef, force bool) (string, string, error)
}

// registry dos adapters (ordem de exibição no wizard).
var engineRegistry = map[string]EngineAdapter{}
var engineOrder = []string{}

func registerEngine(e EngineAdapter) {
	engineRegistry[e.ID()] = e
	engineOrder = append(engineOrder, e.ID())
}

// Engines devolve os ids na ordem de exibição.
func Engines() []string { return engineOrder }

// LookupEngine devolve o adapter pelo id.
func LookupEngine(id string) (EngineAdapter, bool) {
	e, ok := engineRegistry[strings.ToLower(id)]
	return e, ok
}

// EngineSupportsCompleto reporta se o engine tem skills nativas — o
// critério de compatibilidade do perfil completo (derive via SkillsDir;
// NUNCA hardcode da lista).
func EngineSupportsCompleto(engineID, home string) bool {
	adapter, ok := LookupEngine(engineID)
	return ok && adapter.SkillsDir(home) != ""
}

// CompatibleEngineIDs devolve os ids compatíveis com o perfil completo,
// na ordem do registry (mensagens de erro e filtros de lista).
func CompatibleEngineIDs(home string) []string {
	var out []string
	for _, id := range Engines() {
		if EngineSupportsCompleto(id, home) {
			out = append(out, id)
		}
	}
	return out
}

// EngineInfo é a visão de um engine para o wizard/--list-engines.
type EngineInfo struct {
	ID         string `json:"id"`
	Detected   bool   `json:"detected"`
	SkillsDir  string `json:"skills_dir,omitempty"`
	ConfigPath string `json:"config_path,omitempty"`
}

// DetectEngines lista os engines com o estado de detecção.
func DetectEngines(home, workspace string) []EngineInfo {
	var out []EngineInfo
	for _, id := range engineOrder {
		e := engineRegistry[id]
		out = append(out, EngineInfo{
			ID:       id,
			Detected: e.Detect(home, workspace),
		})
	}
	return out
}

// patchJSON aplica a entrada do MCP num arquivo JSON idempotentemente.
// Container key varia por engine ("mcpServers" | "servers"). Se o arquivo
// não existe, cria com apenas o container. Preserva TODO o resto do
// arquivo (encoding/json, sem lib de jsonc — por isso opencode vai SEMPRE
// via CLI do engine: o jsonc dele tem comentários).
func patchJSON(path, containerKey, name string, entry map[string]any, force bool) (string, string, error) {
	root := map[string]any{}
	if data, err := os.ReadFile(path); err == nil {
		if len(strings.TrimSpace(string(data))) > 0 {
			if err := json.Unmarshal(data, &root); err != nil {
				return OutcomeError, "", fmt.Errorf("config existente não é JSON válido (%s): %w", path, err)
			}
		}
	}
	container, _ := root[containerKey].(map[string]any)
	if container == nil {
		container = map[string]any{}
	}
	existing, has := container[name].(map[string]any)
	if has {
		oldJSON, _ := json.Marshal(existing)
		newJSON, _ := json.Marshal(entry)
		if string(oldJSON) == string(newJSON) {
			return OutcomeUnchanged, "bloco já presente com o mesmo conteúdo", nil
		}
		if !force {
			return OutcomeDiverged, fmt.Sprintf("bloco %s.%s existe com conteúdo diferente — pulado sem --force", containerKey, name), nil
		}
	}
	container[name] = entry
	root[containerKey] = container
	data, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return OutcomeError, "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return OutcomeError, "", err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return OutcomeError, "", fmt.Errorf("falha ao escrever %s: %w", path, err)
	}
	if has {
		return OutcomeRegistered, fmt.Sprintf("bloco %s.%s sobrescrito (--force)", containerKey, name), nil
	}
	return OutcomeRegistered, fmt.Sprintf("bloco %s.%s escrito", containerKey, name), nil
}

// readTOMLSections devolve os headers de seção TOML presentes no arquivo
// (match de cabeçalho — sem parser TOML, contrato do v0.1).
func readTOMLSections(path string) map[string]bool {
	out := map[string]bool{}
	data, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(data), "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]") {
			out[strings.Trim(t, "[]")] = true
		}
	}
	return out
}

// appendTOMLSection appenda a seção no config do codex com guarda: só se
// o header ainda não existe (sem parser TOML — append determinístico).
func appendTOMLSection(path, section string, lines []string) (string, string, error) {
	sections := readTOMLSections(path)
	if sections[section] {
		return OutcomeUnchanged, fmt.Sprintf("seção [%s] já existe no config do codex", section), nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return OutcomeError, "", err
	}
	var b strings.Builder
	if existing, err := os.ReadFile(path); err == nil && len(existing) > 0 {
		b.Write(existing)
		if !strings.HasSuffix(string(existing), "\n") {
			b.WriteString("\n")
		}
	}
	fmt.Fprintf(&b, "\n[%s]\n", section)
	for _, l := range lines {
		b.WriteString(l + "\n")
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return OutcomeError, "", fmt.Errorf("falha ao escrever %s: %w", path, err)
	}
	return OutcomeRegistered, fmt.Sprintf("seção [%s] appendada", section), nil
}

// writeEngineSkills instala as skills embutidas no skills dir do engine —
// paridade DR-008: divergente sem --force = preservado com aviso. Diferença
// só de EOL (CRLF/CR vs LF) não é customização: rematerializa em LF e
// conta como escrita (cura silenciosa). Byte-idêntico = no-op (idempotente
// — re-run não conta escrita).
func writeEngineSkills(skillsDir string, skills []update.EmbeddedSkill, force bool) (written, preserved []string, err error) {
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("falha ao criar %s: %w", skillsDir, err)
	}
	for _, sk := range skills {
		target := filepath.Join(skillsDir, sk.Dir, "SKILL.md")
		if data, rerr := os.ReadFile(target); rerr == nil {
			switch {
			case string(data) == string(sk.Content):
				continue // byte-idêntico — nada a fazer
			case !content.SameContentNormalized(data, sk.Content) && !force:
				preserved = append(preserved, sk.Name)
				continue
			}
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return written, preserved, err
		}
		if err := os.WriteFile(target, sk.Content, 0o644); err != nil {
			return written, preserved, fmt.Errorf("falha ao escrever %s: %w", target, err)
		}
		written = append(written, sk.Name)
	}
	sort.Strings(preserved)
	return written, preserved, nil
}
