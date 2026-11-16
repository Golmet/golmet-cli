package rules

import (
	"strings"

	"github.com/Golmet/golmet-cli/internal/core"
)

// S1 — CR implemented com Spec stale = propagação devedora
// (FORMAT §7.4/§9 — tolerada por prática, reportada como AVISO).
func S1(ctx *Context) []core.GolmetError {
	var staleFiles []string
	linked := map[string][]string{}
	for _, sp := range ctx.Specs {
		if sp.Dir == nil || strings.ToLower(sp.Dir.State) != "stale" {
			continue
		}
		staleFiles = append(staleFiles, sp.File)
		if sp.Dir.Reason != "" {
			linked[sp.Dir.Reason] = append(linked[sp.Dir.Reason], sp.File)
		}
	}
	if len(staleFiles) == 0 {
		return nil
	}
	var out []core.GolmetError
	for _, cr := range ctx.CRs {
		if stateOf(cr, "pending") != "implemented" {
			continue
		}
		msg := "CR implemented com Spec stale (propagação devedora)"
		if ls := linked[cr.ID]; len(ls) > 0 {
			msg += ": " + strings.Join(ls, ", ")
		} else {
			msg += " — " + itoa(len(staleFiles)) + " spec(s) stale no workspace"
		}
		out = append(out, warn("S1", "CR", cr.ID, cr.File, cr.Line, msg,
			"atualize as specs stale para current após a propagação"))
	}
	return out
}
