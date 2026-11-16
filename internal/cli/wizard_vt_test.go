package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fakeStdin substitui os.Stdin por arquivo com o input dado (o wizard
// numerado lê de os.Stdin direto; EOF dispararia os.Exit(2) no guard).
func fakeStdin(t *testing.T, input string) {
	t.Helper()
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "stdin.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(input); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	old := os.Stdin
	os.Stdin = f
	t.Cleanup(func() {
		os.Stdin = old
		f.Close()
	})
}

// mocks de terminal para o dispatch: TTY simulado + VT controlado +
// codepage NEVER toca o console real do agente (grava a sequência p/ assert).
func fakeTerminal(t *testing.T, tty, vt bool) *[]string {
	t.Helper()
	oldTTY, oldVT, oldSet, oldRestore := isTTY, enableVT, setOutputCodepage, restoreCodepage
	isTTY = func() bool { return tty }
	enableVT = func() bool { return vt }
	var calls []string
	setOutputCodepage = func() uint32 { calls = append(calls, "set"); return 437 } // codepage legado fake
	restoreCodepage = func(old uint32) { calls = append(calls, "restore") }
	t.Cleanup(func() {
		isTTY, enableVT, setOutputCodepage, restoreCodepage = oldTTY, oldVT, oldSet, oldRestore
	})
	return &calls
}

// decisão pura: tabela completa do dispatch interativo.
func TestTTYWizardMode(t *testing.T) {
	cases := []struct {
		tty, plain, vt bool
		want           string
	}{
		{true, false, true, "tty"},         // Windows Terminal: wizard completo
		{true, false, false, "plain-auto"}, // conhost legado: degradação elegante
		{true, true, true, "plain"},        // --plain força numerado
		{true, true, false, "plain"},       // --plain + VT ausente
		{false, false, true, "plain"},      // pipe/CI: guard non-TTY
		{false, false, false, "plain"},     // pipe/CI
	}
	for _, tc := range cases {
		if got := ttyWizardMode(tc.tty, tc.plain, tc.vt); got != tc.want {
			t.Errorf("ttyWizardMode(tty=%v,plain=%v,vt=%v) = %q, want %q", tc.tty, tc.plain, tc.vt, got, tc.want)
		}
	}
}

// VT indisponível num terminal TTY → linha ℹ️ informando + modo numerado
// automático (o fluxo inteiro até o cancelamento — nada escrito). Ordem
// numerada: perfil (1) → engine (2) → MCPs (3) → confirmação.
func TestInstallDispatchVTFallbackPlainAuto(t *testing.T) {
	virtualHome(t)
	fakeStdin(t, "1\n1\n1\nn\n") // perfil 1 (essencial default), engine 1, mcp 1 (golmet), confirmação n
	fakeTerminal(t, true, false)

	out, err := runCmd(t, "install")
	if err != nil {
		t.Fatalf("install interativo com VT ausente: %v\n%s", err, out)
	}
	if !strings.Contains(out, "terminal sem suporte a VT") {
		t.Errorf("linha ℹ️ de fallback ausente:\n%s", out)
	}
	idxPerfil := strings.Index(out, "Escolha o perfil:")
	idxEngine := strings.Index(out, "Escolha o engine:")
	if idxPerfil < 0 || idxEngine < 0 || idxPerfil > idxEngine {
		t.Errorf("perfil deveria vir antes do engine no modo numerado (perfil=%d engine=%d):\n%s", idxPerfil, idxEngine, out)
	}
	if !strings.Contains(out, "cancelado — nada foi aplicado") {
		t.Errorf("cancelamento do modo numerado ausente:\n%s", out)
	}
	if strings.Contains(out, "◆") && strings.Contains(out, "Plano") {
		// heurística: o wizard de teclas renderiza o plano ◆ — não deveria
		t.Errorf("wizard TTY rodou apesar de VT indisponível:\n%s", out)
	}
}

// VT disponível num terminal TTY → caminho TTY tomado (o MakeRaw falha em
// stdin não-console do teste — a falha PROVA que o wizard de teclas foi
// escolhido; codepage set→restore na ordem certa).
func TestInstallDispatchVTOkTakesTTYPath(t *testing.T) {
	virtualHome(t)
	fakeStdin(t, "")
	calls := fakeTerminal(t, true, true)

	out, err := runCmd(t, "install")
	if err == nil || !strings.Contains(out, "raw mode") {
		t.Fatalf("caminho TTY não tomado (MakeRaw deveria falhar em stdin não-console): %v\n%s", err, out)
	}
	if len(*calls) != 2 || (*calls)[0] != "set" || (*calls)[1] != "restore" {
		t.Errorf("codepage deveria set→restore no exit do wizard: %v", *calls)
	}
}

// enableVT default REAL (não mockado): stdout do go test é pipe → sem
// console mode → false no Windows; POSIX → sempre true.
func TestEnableVTDefaultOnPipe(t *testing.T) {
	got := enableVTWin()
	if runtime.GOOS == "windows" && got {
		t.Errorf("enableVTWin deveria falhar em stdout não-console (pipe)")
	}
	if runtime.GOOS != "windows" && !got {
		t.Errorf("enableVTWin no POSIX deveria ser true (terminal já é VT)")
	}
}
