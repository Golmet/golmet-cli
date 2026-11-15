// init_wizard.go — o wizard interativo do `golmet init` SEM flags (M22 —
// FORMATO §2): pergunta o que inicializar ANTES de criar — nunca mais cria
// no cwd silenciosamente (o caso real: init bare no HOME misturou workspace
// com a instalação global do kit). Terms de usuário: "Organização" é o
// conceito do operador; "container" permanece o termo interno do
// golmet.yaml/FORMATO.
//
// Escada de degradação (o padrão do install — M22 follow-up):
//   - TTY + VT → wizard VISUAL com setas (raw mode — o MESMO padrão do
//     wizard do install: header ◆, ↑/↓ move, enter confirma, esc/q/ctrl-c
//     cancela); os campos de texto e a confirmação final são linha-a-linha
//     (o estilo dos campos livres do install);
//   - sem TTY ou sem VT + --plain → wizard NUMERADO lendo stdin (EOF =
//     erro claro);
//   - sem TTY e sem --plain → fail-closed com o comando a rodar — nada é
//     criado no cwd por acidente.
//
// Confirmação antes de criar SEMPRE (nome/destino exibidos); --yes pula a
// confirmação dentro do wizard. Com flags explícitos (--org | --into) o
// comportamento scriptável atual é intacto — o wizard só roda no init bare.
package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Golmet/golmet-cli/internal/core/container"
	"github.com/Golmet/golmet-cli/internal/core/scaffold"
	"golang.org/x/term"
)

// initWizard — prompts numerados lendo stdin (padrão do installWizard,
// sem os.Exit: erros voltam ao caller para o CLI decidir o exit).
type initWizard struct {
	out io.Writer
	in  *bufio.Reader
	cwd string // cwd absoluto — base de todas as sugestões
}

// initChoiceOptions — as 3 opções do menu (fonte única do wizard visual e
// do numerado): label + detail curto ao lado (o padrão do install —
// "obrigatório", "remoto") + descrição CONTEXTUAL do item ativo (a linha
// embaixo do frame que muda com o cursor — writeActiveFooter). SEM jargão
// de implementação no menu/descrições — golmet.yaml, container: true,
// CONSTITUTION.md são consequências da escolha, mostradas no resumo de
// confirmação (initWizardCollect).
var initChoiceOptions = []struct {
	label, detail, description string
}{
	{"Organização", "a raiz que agrupa os workspaces",
		"Organização é a raiz da empresa que agrupa os workspaces. Crie-a se você está estruturando a empresa — os produtos nascem dentro dela."},
	{"Workspace-produto dentro de uma Organização existente", "projeto novo dentro da raiz",
		"Workspace-produto é um projeto do workflow (charter, requisitos, specs, entrega) — este nasce dentro de uma Organização que você indicar e é registrado na lista de produtos dela."},
	{"Workspace-produto raiz livre", "projeto novo independente",
		"Workspace-produto raiz livre é um projeto independente, sem Organização — para trabalho isolado ou repositório próprio."},
}

func (w *initWizard) prompt(question string) (string, error) {
	fmt.Fprint(w.out, question)
	line, err := w.in.ReadString('\n')
	if err != nil && line == "" {
		return "", fmt.Errorf("stdin fechado (EOF) — o wizard precisa de entrada; em script/CI rode com --org ou --into <container> e --yes")
	}
	return strings.TrimSpace(line), nil
}

// askNumber pergunta um número com default (ENTER vazio = default).
func (w *initWizard) askNumber(question string, min, max, def int) (int, error) {
	for {
		answer, err := w.prompt(question)
		if err != nil {
			return 0, err
		}
		if answer == "" {
			return def, nil
		}
		n, aerr := strconv.Atoi(answer)
		if aerr == nil && n >= min && n <= max {
			return n, nil
		}
		fmt.Fprintf(w.out, "  resposta inválida (%q) — informe um número de %d a %d\n", answer, min, max)
	}
}

// askRequired pergunta até receber algo não-vazio (EOF propaga).
func (w *initWizard) askRequired(question string) (string, error) {
	for {
		answer, err := w.prompt(question)
		if err != nil {
			return "", err
		}
		if answer != "" {
			return answer, nil
		}
		fmt.Fprintln(w.out, "  obrigatório — informe um valor (ENTER não cria nada)")
	}
}

// askWithDefault pergunta com sugestão: ENTER vazio aceita a sugestão;
// sugestão vazia → recai em askRequired (o operador aponta o caminho).
func (w *initWizard) askWithDefault(question, suggestion string) (string, error) {
	if suggestion == "" {
		return w.askRequired(question)
	}
	answer, err := w.prompt(question + " [sugestão: " + suggestion + "]: ")
	if err != nil {
		return "", err
	}
	if answer == "" {
		return suggestion, nil
	}
	return answer, nil
}

// confirm exibe o resumo e pergunta s/n. yes=true pula (o mesmo contrato
// do --yes dos wizards existentes). false = cancelado.
func (w *initWizard) confirm(summary string, yes bool) (bool, error) {
	fmt.Fprintf(w.out, "\nResumo:\n\n%s\n", summary)
	if yes {
		fmt.Fprintln(w.out, "--yes: confirmação pulada")
		return true, nil
	}
	answer, err := w.prompt("Confirmar e criar? (s/n): ")
	if err != nil {
		return false, err
	}
	return strings.EqualFold(answer, "s"), nil
}

// suggestedOrg detecta a sugestão de Organização existente: o dir PAI do
// cwd quando ele é container (golmet.yaml com container: true). Não globa
// o disco (KISS) — o operador aponta caminhos explícitos quando o pai não
// é Organização.
func suggestedOrg(cwd string) string {
	if _, isContainer, err := container.Config(filepath.Dir(cwd)); err == nil && isContainer {
		return filepath.Dir(cwd)
	}
	return ""
}

// ---- passo 1: a ESCOLHA do modo (visual TTY / numerado fallback) ----

// drawInitChoice desenha o frame VISUAL da escolha — o padrão do render
// do install (rawWriter + renderListLineColored: seta ">" fixa, nome bold
// no ativo, detail curto ao lado) + a linha de descrição CONTEXTUAL do
// item ATIVO embaixo do frame (o mesmo mecanismo writeActiveFooter do
// install — muda conforme o cursor navega).
func drawInitChoice(w *rawWriter, cursor int, c ttyColors, width int) {
	fmt.Fprintf(w, "\n%s◆ O que inicializar?%s %s(↑/↓ move · enter confirma · esc cancela)%s\n\n",
		c.bold, c.reset, c.gray, c.reset)
	if cursor < 0 || cursor >= len(initChoiceOptions) {
		cursor = 0 // defensivo — initChoiceApply já constrange
	}
	for i, o := range initChoiceOptions {
		if i > 0 {
			fmt.Fprint(w, "\n") // respiro entre as opções (padrão do install)
		}
		fmt.Fprint(w, renderListLineColored(i == cursor, c, "", o.label, c.gray+o.detail+c.reset, 0)+c.reset+"\n")
	}
	writeActiveFooter(w, c, width, initChoiceOptions[cursor].description)
}

// initChoiceApply — mapeamento puro tecla→transição do passo visual (o
// padrão do applyWizardKey do install — testável sem console real).
// selection = 1..3 quando enter confirma; 0 = continua; cancelled = esc/q/
// ctrl-c/backspace (primeiro passo — não há trás).
func initChoiceApply(cur int, key string) (newCur, selection int, cancelled bool) {
	switch key {
	case keyUp:
		if cur > 0 {
			return cur - 1, 0, false
		}
	case keyDown:
		if cur < len(initChoiceOptions)-1 {
			return cur + 1, 0, false
		}
	case keyEnter:
		return cur, cur + 1, false
	case keyBack, keyCancel, keyCtrlC:
		return cur, 0, true
	}
	return cur, 0, false
}

// runInitChoiceTTY roda o passo VISUAL da escolha em raw mode — o MESMO
// padrão do runInteractiveTTY do install (entra e sai SEMPRE com o
// terminal restaurado; codepage UTF-8 para os glifos). Retorna
// (1..3, false) selecionado, (0, true) cancelado.
func runInitChoiceTTY() (int, bool, error) {
	oldCP := setOutputCodepage()
	defer restoreCodepage(oldCP)
	state, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return 0, false, fmt.Errorf("falha ao entrar em raw mode: %w", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), state)
	w := &rawWriter{out: os.Stdout, c: newTTYColors(true)}

	cur := 0
	drawInitChoice(w, cur, w.c, terminalWidth())
	single := make([]byte, 1)
	for {
		if _, rerr := os.Stdin.Read(single); rerr != nil {
			return 0, true, nil
		}
		key := parseKey(single)
		// setas chegam como sequência: lê o resto do frame ESC [ X
		if single[0] == 27 {
			next := make([]byte, 2)
			if n2, _ := os.Stdin.Read(next); n2 == 2 && next[0] == '[' {
				switch next[1] {
				case 'A', 'H':
					key = keyUp
				case 'B', 'F':
					key = keyDown
				}
			}
		}
		newCur, selection, cancelled := initChoiceApply(cur, key)
		cur = newCur
		if cancelled {
			w.erase()
			return 0, true, nil
		}
		if selection > 0 {
			w.erase()
			return selection, false, nil
		}
		w.erase()
		drawInitChoice(w, cur, w.c, terminalWidth())
	}
}

// printNumberedMenu — o menu NUMERADO do fallback (--plain/sem VT), os
// mesmos itens com detail curto ao lado (a descrição contextual é do
// render visual; o numerado é o fallback compacto).
func (w *initWizard) printNumberedMenu() {
	fmt.Fprintf(w.out, "\n%s◆ O que inicializar?%s %s(ENTER = opção 1)%s\n\n",
		wizardANSI("\x1b[1m"), wizardANSI("\x1b[0m"), wizardANSI("\x1b[90m"), wizardANSI("\x1b[0m"))
	for i, o := range initChoiceOptions {
		fmt.Fprintf(w.out, "  %d. %s — %s\n", i+1, o.label, o.detail)
	}
}

// ---- fluxo comum: campos + confirmação por opção escolhida ----

// runInitWizard roda o wizard completo: escolha (visual quando visualTTY;
// numerado no fallback) + campos/destino + confirmação. Cancelado →
// (nil, nil).
func runInitWizard(w *initWizard, yes, visualTTY bool) (*scaffold.Options, string, string, error) {
	choice := 0
	if visualTTY {
		sel, cancelled, err := runInitChoiceTTY()
		if err != nil {
			return nil, "", "", err
		}
		if cancelled {
			return nil, "", "", nil
		}
		choice = sel
	} else {
		w.printNumberedMenu()
		n, err := w.askNumber("número [1-3] (default: 1): ", 1, 3, 1)
		if err != nil {
			return nil, "", "", err
		}
		choice = n
	}
	return initWizardCollect(w, choice, yes)
}

// initWizardCollect — os campos de texto e a confirmação final (linha-a-
// linha, o estilo dos campos livres do install) para a opção escolhida.
func initWizardCollect(w *initWizard, choice int, yes bool) (*scaffold.Options, string, string, error) {
	suggestion := suggestedOrg(w.cwd)
	switch choice {
	case 1: // Organização
		name, err := w.prompt(fmt.Sprintf("Nome da Organização (vazio = usar o próprio dir atual %q): ", filepath.Base(w.cwd)))
		if err != nil {
			return nil, "", "", err
		}
		root := w.cwd
		if name != "" {
			root = filepath.Join(w.cwd, name)
		}
		summary := fmt.Sprintf("  tipo:      Organização (container: true)\n  raiz:      %s\n  arquivos:  golmet.yaml · CONSTITUTION.md · INDEX.md", root)
		ok, err := w.confirm(summary, yes)
		if err != nil {
			return nil, "", "", err
		}
		if !ok {
			return nil, "", "", nil
		}
		return &scaffold.Options{Org: true}, w.cwd, name, nil
	case 2: // workspace dentro de Organização existente
		org, err := w.askWithDefault("Caminho da Organização existente", suggestion)
		if err != nil {
			return nil, "", "", err
		}
		org = filepath.Clean(org)
		name, err := w.askRequired("Nome do workspace-produto: ")
		if err != nil {
			return nil, "", "", err
		}
		summary := fmt.Sprintf("  tipo:      Workspace-produto dentro de Organização\n  org:       %s\n  produto:   %s (products: da Organização ganha o produto)", org, name)
		ok, err := w.confirm(summary, yes)
		if err != nil {
			return nil, "", "", err
		}
		if !ok {
			return nil, "", "", nil
		}
		return &scaffold.Options{Into: org}, w.cwd, name, nil
	default: // raiz livre
		name, err := w.prompt(fmt.Sprintf("Nome do workspace-produto (vazio = usar o próprio dir atual %q): ", filepath.Base(w.cwd)))
		if err != nil {
			return nil, "", "", err
		}
		root := w.cwd
		if name != "" {
			root = filepath.Join(w.cwd, name)
		}
		summary := fmt.Sprintf("  tipo:      Workspace-produto raiz livre\n  raiz:      %s", root)
		ok, err := w.confirm(summary, yes)
		if err != nil {
			return nil, "", "", err
		}
		if !ok {
			return nil, "", "", nil
		}
		return &scaffold.Options{}, w.cwd, name, nil
	}
}

// cores do wizard init — reuso leve do paletinha (os prompts linha-a-
// linha só colore quando stdout é TTY e sem NO_COLOR; o frame visual usa
// o ttyColors do render do install).
func wizardANSI(seq string) string {
	if os.Getenv("NO_COLOR") != "" || !isTTY() {
		return ""
	}
	return seq
}

// stdinReader injetável (testes) — o default é o stdin real do processo.
var stdinReader = func() *bufio.Reader { return bufio.NewReader(os.Stdin) }
