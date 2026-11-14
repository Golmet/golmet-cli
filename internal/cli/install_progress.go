// install_progress.go — execução progressiva estilo npm/cargo do
// `golmet install`: um spinner braille por passo numa ÚNICA linha em
// atualização (\r + \x1b[K apaga a linha — SEM full-clear; o raw mode é
// só para teclas do wizard, spinner é goroutine + ANSI puro). Enquanto um
// spinner está ativo ele é o DONO da linha: finish() espera a goroutine
// terminar antes de escrever o resultado — nenhum write concorrente.
// Não-TTY (pipe/CI) e NO_COLOR: sem spinner e sem ANSI — as MESMAS linhas
// de progresso em texto plano.
package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/Golmet/golmet-cli/internal/core/install"
)

// spinnerFrames — ciclo braille (~80-100ms por frame).
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// spinnerInterval injetável (testes); default 90ms.
var spinnerInterval = 90 * time.Millisecond

type spinner struct {
	out   io.Writer
	label string
	stop  chan struct{}
	done  chan struct{}
}

// startSpinner cria o spinner com o label truncado na largura do terminal
// (o resultado usa truncateLine; o label TAMBÉM — em janelas estreitas um
// label largo wrapa e suja a linha até o finish).
func startSpinner(out io.Writer, label string) *spinner {
	s := &spinner{
		out:   out,
		label: truncateLine(label, terminalWidth()-8),
		stop:  make(chan struct{}),
		done:  make(chan struct{}),
	}
	go s.loop()
	return s
}

func (s *spinner) loop() {
	defer close(s.done)
	for i := 0; ; i = (i + 1) % len(spinnerFrames) {
		select {
		case <-s.stop:
			return
		default:
		}
		fmt.Fprintf(s.out, "\r  %s %s\x1b[K", spinnerFrames[i], s.label)
		time.Sleep(spinnerInterval)
	}
}

// finish para a goroutine, ESPERA o frame corrente terminar (o spinner é
// dono da linha) e escreve a linha final do passo.
func (s *spinner) finish(final string) {
	close(s.stop)
	<-s.done
	fmt.Fprintf(s.out, "\r%s\x1b[K\n", final)
}

// installProgress consome os StepEvents do install.Run: spinner por passo
// (TTY) ou linha plana (pipe/CI). Um spinner por vez — invariantes do
// core garantem o par start→done em ordem.
type installProgress struct {
	out    io.Writer
	tty    bool
	colors ttyColors
	active *spinner
}

func (p *installProgress) consume(ev install.StepEvent) {
	switch ev.Kind {
	case "start":
		p.stopActive() // defensivo: o core emite pares balanceados
		if p.tty {
			p.active = startSpinner(p.out, ev.Label)
		} else {
			fmt.Fprintf(p.out, "  … %s\n", ev.Label)
		}
	case "done":
		final := p.finalLine(ev)
		if p.active != nil {
			p.active.finish(final)
			p.active = nil
			return
		}
		fmt.Fprintln(p.out, final)
	}
}

// stopActive fecha um spinner em curso sem resultado (não deveria
// acontecer — o core emite pares; mantido para nunca vazar goroutine).
func (p *installProgress) stopActive() {
	if p.active != nil {
		p.active.finish(truncateLine(p.active.label, terminalWidth()-6))
		p.active = nil
	}
}

// abort fecha um spinner ativo quando o Run falha (o erro do CLI não pode
// disputar a linha com o spinner).
func (p *installProgress) abort() {
	if p.active != nil {
		p.active.finish(p.colors.yellow + "  ✱ interrompido — " + p.colors.reset + truncateLine(p.active.label, terminalWidth()-22))
		p.active = nil
	}
}

// finalLine monta a linha de resultado do passo (✔ verde / ⚠ amarelo /
// ❌ vermelho — cores respeitam NO_COLOR via ttyColors).
func (p *installProgress) finalLine(ev install.StepEvent) string {
	mark, color := "· ", ""
	switch ev.Outcome {
	case "ok":
		mark, color = "✔ ", p.colors.green
	case "warn":
		mark, color = "⚠ ", p.colors.yellow
	case "fail":
		mark, color = "❌ ", p.colors.red
	}
	return "  " + color + mark + p.colors.reset + truncateLine(ev.Detail, terminalWidth()-8)
}

// truncateLine corta por RUNES (labels têm acentos e emoji) para manter o
// resultado numa única linha de terminal.
func truncateLine(s string, max int) string {
	if max < 4 {
		max = 4
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}
