// init_wizard.go — o wizard interativo do `golmet init` SEM flags (M22 —
// FORMAT §2): pergunta o que inicializar ANTES de criar — nunca mais cria
// no cwd silenciosamente (o caso real: init bare no HOME misturou workspace
// com a instalação global do kit). Terms de usuário: "Organização" é o
// conceito do operador; "container" permanece o termo interno do
// golmet.yaml/FORMAT.
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
)

// initWizard — prompts de linha lendo stdin (padrão do installWizard,
// sem os.Exit: erros voltam ao caller para o CLI decidir o exit). tty é o
// contexto do wizard (isTTY na entrada) — define o comportamento do
// cancelamento cooked (o padrão único de wizard_read.go). raw = TTY+VT:
// os campos de texto vão pelo EDITOR RAW (wizard_rawedit.go) — em cooked
// o console do Windows captura o ESC antes do programa; em raw mode o
// editor entrega o contrato completo.
type initWizard struct {
	out io.Writer
	in  *bufio.Reader
	cwd string // cwd absoluto — base de todas as sugestões
	tty bool
	raw bool // TTY+VT → campos em raw mode
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

func (w *initWizard) prompt(question, initial string) (string, error) {
	if w.raw {
		// editor raw (o padrão da M26 revisada): em cooked o console do
		// Windows captura o ESC — em raw mode o editor entrega ESC/CTRL+C
		// como cancelamento, ← no texto vazio como back (o sentinel
		// errWizardBack) e o texto com UTF-8/backspace/setas-ignoradas; o
		// initial vem PRÉ-ECADO e editável (a navegação preserva o valor)
		return rawEditLine(question, initial)
	}
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
		answer, err := w.prompt(question, "")
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

// askStepText — UM campo de texto do wizard com navegação (M27): o valor
// anterior vem PRÉ-ECADO e editável (raw editor com initial; no cooked o
// ENTER mantém o anterior — o console não entrega ←). ← no texto vazio
// volta ao passo anterior (errWizardBack); esc/ctrl-c cancelam TUDO;
// required = ENTER vazio (sem anterior) re-pergunta.
func (w *initWizard) askStepText(question, initial string, required bool) (string, bool, error) {
	for {
		line, err := w.prompt(question, initial)
		if err != nil {
			if errors.Is(err, errWizardBack) {
				return "", true, nil // ← no texto vazio — volta ao passo anterior
			}
			return "", false, err // errWizardCancelled ou raw indisponível
		}
		if line == "" {
			if !w.raw && initial != "" {
				return initial, false, nil // cooked: ENTER mantém o anterior
			}
			if required {
				fmt.Fprintln(w.out, "  obrigatório — informe um valor (ENTER não cria nada)")
				initial = "" // o valor anterior foi limpo de propósito
				continue
			}
			return line, false, nil // vazio permitido (= usar o dir atual)
		}
		return line, false, nil
	}
}

// confirmStep — o confirm NAVEGÁVEL (M27) com o desenho VISUAL do
// install (M28): o frame ◆ fica ISOLADO embaixo do resumo (linha em
// branco entre eles — nunca colado). O input: ENTER confirma e cria (o
// default — a ação principal sem letra); n/← voltam a editar (M27 — o
// valor anterior fica pré-ecado no passo anterior); esc/ctrl-c cancelam
// TUDO (o invariante: só esc/ctrl-c cancelam); s confirma como ALIAS
// (compat com quem digita s — documentado); qualquer outra resposta =
// re-pergunta. true = confirmado; false+nil = voltar a editar; err =
// cancelamento/script.
func (w *initWizard) confirmStep(summary string, yes bool) (bool, error) {
	fmt.Fprintf(w.out, "\nResumo:\n\n%s\n\n", summary)
	if yes {
		fmt.Fprintln(w.out, "--yes: confirmação pulada")
		return true, nil
	}
	for {
		// o frame ◆ re-renderiza a cada re-pergunta (o padrão do install)
		fmt.Fprintf(w.out, "%s◆ Confirmar e criar?%s %s(ENTER confirma · n volta a editar · esc cancela)%s\n\n",
			wizardANSI("\x1b[1m"), wizardANSI("\x1b[0m"), wizardANSI("\x1b[90m"), wizardANSI("\x1b[0m"))
		answer, err := w.prompt("> ", "")
		if err != nil {
			if errors.Is(err, errWizardBack) {
				return false, nil // ← = voltar a editar
			}
			return false, err
		}
		switch {
		case answer == "":
			return true, nil // ENTER confirma (o default)
		case strings.EqualFold(answer, "s"):
			return true, nil // alias do ENTER (compat)
		case strings.EqualFold(answer, "n"):
			return false, nil
		default:
			fmt.Fprintln(w.out, "  resposta inválida — ENTER confirma · n volta a editar · esc cancela")
		}
	}
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
// restaurado; codepage UTF-8 para os glifos; as dependências de raw mode
// são as injetáveis de wizard_rawedit.go — rawEnter/rawStdin/rawEcho).
// Retorna (1..len(items), false) selecionado, (0, true) cancelado.
func runPickTTY(header, hint string, items []pickItem, note string) (int, bool, error) {
	restore, err := rawEnter()
	if err != nil {
		return 0, false, fmt.Errorf("falha ao entrar em raw mode: %w", err)
	}
	defer restore()
	oldCP := setOutputCodepage()
	defer restoreCodepage(oldCP)
	w := &rawWriter{alt: rawEcho, c: newTTYColors(true)}

	cur := 0
	drawPick(w, header, hint, items, cur, w.c, terminalWidth(), note)
	single := make([]byte, 1)
	for {
		if _, rerr := rawStdin.Read(single); rerr != nil {
			return 0, true, nil
		}
		key := parseKey(single)
		// ESC: a sequência/seta via peek com janela curta (o peek
		// bloqueante antigo pendurava o ESC sozinho); ← no menu =
		// ignorado (o primeiro passo não tem anterior)
		if single[0] == 27 {
			switch p := rawEscPeek(); p {
			case escUp:
				key = keyUp
			case escDown:
				key = keyDown
			case escLeft, escOther:
				key = "" // no-op no menu (← não tem para onde voltar)
			default: // escSolo
				key = keyCancel
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

// ---- o loop de STEPS com back (M27) ----

// initStep — os passos do wizard do init na cadeia next/back.
type initStep int

const (
	initStepChoice      initStep = iota // o menu — o PRIMEIRO passo (← ignorado)
	initStepOrgName                     // opção 1: nome da Organização
	initStepOrgPath                     // opção 2: caminho/seleção da Organização
	initStepProductName                 // opção 2/3: nome do workspace-produto
	initStepConfirm                     // confirmação (s cria · n/← voltam a editar)
)

// runInitWizard roda o wizard inteiro como LOOP DE STEPS com transições
// next/back/cancel: enter/validação avança; **← no texto VAZIO volta ao
// passo anterior** (o valor anterior pré-ecado e editável — com conteúdo
// a seta é ignorada, o backspace apaga); **n/← no confirm voltam a
// editar**; esc/ctrl-c cancelam TUDO — nada é criado antes da confirmação
// final (invariante §2). A pré-checagem de duplicado do catálogo acontece
// antes de avançar (voltar não a re-dispara até chegar no passo de novo).
func runInitWizard(w *initWizard, yes, visual bool) (*scaffold.Options, string, string, error) {
	step := initStepChoice
	var (
		choice   int
		orgName  string
		orgPath  string
		prodName string
	)
	for {
		switch step {
		case initStepChoice:
			c, err := w.pickChoice(visual)
			if err != nil {
				return nil, "", "", err
			}
			choice = c
			switch c {
			case 1:
				step = initStepOrgName
			case 2:
				step = initStepOrgPath
			default:
				step = initStepProductName
			}
		case initStepOrgName:
			name, back, err := w.askStepText(
				fmt.Sprintf("Nome da Organização (vazio = usar o próprio dir atual %q): ", filepath.Base(w.cwd)),
				orgName, false)
			if err != nil {
				return nil, "", "", err
			}
			if back { // ← — o valor do passo fica preservado para o retorno
				step = initStepChoice
				continue
			}
			root := w.cwd
			if name != "" {
				root = filepath.Join(w.cwd, name)
			}
			// a pré-checagem acontece ANTES de avançar (o kit nunca cria
			// organização que não possa registrar)
			if cerr := checkOrgCatalogName(root); cerr != nil {
				return nil, "", "", cerr
			}
			orgName = name
			step = initStepConfirm
		case initStepOrgPath:
			org, back, err := w.pickOrg(suggestedOrg(w.cwd), visual)
			if err != nil {
				return nil, "", "", err
			}
			if back {
				step = initStepChoice
				continue
			}
			orgPath = org
			step = initStepProductName
		case initStepProductName:
			var question string
			if choice == 2 {
				question = "Nome do workspace-produto: "
			} else {
				question = fmt.Sprintf("Nome do workspace-produto (vazio = usar o próprio dir atual %q): ", filepath.Base(w.cwd))
			}
			name, back, err := w.askStepText(question, prodName, choice == 2)
			if err != nil {
				return nil, "", "", err
			}
			if back {
				if choice == 2 {
					step = initStepOrgPath // ← volta ao passo da Organização
				} else {
					step = initStepChoice
				}
				continue
			}
			prodName = name
			step = initStepConfirm
		case initStepConfirm:
			var opt scaffold.Options
			var summary string
			switch choice {
			case 1:
				root := w.cwd
				if orgName != "" {
					root = filepath.Join(w.cwd, orgName)
				}
				opt = scaffold.Options{Org: true}
				summary = fmt.Sprintf("  tipo:      Organização (container: true)\n  raiz:      %s\n  arquivos:  golmet.yaml · CONSTITUTION.md · INDEX.md", root)
			case 2:
				opt = scaffold.Options{Into: orgPath}
				summary = fmt.Sprintf("  tipo:      Workspace-produto dentro de Organização\n  org:       %s\n  produto:   %s (products: da Organização ganha o produto)", orgPath, prodName)
			default:
				root := w.cwd
				if prodName != "" {
					root = filepath.Join(w.cwd, prodName)
				}
				summary = fmt.Sprintf("  tipo:      Workspace-produto raiz livre\n  raiz:      %s", root)
			}
			confirmed, err := w.confirmStep(summary, yes)
			if err != nil {
				return nil, "", "", err
			}
			if !confirmed { // n/← — volta ao campo anterior para editar
				if choice == 2 {
					step = initStepProductName
				} else {
					step = initStepOrgName
				}
				continue
			}
			name := orgName
			if choice != 1 {
				name = prodName
			}
			return &opt, w.cwd, name, nil
		}
	}
}

// pickChoice — o passo da ESCOLHA (o primeiro — ← ignorado nele: não há
// anterior; o cancelamento segue esc/ctrl-c).
func (w *initWizard) pickChoice(visual bool) (int, error) {
	if visual {
		sel, cancelled, err := runPickTTY("O que inicializar?",
			"↑/↓ move · enter confirma · esc cancela", initChoiceOptions, "")
		if err != nil {
			return 0, err
		}
		if cancelled {
			return 0, errWizardCancelled
		}
		return sel, nil
	}
	return w.pickNumbered("O que inicializar? (ENTER = opção 1)", initChoiceOptions, "")
}

// askOrgPathManual — o input MANUAL do caminho da Organização (opção 2 —
// "Outra" ou catálogo vazio): instrução clara + exemplo PLACEHOLDER
// GENÉRICO — NUNCA um path do catálogo (M24: privacidade — o exemplo do
// prompt não expõe username/estrutura de pastas do usuário; o catálogo
// alimenta APENAS a lista de seleção, onde o path é funcional). Validação
// com loop de re-pergunta — mensagens em linguagem de usuário, nada de
// jargão. `esc`/CTRL+C cancelam o wizard inteiro (errWizardCancelled); ←
// no texto vazio VOLTA ao passo da escolha (errWizardBack); ENTER aceita
// a sugestão (quando há — também validada); o valor inválido digitado
// volta pré-ecado para correção. Devolve o caminho ABSOLUTO resolvido
// (o que aparece na confirmação).
func (w *initWizard) askOrgPathManual(suggestion string) (string, bool, error) {
	const example = `C:\Projetos\MinhaOrganizacao` // placeholder genérico — nunca path de catálogo
	fmt.Fprintf(w.out, "\nCaminho completo da pasta que contém o golmet.yaml da Organização\n")
	fmt.Fprintf(w.out, "(ex.: %s · esc cancela · ← volta)\n", example)
	last := suggestion // o valor pré-ecado (a sugestão ou o último digitado)
	for {
		raw, err := w.prompt("Caminho da Organização: ", last)
		if err != nil {
			if errors.Is(err, errWizardBack) {
				return "", true, nil // ← no texto vazio — volta à escolha
			}
			return "", false, err // errWizardCancelled ou raw indisponível
		}
		if raw == "" && !w.raw && suggestion != "" {
			raw = suggestion // cooked: ENTER = sugestão do pai (também validada)
		}
		p := normalizeOrgPathInput(w.cwd, raw)
		if p == "" {
			fmt.Fprintln(w.out, "obrigatório — informe o caminho da pasta da Organização")
			last = ""
			continue
		}
		if valid, msg := validateOrgPath(p); !valid {
			fmt.Fprintln(w.out, msg)
			last = p // o inválido volta pré-ecado para correção
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
// manual (o fluxo anterior) com o aviso de catálogo vazio. Cancelado =
// errWizardCancelled (o wizard inteiro cancela — nada criado); back =
// ← no campo manual (volta à escolha).
func (w *initWizard) pickOrg(suggestion string, visualTTY bool) (org string, back bool, err error) {
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
			return "", false, errWizardCancelled
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
