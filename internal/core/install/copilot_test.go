package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// O Copilot CLI respeita COPILOT_HOME (default ~/.copilot); o config é
// mcp-config.json com top-level mcpServers (forma do Claude) e entradas
// {"type":"local"|"http", ...} (docs.github.com copilot-cli).
func TestCopilotPatchIdempotentCOPILOTHomeAndRemote(t *testing.T) {
	// COPILOT_HOME virtual injetada — nunca toca no ~/.copilot real
	copilotHome := t.TempDir()
	t.Setenv("COPILOT_HOME", copilotHome)
	noEngineCLI(t) // patch é o caminho exercitado (CLI real nunca nos testes)
	e := engineCopilot{}

	// detect pela pasta COPILOT_HOME
	if !e.Detect(newHome(t), "") {
		t.Errorf("detect deveria ser true com COPILOT_HOME presente")
	}

	// stdio: patch com forma {"type":"local","command","args"}
	def, _ := Lookup("playwright")
	entry := map[string]any{"command": def.Command, "args": def.Args}
	o, _, err := e.RegisterMCP(newHome(t), entry, def, false)
	if err != nil || o != OutcomeRegistered {
		t.Fatalf("stdio patch: %s %v", o, err)
	}
	cfg := filepath.Join(copilotHome, "mcp-config.json")
	got := entryOf(t, cfg, "mcpServers", "playwright")
	if got["type"] != "local" || got["command"] != def.Command {
		t.Errorf("bloco copilot malformado: %+v", got)
	}
	// idempotente
	o2, _, err := e.RegisterMCP(newHome(t), entry, def, false)
	if err != nil || o2 != OutcomeUnchanged {
		t.Errorf("re-run deveria ser unchanged: %s %v", o2, err)
	}
	// divergência sem force → pulado
	entryDiv := map[string]any{"command": "outro", "args": def.Args}
	o3, _, err := e.RegisterMCP(newHome(t), entryDiv, def, false)
	if err != nil || o3 != OutcomeDiverged {
		t.Errorf("divergência sem force = %s %v", o3, err)
	}

	// remote: forma {"type":"http","url"} (docs.github copilot-cli)
	hostinger, _ := Lookup("hostinger")
	o4, _, err := e.RegisterMCP(newHome(t), nil, hostinger, false)
	if err != nil || o4 != OutcomeRegistered {
		t.Fatalf("remote patch: %s %v", o4, err)
	}
	got4 := entryOf(t, cfg, "mcpServers", "hostinger")
	if got4["type"] != "http" || got4["url"] != hostinger.URL {
		t.Errorf("remote copilot malformado: %+v", got4)
	}
}

// COPILOT_HOME ausente → default ~/.copilot (home injetada).
func TestCopilotDefaultHome(t *testing.T) {
	noEngineCLI(t) // patch direto (CLI-first exige lookPath injetado nos testes de CLI)
	home := newHome(t)
	os.Unsetenv("COPILOT_HOME")
	e := engineCopilot{}
	def, _ := Lookup("postgres")
	entry := map[string]any{"command": def.Command, "args": def.Args}
	if o, _, err := e.RegisterMCP(home, entry, def, false); err != nil || o != OutcomeRegistered {
		t.Fatalf("patch: %s %v", o, err)
	}
	cfg := filepath.Join(home, ".copilot", "mcp-config.json")
	got := entryOf(t, cfg, "mcpServers", "postgres")
	if got["command"] != "npx" {
		t.Errorf("bloco no default home malformado: %+v", got)
	}
	argsJSON, _ := json.Marshal(got["args"])
	if string(argsJSON) != `["-y","@microsoft/postgres-mcp","run"]` {
		t.Errorf("args do postgres incorretos: %s", argsJSON)
	}
}

// SkillsDir vazio (sem sistema de skills) e detect via lookPath quando a
// pasta COPILOT_HOME não existe.
func TestCopilotDetectViaLookPathAndNoSkills(t *testing.T) {
	home := newHome(t)
	os.Unsetenv("COPILOT_HOME")
	old := lookPathFn
	lookPathFn = func(string) (string, error) { return "copilot-fake", nil }
	t.Cleanup(func() { lookPathFn = old })

	e := engineCopilot{}
	if !e.Detect(home, "") {
		t.Errorf("detect deveria ser true com copilot no PATH")
	}
	if e.SkillsDir(home) != "" {
		t.Errorf("copilot não tem sistema de skills — SkillsDir deveria ser vazio")
	}
}

// env do stdio entra no patch com valor literal (mesmo padrão do claude).
func TestCopilotEnvLiteralInPatch(t *testing.T) {
	noEngineCLI(t) // patch direto
	home := newHome(t)
	os.Unsetenv("COPILOT_HOME")
	t.Setenv("AWS_PROFILE", "perfil-local")
	e := engineCopilot{}
	def, _ := Lookup("aws")
	entry := map[string]any{"command": def.Command, "args": def.Args,
		"env": map[string]string{"AWS_PROFILE": os.Getenv("AWS_PROFILE")}}
	if o, _, err := e.RegisterMCP(home, entry, def, false); err != nil || o != OutcomeRegistered {
		t.Fatalf("patch aws: %s %v", o, err)
	}
	cfg := filepath.Join(home, ".copilot", "mcp-config.json")
	raw := readFileT(t, cfg)
	if !strings.Contains(raw, `"AWS_PROFILE": "perfil-local"`) {
		t.Errorf("env literal ausente do config: %s", raw)
	}
}

// CLI-first: copilot no PATH (injetado) → registro via CLI com env por
// --env (valor literal) e verificação pós-add via `copilot mcp get`.
func TestCopilotCLIFirstWithEnvAndPostVerify(t *testing.T) {
	home := newHome(t)
	os.Unsetenv("COPILOT_HOME")
	old := lookPathFn
	lookPathFn = func(string) (string, error) { return "copilot-fake", nil }
	t.Cleanup(func() { lookPathFn = old })

	var commands [][]string
	oldRun := runEngineCmd
	runEngineCmd = func(name string, args []string) (string, error) {
		commands = append(commands, args)
		if len(args) > 1 && args[0] == "mcp" && args[1] == "get" {
			// `copilot mcp get <nome>` confirma a entrada adicionada
			return "def " + args[2], nil
		}
		return "", nil // add sai 0
	}
	t.Cleanup(func() { runEngineCmd = oldRun })

	t.Setenv("AWS_PROFILE", "perfil-cli")
	e := engineCopilot{}
	def, _ := Lookup("aws")
	o, detail, err := e.RegisterMCP(home, nil, def, false)
	if err != nil || o != OutcomeRegistered {
		t.Fatalf("CLI-first: %s %v (%s)", o, err, detail)
	}
	// add com --env KEY=valor literal
	var addCmd []string
	for _, c := range commands {
		if len(c) > 1 && c[0] == "mcp" && c[1] == "add" {
			addCmd = c
		}
	}
	joined := strings.Join(addCmd, " ")
	if !strings.Contains(joined, "--env AWS_PROFILE=perfil-cli") {
		t.Errorf("--env com valor literal ausente: %s", joined)
	}
	// verificação pós-add aconteceu
	var gotGet bool
	for _, c := range commands {
		if len(c) > 2 && c[0] == "mcp" && c[1] == "get" && c[2] == def.Name {
			gotGet = true
		}
	}
	if !gotGet {
		t.Errorf("verificação pós-add (mcp get) não executou: %v", commands)
	}

	// get falha (sai 0 mas sem a entrada) → OutcomeError
	runEngineCmd = func(name string, args []string) (string, error) {
		if len(args) > 1 && args[0] == "mcp" && args[1] == "get" {
			return "não encontrado", os.ErrNotExist
		}
		return "", nil
	}
	o2, d2, err2 := e.RegisterMCP(home, nil, def, false)
	if o2 != OutcomeError || err2 == nil || !strings.Contains(err2.Error(), "mcp get") {
		t.Errorf("falso-positivo deveria virar error citando o get: %s (%v) (%s)", o2, err2, d2)
	}
}
