// wizard_rawedit.go — o EDITOR DE LINHA em RAW MODE (M26 follow-up): em
// modo COOKED o console do Windows captura o ESC ANTES do programa
// (comando de limpar-a-linha-em-edição) — a leitura nunca o vê. A solução
// de verdade: o wizard do init inteiro em raw mode (o mesmo mecanismo do
// menu visual, que funciona porque é raw). Contrato de bytes:
//
//	imprimíveis (UTF-8 multibyte — é/ç funcionam com o codepage UTF-8)
//	  ecoam no fim da linha (redraw \r + \x1b[K — o texto digita-se no
//	  fim da linha, sem cursor móvel no meio);
//	ENTER (\r/\n) termina a linha (devolve o texto digitado);
//	ESC (\x1b) e CTRL+C (\x03) cancelam o wizard inteiro
//	  (errWizardCancelled — a linha nunca vaza; em raw mode o console
//	  NÃO gera o evento de interrupção do CTRL+C — é o byte tratado);
//	backspace (\x08/\x7f) apaga o último caractere impresso;
//	setas/f-keys (ESC [ X) ignoradas — sem histórico nesta versão.
package cli

import (
	"fmt"
	"io"
	"os"
	"time"
	"unicode/utf8"

	"golang.org/x/term"
)

// dependências injetáveis (testes) — o padrão do runInteractiveTTY: o
// teste injeta stdin fake e entra/sai de raw mode como no-op.
var (
	rawEnter = func() (func(), error) {
		state, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			return nil, err
		}
		return func() { term.Restore(int(os.Stdin.Fd()), state) }, nil
	}
	rawStdin io.Reader = os.Stdin
	rawEcho  io.Writer = os.Stdout
)

// rawEscPeek — a janela do peek da sequência ESC [ (setas): sem byte de
// continuação dentro dela = ESC sozinho (cancelar — a resposta tem de ser
// imediata; o peek bloqueante do menu visual não serve no campo de
// texto).
const rawEscPeek = 50 * time.Millisecond

// rawEditLine edita UMA linha em raw mode. Erro de retorno:
// errWizardCancelled (ESC/CTRL+C/EOF) — o wizard inteiro cancela (nada é
// criado antes da confirmação final; invariante §2).
func rawEditLine(prompt string) (string, error) {
	restore, err := rawEnter()
	if err != nil {
		return "", err
	}
	defer restore()
	oldCP := setOutputCodepage()
	defer restoreCodepage(oldCP)
	w := &rawWriter{alt: rawEcho}
	fmt.Fprint(w, prompt)
	var runes []rune
	var pending []byte // sequência UTF-8 multibyte em construção (é/ç)
	buf := make([]byte, 1)
	redraw := func() {
		fmt.Fprintf(w, "\r%s%s\x1b[K", prompt, string(runes))
	}
	for {
		if _, rerr := rawStdin.Read(buf); rerr != nil {
			return "", errWizardCancelled // EOF no raw = console fechou
		}
		b := buf[0]
		switch b {
		case 3: // CTRL+C — em raw mode o console NÃO gera o evento de
			// interrupção: é o byte tratado aqui
			fmt.Fprintln(w)
			return "", errWizardCancelled
		case 27: // ESC — cancela, MAS ESC [ X (seta/f-key) é sequência
			// ignorada, não cancelamento (peek com janela curta)
			if rawEscArrowPeek() {
				fmt.Fprintln(w)
				return "", errWizardCancelled
			}
			continue
		case 13, 10: // ENTER termina a linha
			fmt.Fprintln(w)
			return string(runes), nil
		case 8, 127: // backspace apaga o último caractere impresso
			if n := len(runes); n > 0 {
				runes = runes[:n-1]
				redraw()
			}
			continue
		}
		if b < 32 {
			pending = nil // demais controles — ignorados
			continue
		}
		pending = append(pending, b)
		// multibyte incompleto (é/ç): FullRune=false → continua lendo os
		// bytes da rune ANTES de decodificar — DecodeRune com sequência
		// incompleta devolve (RuneError, 1) e perderia o byte líder
		if !utf8.FullRune(pending) {
			continue
		}
		r, size := utf8.DecodeRune(pending)
		if size == 0 || r == utf8.RuneError {
			pending = nil // byte inválido — descarta
			continue
		}
		pending = nil
		runes = append(runes, r)
		redraw()
	}
}

// rawEscArrowPeek — ESC seguido de "[" (setas/f-keys: ESC [ X) é
// sequência ignorada, não cancelamento. Peek com janela curta
// (rawEscPeek): sem continuação a tempo = ESC sozinho (cancela). A
// goroutine que perde a corrida do timeout só fica bloqueada quando o
// wizard já cancelou — nenhum byte é roubado do fluxo que continua.
func rawEscArrowPeek() (cancelled bool) {
	type peek struct {
		n int
		b []byte
	}
	next := make([]byte, 2)
	ch := make(chan peek, 1)
	go func() {
		n, _ := rawStdin.Read(next)
		ch <- peek{n, next}
	}()
	select {
	case p := <-ch:
		// ESC + os 2 bytes lidos = a sequência completa da seta/f-key
		// (ESC [ X) — ignorada inteira (sem histórico nesta versão)
		if p.n == 2 && p.b[0] == '[' {
			return false
		}
		return true
	case <-time.After(rawEscPeek):
		return true
	}
}
