// gate do gm_lint — read-only (sem confirm), sandbox kit com ref morta
// vira finding error (ok=false); workspace inexistente vira tool-error.
package mcpserve

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGmLintViaMCP(t *testing.T) {
	cs := session(t, context.Background())

	ws := t.TempDir()
	os.MkdirAll(filepath.Join(ws, "coredata", "sources"), 0o755)
	os.WriteFile(filepath.Join(ws, "coredata", "registry.yaml"),
		[]byte("version: pilot-1\nitems:\n  - id: gm-a\n    kind: skill\n    body: sources/gm-a.md\n"), 0o644)
	os.WriteFile(filepath.Join(ws, "coredata", "sources", "gm-a.md"),
		[]byte("- `sub/nada-disso.md` — ref morta (com localização)\n"), 0o644)

	_, out := callTool(t, cs, "gm_lint", map[string]any{"workspace": ws, "home": t.TempDir()})
	var rep struct {
		OK       bool `json:"ok"`
		Errors   int  `json:"errors"`
		Warnings int  `json:"warnings"`
		KitMode  bool `json:"kit_mode"`
		Findings []struct {
			Check    string `json:"check"`
			Severity string `json:"severity"`
		} `json:"findings"`
	}
	if err := extractJSON(out, &rep); err != nil {
		t.Fatalf("payload gm_lint ilegível: %v — %s", err, out)
	}
	if rep.OK || !rep.KitMode || rep.Errors < 1 {
		t.Fatalf("gm_lint deveria flagar a ref morta (ok=false): %+v — %s", rep, out)
	}
	found := false
	for _, f := range rep.Findings {
		if f.Check == "L1" && f.Severity == "error" {
			found = true
		}
	}
	if !found {
		t.Errorf("finding L1/error ausente: %+v", rep.Findings)
	}
	if !strings.Contains(out, `"errors":`) {
		t.Errorf("contagens ausentes do payload: %s", out)
	}

	// workspace inexistente → tool-error
	res, _ := callTool(t, cs, "gm_lint", map[string]any{"workspace": filepath.Join(t.TempDir(), "nada")})
	if !res.IsError {
		t.Errorf("workspace inválido deveria virar tool-error")
	}
}
