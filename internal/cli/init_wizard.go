// init_wizard.go — o wizard interativo do `golmet init` SEM flags (M22 —
// FORMATO §2): pergunta o que inicializar ANTES de criar — nunca mais cria
// no cwd silenciosamente (o caso real: init bare no HOME misturou workspace
// com a instalação global do kit). Terms de usuário: "Organização" é o
// conceito do operador; "container" permanece o termo interno do
// golmet.yaml/FORMATO.
//
// Escada de degradação (o padrão do install):
//   - TTY + VT → wizard VISUAL com setas (raw mode — o MESMO padrão do
//     wizard do install: header ◆, ↑/↓ move, enter confirma, esc/q/ctrl-c
//     cancela; detail curto ao lado + descrição contextual do item ativo
//     no rodapé); os campos de texto e a confirmação final são linha-a-
//     linha (o estilo dos campos livres do install);
//   - sem TTY ou sem VT + --plain → wizard NUMERADO lendo stdin (EOF =
//     erro claro);
//   - sem TTY e sem --plain → fail-closed com o comando a rodar — nada é
//     criado no cwd por acidente.
//
// A opção "Workspace-produto dentro de uma Organização existente" lista as
// organizações do CATÁLOGO GLOBAL (~/.golmet/organizations.json — o
// registro que o kit mantém ao criar/adotar orgs) com o mesmo picker; path
// morto é pulado com aviso; a última opção é "Outra — caminho manual".
//
// Confirmação antes de criar SEMPRE (nome/destino exibidos); --yes pula a
// confirmação dentro do wizard. Com flags explícitos (--org | --into) o
// comportamento scriptável atual é intacto — o wizard só roda no init bare.
package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Golmet/golmet-cli/internal/core/container"
	"github.com/Golmet/golmet-cli/internal/core/install"
	"github.com/Golmet/golmet-cli/internal/core/orgs"
	"github.com/Golmet/golmet-cli/internal/core/scaffold"
	"golang.org/x/term"
)

// initWizard — prompts de linha lendo stdin (padrão do installWizard,
// sem os.Exit: erros voltam ao caller para o CLI decidir o exit). tty é o
// contexto do wizard (isTTY na entrada) — define o comportamento do
// cancelamento (o padrão único de wizard_read.go).
type initWizard struct {
	out io.Writer
	in  *bufio.Reader
	cwd string // cwd absoluto — base de todas as sugestões
	tty bool
}

// userHomeFn injetável (testes) — o catálogo global de organizações vive
// no home (~/.golmet/organizations.json).
var userHomeFn = os.UserHomeDir

// pickItem — uma linha de seleção visual: label + detail curto ao lado +
// descrição contextual do item ativo (o padrão do install).
type pickItem struct {
	label, detail, description string
}

// initChoiceOptions — as opções do menu do init (fonte única do visual e
// do numerado). SEM jargão de implementação no menu/descrições —
// golmet.yaml, container: true, CONSTITUTION.md são consequências da
// escolha, mostradas no resumo de confirmação (initWizardCollect).
var initChoiceOptions = []pickItem{
	{"Organização", "a raiz que agrupa os workspaces",
		"Organização é a raiz da empresa que agrupa os workspaces. Crie-a se você está estruturando a empresa — os produtos nascem dentro dela."},
	{"Workspace-produto dentro de uma Organização existente", "projeto novo dentro da raiz",
		"Workspace-produto é um projeto do workflow (charter, requisitos, specs, entrega) — este nasce dentro de uma Organização que você indicar e é registrado na lista de produtos dela."},
	{"Workspace-produto raiz livre", "projeto novo independente",
		"Workspace-produto raiz livre é um projeto independente, sem Organização — para trabalho isolado ou repositório próprio."},
}

func (w *initWizard) prompt(question string) (string, error) {
	fmt.Fprint(w.out, question)
	line, cancelled, serr := readWizardLine(w.in, w.tty,
		"rode com --org ou --into <container> e --yes")
	switch {
	case cancelled:
		return "", errWizardCancelled
	case serr != nil:
		return "", serr
	}
	return line, nil
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

// ---- componente de SELEÇÃO VISUAL (o padrão do wizard do install) ----

// drawPick desenha o frame de seleção: header ◆ + hint, itens com seta
// ">" de largura fixa / nome bold no ativo / detail curto ao lado, nota
// discreta (quando há) e a linha de descrição CONTEXTUAL do item ativo
// embaixo do frame (writeActiveFooter — muda com o cursor).
func drawPick(w *rawWriter, header, hint string, items []pickItem, cursor int, c ttyColors, width int, note string) {
	fmt.Fprintf(w, "\n%s◆ %s%s %s(%s)%s\n\n", c.bold, header, c.reset, c.gray, hint, c.reset)
	if cursor < 0 || cursor >= len(items) {
		cursor = 0 // defensivo — pickApply já constrange
	}
	for i, o := range items {
		if i > 0 {
			fmt.Fprint(w, "\n") // respiro entre as opções (padrão do install)
		}
		fmt.Fprint(w, renderListLineColored(i == cursor, c, "", o.label, c.gray+o.detail+c.reset, 0)+c.reset+"\n")
	}
	if note != "" {
		fmt.Fprintf(w, "\n%s%s%s\n", c.gray, install.TruncateWidth(note, width), c.reset)
	}
	writeActiveFooter(w, c, width, items[cursor].description)
}

// pickApply — mapeamento puro tecla→transição da seleção (o padrão do
// applyWizardKey do install — testável sem console real). selection =
// 1..len(items) quando enter confirma; 0 = continua; cancelled = esc/q/
// ctrl-c/backspace.
func pickApply(cur, n int, key string) (newCur, selection int, cancelled bool) {
	switch key {
	case keyUp:
		if cur > 0 {
			return cur - 1, 0, false
		}
	case keyDown:
		if cur < n-1 {
			return cur + 1, 0, false
		}
	case keyEnter:
		return cur, cur + 1, false
	case keyBack, keyCancel, keyCtrlC:
		return cur, 0, true
	}
	return cur, 0, false
}

// runPickTTY roda a seleção VISUAL em raw mode — o MESMO padrão do
// runInteractiveTTY do install (entra e sai SEMPRE com o terminal
// restaurado; codepage UTF-8 para os glifos). Retorna (1..len(items),
// false) selecionado, (0, true) cancelado.
func runPickTTY(header, hint string, items []pickItem, note string) (int, bool, error) {
	oldCP := setOutputCodepage()
	defer restoreCodepage(oldCP)
	state, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return 0, false, fmt.Errorf("falha ao entrar em raw mode: %w", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), state)
	w := &rawWriter{out: os.Stdout, c: newTTYColors(true)}

	cur := 0
	drawPick(w, header, hint, items, cur, w.c, terminalWidth(), note)
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
		newCur, selection, cancelled := pickApply(cur, len(items), key)
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
		drawPick(w, header, hint, items, cur, w.c, terminalWidth(), note)
	}
}

// pickNumbered — o fallback NUMERADO do mesmo picker (--plain/sem VT):
// os mesmos itens, com a nota discreta quando houver.
func (w *initWizard) pickNumbered(header string, items []pickItem, note string) (int, error) {
	fmt.Fprintf(w.out, "\n%s◆ %s%s\n\n", wizardANSI("\x1b[1m"), header, wizardANSI("\x1b[0m"))
	for i, o := range items {
		fmt.Fprintf(w.out, "  %d. %s — %s\n", i+1, o.label, o.detail)
	}
	if note != "" {
		fmt.Fprintf(w.out, "%s\n", note)
	}
	return w.askNumber(fmt.Sprintf("número [1-%d] (default: 1): ", len(items)), 1, len(items), 1)
}

// ---- fluxo comum: campos + confirmação por opção escolhida ----

// runInitWizard roda o wizard completo: escolha (visual quando visualTTY;
// numerado no fallback) + campos/destino + confirmação. Cancelado →
// (nil, nil).
func runInitWizard(w *initWizard, yes, visualTTY bool) (*scaffold.Options, string, string, error) {
	choice := 0
	if visualTTY {
		sel, cancelled, err := runPickTTY("O que inicializar?",
			"↑/↓ move · enter confirma · esc cancela", initChoiceOptions, "")
		if err != nil {
			return nil, "", "", err
		}
		if cancelled {
			return nil, "", "", nil
		}
		choice = sel
	} else {
		n, err := w.pickNumbered("O que inicializar? (ENTER = opção 1)", initChoiceOptions, "")
		if err != nil {
			return nil, "", "", err
		}
		choice = n
	}
	return initWizardCollect(w, choice, yes, visualTTY)
}

// initWizardCollect — os campos de texto e a confirmação final (linha-a-
// linha, o estilo dos campos livres do install) para a opção escolhida.
// A pré-checagem de nome duplicado no catálogo acontece ANTES da
// confirmação — o kit nunca cria organização que não possa registrar.
func initWizardCollect(w *initWizard, choice int, yes, visualTTY bool) (*scaffold.Options, string, string, error) {
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
		if err := checkOrgCatalogName(root); err != nil {
			return nil, "", "", err
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
		org, cancelled, err := w.pickOrg(suggestion, visualTTY)
		if err != nil {
			return nil, "", "", err
		}
		if cancelled {
			return nil, "", "", nil
		}
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

// askOrgPathManual — o input MANUAL do caminho da Organização (opção 2 —
// "Outra" ou catálogo vazio): instrução clara + exemplo PLACEHOLDER
// GENÉRICO — NUNCA um path do catálogo (M24: privacidade — o exemplo do
// prompt não expõe username/estrutura de pastas do usuário; o catálogo
// alimenta APENAS a lista de seleção, onde o path é funcional). Validação
// com loop de re-pergunta — mensagens em linguagem de usuário, nada de
// jargão. `esc` cancela o wizard inteiro; ENTER aceita a sugestão (quando
// há — e ela também é validada). Devolve o caminho ABSOLUTO resolvido
// (o que aparece na confirmação).
func (w *initWizard) askOrgPathManual(suggestion string) (string, bool, error) {
	const example = `C:\Projetos\MinhaOrganizacao` // placeholder genérico — nunca path de catálogo
	fmt.Fprintf(w.out, "\nCaminho completo da pasta que contém o golmet.yaml da Organização\n")
	fmt.Fprintf(w.out, "(ex.: %s · esc cancela)\n", example)
	for {
		raw, err := w.prompt("Caminho da Organização: ")
		if err != nil {
			if errors.Is(err, errWizardCancelled) {
				return "", true, nil // ESC/CTRL+C — o wizard inteiro cancela
			}
			return "", false, err
		}
		if strings.EqualFold(strings.TrimSpace(raw), "esc") {
			return "", true, nil
		}
		if raw == "" && suggestion != "" {
			raw = suggestion // ENTER = sugestão do pai (também validada)
		}
		p := normalizeOrgPathInput(w.cwd, raw)
		if p == "" {
			fmt.Fprintln(w.out, "obrigatório — informe o caminho da pasta da Organização")
			continue
		}
		if valid, msg := validateOrgPath(p); !valid {
			fmt.Fprintln(w.out, msg)
			continue
		}
		return p, false, nil
	}
}

// validateOrgPath — a pasta existe, tem golmet.yaml e declara a
// Organização (container: true — a fonte da verdade). Mensagens literais
// do formato (§2 — re-pergunta em cada erro).
func validateOrgPath(p string) (bool, string) {
	st, serr := os.Stat(p)
	if serr != nil || !st.IsDir() {
		return false, "a pasta não existe — verifique o caminho"
	}
	if _, ferr := os.Stat(filepath.Join(p, "golmet.yaml")); ferr != nil {
		return false, "a pasta não é uma Organização declarada — rode golmet init --org nela, ou informe outra"
	}
	if _, isContainer, cerr := container.Config(p); cerr != nil || !isContainer {
		return false, "essa pasta é um workspace-produto, não uma Organização"
	}
	return true, ""
}

// normalizeOrgPathInput — aspas simples/duplas removidas (colar do
// explorer às vezes traz aspas), trailing slashes trim (exceto raiz de
// drive) e caminho relativo resolvido contra o cwd — o ABSOLUTO resolvido
// é o que segue para o fluxo e aparece na confirmação.
func normalizeOrgPathInput(cwd, raw string) string {
	p := strings.TrimSpace(raw)
	p = strings.Trim(p, `"'`)
	p = strings.TrimSpace(p)
	for len(p) > 3 && (strings.HasSuffix(p, "/") || strings.HasSuffix(p, "\\")) {
		p = p[:len(p)-1]
	}
	if p == "" {
		return ""
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(cwd, p)
	}
	return filepath.Clean(p)
}

// pickOrg — o passo de Organização da opção 2: seleção das organizações
// do catálogo global (path vivo; morta = pulada com aviso discreto) +
// "Outra — caminho manual" por último. 0 registradas → direto no caminho
// manual (o fluxo anterior) com o aviso de catálogo vazio. Cancelado →
// cancelled (o wizard inteiro cancela — nada criado).
func (w *initWizard) pickOrg(suggestion string, visualTTY bool) (org string, cancelled bool, err error) {
	var alive []orgs.Entry
	dead := 0
	registered := false
	if home, herr := userHomeFn(); herr == nil {
		if c, lerr := orgs.Load(home); lerr == nil {
			registered = len(c.Organizations) > 0
			for _, e := range c.Organizations {
				if e.Alive() {
					alive = append(alive, e)
				} else {
					dead++
				}
			}
		}
	}
	deadNote := ""
	if dead > 0 {
		deadNote = fmt.Sprintf("%d organização(ões) do catálogo com caminho ausente — remova do catálogo (%s) ou restaure o caminho", dead, orgs.FileName)
	}
	if len(alive) == 0 {
		// nenhuma org viva registrada → caminho manual direto (o fluxo
		// anterior), com o aviso quando há catálogo vazio
		if registered {
			// catálogo só com paths mortos — o aviso discreto explica
			fmt.Fprintf(w.out, "%s\n", deadNote)
		} else {
			fmt.Fprintf(w.out, "nenhuma Organização no catálogo — crie com 'golmet init --org'\n")
		}
		return w.askOrgPathManual(suggestion)
	}
	items := make([]pickItem, 0, len(alive)+1)
	for _, e := range alive {
		items = append(items, pickItem{label: e.Name, detail: e.Path, description: e.Path})
	}
	items = append(items, pickItem{label: "Outra — caminho manual",
		detail:      "a Organização não está no catálogo",
		description: "o próximo passo pede o caminho da pasta da Organização"})
	var idx int
	if visualTTY {
		sel, canc, perr := runPickTTY("Escolha a Organização",
			"↑/↓ move · enter confirma · esc cancela", items, deadNote)
		if perr != nil {
			return "", false, perr
		}
		if canc {
			return "", true, nil
		}
		idx = sel
	} else {
		n, perr := w.pickNumbered("Escolha a Organização", items, deadNote)
		if perr != nil {
			return "", false, perr
		}
		idx = n
	}
	if idx == len(items) { // "Outra — caminho manual"
		return w.askOrgPathManual(suggestion)
	}
	return alive[idx-1].Path, false, nil
}

// checkOrgCatalogName — pré-checagem do nome de organização no catálogo
// global ANTES de criar (o kit nunca cria organização que não possa
// registrar: mesmo nome com path diferente = erro claro). Home
// indisponível → sem checagem (o Register na criação volta a validar).
func checkOrgCatalogName(root string) error {
	home, herr := userHomeFn()
	if herr != nil {
		return nil
	}
	return orgs.CheckName(home, filepath.Base(root), root)
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
