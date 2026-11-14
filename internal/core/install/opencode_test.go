package install

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// O opencode recebe a CLI com placeholders JÁ resolvidos (@self → exe
// corrente; @codegraph → binário do bloco 3) e o registro só é marcado
// como registered quando `opencode mcp list` lista o nome (exit 0 da CLI
// não garante o bloco — regressão observada em e2e real). A resolução
// acontece no Run/resolveDef — este teste exercita o fluxo completo.
func TestOpenCodePlaceholdersResolvedAndPostVerify(t *testing.T) {
	lookPathFn = func(string) (string, error) { return "opencode-fake", nil }
	t.Cleanup(func() { lookPathFn = exec.LookPath })
	home := newHome(t)
	os.MkdirAll(filepath.Join(home, ".config", "opencode"), 0o755)

	selfExe := selfExeForTest()
	codegraphBin := filepath.Join(home, ".golmet", "tools", "codegraph", "codebase-memory-mcp.exe")
	os.MkdirAll(filepath.Dir(codegraphBin), 0o755)
	os.WriteFile(codegraphBin, []byte("fake"), 0o755)

	var commands [][]string
	mock := newOpencodeMock()
	oldRun := runEngineCmd
	runEngineCmd = func(name string, args []string) (string, error) {
		commands = append(commands, args)
		out, rerr := mock.run(name, args)
		// o `mcp list` do mock stateful lista o que foi adicionado ✓
		return out, rerr
	}
	t.Cleanup(func() { runEngineCmd = oldRun })

	res, err := Run(Options{
		EngineID:  "opencode",
		Profile:   "base",
		MCPs:      []string{"golmet", "codegraph", "playwright"},
		Workspace: home,
		Home:      home,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Partial {
		t.Errorf("install não deveria ser parcial: %+v", res.Report.MCPs)
	}
	// nenhum placeholder cruza a fronteira da CLI
	for _, c := range commands {
		for _, a := range c {
			if a == "@self" || a == "@codegraph" {
				t.Errorf("placeholder %q vazou para a CLI do engine: %v", a, c)
			}
		}
	}
	// add do golmet recebeu o exe corrente
	foundSelf, foundCodegraph, foundPlaywright := false, false, false
	for _, c := range commands {
		j := strings.Join(c, " ")
		if strings.Contains(j, "-- "+selfExe+" mcp") {
			foundSelf = true
		}
		if strings.Contains(j, "-- "+codegraphBin) {
			foundCodegraph = true
		}
		if strings.Contains(j, "npx -y @playwright/mcp") {
			foundPlaywright = true
		}
	}
	if !foundSelf {
		t.Errorf("add do golmet sem o exe resolvido: %v", commands)
	}
	if !foundCodegraph {
		t.Errorf("add do codegraph sem o binário resolvido: %v", commands)
	}
	if !foundPlaywright {
		t.Errorf("add do playwright sem os args do catálogo: %v", commands)
	}
	// registered com verificação pós-add (sem aviso — list disponível)
	for _, m := range res.Report.MCPs {
		if strings.Contains(m.Detail, "confira com") {
			t.Errorf("list disponível não deveria degradar o detail: %s", m.Detail)
		}
	}
}

// CLI que falha no list (indisponível) → registered com aviso declarado.
func TestOpenCodeListUnavailableDegradesDeclared(t *testing.T) {
	lookPathFn = func(string) (string, error) { return "opencode-fake", nil }
	t.Cleanup(func() { lookPathFn = exec.LookPath })
	home := newHome(t)
	os.MkdirAll(filepath.Join(home, ".config", "opencode"), 0o755)
	oldRun := runEngineCmd
	runEngineCmd = func(name string, args []string) (string, error) {
		if strings.Contains(strings.Join(args, " "), "list") {
			return "", context.DeadlineExceeded
		}
		return "", nil
	}
	t.Cleanup(func() { runEngineCmd = oldRun })

	e := engineOpenCode{}
	def, _ := Lookup("playwright")
	o, detail, err := e.RegisterMCP(home, nil, def, false)
	if err != nil || o != OutcomeRegistered {
		t.Fatalf("registered com aviso = %s (%v)", o, err)
	}
	if !strings.Contains(detail, "não foi possível confirmar via 'opencode mcp list'") {
		t.Errorf("aviso declarado ausente: %s", detail)
	}
}

// CLI que sai 0 mas não lista o MCP → OutcomeError (falso-positivo proibido).
func TestOpenCodePostVerifyCatchSilentFailure(t *testing.T) {
	lookPathFn = func(string) (string, error) { return "opencode-fake", nil }
	t.Cleanup(func() { lookPathFn = exec.LookPath })
	home := newHome(t)
	os.MkdirAll(filepath.Join(home, ".config", "opencode"), 0o755)
	oldRun := runEngineCmd
	runEngineCmd = func(name string, args []string) (string, error) {
		return "nenhum mcp registrado", nil // add sai 0, list não mostra
	}
	t.Cleanup(func() { runEngineCmd = oldRun })

	e := engineOpenCode{}
	def, _ := Lookup("playwright")
	o, detail, err := e.RegisterMCP(home, nil, def, false)
	if o != OutcomeError || err == nil || !strings.Contains(err.Error(), "não aparece em 'opencode mcp list'") {
		t.Errorf("falso-positivo deveria virar error: %s (%v) (%s)", o, err, detail)
	}
}
