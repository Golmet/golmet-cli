package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/Golmet/golmet-cli/internal/core"
	"github.com/Golmet/golmet-cli/internal/core/scan"
)

func TestExitCodeOf(t *testing.T) {
	if c, ok := ExitCodeOf(exitError(1)); !ok || c != 1 {
		t.Errorf("ExitCodeOf(exitError(1)) = %d, %v", c, ok)
	}
	if _, ok := ExitCodeOf(errors.New("qualquer")); ok {
		t.Errorf("erro genérico não deveria ter exit code")
	}
}

func TestPrintHuman(t *testing.T) {
	res := &scan.Result{
		Workspace: "ws",
		Files:     3,
		Counts:    map[string]int{"rn": 2},
		Errors: []core.GolmetError{{
			File: "REQUIREMENTS.md", Line: 12, Entity: "RN", ID: "RN-042",
			Rule: "H4", Severity: core.SeverityError,
			Message: "RN active sem evidência parseável",
			Fix:     "adicione [caminho:linha @hash]",
		}},
		Warnings: []core.GolmetError{{
			File: "REQUIREMENTS.md", Line: 5, Entity: "Evidence",
			Rule: "S4", Severity: core.SeverityWarning,
			Message: "evidência legada sem hash: [a:1]", Fix: "refresh",
		}},
	}
	var buf bytes.Buffer
	printHuman(&buf, res)
	out := buf.String()
	for _, want := range []string{
		"[REQUIREMENTS.md:12] RN RN-042 [H4] RN active sem evidência parseável",
		"  fix: adicione [caminho:linha @hash]",
		"AVISO [REQUIREMENTS.md:5] Evidence [S4]",
		"1 erro(s), 1 aviso(s)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("saída não contém %q:\n%s", want, out)
		}
	}
}

func TestPrintHumanOk(t *testing.T) {
	var buf bytes.Buffer
	printHuman(&buf, &scan.Result{Counts: map[string]int{}})
	if !strings.Contains(buf.String(), "ok — workspace conforme") {
		t.Errorf("saída ok ausente:\n%s", buf.String())
	}
}
