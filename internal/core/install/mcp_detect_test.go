package install

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// cascata opencode: list COM entradas → status real da sondagem; o config
// AUSENTE no home prova que a fonte 2 não foi consultada.
func TestInstalledMCPsOpenCodeListWins(t *testing.T) {
	home := newHome(t) // SEM opencode.jsonc — se a fonte 2 rodasse, viria vazio
	old := runEngineCmd
	runEngineCmd = func(name string, args []string) (string, error) {
		return "✓ figma  connected\n✗ sqlite  failed: exit 1\n", nil
	}
	t.Cleanup(func() { runEngineCmd = old })

	m := InstalledMCPs(home, newHome(t), "opencode")
	if m["figma"] != "connected" || m["sqlite"] != "failed: exit 1" {
		t.Errorf("status real da sondagem = %v", m)
	}
}

// sondagem NÃO-determinística: list vazio (exit 0, "No MCP servers
// configured") com 13 servidores no config → fallback ao config: TODOS os
// nomes do jsonc viram badges (fixture real: comentários, nesting
// mcp.servers, hífens em nomes).
func TestInstalledMCPsOpenCodeConfigFallback(t *testing.T) {
	home := newHome(t)
	os.MkdirAll(filepath.Join(home, ".config", "opencode"), 0o755)
	os.WriteFile(filepath.Join(home, ".config", "opencode", "opencode.jsonc"),
		[]byte(opencodeConfigFixture), 0o644)

	old := runEngineCmd
	runEngineCmd = func(name string, args []string) (string, error) {
		return "", nil // exit 0, lista VAZIA (a mentira do engine)
	}
	t.Cleanup(func() { runEngineCmd = old })

	m := InstalledMCPs(home, newHome(t), "opencode")
	for _, name := range []string{"codebase-memory-mcp", "golmet-x", "playwright"} {
		if m[name] != "config" {
			t.Errorf("%s deveria vir do config com status \"config\", veio %q", name, m[name])
		}
	}
}

// a mentira do engine com TEXTO (caso real capturado: exit 0 com "No MCP
// servers configured" minutos depois de listar 13) NÃO vira entrada falsa
// no parser — lista vazia → fallback ao config intacto.
func TestParseOpencodeListEmptyTextIsNotAnEntry(t *testing.T) {
	if e := parseOpencodeList("No MCP servers configured\r\n"); len(e) != 0 {
		t.Errorf("texto de lista vazia deveria devolver 0 entradas, veio %v", e)
	}
	home := newHome(t)
	os.MkdirAll(filepath.Join(home, ".config", "opencode"), 0o755)
	os.WriteFile(filepath.Join(home, ".config", "opencode", "opencode.jsonc"),
		[]byte(opencodeConfigFixture), 0o644)
	old := runEngineCmd
	runEngineCmd = func(name string, args []string) (string, error) {
		return "No MCP servers configured\r\n", nil
	}
	t.Cleanup(func() { runEngineCmd = old })

	m := InstalledMCPs(home, newHome(t), "opencode")
	if m["codebase-memory-mcp"] != "config" {
		t.Errorf("fallback ao config deveria preservar os badges, veio %v", m)
	}
}

// list vazio + config SEM MCPs → vazio (nenhum badge inventado).
func TestInstalledMCPsOpenCodeEmptyConfig(t *testing.T) {
	home := newHome(t)
	os.MkdirAll(filepath.Join(home, ".config", "opencode"), 0o755)
	os.WriteFile(filepath.Join(home, ".config", "opencode", "opencode.jsonc"),
		[]byte("{\n  // só tema, sem MCPs\n  \"theme\": \"dark\"\n}\n"), 0o644)

	old := runEngineCmd
	runEngineCmd = func(name string, args []string) (string, error) {
		return "", nil // exit 0, vazio
	}
	t.Cleanup(func() { runEngineCmd = old })

	if m := InstalledMCPs(home, newHome(t), "opencode"); m != nil {
		t.Errorf("sem MCPs em lugar nenhum deveria devolver nil, veio %v", m)
	}
}

// list indisponível (erro de ambiente) + config AUSENTE → vazio.
func TestInstalledMCPsOpenCodeListErrNoConfig(t *testing.T) {
	old := runEngineCmd
	runEngineCmd = func(name string, args []string) (string, error) {
		return "", context.DeadlineExceeded
	}
	t.Cleanup(func() { runEngineCmd = old })

	if m := InstalledMCPs(newHome(t), newHome(t), "opencode"); m != nil {
		t.Errorf("list err + config ausente deveria devolver nil, veio %v", m)
	}
}

// DRY: opencodeConfigHasMCP continua funcionando via a MESMA enumeração —
// nome ∈ enumeração; ausente/valor-não-objeto → false.
func TestOpencodeConfigHasMCPViaEnumeration(t *testing.T) {
	home := newHome(t)
	os.MkdirAll(filepath.Join(home, ".config", "opencode"), 0o755)
	os.WriteFile(filepath.Join(home, ".config", "opencode", "opencode.jsonc"),
		[]byte(opencodeConfigFixture), 0o644)

	if !opencodeConfigHasMCP(home, "codebase-memory-mcp") {
		t.Errorf("hasMCP deveria achar o bloco com command (nome com hífen)")
	}
	if !opencodeConfigHasMCP(home, "golmet-x") {
		t.Errorf("hasMCP deveria achar o bloco remote com url")
	}
	if opencodeConfigHasMCP(home, "postgres") {
		t.Errorf("nome ausente deveria dar false")
	}
	// nome-como-VALOR (não chave de objeto) não conta
	os.WriteFile(filepath.Join(home, ".config", "opencode", "opencode.jsonc"),
		[]byte(`{"mcp": {"other": {"description": "veja golmet-x no docs"}}}`), 0o644)
	if opencodeConfigHasMCP(home, "golmet-x") {
		t.Errorf("nome como valor não deveria contar como bloco")
	}
}
