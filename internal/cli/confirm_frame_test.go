package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---- M28 — o confirm-frame isolado com ENTER como default ----

// O render: o frame ◆ vem ISOLADO embaixo do resumo (linha em branco
// entre eles — nunca colado) com o hint entre parênteses.
func TestConfirmStepFrameIsolatedFromSummary(t *testing.T) {
	cwd := t.TempDir()
	isolatedHome(t)
	w, out := newTestWizard(cwd, "\n") // ENTER confirma
	confirmed, err := w.confirmStep("  tipo:      Organização (container: true)\n  raiz:      X", false)
	if err != nil || !confirmed {
		t.Fatalf("ENTER deveria confirmar: %v/%v\n%s", confirmed, err, out)
	}
	s := out.String()
	for _, want := range []string{
		"  raiz:      X",
		"\n\n◆ Confirmar e criar?", // linha em branco ENTRE resumo e frame
		"(ENTER confirma · n volta a editar · esc cancela)",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("render do confirm sem %q:\n%s", want, s)
		}
	}
	// colado NUNCA: o frame não herda linha do resumo
	if strings.Contains(s, "X\n◆ Confirmar") {
		t.Errorf("frame colado ao resumo:\n%s", s)
	}
}

// ENTER vazio = confirma e cria (o default — a ação principal sem letra).
func TestConfirmStepEnterCreates(t *testing.T) {
	base := inTempDir(t)
	isolatedHome(t)
	w, out := newTestWizard(base, "1\n\n\n") // menu enter → nome vazio (= cwd) → confirm ENTER
	opt, _, _, err := runInitWizard(w, false, false)
	if err != nil {
		t.Fatalf("ENTER deveria confirmar: %v\n%s", err, out)
	}
	if opt == nil || !opt.Org {
		t.Errorf("ENTER deveria criar a Organização: %+v", opt)
	}
}

// s = confirma (alias do ENTER, compat com quem digita s).
func TestConfirmStepSAlias(t *testing.T) {
	cwd := t.TempDir()
	isolatedHome(t)
	w, out := newTestWizard(cwd, "s\n")
	if confirmed, err := w.confirmStep("resumo", false); err != nil || !confirmed {
		t.Fatalf("s (alias) deveria confirmar: %v/%v\n%s", confirmed, err, out)
	}
}

// n = volta a editar (M27 inalterado); ← idem (raw — coberto no editor).
func TestConfirmStepNGoesBack(t *testing.T) {
	cwd := t.TempDir()
	isolatedHome(t)
	w, out := newTestWizard(cwd, "n\n")
	confirmed, err := w.confirmStep("resumo", false)
	if err != nil || confirmed {
		t.Fatalf("n deveria voltar a editar: %v/%v\n%s", confirmed, err, out)
	}
}

// esc = cancela o wizard (o sentinel — nada é criado).
func TestConfirmStepEscCancels(t *testing.T) {
	cwd := t.TempDir()
	isolatedHome(t)
	w, out := newTestWizard(cwd, "esc\n")
	_, err := w.confirmStep("resumo", false)
	if !errors.Is(err, errWizardCancelled) {
		t.Fatalf("esc deveria cancelar, got %v\n%s", err, out)
	}
	if _, serr := os.Stat(filepath.Join(cwd, "golmet.yaml")); serr == nil {
		t.Errorf("cancelado não deveria criar nada")
	}
}

// Texto aleatório = re-pergunta (o padrão vigente).
func TestConfirmStepRandomReasks(t *testing.T) {
	cwd := t.TempDir()
	isolatedHome(t)
	// "x" → re-pergunta → ENTER confirma
	w, out := newTestWizard(cwd, "x\n\n")
	confirmed, err := w.confirmStep("resumo", false)
	if err != nil || !confirmed {
		t.Fatalf("após o re-prompt o ENTER deveria confirmar: %v/%v\n%s", confirmed, err, out)
	}
	if !strings.Contains(out.String(), "ENTER confirma · n volta a editar · esc cancela") {
		t.Errorf("re-pergunta sem o hint:\n%s", out)
	}
	if strings.Count(out.String(), "◆ Confirmar e criar?") < 2 {
		t.Errorf("o frame deveria re-renderizar no re-prompt:\n%s", out)
	}
}
