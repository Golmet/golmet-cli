package cli

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"github.com/Golmet/golmet-cli/internal/core/install"
)

// passo MCPs do wizard TTY: MCP já registrado ganha o badge dim "já
// registrado (status)" no lugar dos atributos; não-registrados mantêm attrs.
func TestMCPStepBadgeRegistered(t *testing.T) {
	m := install.NewWizardModel(t.TempDir(), ".")
	m.Step = install.StepMCPs // desenha o passo MCPs explicitamente (o default do modelo é o perfil)
	m.Registered = map[string]string{"figma": "connected", "hostinger": "needs authentication"}
	c := newTTYColors(true)
	var buf bytes.Buffer
	drawMCPStep(&rawWriter{alt: &buf, c: c}, m, c, 100)

	out := buf.String()
	if !strings.Contains(out, "já registrado (connected)") {
		t.Errorf("badge do figma ausente do render TTY:\n%s", out)
	}
	if !strings.Contains(out, "já registrado (needs authentication)") {
		t.Errorf("badge do hostinger (status textual real) ausente:\n%s", out)
	}
	// não-registrado mantém os atributos normais (remote do catálogo)
	if !strings.Contains(out, "remoto") {
		t.Errorf("atributos dos não-registrados sumiram:\n%s", out)
	}
}

// badge também no modo plano (NO_COLOR): texto sem ANSI.
func TestMCPStepBadgePlain(t *testing.T) {
	m := install.NewWizardModel(t.TempDir(), ".")
	m.Step = install.StepMCPs
	m.Registered = map[string]string{"figma": "connected"}
	c := newTTYColors(false)
	var buf bytes.Buffer
	drawMCPStep(&rawWriter{alt: &buf, c: c}, m, c, 100)

	out := buf.String()
	if strings.Contains(out, "\x1b") {
		t.Errorf("ANSI vazou no render plano")
	}
	if !strings.Contains(out, "já registrado (connected)") {
		t.Errorf("badge plano ausente:\n%s", out)
	}
}

// pipe (modo numerado): badge na listagem — o usuário VÊ o que já existe
// antes de digitar; default continua golmet-only (registrados são opt-in).
func TestPickMCPsBadgePipe(t *testing.T) {
	var buf bytes.Buffer
	w := &installWizard{out: &buf, in: bufio.NewReader(strings.NewReader("1\n"))}
	mcps, err := w.pickMCPs(map[string]string{"figma": "registered", "hostinger": "needs authentication"})
	if err != nil {
		t.Fatalf("pickMCPs: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "já registrado (registered)") {
		t.Errorf("badge do pipe ausente:\n%s", out)
	}
	if !strings.Contains(out, "já registrado (needs authentication)") {
		t.Errorf("badge com status textual ausente:\n%s", out)
	}
	if len(mcps) != 1 || !strings.HasPrefix(mcps[0], "golmet") {
		t.Errorf("default deveria ser golmet-only (registrados são opt-in): %v", mcps)
	}
}

// list indisponível (installed nil) → lista SEM badges, nada quebra.
func TestPickMCPsNoBadgeWhenUnavailable(t *testing.T) {
	var buf bytes.Buffer
	w := &installWizard{out: &buf, in: bufio.NewReader(strings.NewReader("1\n"))}
	mcps, err := w.pickMCPs(nil)
	if err != nil {
		t.Fatalf("pickMCPs sem detecção: %v", err)
	}
	if strings.Contains(buf.String(), "já registrado") {
		t.Errorf("badge presente com detecção indisponível:\n%s", buf.String())
	}
	if len(mcps) != 1 {
		t.Errorf("seleção default = %v, want golmet only", mcps)
	}
}
