package cli

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---- o padrão ÚNICO de cancelamento (FORMAT §2/§9 — wizard_read.go) ----

func TestReadWizardLineEscKeyAndLiteral(t *testing.T) {
	// ESC de teclado: a linha \x1b (ESC sozinho vaza como byte em alguns
	// terminais) cancela — o ESC NUNCA vaza como caractere no input
	r := bufio.NewReader(strings.NewReader("\x1b\n"))
	line, cancelled, serr := readWizardLine(r, true, "fix")
	if !cancelled || line != "" || serr != nil {
		t.Errorf("ESC de teclado = %q/%v/%v, want cancelado", line, cancelled, serr)
	}
	// a linha literal "esc" idem (o fallback de linha)
	r2 := bufio.NewReader(strings.NewReader("esc\n"))
	if _, cancelled2, _ := readWizardLine(r2, false, "fix"); !cancelled2 {
		t.Errorf("linha literal esc deveria cancelar")
	}
	// "ESC" em maiúsculas idem
	r3 := bufio.NewReader(strings.NewReader("ESC\n"))
	if _, cancelled3, _ := readWizardLine(r3, false, "fix"); !cancelled3 {
		t.Errorf("linha literal ESC deveria cancelar")
	}
}

func TestReadWizardLineEOFByContext(t *testing.T) {
	// EOF NO TTY (o CTRL+C chega como EOF na leitura) = cancelamento LIMPO
	// — NUNCA a mensagem de script/CI no meio de uma sessão interativa
	r := bufio.NewReader(strings.NewReader(""))
	_, cancelled, serr := readWizardLine(r, true, "fix de script")
	if !cancelled || serr != nil {
		t.Errorf("EOF no TTY = cancelled %v, scriptErr %v — want cancelado limpo", cancelled, serr)
	}
	// EOF FORA de TTY (script/CI) = a mensagem de script com o comando
	r2 := bufio.NewReader(strings.NewReader(""))
	_, cancelled2, serr2 := readWizardLine(r2, false, "rode com flags e --yes")
	if cancelled2 || serr2 == nil {
		t.Fatalf("EOF fora de TTY deveria dar erro de script: %v", serr2)
	}
	if !strings.Contains(serr2.Error(), "stdin fechado (EOF)") || !strings.Contains(serr2.Error(), "rode com flags e --yes") {
		t.Errorf("mensagem de script incompleta: %v", serr2)
	}
}

func TestReadWizardLineData(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("  C:\\Projetos\\Org  \n"))
	line, cancelled, serr := readWizardLine(r, true, "fix")
	if cancelled || serr != nil || line != `C:\Projetos\Org` {
		t.Errorf("linha de dados = %q/%v/%v — want trimada, sem cancelamento", line, cancelled, serr)
	}
	// texto com "esc" DENTRO não cancela (é dado — "pescaria" etc.)
	r2 := bufio.NewReader(strings.NewReader("pescaria\n"))
	line2, cancelled2, _ := readWizardLine(r2, true, "fix")
	if cancelled2 || line2 != "pescaria" {
		t.Errorf("esc dentro de palavra não deveria cancelar: %q/%v", line2, cancelled2)
	}
}

// ---- init: ESC/CTRL+C no campo de texto = cancelado limpo (exit 0) ----

// ESC (linha \x1b) no campo de texto: cancelado, NADA criado, sem a
// mensagem de script/CI.
func TestInitEscInTextFieldCancelsClean(t *testing.T) {
	cwd := inTempDir(t)
	fakeStdin(t, "\x1b\n")
	out, err := runCmd(t, "init", "--plain")
	if err != nil {
		t.Fatalf("ESC deveria cancelar limpo (exit 0): %v\n%s", err, out)
	}
	if !strings.Contains(out, "cancelado — nada foi criado") {
		t.Errorf("mensagem de cancelamento limpo ausente:\n%s", out)
	}
	if strings.Contains(out, "stdin fechado") || strings.Contains(out, "script/CI") {
		t.Errorf("mensagem de script/CI NUNCA no cancelamento:\n%s", out)
	}
	if _, serr := os.Stat(filepath.Join(cwd, "golmet.yaml")); serr == nil {
		t.Errorf("cancelado não deveria criar nada")
	}
}

// CTRL+C simulado (EOF na leitura COM TTY): cancelamento limpo — a
// mensagem de script/CI nunca aparece (bug 2 do operador).
func TestInitCtrlCSimulatedCancelsClean(t *testing.T) {
	inTempDir(t)
	fakeStdin(t, "")             // EOF na leitura
	fakeTerminal(t, true, false) // TTY sem VT → modo numerado
	out, err := runCmd(t, "init")
	if err != nil {
		t.Fatalf("CTRL+C (EOF no TTY) deveria cancelar limpo: %v\n%s", err, out)
	}
	if !strings.Contains(out, "cancelado — nada foi criado") {
		t.Errorf("mensagem de cancelamento limpo ausente:\n%s", out)
	}
	if strings.Contains(out, "script/CI") || strings.Contains(out, "stdin fechado") {
		t.Errorf("mensagem de script/CI NUNCA no TTY:\n%s", out)
	}
}

// EOF FORA de TTY (script/CI): a mensagem de script com o comando (o
// comportamento de M22 — só quando o contexto não é TTY).
func TestInitEOFNonTTYScriptMessage(t *testing.T) {
	inTempDir(t)
	fakeStdin(t, "")
	fakeTerminal(t, false, false)
	out, err := runCmd(t, "init", "--plain")
	if err == nil {
		t.Fatalf("EOF sem TTY deveria falhar com a mensagem de script")
	}
	if code, ok := ExitCodeOf(err); !ok || code != 2 {
		t.Errorf("exit = %v, want 2", err)
	}
	if !strings.Contains(out, "stdin fechado (EOF)") || !strings.Contains(out, "script/CI") {
		t.Errorf("mensagem de script esperada fora de TTY:\n%s", out)
	}
}

// ---- install e uninstall: o MESMO padrão de cancelamento ----

// installWizard: EOF no TTY = cancelamento limpo (sem mensagem de
// script); EOF fora de TTY = mensagem de script.
func TestInstallWizardPromptCancelByContext(t *testing.T) {
	wTTY := &installWizard{out: &strings.Builder{}, in: bufio.NewReader(strings.NewReader("")), tty: true}
	_, err := wTTY.prompt("pergunta: ")
	if !errors.Is(err, errWizardCancelled) {
		t.Errorf("EOF no TTY deveria cancelar limpo, got %v", err)
	}
	wPipe := &installWizard{out: &strings.Builder{}, in: bufio.NewReader(strings.NewReader("")), tty: false}
	_, err2 := wPipe.prompt("pergunta: ")
	if err2 == nil || errors.Is(err2, errWizardCancelled) || !strings.Contains(err2.Error(), "--engine/--mcp/--yes") {
		t.Errorf("EOF fora de TTY deveria dar a mensagem de script: %v", err2)
	}
}

// uninstall: ESC na seleção = cancelado limpo "cancelado — nada foi
// removido" (exit 0); EOF sem TTY = mensagem de script (exit 2).
func TestUninstallWizardCancelByContext(t *testing.T) {
	oldTTY := isTTY
	isTTY = func() bool { return false }
	t.Cleanup(func() { isTTY = oldTTY })

	// ESC na seleção (home de teste COM config de engine para o plano)
	home := t.TempDir()
	cfg := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(cfg, 0o755); err != nil {
		t.Fatal(err)
	}
	fakeStdin(t, "esc\n")
	out, err := runCmd(t, "uninstall", "--home", home)
	if err != nil {
		t.Fatalf("ESC deveria cancelar limpo: %v\n%s", err, out)
	}
	if !strings.Contains(out, "cancelado — nada foi removido") {
		t.Errorf("mensagem de cancelamento do uninstall ausente:\n%s", out)
	}
	if strings.Contains(out, "script/CI") || strings.Contains(out, "stdin fechado") {
		t.Errorf("mensagem de script/CI NUNCA no cancelamento:\n%s", out)
	}

	// EOF sem TTY → mensagem de script
	fakeStdin(t, "")
	_, err2 := runCmd(t, "uninstall", "--home", home)
	if err2 == nil {
		t.Fatalf("EOF sem TTY deveria falhar com a mensagem de script")
	}
	if code, ok := ExitCodeOf(err2); !ok || code != 2 {
		t.Errorf("exit = %v, want 2", err2)
	}
}
