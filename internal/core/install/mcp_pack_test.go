// mcp_pack_test.go — server MCP de state packs: status read-only (cascata
// do engine, node do PATH), sync idempotente pelo pipeline do adapter
// (opencode: pré-check → `mcp add` → pós-verify) e elegibilidade. Seeds
// com os.WriteFile + newHome(t); runEngineCmd/lookPathFn injetados — os
// testes NUNCA executam os CLIs reais dos engines nem o node.
package install

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedStatePack cria um working copy de state pack no root do opencode
// (pack.yaml kind: state + mcp/server.js opcional). ListState lê apenas o
// pack.yaml — sem git real.
func seedStatePack(t *testing.T, home, id string, withServer bool) string {
	t.Helper()
	dir := filepath.Join(home, ".config", "opencode", "packs", id)
	if err := os.MkdirAll(filepath.Join(dir, "mcp"), 0o755); err != nil {
		t.Fatal(err)
	}
	spec := "id: " + id + "\nversion: 1.0.0\nname: Pack " + id + "\nkind: state\n"
	if err := os.WriteFile(filepath.Join(dir, "pack.yaml"), []byte(spec), 0o644); err != nil {
		t.Fatal(err)
	}
	if withServer {
		if err := os.WriteFile(filepath.Join(dir, "mcp", "server.js"), []byte("// server do pack"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// injectEngineCLI injeta a CLI falsa dos engines: `opencode mcp list`
// devolve o snapshot informado; `opencode mcp add ...` registra no
// snapshot (comportamento do pipeline real) EXCETO para nomes em
// addFailFor (falha determinística — caminho de erro do pipeline);
// `node --version` devolve a versão (ou nodeErr). Qualquer outra chamada
// FALHA o teste (nada de binário real).
type fakeEngineCLI struct {
	list       string // snapshot do `opencode mcp list` ("" = vazio)
	adds       [][]string
	addFailFor map[string]bool // nome do server cujo `mcp add` falha
	nodeVer    string
	nodeErr    error
	t          *testing.T
}

func (f *fakeEngineCLI) run(name string, args []string) (string, error) {
	switch {
	case name == "opencode" && len(args) >= 2 && args[0] == "mcp" && args[1] == "list":
		return f.list, nil
	case name == "opencode" && len(args) >= 2 && args[0] == "mcp" && args[1] == "add":
		if f.addFailFor[args[2]] {
			return "", fmt.Errorf("opencode mcp add falhou (fake): %s", args[2])
		}
		f.adds = append(f.adds, args)
		f.list += "✓ " + args[2] + "  connected\n"
		return "", nil
	case name == "node" && len(args) == 1 && args[0] == "--version":
		return f.nodeVer, f.nodeErr
	}
	f.t.Fatalf("comando inesperado nos testes: %s %v", name, args)
	return "", fmt.Errorf("inesperado: %s %v", name, args)
}

// inject simula os binários no PATH: opencode sempre presente (o pipeline
// de registro exige a CLI do engine), node conforme nodePathOK.
func (f *fakeEngineCLI) inject(t *testing.T, nodePathOK bool) {
	t.Helper()
	oldRun := runEngineCmd
	runEngineCmd = f.run
	t.Cleanup(func() { runEngineCmd = oldRun })
	oldLook := lookPathFn
	lookPathFn = func(name string) (string, error) {
		if name == "opencode" {
			return filepath.Join("fake", "opencode.exe"), nil
		}
		if name == "node" && nodePathOK {
			return filepath.Join("fake", "node.exe"), nil
		}
		return "", os.ErrNotExist
	}
	t.Cleanup(func() { lookPathFn = oldLook })
}

// ---- status ----

// pack elegível registrado + node ok → estado ok com status real da
// sondagem do engine (cascata: list vence).
func TestPackMCPStatusEligibleRegistered(t *testing.T) {
	home := newHome(t)
	dir := seedStatePack(t, home, "techminds", true)
	fake := &fakeEngineCLI{list: "✓ pack-techminds  connected\n", nodeVer: "v22.14.0\n", t: t}
	fake.inject(t, true)

	statuses, err := PackMCPStatuses(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 1 {
		t.Fatalf("1 status esperado, veio %d: %+v", len(statuses), statuses)
	}
	s := statuses[0]
	if s.Pack != "techminds" || s.Engine != "opencode" {
		t.Errorf("pack/engine = %s/%s", s.Pack, s.Engine)
	}
	if s.ServerName != "pack-techminds" {
		t.Errorf("nome do server = %s", s.ServerName)
	}
	if s.ServerPath != filepath.Join(dir, "mcp", "server.js") {
		t.Errorf("caminho do server = %s", s.ServerPath)
	}
	if !s.Registered || s.RegStatus != "connected" {
		t.Errorf("registro = %v (%q)", s.Registered, s.RegStatus)
	}
	if !s.NodePresent || !s.NodeRecent || s.NodeVersion != "v22.14.0" {
		t.Errorf("node = %v/%v/%q", s.NodePresent, s.NodeRecent, s.NodeVersion)
	}
	if s.State != PackMCPStateOK {
		t.Errorf("estado = %s", s.State)
	}
}

// pack elegível NÃO registrado (list sem o nome, config sem o bloco) →
// estado nao-registrado com a remediação do sync.
func TestPackMCPStatusEligibleNotRegistered(t *testing.T) {
	home := newHome(t)
	seedStatePack(t, home, "techminds", true)
	fake := &fakeEngineCLI{list: "✓ outro-server  connected\n", nodeVer: "v20.11.1", t: t}
	fake.inject(t, true)

	statuses, err := PackMCPStatuses(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 1 || statuses[0].Registered {
		t.Fatalf("status = %+v", statuses)
	}
	if statuses[0].State != PackMCPStateNaoRegistro {
		t.Errorf("estado = %s", statuses[0].State)
	}
}

// state pack SEM mcp/server.js aparece com estado server-ausente e NÃO
// sonda engine/node; pack overlay no cache (~/.golmet/packs) é ignorado.
func TestPackMCPStatusServerAusenteIgnoraOverlay(t *testing.T) {
	home := newHome(t)
	seedStatePack(t, home, "semserver", false)
	// overlay no cache — nunca aparece na varredura de state packs
	overlay := filepath.Join(home, ".golmet", "packs", "overlay-x")
	os.MkdirAll(overlay, 0o755)
	os.WriteFile(filepath.Join(overlay, "pack.yaml"),
		[]byte("id: overlay-x\nversion: 1.0.0\nname: Overlay\nprovides:\n  - skills/gm-x/SKILL.md\n"), 0o644)
	fake := &fakeEngineCLI{t: t}
	fake.inject(t, true) // qualquer sondagem aqui falha o teste

	statuses, err := PackMCPStatuses(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 1 {
		t.Fatalf("1 status esperado (sem mcp/server.js), veio %d: %+v", len(statuses), statuses)
	}
	s := statuses[0]
	if s.Pack != "semserver" || s.HasServer || s.State != PackMCPStateServerAusente {
		t.Errorf("status = %+v", s)
	}
	if s.Registered || s.NodePresent {
		t.Errorf("pack sem server não deveria sondar engine/node: %+v", s)
	}
}

// precedência do estado: registrado + node ausente → node-ausente; e node
// presente mas antigo → node-antigo (registro vem primeiro no estado —
// é o que o sync resolve; node fica nas colunas).
func TestPackMCPStatusNodeStates(t *testing.T) {
	seed := func() string {
		home := newHome(t)
		seedStatePack(t, home, "techminds", true)
		return home
	}
	// node ausente no PATH (registro OK via list) — estado: node-ausente
	home := seed()
	fake := &fakeEngineCLI{list: "✓ pack-techminds  connected\n", t: t}
	fake.inject(t, false)
	statuses, err := PackMCPStatuses(home)
	if err != nil {
		t.Fatal(err)
	}
	if statuses[0].State != PackMCPStateNodeAusente || !statuses[0].Registered {
		t.Errorf("node ausente: %+v", statuses[0])
	}
	if statuses[0].NodeVersion != "" {
		t.Errorf("versão sem node = %q", statuses[0].NodeVersion)
	}

	// node presente no PATH mas `node --version` falha → runtime indisponível
	// (present=false; o estado degrada para node-ausente — nunca engolido)
	home = seed()
	fake = &fakeEngineCLI{list: "✓ pack-techminds  connected\n", nodeErr: context.DeadlineExceeded, t: t}
	fake.inject(t, true)
	statuses, err = PackMCPStatuses(home)
	if err != nil {
		t.Fatal(err)
	}
	if statuses[0].NodePresent || statuses[0].NodeRecent || statuses[0].NodeVersion != "" {
		t.Errorf("node --version falhou deveria deixar o runtime indisponível: %+v", statuses[0])
	}
	if statuses[0].State != PackMCPStateNodeAusente {
		t.Errorf("estado = %s, want node-ausente", statuses[0].State)
	}

	// node antigo (v18 — abaixo do mínimo 20)
	home = seed()
	fake = &fakeEngineCLI{list: "✓ pack-techminds  connected\n", nodeVer: "v18.20.4\n", t: t}
	fake.inject(t, true)
	statuses, err = PackMCPStatuses(home)
	if err != nil {
		t.Fatal(err)
	}
	if statuses[0].State != PackMCPStateNodeAntigo || statuses[0].NodeRecent {
		t.Errorf("node antigo: %+v", statuses[0])
	}
}

// home sem state packs → lista vazia (sem erro).
func TestPackMCPStatusEmpty(t *testing.T) {
	fake := &fakeEngineCLI{t: t}
	fake.inject(t, true)
	statuses, err := PackMCPStatuses(newHome(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 0 {
		t.Errorf("esperado vazio, veio %+v", statuses)
	}
}

// ---- sync ----

// sync registra pelo pipeline (1 chamada `opencode mcp add pack-techminds
// --global -- node <server.js>`) e é IDEMPOTENTE: 2º sync → unchanged sem
// nova chamada de add.
func TestPackMCPSyncRegistersAndIsIdempotent(t *testing.T) {
	home := newHome(t)
	dir := seedStatePack(t, home, "techminds", true)
	fake := &fakeEngineCLI{nodeVer: "v22.0.0", t: t}
	fake.inject(t, true)

	res, err := PackMCPSync(home, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 {
		t.Fatalf("1 resultado esperado, veio %d: %+v", len(res), res)
	}
	if res[0].Status != OutcomeRegistered {
		t.Errorf("1º sync = %s (%s)", res[0].Status, res[0].Detail)
	}
	if len(fake.adds) != 1 {
		t.Fatalf("1 add esperado, veio %d: %v", len(fake.adds), fake.adds)
	}
	want := []string{"mcp", "add", "pack-techminds", "--global", "--", "node", filepath.Join(dir, "mcp", "server.js")}
	if strings.Join(fake.adds[0], "\x00") != strings.Join(want, "\x00") {
		t.Errorf("args do add = %v; queria %v", fake.adds[0], want)
	}

	// 2º sync: pré-check em cascata acha o nome na lista → unchanged
	res, err = PackMCPSync(home, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 || res[0].Status != OutcomeUnchanged {
		t.Errorf("2º sync deveria ser unchanged: %+v (adds=%d)", res, len(fake.adds))
	}
	if len(fake.adds) != 1 {
		t.Errorf("2º sync não deveria re-add (adds=%d)", len(fake.adds))
	}
}

// sync --force repassa o re-add ao pipeline (add executado mesmo já
// registrado; pós-verify confirma → registered).
func TestPackMCPSyncForceReAdds(t *testing.T) {
	home := newHome(t)
	seedStatePack(t, home, "techminds", true)
	fake := &fakeEngineCLI{list: "✓ pack-techminds  connected\n", nodeVer: "v22.0.0", t: t}
	fake.inject(t, true)

	res, err := PackMCPSync(home, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 || res[0].Status != OutcomeRegistered {
		t.Fatalf("sync --force = %+v", res)
	}
	if len(fake.adds) != 1 {
		t.Errorf("--force deveria executar o add mesmo registrado (adds=%d)", len(fake.adds))
	}
}

// sync com opencode ausente do PATH → outcome manual com instruções
// (o kit NUNCA faz patch no jsonc).
func TestPackMCPSyncManualWithoutEngineCLI(t *testing.T) {
	home := newHome(t)
	seedStatePack(t, home, "techminds", true)
	fake := &fakeEngineCLI{nodeVer: "v22.0.0", t: t}
	fake.inject(t, true)
	// sobrescreve a injeção: opencode AUSENTE do PATH (só o lookPath decide)
	oldLook := lookPathFn
	lookPathFn = func(name string) (string, error) {
		if name == "node" {
			return filepath.Join("fake", "node.exe"), nil
		}
		return "", os.ErrNotExist
	}
	t.Cleanup(func() { lookPathFn = oldLook })

	res, err := PackMCPSync(home, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 || res[0].Status != OutcomeManual {
		t.Fatalf("esperado manual: %+v", res)
	}
	if !strings.Contains(res[0].Detail, "rode manualmente") || !strings.Contains(res[0].Detail, "pack-techminds") {
		t.Errorf("instruções manuais = %q", res[0].Detail)
	}
}

// sync com resultado MISTO (1 registered + 1 error): o erro de UM pack não
// interrompe os demais — todos os elegíveis são processados e o agregado
// carrega os dois outcomes (o CLI sinaliza parcial → exit 1).
func TestPackMCPSyncMixedOutcomes(t *testing.T) {
	home := newHome(t)
	seedStatePack(t, home, "okpack", true)
	seedStatePack(t, home, "badpack", true)
	fake := &fakeEngineCLI{
		nodeVer:    "v22.0.0",
		addFailFor: map[string]bool{"pack-badpack": true},
		t:          t,
	}
	fake.inject(t, true)

	res, err := PackMCPSync(home, false)
	if err != nil {
		t.Fatal(err) // erro de UM pack não vira erro do aggregate
	}
	if len(res) != 2 {
		t.Fatalf("os 2 packs deveriam ser processados, veio %d: %+v", len(res), res)
	}
	byPack := map[string]string{}
	for _, r := range res {
		byPack[r.Pack] = r.Status
	}
	if byPack["okpack"] != OutcomeRegistered {
		t.Errorf("okpack = %s, want registered", byPack["okpack"])
	}
	if byPack["badpack"] != OutcomeError {
		t.Errorf("badpack = %s, want error", byPack["badpack"])
	}
}

// sync sem elegível (home vazio OU só packs sem server) → vazio, nil
// (erro suave — o CLI imprime a mensagem; não crasha).
func TestPackMCPSyncNoEligible(t *testing.T) {
	fake := &fakeEngineCLI{t: t}
	fake.inject(t, true)

	if res, err := PackMCPSync(newHome(t), false); err != nil || len(res) != 0 {
		t.Errorf("home vazio: %+v (%v)", res, err)
	}
	home := newHome(t)
	seedStatePack(t, home, "semserver", false)
	if res, err := PackMCPSync(home, false); err != nil || len(res) != 0 {
		t.Errorf("pack sem server: %+v (%v)", res, err)
	}
}

// ---- helpers ----

func TestParseNodeMajor(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"v22.14.0", 22}, {"22.14.0", 22}, {"v20", 20}, {"20.11.1\n", 20},
		{"v18.20.4", 18}, {"", 0}, {"garbage", 0}, {"v", 0},
	}
	for _, c := range cases {
		if got := parseNodeMajor(c.in); got != c.want {
			t.Errorf("parseNodeMajor(%q) = %d; queria %d", c.in, got, c.want)
		}
	}
}
