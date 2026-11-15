// init_wizard.go — o wizard interativo do `golmet init` SEM flags (M22 —
// FORMATO §2): pergunta o que inicializar ANTES de criar — nunca mais cria
// no cwd silenciosamente (o caso real: init bare no HOME misturou workspace
// com a instalação global do kit). Terms de usuário: "Organização" é o
// conceito do operador; "container" permanece o termo interno do
// golmet.yaml/FORMATO.
//
// Escada de degradação (o padrão do install):
//   - TTY → wizard numerado lendo stdin (o mesmo estilo dos pickers do
//     install — KISS: init não precisa de raw mode para 3 perguntas);
//   - sem TTY + --plain → o MESMO wizard numerado (lê stdin; EOF = erro);
//   - sem TTY e sem --plain → fail-closed com o comando a rodar — nada é
//     criado no cwd por acidente.
//
// Confirmção antes de criar SEMPRE (nome/destino exibidos); --yes pula a
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
)

// initWizard — prompts numerados lendo stdin (padrão do installWizard,
// sem os.Exit: erros voltam ao caller para o CLI decidir o exit).
type initWizard struct {
	out io.Writer
	in  *bufio.Reader
	cwd string // cwd absoluto — base de todas as sugestões
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

// runInitWizard roda o wizard completo e devolve os parâmetros resolvidos
// do scaffold. Cancelado → (nil, nil).
func runInitWizard(w *initWizard, yes bool) (*scaffold.Options, string, string, error) {
	fmt.Fprintf(w.out, "\n%s◆ O que inicializar?%s %s(ENTER = opção 1)%s\n\n",
		ttyColorsBoldForWizard(), ttyColorsResetForWizard(), ttyColorsGrayForWizard(), ttyColorsResetForWizard())
	fmt.Fprintf(w.out, "  1. Organização — a raiz que agrupa os workspaces da empresa\n")
	fmt.Fprintf(w.out, "     (cria golmet.yaml com container: true · CONSTITUTION.md · INDEX.md)\n")
	fmt.Fprintf(w.out, "  2. Workspace-produto dentro de uma Organização existente\n")
	fmt.Fprintf(w.out, "  3. Workspace-produto raiz livre (sem Organização)\n")

	choice, err := w.askNumber("número [1-3] (default: 1): ", 1, 3, 1)
	if err != nil {
		return nil, "", "", err
	}

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

// cores do wizard init — reuso leve do paletinha (o init wizard roda em
// modo linha, sem raw mode; cores só quando stdout é TTY e sem NO_COLOR).
func ttyColorsBoldForWizard() string  { return wizardANSI("\x1b[1m") }
func ttyColorsResetForWizard() string { return wizardANSI("\x1b[0m") }
func ttyColorsGrayForWizard() string  { return wizardANSI("\x1b[90m") }
func wizardANSI(seq string) string {
	if os.Getenv("NO_COLOR") != "" || !isTTY() {
		return ""
	}
	return seq
}

// stdinReader injetável (testes) — o default é o stdin real do processo.
var stdinReader = func() *bufio.Reader { return bufio.NewReader(os.Stdin) }
