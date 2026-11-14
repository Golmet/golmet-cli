package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/Golmet/golmet-cli/internal/core/install"
)

// spinner em intervalo rápido para o teste; restaura no fim.
func fastSpinner(t *testing.T) {
	t.Helper()
	old := spinnerInterval
	spinnerInterval = 1 * time.Millisecond
	t.Cleanup(func() { spinnerInterval = old })
}

// spinner: frames braille numa única linha (\r + \x1b[K) e finish espera
// a goroutine antes da linha final — sem write concorrente.
func TestSpinnerFinishWritesFinal(t *testing.T) {
	fastSpinner(t)
	var buf bytes.Buffer
	s := startSpinner(&buf, "golmet (mcp add)")
	time.Sleep(30 * time.Millisecond) // deixa girar alguns frames
	s.finish("  ✔ golmet — registrado")

	out := buf.String()
	if !strings.Contains(out, "golmet (mcp add)") {
		t.Errorf("label do passo ausente do output")
	}
	if strings.HasSuffix(out, "\x1b[K") {
		t.Errorf("output termina sem a linha final: %q", out[len(out)-40:])
	}
	if !strings.Contains(out, "✔ golmet — registrado") {
		t.Errorf("linha final ausente: %q", out)
	}
	// a linha final começa com \r (sobrescreve o frame corrente)
	if !strings.Contains(out, "\r  ✔ golmet — registrado\x1b[K\n") {
		t.Errorf("linha final malformada (sem \\r + erase): %q", out[len(out)-60:])
	}
}

// fallback NÃO-TTY: as mesmas linhas de progresso em texto plano — sem
// ANSI, sem spinner, sem full-clear.
func TestInstallProgressPlainNoANSI(t *testing.T) {
	var buf bytes.Buffer
	p := &installProgress{out: &buf, tty: false}
	p.consume(install.StepEvent{Kind: "start", Step: "mcp:golmet", Label: "golmet (mcp add)"})
	p.consume(install.StepEvent{Kind: "done", Step: "mcp:golmet", Detail: "golmet — registrado", Outcome: "ok"})
	p.consume(install.StepEvent{Kind: "start", Step: "doctor", Label: "doctor — verificando workspace..."})
	p.consume(install.StepEvent{Kind: "done", Step: "doctor", Detail: "doctor — 8 ok, 1 aviso(s)", Outcome: "warn"})

	out := buf.String()
	if strings.Contains(out, "\x1b") {
		t.Errorf("ANSI vazou no modo plano: %q", out)
	}
	for _, f := range spinnerFrames {
		if strings.Contains(out, f) {
			t.Errorf("frame braille %q vazou no modo plano", f)
		}
	}
	for _, want := range []string{"  … golmet (mcp add)\n", "  ✔ golmet — registrado\n", "  ⚠ doctor — 8 ok, 1 aviso(s)\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("linha plana %q ausente: %q", want, out)
		}
	}
}

// TTY: pares start→done geram as linhas finais (✔/⚠) construídas
// progressivamente — sem linha plana de start.
func TestInstallProgressTTYPairs(t *testing.T) {
	var buf bytes.Buffer
	p := &installProgress{out: &buf, tty: true} // colors zerados = sem ANSI
	p.consume(install.StepEvent{Kind: "start", Step: "skills", Label: "skills gm-* → skills"})
	p.consume(install.StepEvent{Kind: "done", Step: "skills", Detail: "skills gm-* — 16 escrita(s)", Outcome: "ok"})
	p.consume(install.StepEvent{Kind: "start", Step: "mcp:supabase", Label: "supabase (mcp add)"})
	p.consume(install.StepEvent{Kind: "done", Step: "mcp:supabase", Detail: "supabase — registrado · ⚠ autentique no engine", Outcome: "ok"})

	out := buf.String()
	if strings.Contains(out, "  … ") {
		t.Errorf("linha plana de start vazou no modo TTY: %q", out)
	}
	for _, want := range []string{"  ✔ skills gm-* — 16 escrita(s)", "  ✔ supabase — registrado · ⚠ autentique no engine"} {
		if !strings.Contains(out, want) {
			t.Errorf("linha final %q ausente: %q", want, out)
		}
	}
}

// abort: spinner ativo é fechado com linha de interrupção (o erro do Run
// não disputa a linha com o spinner).
func TestInstallProgressAbortClosesSpinner(t *testing.T) {
	fastSpinner(t)
	var buf bytes.Buffer
	p := &installProgress{out: &buf, tty: true}
	p.consume(install.StepEvent{Kind: "start", Step: "codegraph-download", Label: "codegraph — baixando..."})
	time.Sleep(20 * time.Millisecond)
	p.abort()
	out := buf.String()
	if !strings.Contains(out, "interrompido") {
		t.Errorf("linha de interrupção ausente: %q", out[len(out)-60:])
	}
	// p.abort de novo é no-op (sem panic de double-close)
	p.abort()
}

// truncamento por runes (acentos e emoji não viram bytes quebrados).
func TestTruncateLineRunesSafe(t *testing.T) {
	s := "codegraph — baixando binário verificado (pode levar um minuto) ✔"
	got := truncateLine(s, 20)
	if len([]rune(got)) != 20 || !strings.HasSuffix(got, "…") {
		t.Errorf("truncateLine = %q (%d runes), want 20 runes terminando em …", got, len([]rune(got)))
	}
	if truncateLine("curto", 20) != "curto" {
		t.Errorf("linha curta não deveria ser truncada")
	}
}

// LABEL do spinner também trunca na largura do terminal (o resultado já
// truncava — em janelas estreitas o label largo wrapa e suja a linha até o
// finish; risco documentado do bloco loading).
func TestSpinnerLabelTruncated(t *testing.T) {
	old := termWidthFn
	termWidthFn = func() int { return 40 }
	t.Cleanup(func() { termWidthFn = old })

	fastSpinner(t)
	var buf bytes.Buffer
	long := "codegraph — baixando binário verificado (pode levar um minuto) com sufixo bem comprido mesmo"
	s := startSpinner(&buf, long)
	time.Sleep(20 * time.Millisecond)
	s.finish("  ✔ pronto")

	out := buf.String()
	for _, frameLine := range strings.Split(out, "\r") {
		line := strings.TrimSuffix(frameLine, "\x1b[K\n")
		if !strings.Contains(line, "codegraph") || strings.Contains(line, long) {
			continue
		}
		if len([]rune(line)) > 44 { // 40 cols - 8 de margem + arte do frame
			t.Errorf("label do spinner wrapando (linha longa demais): %d runes", len([]rune(line)))
		}
	}
	if !strings.Contains(out, "…") {
		t.Errorf("label truncado deveria terminar com …")
	}
}
