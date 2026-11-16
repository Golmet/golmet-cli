package rules

import (
	"strings"

	"github.com/Golmet/golmet-cli/internal/core"
)

// H9 — Spec stale DEVE carregar reason: CR-XXX (FORMAT §7.7/§9).
func H9(ctx *Context) []core.GolmetError {
	var out []core.GolmetError
	for _, sp := range ctx.Specs {
		if sp.Dir == nil || strings.ToLower(sp.Dir.State) != "stale" {
			continue
		}
		if sp.Dir.Reason == "" || !strings.HasPrefix(sp.Dir.Reason, "CR-") {
			out = append(out, verr("H9", "Spec", sp.ID, sp.File, sp.Dir.Line,
				"Spec stale sem reason: CR-XXX",
				"escreva '> estado: stale · reason: CR-XXX' citando o CR que invalidou"))
		}
	}
	return out
}
