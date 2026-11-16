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
	"errors"
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
// imediata; o peek bloqueante não serve em campo de texto).
const rawEscPeekWindow = 50 * time.Millisecond

// escPeek — a classificação do byte pós-ESC: ESC sozinho (ou sequência
// desconhecida) e as setas ↑/↓/←.
type escPeek int

const (
	escSolo  escPeek = iota // ESC sozinho → cancelar
	escUp                   // ESC [ A/H
	escDown                 // ESC [ B/F
	escLeft                 // ESC [ D — o ← da navegação multi-step (M27)
	escOther                // ESC [ X desconhecido
)

// rawEscPeek lê os 2 bytes de continuação do ESC com janela curta e
// classifica. A goroutine que perde a corrida do timeout só fica
// bloqueada quando o wizard já cancelou — nenhum byte é roubado do fluxo
// que continua.
func rawEscPeek() escPeek {
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
		if p.n == 2 && p.b[0] == '[' {
			switch p.b[1] {
			case 'A', 'H':
				return escUp
			case 'B', 'F':
				return escDown
			case 'D':
				return escLeft
			default:
				return escOther
			}
		}
		return escSolo
	case <-time.After(rawEscPeekWindow):
		return escSolo
	}
}

// errWizardBack — ← no texto VAZIO: volta ao passo anterior (M27). O
// caller decide — nunca cancela, nunca cria; o passo anterior re-renderiza
// com o valor anterior PRÉ-ECADO e editável.
var errWizardBack = errors.New("wizard back")

// rawEditLine edita UMA linha em raw mode. O initial (M27) vem PRÉ-ECADO
// e editável (a navegação multi-step preserva o valor do passo). Erro de
// retorno: errWizardCancelled (ESC/CTRL+C/EOF) ou errWizardBack (← no
// texto vazio) — o wizard inteiro cancela no primeiro; nada é criado
// antes da confirmação final (invariante §2).
func rawEditLine(prompt, initial string) (string, error) {
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
	redraw := func() {
		fmt.Fprintf(w, "\r%s%s\x1b[K", prompt, string(runes))
	}
	if initial != "" {
		// M27 — o valor anterior do passo pré-ecado e editável
		runes = []rune(initial)
		redraw()
	}
	var pending []byte // sequência UTF-8 multibyte em construção (é/ç)
	buf := make([]byte, 1)
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
		case 27: // ESC — cancela, MAS ESC [ X (seta) é sequência: o ← no
			// texto VAZIO volta ao passo anterior (M27); com conteúdo é
			// ignorado (o backspace apaga); ↑/↓/f-keys ignorados
			switch p := rawEscPeek(); p {
			case escLeft:
				if len(runes) == 0 {
					fmt.Fprintln(w)
					return "", errWizardBack
				}
				continue // ← com conteúdo = ignorado
			case escUp, escDown, escOther:
				continue // ignorados num campo de texto
			default: // escSolo — ESC sozinho cancela
				fmt.Fprintln(w)
				return "", errWizardCancelled
			}
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
