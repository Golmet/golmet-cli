package install

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func newHome(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// noEngineCLI garante que os CLIs reais dos engines NUNCA são executados
// nos testes (o operador pode ter opencode/claude no PATH — o registro
// via CLI cai no outcome manual/patch com injeção de LookPath).
func noEngineCLI(t *testing.T) {
	t.Helper()
	old := lookPathFn
	lookPathFn = func(string) (string, error) { return "", os.ErrNotExist }
	t.Cleanup(func() { lookPathFn = old })
}

func readFileT(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func catJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("arquivo ausente: %s (%v)", path, err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json inválido em %s: %v", path, err)
	}
	return m
}

func entryOf(t *testing.T, path, container, name string) map[string]any {
	t.Helper()
	root := catJSON(t, path)
	containerMap, ok := root[container].(map[string]any)
	if !ok {
		t.Fatalf("container %q ausente em %s", container, path)
	}
	entry, ok := containerMap[name].(map[string]any)
	if !ok {
		t.Fatalf("entrada %q ausente em %s.%s", name, container, path)
	}
	return entry
}

func runtimeGOOS() string { return runtime.GOOS }

// ---- detecção ----

func TestEngineDetect(t *testing.T) {
	noEngineCLI(t) // copilot detecta por CLI — sem injeção, vazaria do PATH real
	home := newHome(t)
	ws := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".config", "opencode"), 0o755)
	os.WriteFile(filepath.Join(home, ".config", "opencode", "opencode.jsonc"), []byte("// jsonc\n{}"), 0o644)
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	os.MkdirAll(filepath.Join(home, ".cursor"), 0o755)
	os.MkdirAll(filepath.Join(home, ".gemini"), 0o755)
	os.MkdirAll(filepath.Join(home, ".codex"), 0o755)
	os.MkdirAll(filepath.Join(ws, ".vscode"), 0o755)

	infos := DetectEngines(home, ws)
	byID := map[string]bool{}
	for _, info := range infos {
		byID[info.ID] = info.Detected
	}
	for _, id := range []string{"opencode", "claude-code", "cursor", "gemini-cli", "codex", "vscode-copilot"} {
		if !byID[id] {
			t.Errorf("engine %s deveria estar detectado: %+v", id, byID)
		}
	}
	// home vazia → nada detectado
	empty := DetectEngines(newHome(t), t.TempDir())
	for _, info := range empty {
		if info.Detected {
			t.Errorf("engine %s detectado em home vazia", info.ID)
		}
	}
}

// ---- patch JSON idempotente (claude-code fallback / vscode / cursor / gemini) ----

func TestClaudePatchIdempotentAndDiverge(t *testing.T) {
	noEngineCLI(t)
	home := newHome(t)
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755) // detect
	def, _ := Lookup("playwright")
	entry := map[string]any{"command": def.Command, "args": def.Args}

	e := engineClaude{}
	o1, d1, err := e.RegisterMCP(home, entry, def, false)
	if err != nil || o1 != OutcomeRegistered {
		t.Fatalf("1º registro: %s %s %v", o1, d1, err)
	}
	// idempotente: mesmo conteúdo → unchanged
	o2, _, err := e.RegisterMCP(home, entry, def, false)
	if err != nil || o2 != OutcomeUnchanged {
		t.Fatalf("2º registro deveria ser unchanged: %s %v", o2, err)
	}
	// conteúdo divergente sem --force → diverged (pula)
	entry2 := map[string]any{"command": "outro", "args": def.Args}
	o3, d3, err := e.RegisterMCP(home, entry2, def, false)
	if err != nil || o3 != OutcomeDiverged {
		t.Fatalf("divergência sem force = %s (%s) %v", o3, d3, err)
	}
	got := entryOf(t, filepath.Join(home, ".claude.json"), "mcpServers", def.Name)
	if got["command"] != def.Command {
		t.Errorf("bloco divergente foi sobrescrito sem --force: %+v", got)
	}
	// com --force → sobrescreve
	o4, _, err := e.RegisterMCP(home, entry2, def, true)
	if err != nil || o4 != OutcomeRegistered {
		t.Fatalf("--force = %s %v", o4, err)
	}
	got = entryOf(t, filepath.Join(home, ".claude.json"), "mcpServers", def.Name)
	if got["command"] != "outro" {
		t.Errorf("--force não sobrescreveu: %+v", got)
	}
	// resto do arquivo preservado (chave alheia no mesmo config)
	os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{"outraChave": true, "mcpServers": {}}`), 0o644)
	e.RegisterMCP(home, entry, def, true)
	root := catJSON(t, filepath.Join(home, ".claude.json"))
	if root["outraChave"] != true {
		t.Errorf("patch JSON perdeu chaves alheias: %+v", root)
	}
}

func TestCodexTOMLAppendSectionGuard(t *testing.T) {
	noEngineCLI(t)
	home := newHome(t)
	os.MkdirAll(filepath.Join(home, ".codex"), 0o755)
	def, _ := Lookup("playwright")
	entry := map[string]any{"command": def.Command, "args": def.Args}
	e := engineCodex{}

	o1, _, err := e.RegisterMCP(home, entry, def, false)
	if err != nil || o1 != OutcomeRegistered {
		t.Fatalf("append 1: %s %v", o1, err)
	}
	tomlPath := filepath.Join(home, ".codex", "config.toml")
	content := readFileT(t, tomlPath)
	if !strings.Contains(content, "[mcp_servers."+def.Name+"]") ||
		!strings.Contains(content, `"npx"`) {
		t.Errorf("seção TOML incompleta:\n%s", content)
	}
	// wire codex: command e args SEPARADOS — o command não se duplica no args
	if strings.Contains(content, `args = ["`+def.Command+`"`) {
		t.Errorf("command duplicado dentro de args (wire incorreto):\n%s", content)
	}
	// guarda: re-run → unchanged, sem duplicar seção
	o2, _, err := e.RegisterMCP(home, entry, def, false)
	if err != nil || o2 != OutcomeUnchanged {
		t.Fatalf("append 2 = %s (%v)", o2, err)
	}
	content = readFileT(t, tomlPath)
	if strings.Count(content, "[mcp_servers."+def.Name+"]") != 1 {
		t.Errorf("seção duplicada no re-run:\n%s", content)
	}
	// outra skill → nova seção no mesmo arquivo
	def2, _ := Lookup("figma")
	e.RegisterMCP(home, map[string]any{"command": def2.Command, "args": def2.Args}, def2, false)
	content = readFileT(t, tomlPath)
	if !strings.Contains(content, "[mcp_servers.figma]") {
		t.Errorf("segunda seção ausente:\n%s", content)
	}
	// remote não suportado no codex v0.1
	hostinger, _ := Lookup("hostinger")
	o3, _, _ := e.RegisterMCP(home, map[string]any{}, hostinger, false)
	if o3 != OutcomeUnsupported {
		t.Errorf("codex remote = %s, want unsupported", o3)
	}
}

func TestVSCodeWorkspaceConfigAndCursorRemoteGemini(t *testing.T) {
	home := newHome(t)
	ws := t.TempDir()
	os.MkdirAll(filepath.Join(ws, ".vscode"), 0o755)
	def, _ := Lookup("golmet")
	entry := map[string]any{"command": selfExeForTest(), "args": []string{"mcp"}}

	vs := engineVSCode{}
	o, _, err := vs.RegisterMCPWorkspace(ws, entry, def, false)
	if err != nil || o != OutcomeRegistered {
		t.Fatalf("vscode: %s %v", o, err)
	}
	got := entryOf(t, filepath.Join(ws, ".vscode", "mcp.json"), "servers", "golmet")
	if got["type"] != "stdio" || got["command"] == "" {
		t.Errorf("bloco vscode malformado: %+v", got)
	}

	cur := engineCursor{}
	os.MkdirAll(filepath.Join(home, ".cursor"), 0o755)
	hostinger, _ := Lookup("hostinger")
	o, _, err = cur.RegisterMCP(home, nil, hostinger, false)
	if err != nil || o != OutcomeRegistered {
		t.Fatalf("cursor remote: %s %v", o, err)
	}
	got = entryOf(t, filepath.Join(home, ".cursor", "mcp.json"), "mcpServers", "hostinger")
	if got["url"] != hostinger.URL {
		t.Errorf("cursor remote sem url: %+v", got)
	}

	gem := engineGemini{}
	o, _, err = gem.RegisterMCP(home, entry, def, false)
	if err != nil || o != OutcomeRegistered {
		t.Fatalf("gemini: %s %v", o, err)
	}
	entryOf(t, filepath.Join(home, ".gemini", "settings.json"), "mcpServers", "golmet")

	// remote não suportado no gemini v0.1
	o, _, err = gem.RegisterMCP(home, nil, hostinger, false)
	if o != OutcomeUnsupported {
		t.Errorf("gemini remote = %s, want unsupported", o)
	}
}

func selfExeForTest() string {
	p, _ := os.Executable()
	return p
}

// ---- Run ponta a ponta (flags não-interativas, HOME virtual) ----

func TestRunSkillsAndMCPPatch(t *testing.T) {
	noEngineCLI(t)
	home := newHome(t)
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	res, err := Run(Options{
		EngineID:  "claude-code",
		Profile:   "base",
		MCPs:      []string{"golmet", "playwright"},
		Workspace: home, // doctor roda e falha suave (sem golmet.yaml)
		Home:      home,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !res.Report.SkillsSupported || len(res.Report.SkillsWritten) != 16 {
		t.Errorf("skills: supported=%v written=%d", res.Report.SkillsSupported, len(res.Report.SkillsWritten))
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "skills", "gm-charter", "SKILL.md")); err != nil {
		t.Errorf("gm-charter não instalada no engine: %v", err)
	}
	// golmet MCP com command resolvido para o exe corrente + doctor summary
	got := entryOf(t, filepath.Join(home, ".claude.json"), "mcpServers", "golmet")
	if got["command"] != selfExeForTest() {
		t.Errorf("command do golmet = %v, want o exe corrente", got["command"])
	}
	if res.Report.Doctor == nil || res.Report.Doctor.Workspace == "" {
		t.Errorf("doctor final ausente: %+v", res.Report.Doctor)
	}
	if res.Partial {
		t.Errorf("install completo não deveria ser parcial: %+v", res.Report.MCPs)
	}
}

func TestRunFullMaterializesAgentsSkillsAndLayerManifest(t *testing.T) {
	noEngineCLI(t)
	home := newHome(t)
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	res, err := Run(Options{
		EngineID: "claude-code", Profile: "base", Full: true,
		MCPs: []string{"golmet"}, Home: home,
	})
	if err != nil {
		t.Fatalf("Run FULL: %v", err)
	}
	if res.Report.FullTotal != 125 || len(res.Report.FullWritten) != 125 {
		t.Fatalf("FULL: total=%d written=%d conflicts=%d", res.Report.FullTotal, len(res.Report.FullWritten), len(res.Report.FullConflicts))
	}
	for _, rel := range []string{
		"agents/gm-dev.md",
		"agents/gm-dev-backend.md",
		"skills/gm-reverse/SKILL.md",
		".golmet/manifest.json",
	} {
		if _, err := os.Stat(filepath.Join(home, ".claude", rel)); err != nil {
			t.Errorf("FULL ausente %s: %v", rel, err)
		}
	}
}

func TestRunSkillsPreserveModifiedWithoutForce(t *testing.T) {
	noEngineCLI(t)
	home := newHome(t)
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	req := Options{EngineID: "claude-code", MCPs: []string{"golmet"}, Home: home}
	if _, err := Run(req); err != nil {
		t.Fatalf("Run 1: %v", err)
	}
	target := filepath.Join(home, ".claude", "skills", "gm-charter", "SKILL.md")
	os.WriteFile(target, []byte("---\nname: gm-charter\n---\n# adaptação local\n"), 0o644)

	res, err := Run(req) // sem --force
	if err != nil {
		t.Fatalf("Run 2: %v", err)
	}
	if len(res.Report.SkillsPreserved) != 1 || res.Report.SkillsPreserved[0] != "gm-charter" {
		t.Errorf("preserved = %+v", res.Report.SkillsPreserved)
	}
	if got := readFileT(t, target); !strings.Contains(got, "adaptação local") {
		t.Errorf("modificação local foi perdida")
	}
	res3, err := Run(Options{EngineID: "claude-code", MCPs: []string{"golmet"}, Home: home, Force: true})
	if err != nil {
		t.Fatalf("Run 3 (force): %v", err)
	}
	if len(res3.Report.SkillsPreserved) != 0 {
		t.Errorf("--force deveria sobrescrever todas: %+v", res3.Report.SkillsPreserved)
	}
}

func TestRunEnvMissingPendingNotRegistered(t *testing.T) {
	noEngineCLI(t)
	t.Setenv("FIGMA_API_KEY", "")
	home := newHome(t)
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	res, err := Run(Options{EngineID: "claude-code", MCPs: []string{"golmet", "figma"}, Home: home})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !res.Partial {
		t.Errorf("env ausente deveria marcar Partial")
	}
	for _, m := range res.Report.MCPs {
		t.Logf("mcp %s -> %s: %s", m.Name, m.Status, m.Detail)
	}
	if len(res.Report.Pending) != 1 || res.Report.Pending[0].Name != "figma" {
		t.Fatalf("pending_env = %+v", res.Report.Pending)
	}
	if !strings.Contains(res.Report.Pending[0].Fix, "FIGMA_API_KEY") {
		t.Errorf("fix sem a env: %+v", res.Report.Pending[0])
	}
	// config quebrada nunca: figma NÃO está no config do engine
	cfgFile := filepath.Join(home, ".claude.json")
	cfg := catJSON(t, cfgFile)
	mcpServers, _ := cfg["mcpServers"].(map[string]any)
	if mcpServers == nil {
		t.Fatalf("mcpServers ausente do config: %+v", cfg)
	}
	if _, ok := mcpServers["figma"]; ok {
		t.Errorf("figma registrada sem env (config quebrada): %+v", cfg)
	}
	if _, ok := mcpServers["golmet"]; !ok {
		t.Errorf("golmet (obrigatória) deveria estar registrada")
	}
}

func TestRunEnvLiteralWrittenNotLogged(t *testing.T) {
	noEngineCLI(t)
	t.Setenv("FIGMA_API_KEY", "segredo-super-fake-123")
	home := newHome(t)
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	res, err := Run(Options{EngineID: "claude-code", MCPs: []string{"golmet", "figma"}, Home: home})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// o VALOR literal vai para o config local (decisão v0.1)
	got := entryOf(t, filepath.Join(home, ".claude.json"), "mcpServers", "figma")
	envMap, _ := got["env"].(map[string]any)
	if envMap["FIGMA_API_KEY"] != "segredo-super-fake-123" {
		t.Errorf("env literal ausente do config: %+v", got)
	}
	// e NUNCA aparece no relatório (json/human)
	repJSON, _ := json.Marshal(res.Report)
	if strings.Contains(string(repJSON), "segredo-super-fake-123") {
		t.Errorf("valor de env vazou para o relatório")
	}
}

func TestCodegraphAutoInstallViaInjectedFn(t *testing.T) {
	noEngineCLI(t)
	home := newHome(t)
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	installed := false
	res, err := Run(Options{
		EngineID: "claude-code",
		MCPs:     []string{"golmet", "codegraph"},
		Home:     home,
		InstallCodegraphFn: func(ctx context.Context) error {
			installed = true
			// simula o download do bloco 3 criando o binário no destino
			dir := filepath.Join(home, ".golmet", "tools", "codegraph")
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
			bin := filepath.Join(dir, "codebase-memory-mcp.exe")
			if runtimeGOOS() != "windows" {
				bin = filepath.Join(dir, "codebase-memory-mcp")
			}
			return os.WriteFile(bin, []byte("fake"), 0o755)
		},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, m := range res.Report.MCPs {
		t.Logf("mcp %s -> %s: %s", m.Name, m.Status, m.Detail)
	}
	if !installed {
		t.Errorf("codegraph ausente deveria disparar o download injetado")
	}
	got := entryOf(t, filepath.Join(home, ".claude.json"), "mcpServers", "codegraph")
	if !strings.Contains(got["command"].(string), "codebase-memory-mcp") {
		t.Errorf("command do codegraph = %v", got["command"])
	}
}

// ---- validações de uso ----

func TestRunProfileAndEngineValidation(t *testing.T) {
	if _, err := Run(Options{EngineID: "warp", Profile: "enterprise", Home: newHome(t)}); err == nil ||
		!strings.Contains(err.Error(), "essencial") || !strings.Contains(err.Error(), "completo") {
		t.Errorf("perfil desconhecido deveria falhar listando os válidos, veio %v", err)
	}
	// os 4 valores aceitos (canônicos + aliases) passam a normalização —
	// a validação seguinte (engine inexistente) prova que o perfil passou
	for _, p := range []string{"", "essencial", "completo", "base", "full"} {
		if _, err := Run(Options{EngineID: "warp", Profile: p, Home: newHome(t)}); err == nil ||
			!strings.Contains(err.Error(), "não existe") {
			t.Errorf("perfil %q deveria ser aceito (falha só no engine), veio %v", p, err)
		}
	}
	if _, err := Run(Options{EngineID: "claude-code", MCPs: []string{"inexistente"}, Home: newHome(t)}); err == nil ||
		!strings.Contains(err.Error(), "não existe no catálogo") {
		t.Errorf("mcp inexistente deveria falhar citando o catálogo, veio %v", err)
	}
}
