// wizard_read.go — o padrão ÚNICO de cancelamento dos wizards de linha do
// CLI (init, install, uninstall) — FORMATO §2/§9:
//   - ESC de teclado: a linha "\x1b" (ESC sozinho vaza como byte em
//     alguns terminais) e a linha literal "esc" CANCELAM — o ESC nunca
//     vaza como caractere no input;
//   - EOF/erro de leitura: no TTY (o CTRL+C chega como EOF na leitura) é
//     cancelamento LIMPO — "cancelado — nada <criado|aplicado|removido>",
//     exit 0, NUNCA a mensagem de script/CI (o pior lugar para ela é o
//     meio de uma sessão interativa); fora de TTY (script/CI) é a
//     mensagem de script com o comando a rodar;
//   - cancelamento em qualquer passo = nada criado (os wizards só criam
//     após a confirmação final — invariante §2).
package cli

import (
	"bufio"
	"errors"
	"fmt"
	"strings"
)

// errWizardCancelled — o sentinel de cancelamento limpo: o CLI imprime
// "cancelado — nada <verbo do comando>" e sai 0.
var errWizardCancelled = errors.New("wizard cancelado")

// readWizardLine — UMA leitura de linha com o padrão de cancelamento.
// Devolve a linha trimada; cancelled = ESC (\x1b ou literal "esc") ou
// EOF/erro de leitura NO TTY; scriptErr = EOF fora de TTY (a mensagem com
// a remediação do comando — nunca no TTY).
func readWizardLine(in *bufio.Reader, tty bool, scriptFix string) (string, bool, error) {
	line, err := in.ReadString('\n')
	if err != nil && line == "" {
		if tty {
			return "", true, nil // CTRL+C no terminal = cancelamento limpo
		}
		return "", false, fmt.Errorf("stdin fechado (EOF) — o wizard precisa de entrada; em script/CI %s", scriptFix)
	}
	s := strings.TrimSpace(line)
	if s == "\x1b" || strings.EqualFold(s, "esc") {
		return "", true, nil
	}
	return s, false, nil
}
