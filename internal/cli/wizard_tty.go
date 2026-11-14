// wizard_tty.go — render raw-mode do wizard interativo do install (camada
// FINA: estado e transições vivem em install.WizardModel; este arquivo só
// mapeia teclas para os mutadores e desenha com ANSI simples). x/term é
// usado APENAS aqui; testes cobrem o modelo e o helper de linha, não o
// desenho. Respeita NO_COLOR e stdout não-TTY (sem cores).
//
// Regras de render em raw mode (bug de frame duplicado corrigido):
//  1. TODA newline impressa é \r\n — em raw mode o terminal não traduz
//     LF; \n puro move linha mantendo a coluna e os frames "entram tortos".
//  2. Redraw = full clear (\x1b[H\x1b[2J) + frame novo — estimativa de
//     altura com wrap (descrições longas quebram) é frágil; o flicker
//     leve do clear-all é aceitável num wizard CLI (decisão documentada).
//  3. Enter chega como \r (Windows) ou \n (alguns terminais) — ambos são
//     enter; ctrl-c e esc/q cancelam com restore limpo do terminal.
//  4. Linhas de lista: prefixo de LARGURA FIXA (❯  / "  " — a seta
//     SUBSTITUI o espaço, nunca soma) + checkbox na mesma coluna + nome
//     padado com gap mínimo de 1 char antes dos atributos.
package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/Golmet/golmet-cli/internal/core/install"
	"golang.org/x/term"
)

// teclas reconhecidas pelo wizard (raw mode, byte a byte).
const (
	keyUp       = "up"
	keyDown     = "down"
	keyEnter    = "enter"
	keySpace    = "space"
	keyA        = "a"
	keyBack     = "back"   // backspace (0x08) ou DEL (0x7F)
	keyCancel   = "cancel" // q ou esc
	keyCtrlC    = "ctrl-c"
	keyQuestion = "?"
)

// parseKey traduz os bytes lidos em raw mode para teclas: sequências de
// seta (ESC [ A/B), enter (\r no Windows e \n em terminais que entregam
// LF), espaço (32), backspace (8) e DEL (127), letras e esc (ESC sozinho).
// TECLA DESCONHECIDA → "" (no-op no caller, NUNCA cancel — só q/esc/ctrl-c
// cancelam).
func parseKey(buf []byte) string {
	switch {
	case len(buf) == 0:
		return ""
	case buf[0] == 3:
		return keyCtrlC
	case buf[0] == 13 || buf[0] == 10: // \r (Windows raw) e \n (LF)
		return keyEnter
	case buf[0] == 32:
		return keySpace
	case buf[0] == 8 || buf[0] == 127: // backspace e DEL
		return keyBack
	case buf[0] == 27:
		if len(buf) >= 3 && buf[1] == '[' && (buf[2] == 'A' || buf[2] == 'H') {
			return keyUp
		}
		if len(buf) >= 3 && buf[1] == '[' && (buf[2] == 'B' || buf[2] == 'F') {
			return keyDown
		}
		return keyCancel // esc
	case buf[0] == 'q' || buf[0] == 'Q':
		return keyCancel
	case buf[0] == 'a' || buf[0] == 'A':
		return keyA
	case buf[0] == '?':
		return keyQuestion
	}
	return "" // desconhecida → no-op
}

// ttyColors é o paletinha ANSI respeitando NO_COLOR e TTY.
type ttyColors struct {
	green, yellow, red, gray, bold, reset string
}

func newTTYColors(enabled bool) ttyColors {
	if !enabled || os.Getenv("NO_COLOR") != "" {
		return ttyColors{}
	}
	return ttyColors{
		green:  "\x1b[32m",
		yellow: "\x1b[33m",
		red:    "\x1b[31m",
		gray:   "\x1b[90m",
		bold:   "\x1b[1m",
		reset:  "\x1b[0m",
	}
}

// isTTY detecta terminal interativo em stdin e stdout (o wizard precisa
// dos dois: teclas de um, render do outro). Var injetável (testes de
// dispatch).
var isTTY = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

// enableVT injetável (testes): habilita ENABLE_VIRTUAL_TERMINAL_PROCESSING
// no stdout do Windows — sem isso cmd/conhost legado não interpreta o
// clear ANSI (\x1b[H\x1b[2J) e os frames do wizard EMPILHAM. Unix = true.
var enableVT = enableVTWin

// codepage de output (Windows): UTF-8 para os glifos do render; guarda e
// restaura no exit do wizard. Injetáveis (testes).
var (
	setOutputCodepage = setOutputCodepageUTF8
	restoreCodepage   = restoreOutputCodepage
)

// termWidthFn é injetável (testes); default = largura do stdout via term.
var termWidthFn = func() int {
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 20 {
		return w
	}
	return 100
}

// terminalWidth devolve a largura do stdout para truncamento do rodapé.
func terminalWidth() int { return termWidthFn() }

// runInteractiveTTY roda o wizard de teclas (raw mode) sobre o modelo.
// banner = composição pre-renderizada UMA vez fora do loop (wordmark +
// golem + tagline — altura fixa, cabe em qualquer terminal, sem cascata);
// o full-clear limpa tudo e REDESENHA banner+frame a cada tecla. String
// vazia = frame puro (--no-banner). Retorna (modelo pós-aprovação,
// cancelado). Entra e sai SEMPRE com o terminal restaurado (MakeRaw/
// Restore no defer — intocado) e a codepage de output restaurada (UTF-8
// para os glifos no cmd legado).
func runInteractiveTTY(model *install.WizardModel, banner string) (*install.WizardModel, bool, error) {
	oldCP := setOutputCodepage()
	defer restoreCodepage(oldCP)
	state, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return nil, false, fmt.Errorf("falha ao entrar em raw mode: %w", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), state)
	w := &rawWriter{out: os.Stdout, c: newTTYColors(true)}

	var step int
	exit := false
	draw := func() {
		w.erase()
		if banner != "" {
			fmt.Fprint(w, banner)
		}
		switch step {
		case 0:
			drawProfileStep(w, model, w.c, terminalWidth())
		case 1:
			drawEngineStep(w, model, w.c, terminalWidth())
		case 2:
			drawMCPStep(w, model, w.c, terminalWidth())
		case 3:
			drawPlanStep(w, model, w.c)
		}
	}
	draw()
	single := make([]byte, 1)
	for {
		if _, rerr := os.Stdin.Read(single); rerr != nil {
			model.Cancel()
			return model, true, nil
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
		if applyWizardKey(model, &step, key) {
			exit = true
		}
		if exit {
			w.erase()
			return model, model.Cancelled, nil
		}
		draw()
	}
}

// applyWizardKey aplica UMA tecla resolvida ao modelo no passo corrente —
// o mapeamento tecla→mutador do wizard (a única lógica do loop raw).
// Ordem dos passos: perfil (0) → engine (1) → MCPs (2) → plano (3) — o
// perfil é a decisão macro e vem primeiro. *step é sincronizado com
// model.Step quando o passo muda pelo modelo (enter/backspace); retorno
// true = sair do loop (cancelamento ou plano aprovado). Função pura sobre
// o modelo — testável sem console real.
func applyWizardKey(m *install.WizardModel, step *int, key string) bool {
	switch key {
	case keyCtrlC:
		m.Cancel()
		return true
	case keyUp:
		switch *step {
		case 0:
			m.ProfileMove(-1)
		case 1:
			m.EngineMove(-1)
		case 2:
			m.MCPMove(-1)
		}
	case keyDown:
		switch *step {
		case 0:
			m.ProfileMove(1)
		case 1:
			m.EngineMove(1)
		case 2:
			m.MCPMove(1)
		}
	case keyEnter:
		switch *step {
		case 0:
			m.ProfileConfirm()
		case 1:
			m.EngineConfirm()
		case 2:
			_ = m.MCPConfirm() // param vazio → pendência no Run; segue ao plano
		case 3:
			return true // plano aprovado
		}
		*step = int(m.Step)
	case keySpace:
		if *step == 0 {
			m.ProfileToggle()
		} else if *step == 2 {
			m.MCPToggle()
		}
	case keyA:
		if *step == 2 {
			m.MCPToggleAll()
		}
	case keyBack:
		m.Back()
		*step = int(m.Step)
		if *step == 0 {
			m.ProfileNotice = "primeiro passo — não há trás"
		}
	case keyQuestion:
		switch *step {
		case 0:
			opts := m.ProfileOptions()
			if m.ProfileCursor < len(opts) {
				m.ProfileNotice = opts[m.ProfileCursor].Blurb
			}
		case 1:
			id := m.Engines[m.EngineCursor].ID
			m.Notice = m.EngineDescription(id)
		}
	case keyCancel: // q ou esc
		m.Cancel()
		return true
	}
	return false
}

// drawEngineStep desenha o passo engine. Com perfil completo, o modelo já
// FILTROU a lista para os compatíveis (skills nativas) — a lista curta
// fala por si, sem nota adicional.
func drawEngineStep(w *rawWriter, m *install.WizardModel, c ttyColors, width int) {
	fmt.Fprintf(w, "\n%s◆ Engine onde as skills gm-* vão rodar%s %s(↑/↓ move · enter confirma · backspace volta · q cancela)%s\n\n",
		c.bold, c.reset, c.gray, c.reset)
	for i, e := range m.Engines {
		if i > 0 {
			fmt.Fprint(w, "\n") // respiro entre as opções
		}
		mark := c.gray + "[ausente]" + c.reset
		if e.Detected {
			mark = c.green + "[detectado]" + c.reset
		}
		fmt.Fprintf(w, "%s\n", renderListLineColored(i == m.EngineCursor, c, "", e.ID, mark, 0))
	}
	if m.Notice != "" {
		fmt.Fprintf(w, "\n%s  %s%s\n", c.yellow, m.Notice, c.reset)
	}
	writeActiveFooter(w, c, width, m.ActiveDescription())
}

// drawProfileStep desenha o passo perfil (PRIMEIRO passo — decisão macro):
// cada opção com a copy de 2 linhas (o que é + limitação principal),
// checkbox na opção ativa.
func drawProfileStep(w *rawWriter, m *install.WizardModel, c ttyColors, width int) {
	fmt.Fprintf(w, "\n%s◆ Perfil — que experiência você quer?%s %s(↑/↓ move · espaço alterna · enter confirma · backspace volta)%s\n\n",
		c.bold, c.reset, c.gray, c.reset)
	choices := install.ProfileChoices()
	for i, ch := range choices {
		if i > 0 {
			fmt.Fprint(w, "\n") // respiro entre as opções
		}
		checkbox := c.gray + "[ ]" + c.reset
		if i == m.ProfileCursor {
			checkbox = c.green + "[x]" + c.reset
		}
		fmt.Fprint(w, renderListLineColored(i == m.ProfileCursor, c, checkbox,
			ch.ID, "", len(ch.ID))+c.reset+"\n")
		fmt.Fprintf(w, "%s      %s%s\n", c.gray, install.TruncateWidth(ch.What, width-6), c.reset)
		fmt.Fprintf(w, "%s      indicado para: %s%s\n", c.gray, install.TruncateWidth(ch.For, width-19), c.reset)
		fmt.Fprintf(w, "%s      limitação: %s%s\n", c.gray, install.TruncateWidth(ch.Limitation, width-13), c.reset)
	}
	if m.ProfileNotice != "" {
		fmt.Fprintf(w, "\n%s  %s%s\n", c.yellow, m.ProfileNotice, c.reset)
	}
}

func drawMCPStep(w *rawWriter, m *install.WizardModel, c ttyColors, width int) {
	// nome em coluna alinhada à largura do maior nome + gap mínimo de 1
	nameWidth := 0
	for _, def := range m.MCPs {
		if len(def.Name) > nameWidth {
			nameWidth = len(def.Name)
		}
	}
	fmt.Fprintf(w, "\n%s◆ MCPs para registrar%s %s(espaço marca · a alterna todos · ↑/↓ move · enter confirma · backspace volta)%s\n\n",
		c.bold, c.reset, c.gray, c.reset)
	for i, def := range m.MCPs {
		if i > 0 {
			fmt.Fprint(w, "\n") // respiro entre as opções
		}
		checkbox := c.gray + "[ ]" + c.reset
		if m.Checked[def.Name] {
			checkbox = c.green + "[x]" + c.reset
		}
		// badge de status: MCP já registrado no engine → "já registrado
		// (status)" no lugar dos atributos — o usuário VÊ o que já existe
		// antes de marcar (detecção em cache, 1 chamada de list por wizard)
		attrs := ""
		if status, registered := m.Registered[def.Name]; registered {
			attrs = c.gray + "já registrado (" + status + ")" + c.reset
		} else {
			attrs = c.gray + strings.Join(install.MCPAttributes(def), " · ") + c.reset
		}
		active := i == m.MCPCursor
		fmt.Fprint(w, renderListLineColored(active, c, checkbox,
			def.Name, attrs, nameWidth)+c.reset+"\n")
	}
	if m.Notice != "" {
		fmt.Fprintf(w, "\n%s  %s%s\n", c.yellow, m.Notice, c.reset)
	}
	// rodapé de contexto: descrição do item ATIVO (frase longa mora aqui)
	writeActiveFooter(w, c, width, m.ActiveDescription())
}

func drawPlanStep(w *rawWriter, m *install.WizardModel, c ttyColors) {
	plan := m.Plan()
	fmt.Fprintf(w, "\n%s◆ Plano%s %s(enter executa · backspace volta · esc/q cancela)%s\n\n", c.bold, c.reset, c.gray, c.reset)
	fmt.Fprintf(w, "  engine:    %s\n", plan.Engine)
	fmt.Fprintf(w, "  perfil:    %s\n", plan.Profile)
	if plan.SkillsSupported {
		fmt.Fprintf(w, "  skills:    16 gm-* em %s\n", plan.SkillsDir)
	} else {
		fmt.Fprintf(w, "%s  skills:    ficam na zona .golmet/ do workspace (engine sem sistema de skills)%s\n", c.gray, c.reset)
	}
	fmt.Fprintf(w, "  mcps:      %s\n", strings.Join(plan.MCPs, ", "))
	fmt.Fprintf(w, "  workspace: %s\n", plan.Workspace)
}

// writeActiveFooter imprime o rodapé de contexto (descrição do item
// ativo truncada na largura do terminal) separado por uma linha em branco.
func writeActiveFooter(w *rawWriter, c ttyColors, width int, desc string) {
	if desc == "" {
		return
	}
	fmt.Fprintf(w, "\n%s%s%s\n", c.gray, install.TruncateWidth(desc, width), c.reset)
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// rawWriter garante \r\n em toda newline impressa em raw mode (bug 1 do
// frame duplicado) e rastreia a ALTURA do último frame para o erase.
// alt é o destino de teste (setOut); out é o stdout real.
type rawWriter struct {
	out   *os.File
	alt   interface{ Write([]byte) (int, error) }
	c     ttyColors
	lines int // altura física estimada do frame corrente
}

func (w *rawWriter) Write(p []byte) (int, error) {
	expanded := strings.ReplaceAll(string(p), "\n", "\r\n")
	target := w.alt
	if target == nil && w.out != nil {
		target = w.out
	}
	n, err := target.Write([]byte(expanded))
	w.lines += strings.Count(expanded, "\r\n")
	return n, err
}

// erase limpa o frame anterior: volta ao topo da tela e apaga tudo
// (opção B do contrato do fix — infalível contra wrap; flicker aceitável).
func (w *rawWriter) erase() {
	w.Write([]byte("\x1b[H\x1b[2J"))
}

// listPrefix é o prefixo de LARGURA FIXA (2 cols) de cada linha de lista:
// ativo = "> " / inativo = "  ". Marcador ASCII deliberado: o ❯ unicode
// renderiza com largura variável em alguns terminais (desalinhamento real
// observado pelo operador no Windows Terminal) — `>` garante 1 coluna.
func listPrefix(active bool) string {
	if active {
		return "> "
	}
	return "  "
}

// renderListLine monta a linha da lista SEM cores (o caller colore por
// partes): prefixo fixo + checkbox + nome padado + gap mínimo de 1 char +
// atributos. Com attrs vazio, o gap mínimo é o único separador (regressão
// do azure-devops — o maior nome da lista precisa do espaço explícito).
func renderListLine(active bool, checkbox, name, attrs string, nameWidth int) string {
	line := listPrefix(active) + checkbox + "  " + padRight(name, nameWidth)
	if attrs != "" {
		line += " " + attrs
	}
	return line
}

// renderListLineColored colore POR PARTE (nunca a linha inteira — regressão
// do "tudo verde"): seta verde + nome bold SÓ no ativo, prefixo cinza no
// inativo; checkbox e attrs chegam pré-coloridos do caller. O nome é padado
// ANTES do bold (ANSI não ocupa coluna, mas também não deve contaminar o
// padRight). Alinhamento idêntico ao renderListLine.
func renderListLineColored(active bool, c ttyColors, checkbox, name, attrs string, nameWidth int) string {
	padded := padRight(stripANSI(name), nameWidth)
	var prefix string
	if active {
		prefix = c.green + listPrefix(true) + c.reset
		padded = c.bold + padded + c.reset
	} else {
		prefix = c.gray + listPrefix(false) + c.reset
	}
	line := prefix + checkbox + "  " + padded
	if attrs != "" {
		line += " " + attrs
	}
	return line
}

// stripANSI remove sequências ANSI (contagem de colunas visíveis).
func stripANSI(s string) string {
	var b strings.Builder
	inSeq := false
	for _, r := range s {
		if r == '\x1b' {
			inSeq = true
			continue
		}
		if inSeq {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inSeq = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
