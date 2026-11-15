package install

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func execLookPathImpl(s string) (string, error) { return exec.LookPath(s) }

// Células da matriz docs/INSTALL_COMPATIBILITY.md que faltavam travar:

// opencode remote: args da CLI com --url (wire) + pós-verify.
func TestOpenCodeRemoteURLWire(t *testing.T) {
	lookPathFn = func(string) (string, error) { return "opencode-fake", nil }
	t.Cleanup(func() { lookPathFn = execLookPathImpl })
	home := newHome(t)
	os.MkdirAll(filepath.Join(home, ".config", "opencode"), 0o755)

	var commands [][]string
	mock := newOpencodeMock() // stateful: pré-check vê lista vazia, add registra
	oldRun := runEngineCmd
	runEngineCmd = func(name string, args []string) (string, error) {
		commands = append(commands, args)
		return mock.run(name, args)
	}
	t.Cleanup(func() { runEngineCmd = oldRun })

	e := engineOpenCode{}
	def, _ := Lookup("hostinger")
	o, _, err := e.RegisterMCP(home, nil, def, false)
	if err != nil || o != OutcomeRegistered {
		t.Fatalf("remote opencode: %s %v", o, err)
	}
	found := false
	for _, c := range commands {
		if strings.Join(c, " ") == "mcp add hostinger --global --url "+def.URL {
			found = true
		}
	}
	if !found {
		t.Errorf("add remote sem --url wire: %v", commands)
	}
}

func execLookPath(s string) (string, error) { return execLookPathImpl(s) }

// claude-code remote SEM CLI no PATH → fallback patch {type:http, url}.
func TestClaudeRemoteFallbackPatch(t *testing.T) {
	noEngineCLI(t)
	home := newHome(t)
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	def, _ := Lookup("hostinger")
	entry := map[string]any{"type": "http", "url": def.URL} // entry montado pelo registerMCP no uso real
	e := engineClaude{}
	o, _, err := e.RegisterMCP(home, entry, def, false)
	if err != nil || o != OutcomeRegistered {
		t.Fatalf("claude remote fallback: %s %v", o, err)
	}
	got := entryOf(t, filepath.Join(home, ".claude.json"), "mcpServers", "hostinger")
	if got["type"] != "http" || got["url"] != def.URL {
		t.Errorf("bloco remote do claude malformado: %+v", got)
	}
}

// env literal escrita no patch dos engines JSON (vscode/cursor/gemini) —
// mesma decisão do claude: valor do ambiente no config local, nunca logado.
func TestEnvLiteralOnJSONEngines(t *testing.T) {
	t.Setenv("FIGMA_API_KEY", "chave-figma-literal")
	cases := []struct {
		engine    EngineAdapter
		cfgPath   func(home string) string
		container string
	}{
		{engineVSCode{}, func(h string) string { return filepath.Join(h, ".vscode", "mcp.json") }, "servers"},
		{engineCursor{}, func(h string) string { return filepath.Join(h, ".cursor", "mcp.json") }, "mcpServers"},
		{engineGemini{}, func(h string) string { return filepath.Join(h, ".gemini", "settings.json") }, "mcpServers"},
	}
	for _, tc := range cases {
		home := newHome(t)
		def, _ := Lookup("figma")
		entry := map[string]any{"command": def.Command, "args": def.Args,
			"env": map[string]string{"FIGMA_API_KEY": os.Getenv("FIGMA_API_KEY")}}
		var o string
		var err error
		target := tc.cfgPath(home)
		if wr, ok := tc.engine.(WorkspaceMCPRegistrar); ok {
			ws2 := t.TempDir()
			o, _, err = wr.RegisterMCPWorkspace(ws2, entry, def, false)
			target = filepath.Join(ws2, ".vscode", "mcp.json")
		} else {
			o, _, err = tc.engine.RegisterMCP(home, entry, def, false)
		}
		if err != nil || o != OutcomeRegistered {
			t.Fatalf("%s: %s %v", tc.engine.ID(), o, err)
		}
		raw := readFileT(t, target)
		if !strings.Contains(raw, `"FIGMA_API_KEY": "chave-figma-literal"`) {
			t.Errorf("%s: env literal ausente do config: %s", tc.engine.ID(), raw)
		}
	}
}

// skills escritas no opencode mesmo SEM a CLI do engine (o write de skills
// não depende da CLI — só o registro de MCP degrada para manual).
func TestOpenCodeSkillsWithoutCLI(t *testing.T) {
	noEngineCLI(t)
	home := newHome(t)
	os.MkdirAll(filepath.Join(home, ".config", "opencode"), 0o755)
	res, err := Run(Options{
		EngineID: "opencode",
		MCPs:     []string{"golmet"},
		Home:     home,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Report.SkillsWritten) != 16 {
		t.Errorf("skills escritas = %d, want 16", len(res.Report.SkillsWritten))
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "opencode", "skills", "gm-charter", "SKILL.md")); err != nil {
		t.Errorf("skill não escrita no skills dir do opencode: %v", err)
	}
	// MCP golmet degrada para manual (CLI ausente) — parcial
	if res.Partial == false {
		t.Errorf("MCP manual deveria marcar Partial")
	}
}

// Matriz completa 7 engines × {essencial, completo} (docs/INSTALL_COMPATIBILITY.md):
// essencial (via Run sem Full) vale para TODOS os engines; completo vale
// APENAS para os engines com skills nativas (opencode, claude-code —
// materialização no config root); engines sem skills nativas falham
// fail-closed (a zona .golmet/ virou comportamento interno do pipeline).
func TestEngineMatrixProfilesAllSeven(t *testing.T) {
	noEngineCLI(t)
	engines := Engines()
	if len(engines) != 7 {
		t.Fatalf("registry tem %d engines, want 7 — atualize a matriz", len(engines))
	}
	for _, id := range engines {
		adapter, ok := LookupEngine(id)
		if !ok {
			t.Fatalf("engine %q ausente do registry", id)
		}
		hasSkills := adapter.SkillsDir(newHome(t)) != ""

		// essencial — caminho válido em todos os engines
		baseHome := newHome(t)
		resBase, err := Run(Options{EngineID: id, Home: baseHome})
		if err != nil {
			t.Fatalf("essencial %s: %v", id, err)
		}
		if hasSkills {
			if !resBase.Report.SkillsSupported || len(resBase.Report.SkillsWritten) != 16 {
				t.Errorf("essencial %s: supported=%v written=%d, want true/16", id, resBase.Report.SkillsSupported, len(resBase.Report.SkillsWritten))
			}
		} else if resBase.Report.SkillsSupported {
			t.Errorf("essencial %s não tem skills nativas mas reportou suporte", id)
		}

		// completo — via Run: nativo (config root) nos engines com skills;
		// ERRO fail-closed nos engines sem skills nativas (a zona virou
		// comportamento interno do pipeline, não exposição de UX)
		fullHome := newHome(t)
		ws := t.TempDir()
		resFull, err := Run(Options{EngineID: id, Full: true, Home: fullHome, Workspace: ws})
		if !hasSkills {
			if err == nil || !strings.Contains(err.Error(), "perfil completo exige engine com skills nativas") {
				t.Fatalf("completo %s deveria falhar fail-closed, veio err=%v report=%+v", id, err, resFull)
			}
			if _, serr := os.Stat(filepath.Join(ws, ".golmet")); !os.IsNotExist(serr) {
				t.Errorf("completo %s escreveu na zona apesar do erro", id)
			}
			continue
		}
		if err != nil {
			t.Fatalf("completo %s: %v", id, err)
		}
		if resFull.Report.FullTotal != 126 {
			t.Errorf("completo %s: total=%d, want 126", id, resFull.Report.FullTotal)
		}
		root := filepath.Dir(adapter.SkillsDir(fullHome))
		if resFull.Report.FullZone != FullZoneEngineConfig || resFull.Report.FullZonePath != root {
			t.Errorf("completo %s: zone=%q path=%q, want engine-config/%q", id, resFull.Report.FullZone, resFull.Report.FullZonePath, root)
		}
		for _, rel := range []string{"agents/gm-dev.md", "skills/gm-reverse/SKILL.md", ".golmet/manifest.json"} {
			if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
				t.Errorf("completo %s ausente %s: %v", id, rel, err)
			}
		}
	}
}
