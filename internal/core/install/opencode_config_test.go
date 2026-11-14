package install

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// jsonc de teste: comentários (// e /* */), nesting mcp.servers e nomes
// com hífen — o formato REAL que o `opencode mcp add` escreve.
const opencodeConfigFixture = `{
  // configuração global do opencode
  "$schema": "https://opencode.ai/config.json",
  "theme": "opencode-dark", // tema do TUI
  "mcp": {
    /* MCPs registrados pelo install */
    "codebase-memory-mcp": {
      "type": "local",
      "command": ["npx", "-y", "codebase-memory-mcp@latest"]
    },
    "golmet-x": {
      "type": "remote",
      "url": "https://mcp.example.invalid/sse"
    },
    "playwright": {
      "type": "local",
      "command": ["npx", "-y", "@playwright/mcp@latest"]
    }
  }
}
`

func writeOpencodeConfig(t *testing.T, home, content string) {
	t.Helper()
	dir := filepath.Join(home, ".config", "opencode")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "opencode.jsonc"), []byte(content), 0o644)
}

// regex do jsonc: o nome como CHAVE de objeto em qualquer nível, com bloco
// contendo "command" (stdio) ou "url" (remote) — tolerante a comentários
// jsonc (// e /* */) e nesting profundo. Nome como VALOR (não chave) não
// conta; arquivo ausente → false.
func TestOpencodeConfigHasMCPRegex(t *testing.T) {
	home := newHome(t)
	writeOpencodeConfig(t, home, opencodeConfigFixture)
	for _, name := range []string{"codebase-memory-mcp", "golmet-x"} {
		if !opencodeConfigHasMCP(home, name) {
			t.Errorf("%s deveria ser detectado no config (bloco com command/url, nesting mcp.servers, nome com hífen)", name)
		}
	}
	if opencodeConfigHasMCP(home, "postgres") {
		t.Error("postgres não deveria ser detectado (nome ausente)")
	}
	// nome presente só como VALOR (não chave de objeto) não conta
	writeOpencodeConfig(t, home, `{"mcp": {"other": {"description": "veja golmet-x no docs"}}}`)
	if opencodeConfigHasMCP(home, "golmet-x") {
		t.Error("nome como valor não deveria contar como bloco")
	}
	// arquivo ausente
	if opencodeConfigHasMCP(newHome(t), "golmet") {
		t.Error("config ausente não deveria detectar nada")
	}
}

// pós-verify: list SEM o nome (sondagem não resolveu o recém-add) + config
// COM o bloco (escrito pelo próprio `mcp add` — o mock escreve como o add
// real faz) → registrado — o config é a autoridade final.
func TestOpenCodeConfigFallbackRegistered(t *testing.T) {
	mock := newOpencodeMock()
	mock.silentFailure = true // sondagem nunca mostra o nome
	old := runEngineCmd
	runEngineCmd = func(name string, args []string) (string, error) {
		if len(args) > 1 && args[1] == "add" {
			// o add real ESCREVE o bloco no jsonc (a sondagem é à parte)
			writeOpencodeConfig(t, mock.home, `{"mcp": {"playwright": {"type": "local", "command": ["npx", "-y", "@playwright/mcp@latest"]}}}`)
		}
		return mock.run(name, args)
	}
	t.Cleanup(func() { runEngineCmd = old })

	mock.home = newHome(t) // config SEM o bloco antes do add (pré-check passa)

	e := engineOpenCode{}
	def, _ := Lookup("playwright")
	o, detail, err := e.RegisterMCP(mock.home, nil, def, false)
	if err != nil || o != OutcomeRegistered {
		t.Fatalf("fallback config = %s (%v)", o, err)
	}
	if !strings.Contains(detail, "confirmado no config") {
		t.Errorf("detail sem a confirmação do config: %s", detail)
	}
}

// pós-verify: list indisponível (lerr != nil) + config COM o bloco (escrito
// pelo próprio add) → registrado ✓ (indisponibilidade da sondagem não veta
// o registro).
func TestOpenCodeListUnavailableConfigFallback(t *testing.T) {
	lookPathFn = func(string) (string, error) { return "opencode-fake", nil }
	t.Cleanup(func() { lookPathFn = execLookPathImpl })
	home := newHome(t)
	oldRun := runEngineCmd
	runEngineCmd = func(name string, args []string) (string, error) {
		if len(args) > 1 && args[1] == "add" {
			writeOpencodeConfig(t, home, `{"mcp": {"playwright": {"type": "local", "command": ["npx", "-y", "@playwright/mcp@latest"]}}}`)
			return "", nil // add sai 0 e escreve o config
		}
		return "", context.DeadlineExceeded // sondagem indisponível
	}
	t.Cleanup(func() { runEngineCmd = oldRun })

	e := engineOpenCode{}
	def, _ := Lookup("playwright")
	o, detail, err := e.RegisterMCP(home, nil, def, false)
	if err != nil || o != OutcomeRegistered {
		t.Fatalf("list indisponível + config = %s (%v)", o, err)
	}
	if !strings.Contains(detail, "confirmado no config") {
		t.Errorf("detail sem a confirmação do config: %s", detail)
	}
}

// pré-check em cascata: config COM o bloco (mesmo com list SEM o nome) →
// skip sem rodar o add (regressão dos re-adds que duplicavam entradas).
func TestOpenCodePrecheckConfigSkip(t *testing.T) {
	mock := newOpencodeMock()
	old := runEngineCmd
	runEngineCmd = mock.run
	t.Cleanup(func() { runEngineCmd = old })

	home := newHome(t)
	writeOpencodeConfig(t, home, opencodeConfigFixture)

	e := engineOpenCode{}
	// MCPDef literal (sem Lookup): o pré-check só usa def.Name — e o nome
	// "codebase-memory-mcp" (com hífen) está no fixture, não no catálogo.
	def := MCPDef{Name: "codebase-memory-mcp", Type: MCPTypeStdio}
	o, detail, err := e.RegisterMCP(home, nil, def, false)
	if err != nil {
		t.Fatalf("RegisterMCP: %v", err)
	}
	if o != OutcomeUnchanged {
		t.Errorf("outcome = %s, want unchanged (config tem o bloco)", o)
	}
	if !strings.Contains(detail, "config") {
		t.Errorf("detail deveria citar o config: %s", detail)
	}
	if mock.addCount() != 0 {
		t.Errorf("add não deveria rodar no pré-check de config (%d adds)", mock.addCount())
	}
}

// status `pending` → registrado com nota de sondagem em curso (a tabela
// de health do `mcp list` não-determinístico).
func TestOpenCodePendingHealthNote(t *testing.T) {
	if note := opencodeHealthNote("pending"); !strings.Contains(note, "sondagem em curso") {
		t.Errorf("healthNote(pending) = %q, want contendo 'sondagem em curso'", note)
	}
}

// fluxo completo com `mcp list` reportando pending (snapshot parcial da
// sondagem): registrado ✓ com a nota.
func TestOpenCodePendingRegisteredViaList(t *testing.T) {
	mock := newOpencodeMock()
	mock.status["figma"] = "pending" // pós-add: sondagem em curso
	old := runEngineCmd
	runEngineCmd = mock.run
	t.Cleanup(func() { runEngineCmd = old })

	e := engineOpenCode{}
	def, _ := Lookup("figma")
	o, detail, err := e.RegisterMCP(newHome(t), nil, def, false)
	if err != nil || o != OutcomeRegistered {
		t.Fatalf("pending = %s (%v)", o, err)
	}
	if !strings.Contains(detail, "sondagem em curso") {
		t.Errorf("detail = %q, want contendo 'sondagem em curso'", detail)
	}
}

// erro final SÓ quando AMBAS as fontes não têm o nome: list OK sem o nome
// + config SEM o bloco → OutcomeError com mensagem citando as duas fontes
// (sem a nota de CLI-ausente — bug de composição da rodada anterior).
func TestOpenCodePostVerifyErrorRequiresBothSources(t *testing.T) {
	mock := newOpencodeMock()
	mock.silentFailure = true // list OK (vazio) + add não registrou
	old := runEngineCmd
	runEngineCmd = mock.run
	t.Cleanup(func() { runEngineCmd = old })

	home := newHome(t) // sem config
	e := engineOpenCode{}
	def, _ := Lookup("golmet")

	o, _, err := e.RegisterMCP(home, nil, def, false)
	if o != OutcomeError {
		t.Fatalf("outcome = %s, want error", o)
	}
	if err == nil {
		t.Fatal("err nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "não aparece em 'opencode mcp list'") || !strings.Contains(msg, "config do engine") {
		t.Errorf("mensagem deveria citar as duas fontes: %s", msg)
	}
	if strings.Contains(msg, "CLI não encontrada") || strings.Contains(msg, "rode manualmente") {
		t.Errorf("mensagem contaminada com a nota de CLI-ausente: %s", msg)
	}
}
