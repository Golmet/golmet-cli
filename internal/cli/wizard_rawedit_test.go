package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---- o editor de linha RAW (o contrato de bytes — M26 follow-up) ----

// fakeRawTerminal injeta o raw mode para testes (rawEnter no-op, stdin
// fake, eco capturado) — o padrão do fakeTerminal, mas para o editor.
func fakeRawTerminal(t *testing.T, input string) *strings.Builder {
	t.Helper()
	var b strings.Builder
	oldEnter, oldIn, oldEcho := rawEnter, rawStdin, rawEcho
	rawEnter = func() (func(), error) { return func() {}, nil }
	rawStdin = strings.NewReader(input)
	rawEcho = &b
	t.Cleanup(func() { rawEnter, rawStdin, rawEcho = oldEnter, oldIn, oldEcho })
	return &b
}

// Imprimíveis com UTF-8 multibyte (é/ç) → o texto correto; ENTER termina.
func TestRawEditLineTextWithAccents(t *testing.T) {
	fakeRawTerminal(t, "São Paulo, ação!\n")
	line, err := rawEditLine("Nome: ")
	if err != nil {
		t.Fatalf("editor: %v", err)
	}
	if line != "São Paulo, ação!" {
		t.Errorf("linha = %q, want %q (UTF-8 multibyte)", line, "São Paulo, ação!")
	}
}

// ENTER (\r e \n) termina a linha; linha vazia = "".
func TestRawEditLineEnterTerminates(t *testing.T) {
	fakeRawTerminal(t, "abc\r")
	if line, err := rawEditLine("x: "); err != nil || line != "abc" {
		t.Errorf("ENTER \\r = %q/%v", line, err)
	}
	fakeRawTerminal(t, "\n")
	if line, err := rawEditLine("x: "); err != nil || line != "" {
		t.Errorf("ENTER \\n linha vazia = %q/%v", line, err)
	}
}

// Backspace (\x08 e \x7f) apaga o último caractere — incl. multibyte
// (é = 1 rune, apaga inteira, não 1 byte).
func TestRawEditLineBackspace(t *testing.T) {
	fakeRawTerminal(t, "abc\x08\x08\n")
	if line, err := rawEditLine("x: "); err != nil || line != "a" {
		t.Errorf("backspace = %q/%v, want a", line, err)
	}
	fakeRawTerminal(t, "ação\x7f\n")
	if line, err := rawEditLine("x: "); err != nil || line != "açã" {
		t.Errorf("backspace multibyte = %q/%v, want açã (o DEL apaga a rune ã inteira)", line, err)
	}
	fakeRawTerminal(t, "\x08\n") // backspace em linha vazia = no-op
	if line, err := rawEditLine("x: "); err != nil || line != "" {
		t.Errorf("backspace em vazia = %q/%v, want \"\"", line, err)
	}
}

// ESC e CTRL+C cancelam o wizard inteiro (o sentinel da M26 — a linha
// nunca vaza). Em raw mode o console NÃO gera o evento do CTRL+C: é o
// byte 0x03 tratado.
func TestRawEditLineEscAndCtrlCCancel(t *testing.T) {
	fakeRawTerminal(t, "\x1b")
	_, err := rawEditLine("x: ")
	if !errors.Is(err, errWizardCancelled) {
		t.Errorf("ESC deveria cancelar, got %v", err)
	}
	fakeRawTerminal(t, "abc\x03")
	if _, err := rawEditLine("x: "); !errors.Is(err, errWizardCancelled) {
		t.Errorf("CTRL+C deveria cancelar, got %v", err)
	}
	// texto digitado + ESC: a linha NUNCA vaza (nada é criado com ela)
	fakeRawTerminal(t, "segredo\x1b")
	if line, err := rawEditLine("x: "); !errors.Is(err, errWizardCancelled) || line != "" {
		t.Errorf("ESC após texto = linha %q/%v — want vazia + cancelado", line, err)
	}
}

// Setas (ESC [ X) são sequências IGNORADAS — o editing continua; ESC
// sozinho cancela.
func TestRawEditLineArrowsIgnored(t *testing.T) {
	fakeRawTerminal(t, "\x1b[Aab\x1b[B\n")
	if line, err := rawEditLine("x: "); err != nil || line != "ab" {
		t.Errorf("setas ignoradas = %q/%v, want ab", line, err)
	}
}

// Bytes de controle desconhecidos ignorados (f-keys etc.).
func TestRawEditLineUnknownControlsIgnored(t *testing.T) {
	fakeRawTerminal(t, "a\x01\x02b\n")
	if line, err := rawEditLine("x: "); err != nil || line != "ab" {
		t.Errorf("controles ignorados = %q/%v, want ab", line, err)
	}
}

// ---- o wizard do init inteiro em raw mode (TTY+VT) ----

// ESC de teclado no CAMPO de texto: cancelado limpo (exit 0), nada
// criado, sem mensagem de script (o bug do operador, corrigido de
// verdade — raw mode, não cooked).
func TestInitRawFieldEscCancels(t *testing.T) {
	cwd := inTempDir(t)
	isolatedHome(t)
	fakeTerminal(t, true, true)  // TTY+VT → wizard inteiro raw
	fakeRawTerminal(t, "\n\x1b") // menu: ENTER (opção 1) → campo: ESC
	out, err := runCmd(t, "init")
	if err != nil {
		t.Fatalf("ESC no campo raw deveria cancelar limpo: %v\n%s", err, out)
	}
	if !strings.Contains(out, "cancelado — nada foi criado") {
		t.Errorf("mensagem de cancelamento limpo ausente:\n%s", out)
	}
	if strings.Contains(out, "script/CI") || strings.Contains(out, "stdin fechado") {
		t.Errorf("mensagem de script/CI NUNCA no cancelamento:\n%s", out)
	}
	if _, serr := os.Stat(filepath.Join(cwd, "MinhaOrg")); serr == nil {
		t.Errorf("cancelado não deveria criar nada")
	}
}

// CTRL+C no campo raw (byte 0x03): idem cancelado limpo.
func TestInitRawFieldCtrlCCancels(t *testing.T) {
	inTempDir(t)
	isolatedHome(t)
	fakeTerminal(t, true, true)
	fakeRawTerminal(t, "\n\x03")
	out, err := runCmd(t, "init")
	if err != nil {
		t.Fatalf("CTRL+C no campo raw deveria cancelar limpo: %v\n%s", err, out)
	}
	if !strings.Contains(out, "cancelado — nada foi criado") {
		t.Errorf("mensagem de cancelamento limpo ausente:\n%s", out)
	}
}

// Campo raw com texto válido: o fluxo SEGUE (criação confirmada).
func TestInitRawFieldTextFlow(t *testing.T) {
	base := inTempDir(t)
	isolatedHome(t)
	fakeTerminal(t, true, true)
	// menu ENTER (opção 1) → nome "MinhaOrg" → confirmação "s"
	fakeRawTerminal(t, "\nMinhaOrg\ns\n")
	out, err := runCmd(t, "init")
	if err != nil {
		t.Fatalf("fluxo raw: %v\n%s", err, out)
	}
	if _, serr := os.Stat(filepath.Join(base, "MinhaOrg", "golmet.yaml")); serr != nil {
		t.Fatalf("Organização não criada pelo fluxo raw: %v\n%s", serr, out)
	}
	if !strings.Contains(out, "inicializado em") {
		t.Errorf("relatório do scaffold ausente:\n%s", out)
	}
}
