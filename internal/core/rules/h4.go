package rules

import (
	"github.com/Golmet/golmet-cli/internal/core"
	"github.com/Golmet/golmet-cli/internal/core/evidence"
)

// H4 — RN active exige ≥1 evidência parseável na seção (FORMATO §7.2/§9).
// Com --tolerant degrada para aviso com worklist.
func H4(ctx *Context) []core.GolmetError {
	var out []core.GolmetError
	for _, s := range ctx.RNs {
		if stateOf(s, "draft") != "active" {
			continue
		}
		has := false
		for _, f := range evidence.Scan(s.File, s.Lines) {
			if f.Kind == evidence.KindCanonical {
				has = true
				break
			}
		}
		if has {
			continue
		}
		sev := core.SeverityError
		suffix := ""
		if ctx.Tolerant {
			sev = core.SeverityWarning
			suffix = " (worklist — --tolerant)"
		}
		out = append(out, core.GolmetError{Rule: "H4", Entity: "RN", ID: s.ID,
			File: s.File, Line: s.Line, Severity: sev,
			Message: "RN active sem evidência parseável" + suffix,
			Fix:     "adicione [caminho:linha @hash] ao corpo da seção"})
	}
	return out
}
