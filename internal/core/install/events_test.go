package install

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Golmet/golmet-cli/internal/core/doctor"
	"github.com/Golmet/golmet-cli/internal/core/tools"
)

// captureProgress registra os eventos do Run para asserts de ordem/paridade.
type captureProgress struct {
	events []StepEvent
}

func (c *captureProgress) fn(ev StepEvent) { c.events = append(c.events, ev) }

// assertBalanced valida a invariante do modelo: cada done tem um start
// anterior com o mesmo Step, sem overlap (spinner único por vez).
func assertBalanced(t *testing.T, events []StepEvent) {
	t.Helper()
	open := map[string]bool{}
	for i, ev := range events {
		switch ev.Kind {
		case "start":
			if open[ev.Step] {
				t.Errorf("evento %d: start de %q com passo já aberto (overlap)", i, ev.Step)
			}
			open[ev.Step] = true
			if ev.Label == "" {
				t.Errorf("evento %d: start de %q sem label", i, ev.Step)
			}
			if ev.Outcome != "" {
				t.Errorf("evento %d: start de %q com outcome %q", i, ev.Step, ev.Outcome)
			}
		case "done":
			if !open[ev.Step] {
				t.Errorf("evento %d: done de %q sem start anterior", i, ev.Step)
			}
			delete(open, ev.Step)
			if ev.Outcome != "ok" && ev.Outcome != "warn" && ev.Outcome != "fail" {
				t.Errorf("evento %d: done de %q com outcome inválido %q", i, ev.Step, ev.Outcome)
			}
			if ev.Detail == "" {
				t.Errorf("evento %d: done de %q sem detail", i, ev.Step)
			}
		default:
			t.Errorf("evento %d: kind inválido %q", i, ev.Kind)
		}
	}
	for step := range open {
		t.Errorf("passo %q aberto sem done", step)
	}
}

// Run com engine de patch (claude-code — sem CLI externa) emite os passos
// em ordem: skills → mcp:golmet → doctor, todos com pares balanceados.
func TestRunEmitsStepsInOrder(t *testing.T) {
	noEngineCLI(t)
	home := newHome(t)
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	cap := &captureProgress{}
	_, err := Run(Options{
		EngineID:  "claude-code",
		MCPs:      []string{"golmet"},
		Home:      home,
		Workspace: "ws-fake", // doctor emite os eventos de fechamento
		DoctorFn: func(ws string) (*doctor.Report, error) {
			return &doctor.Report{OKs: 8, Warns: 1}, nil
		},
		Progress: cap.fn,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	assertBalanced(t, cap.events)

	var steps []string
	for _, ev := range cap.events {
		steps = append(steps, ev.Step)
	}
	want := []string{"skills", "skills", "mcp:golmet", "mcp:golmet", "doctor", "doctor"}
	if strings.Join(steps, ",") != strings.Join(want, ",") {
		t.Errorf("ordem dos passos = %v, want %v", steps, want)
	}
	// skills escreve 16 → done ok; doctor 8 ok 1 aviso → done warn
	if got := cap.events[1].Detail; !strings.Contains(got, "16 escrita(s)") {
		t.Errorf("detail do skills = %q", got)
	}
	if cap.events[1].Outcome != "ok" {
		t.Errorf("outcome do skills = %q, want ok", cap.events[1].Outcome)
	}
	if got := cap.events[5].Detail; !strings.Contains(got, "8 ok") || !strings.Contains(got, "1 aviso") {
		t.Errorf("detail do doctor = %q", got)
	}
	if cap.events[5].Outcome != "warn" {
		t.Errorf("outcome do doctor = %q, want warn (1 aviso)", cap.events[5].Outcome)
	}
}

// download do codegraph (binário ausente + InstallCodegraphFn fake) emite
// o passo codegraph-download ANTES do passo mcp:codegraph.
func TestRunCodegraphDownloadEvents(t *testing.T) {
	noEngineCLI(t)
	home := newHome(t)
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	cap := &captureProgress{}
	_, err := Run(Options{
		EngineID: "claude-code",
		MCPs:     []string{"golmet", "codegraph"},
		Home:     home,
		InstallCodegraphFn: func(ctx context.Context) error {
			dir := filepath.Join(home, ".golmet", "tools", "codegraph")
			os.MkdirAll(dir, 0o755)
			bin := filepath.Join(dir, "codebase-memory-mcp.exe")
			if runtimeGOOS() != "windows" {
				bin = filepath.Join(dir, "codebase-memory-mcp")
			}
			return os.WriteFile(bin, []byte("fake"), 0o755)
		},
		Progress: cap.fn,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	assertBalanced(t, cap.events)

	var codegraph, mcpCodegraph []int
	for i, ev := range cap.events {
		if ev.Step == "codegraph-download" {
			codegraph = append(codegraph, i)
			if ev.Kind == "done" && ev.Outcome != "ok" {
				t.Errorf("download ok deveria ser outcome ok, veio %q", ev.Outcome)
			}
			if ev.Kind == "done" && !strings.Contains(ev.Detail, "SHA-256") {
				t.Errorf("detail do download = %q (sem verificação citada)", ev.Detail)
			}
		}
		if ev.Step == "mcp:codegraph" {
			mcpCodegraph = append(mcpCodegraph, i)
		}
	}
	if len(codegraph) != 2 {
		t.Fatalf("codegraph-download = %d eventos, want 2 (start+done)", len(codegraph))
	}
	if len(mcpCodegraph) != 2 || mcpCodegraph[0] < codegraph[1] {
		t.Errorf("mcp:codegraph deveria vir depois do download (mcp=%v, codegraph=%v)", mcpCodegraph, codegraph)
	}
}

// download que falha emite done fail (a linha de passo reporta o erro).
func TestRunCodegraphDownloadFailEvent(t *testing.T) {
	noEngineCLI(t)
	home := newHome(t)
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	cap := &captureProgress{}
	_, err := Run(Options{
		EngineID: "claude-code",
		MCPs:     []string{"golmet", "codegraph"},
		Home:     home,
		InstallCodegraphFn: func(ctx context.Context) error {
			return context.DeadlineExceeded
		},
		Progress: cap.fn,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	assertBalanced(t, cap.events)
	var doneFail bool
	for _, ev := range cap.events {
		if ev.Step == "codegraph-download" && ev.Kind == "done" {
			doneFail = ev.Outcome == "fail" && strings.Contains(ev.Detail, "falhou")
		}
	}
	if !doneFail {
		t.Errorf("download falho deveria emitir done fail com o motivo: %+v", cap.events)
	}
}

// (o caso "pendência SEM execução" morreu com o auto-install do sqlite:
// hoje o sqlite ausente SEMPRE tenta o download com eventos
// sqlite-download start/done — o falho é coberto por
// TestSqliteAutoInstallFailEvent; o param ausente é erro de parse,
// TestRunProfileAndEngineValidation/ParseMCPSelection)

// health note inline: MCP registrado com nota de autenticação vira linha
// "nome — registrado · ⚠ autentique no engine" com outcome ok (o install
// registra; a saúde é do doctor).
func TestMCPStepDetailHealthNoteInline(t *testing.T) {
	detail, outcome := mcpStepDetail(MCPResult{
		Name:   "supabase",
		Status: OutcomeRegistered,
		Detail: "registrado via CLI do engine — registrado — autentique no engine (status: needs authentication)",
	})
	if outcome != "ok" {
		t.Errorf("outcome = %q, want ok", outcome)
	}
	if !strings.Contains(detail, "registrado · ⚠ autentique no engine") {
		t.Errorf("detail = %q, want contendo 'registrado · ⚠ autentique no engine'", detail)
	}
	// registered limpo → resumo curto sem nota
	detail2, _ := mcpStepDetail(MCPResult{Name: "golmet", Status: OutcomeRegistered, Detail: "registrado via CLI do engine (status: connected)"})
	if detail2 != "golmet — registrado" {
		t.Errorf("detail limpo = %q", detail2)
	}
	// erro → fail
	detail3, outcome3 := mcpStepDetail(MCPResult{Name: "x", Status: OutcomeError})
	if outcome3 != "fail" || !strings.Contains(detail3, "falhou") {
		t.Errorf("erro = %q %q", detail3, outcome3)
	}
}

// auto-install do sqlite: binário ausente + InstallSqliteFn fake (cria o
// binário) → eventos sqlite-download ok ANTES do mcp:sqlite registered.
func TestSqliteAutoInstallViaInjectedFn(t *testing.T) {
	noEngineCLI(t)
	home := newHome(t)
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	cap := &captureProgress{}
	_, err := Run(Options{
		EngineID:     "claude-code",
		MCPs:         []string{"golmet", "sqlite"},
		Home:         home,
		LocalAppData: filepath.Join(home, "lad"),
		InstallSqliteFn: func(ctx context.Context) error {
			bin := filepath.Join(home, "lad", "Programs", "mcp-sqlite", tools.SqliteBinaryName())
			os.MkdirAll(filepath.Dir(bin), 0o755)
			return os.WriteFile(bin, []byte("fake"), 0o755)
		},
		Progress: cap.fn,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	assertBalanced(t, cap.events)
	var downloadOK, hasMCPEvent bool
	for i, ev := range cap.events {
		if ev.Step == "sqlite-download" && ev.Kind == "done" {
			downloadOK = ev.Outcome == "ok" && strings.Contains(ev.Detail, "SHA-256")
		}
		if ev.Step == "mcp:sqlite" && !hasMCPEvent {
			hasMCPEvent = true
			if i < 1 || cap.events[i-1].Step != "sqlite-download" {
				t.Errorf("o PRIMEIRO evento mcp:sqlite deveria vir depois do sqlite-download")
			}
		}
	}
	if !downloadOK || !hasMCPEvent {
		t.Errorf("eventos do sqlite ausentes/errados (downloadOK=%v, mcp=%v)", downloadOK, hasMCPEvent)
	}
}

// download do sqlite falha → done fail + pendência declarada (config
// quebrada nunca).
func TestSqliteAutoInstallFailEvent(t *testing.T) {
	noEngineCLI(t)
	home := newHome(t)
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	cap := &captureProgress{}
	res, err := Run(Options{
		EngineID:     "claude-code",
		MCPs:         []string{"golmet", "sqlite"},
		Home:         home,
		LocalAppData: filepath.Join(home, "lad"),
		InstallSqliteFn: func(ctx context.Context) error {
			return context.DeadlineExceeded
		},
		Progress: cap.fn,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	assertBalanced(t, cap.events)
	var failSeen, pending bool
	for _, ev := range cap.events {
		if ev.Step == "sqlite-download" && ev.Kind == "done" && ev.Outcome == "fail" {
			failSeen = true
		}
	}
	for _, p := range res.Report.Pending {
		if p.Name == "sqlite" {
			pending = true
		}
	}
	if !failSeen || !pending {
		t.Errorf("download falho deveria emitir done fail + pendência (fail=%v, pending=%v)", failSeen, pending)
	}
}
