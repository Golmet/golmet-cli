package install

import (
	"strings"
	"testing"
)

// opencodeMock simula a CLI do opencode de forma STATEFUL: `mcp add`
// adiciona o nome ao set com o STATUS configurado por nome e `mcp list`
// renderiza o formato REAL (`<glifo> <nome> <status>`).
type opencodeMock struct {
	commands      [][]string
	status        map[string]string // nome → status (default connected)
	servers       map[string]string // espelho do set (para asserts)
	order         []string
	silentFailure bool   // quando true, o add sai 0 SEM registrar (regressão)
	home          string // home do teste (mocks que escrevem config no add)
}

func newOpencodeMock() *opencodeMock {
	return &opencodeMock{status: map[string]string{}, servers: map[string]string{}}
}

func (m *opencodeMock) statusOf(name string) string {
	if st, ok := m.status[name]; ok {
		return st
	}
	return "connected"
}

func (m *opencodeMock) glyph(name string) string {
	switch m.statusOf(name) {
	case "needs authentication":
		return "⚠"
	case "disabled":
		return "○"
	}
	if strings.HasPrefix(m.statusOf(name), "failed") {
		return "✗"
	}
	return "✓"
}

func (m *opencodeMock) run(name string, args []string) (string, error) {
	m.commands = append(m.commands, args)
	if len(args) > 1 && args[0] == "mcp" {
		switch args[1] {
		case "add":
			if !m.silentFailure {
				m.order = append(m.order, args[2])
				m.servers[args[2]] = m.statusOf(args[2])
			}
			return "", nil // add "sai 0" mesmo em silentFailure (regressão simulada)
		case "list":
			var b strings.Builder
			for _, n := range m.order {
				b.WriteString(m.glyph(n) + " " + n + "  " + m.statusOf(n) + "\n")
			}
			return b.String(), nil
		}
	}
	return "", nil
}

func (m *opencodeMock) addCount() int {
	n := 0
	for _, c := range m.commands {
		if len(c) > 1 && c[0] == "mcp" && c[1] == "add" {
			n++
		}
	}
	return n
}

// fixture verbatim do `opencode mcp list` REAL (máquina do operador,
// colada char-a-char — glifos ✓ ⚠ ✗ ○ e status connected · needs
// authentication · failed: <motivo> · disabled).
const opencodeListFixture = `○ aws-mcp              disabled
○ azure-devops         disabled
⚠ cloudflare           needs authentication
✓ codebase-memory-mcp  connected
✓ codegraph            connected
✓ figma                connected
✗ golmet               failed: Connection closed: MCP server process exited with code 255
⚠ hostinger            needs authentication
✓ playwright           connected
✓ postgres             connected
✓ sqlite               connected
○ ssh-manager          disabled
⚠ supabase             needs authentication
`

// fixture verbatim: o parser extrai os 13 pares (nome, status) — incluindo
// o golmet com status failed multivocal ("failed: Connection closed: ...").
func TestParseOpencodeListVerbatimFixture(t *testing.T) {
	entries := parseOpencodeList(opencodeListFixture)
	if len(entries) != 13 {
		t.Fatalf("parse = %d entradas, want 13", len(entries))
	}
	byName := map[string]opencodeListEntry{}
	for _, e := range entries {
		byName[e.name] = e
	}
	for name, wantStatus := range map[string]string{
		"aws-mcp":             "disabled",
		"azure-devops":        "disabled",
		"cloudflare":          "needs authentication",
		"codebase-memory-mcp": "connected",
		"codegraph":           "connected",
		"figma":               "connected",
		"golmet":              "failed: Connection closed: MCP server process exited with code 255",
		"hostinger":           "needs authentication",
		"playwright":          "connected",
		"postgres":            "connected",
		"sqlite":              "connected",
		"ssh-manager":         "disabled",
		"supabase":            "needs authentication",
	} {
		e, ok := byName[name]
		if !ok {
			t.Errorf("nome %s ausente do parse", name)
			continue
		}
		if e.status != wantStatus {
			t.Errorf("%s status = %q, want %q", name, e.status, wantStatus)
		}
	}
	// glifos foram stripados (nunca viram nome)
	for _, e := range entries {
		if strings.ContainsAny(e.name, "✓⚠✗○") {
			t.Errorf("glifo vazou para o nome: %q", e.name)
		}
	}
}

// formato simples (mocks de teste sem glifo) também parseia — o nome é o
// primeiro token e o resto é o status.
func TestParseOpencodeListSimpleFormat(t *testing.T) {
	entries := parseOpencodeList("golmet\ncodegraph connected\n")
	if len(entries) != 2 {
		t.Fatalf("parse = %d, want 2", len(entries))
	}
	if entries[0].name != "golmet" || entries[0].status != "" {
		t.Errorf("golmet = %+v", entries[0])
	}
	if entries[1].name != "codegraph" || entries[1].status != "connected" {
		t.Errorf("codegraph = %+v", entries[1])
	}
	// cabeçalho sem glifo com múltiplas palavras: o primeiro token é o nome
	entries2 := parseOpencodeList("nenhum mcp registrado")
	if len(entries2) != 1 || entries2[0].name != "nenhum" {
		t.Errorf("cabeçalho parseado como entrada com nome errado: %+v", entries2)
	}
}

// pré-check de duplicata: nome JÁ presente em QUALQUER status → skip
// silencioso (OutcomeUnchanged) — o `mcp add` NUNCA é executado de novo
// (regressão dos ×2 que o config do operador ganhou).
func TestOpenCodeDuplicatePrecheckSkip(t *testing.T) {
	mock := newOpencodeMock()
	mock.status["golmet"] = "failed: Connection closed: MCP server process exited with code 255"
	mock.order = []string{"golmet"}
	lookPathFn = func(string) (string, error) { return "opencode-fake", nil }
	t.Cleanup(func() { lookPathFn = execLookPathImpl })
	old := runEngineCmd
	runEngineCmd = mock.run
	t.Cleanup(func() { runEngineCmd = old })

	e := engineOpenCode{}
	def, _ := Lookup("golmet") // status REAL: failed (o caso do operador)

	o, detail, err := e.RegisterMCP(newHome(t), nil, def, false)
	if err != nil {
		t.Fatalf("RegisterMCP: %v", err)
	}
	if o != OutcomeUnchanged {
		t.Errorf("outcome = %s, want unchanged (já registrado)", o)
	}
	if !strings.Contains(detail, "failed") {
		t.Errorf("detail deveria citar o status atual do engine: %s", detail)
	}
	if mock.addCount() != 0 {
		t.Errorf("mcp add não deveria rodar no pré-check de duplicata (%d adds)", mock.addCount())
	}
}

// --force re-adda mesmo com o nome já presente (sobrescrita intencional).
func TestOpenCodeForceReAdds(t *testing.T) {
	mock := newOpencodeMock()
	mock.status["golmet"] = "failed: Connection closed: MCP server process exited with code 255"
	mock.order = []string{"golmet"}
	lookPathFn = func(string) (string, error) { return "opencode-fake", nil }
	t.Cleanup(func() { lookPathFn = execLookPathImpl })
	old := runEngineCmd
	runEngineCmd = mock.run
	t.Cleanup(func() { runEngineCmd = old })

	e := engineOpenCode{}
	def, _ := Lookup("golmet")
	o, _, err := e.RegisterMCP(newHome(t), nil, def, true)
	if err != nil || o != OutcomeRegistered {
		t.Fatalf("force re-add = %s %v", o, err)
	}
	if mock.addCount() != 1 {
		t.Errorf("--force deveria executar o mcp add (%d)", mock.addCount())
	}
}

// pós-verify: o add adiciona o nome ao set do mock e o `mcp list`
// subsequente o lista — registered com a health note por status.
func TestOpenCodeRegisterStatusesFromFixture(t *testing.T) {
	e := engineOpenCode{}
	home := newHome(t)

	cases := []struct {
		name       string
		mockStatus string // status que o mock reporta para esse nome
		wantNote   string // substring esperada no detail
	}{
		{"codegraph", "connected", "status: connected"},
		{"postgres", "connected", "status: connected"},
		{"supabase", "needs authentication", "autentique no engine"},
		{"golmet", "failed: Connection closed: MCP server process exited with code 255", "não conseguiu conectar"},
	}
	for _, tc := range cases {
		mock := newOpencodeMock()            // nasce VAZIO — o pré-check vê lista vazia
		mock.status[tc.name] = tc.mockStatus // status que o `add` registra
		lookPathFn = func(string) (string, error) { return "opencode-fake", nil }
		t.Cleanup(func() { lookPathFn = execLookPathImpl })
		old := runEngineCmd
		runEngineCmd = mock.run
		def, ok := Lookup(tc.name)
		if !ok {
			t.Fatalf("mcp %s ausente do catálogo", tc.name)
		}
		o, detail, err := e.RegisterMCP(home, nil, def, false)
		runEngineCmd = old
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		if o != OutcomeRegistered {
			t.Errorf("%s outcome = %s, want registered", tc.name, o)
		}
		if !strings.Contains(detail, tc.wantNote) {
			t.Errorf("%s detail = %q, want contendo %q", tc.name, detail, tc.wantNote)
		}
	}
}

// pós-verify com nome ausente da lista → OutcomeError com mensagem APENAS
// sobre o pós-verify (sem a nota de CLI-ausente — bug de composição do
// operador: a nota de fallback foi anexada a um outcome que RODOU a CLI).
func TestOpenCodePostVerifyErrorMessageClean(t *testing.T) {
	mock := newOpencodeMock()
	mock.silentFailure = true // add sai 0 sem registrar (regressão simulada)
	lookPathFn = func(string) (string, error) { return "opencode-fake", nil }
	t.Cleanup(func() { lookPathFn = execLookPathImpl })
	old := runEngineCmd
	runEngineCmd = mock.run
	t.Cleanup(func() { runEngineCmd = old })

	e := engineOpenCode{}
	def, _ := Lookup("golmet")

	o, _, err := e.RegisterMCP(newHome(t), nil, def, false)
	if o != OutcomeError {
		t.Fatalf("outcome = %s, want error", o)
	}
	if err == nil {
		t.Fatal("err nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "não aparece em 'opencode mcp list'") {
		t.Errorf("mensagem sem o pós-verify: %s", msg)
	}
	if strings.Contains(msg, "CLI não encontrada") || strings.Contains(msg, "rode manualmente") {
		t.Errorf("mensagem contaminada com a nota de CLI-ausente: %s", msg)
	}
}
