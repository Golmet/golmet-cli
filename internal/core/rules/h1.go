package rules

import (
	"strings"

	"github.com/Golmet/golmet-cli/internal/core"
)

// H1 — PEND nunca ancora Feature (FORMAT §9 / §7.3, invariante hard).
func H1(ctx *Context) []core.GolmetError {
	var out []core.GolmetError
	for i := range ctx.Features {
		f := &ctx.Features[i]
		anchor := strings.ToUpper(strings.TrimSpace(f.FMString("anchor")))
		if !strings.HasPrefix(anchor, "PEND-") {
			continue
		}
		line := 1
		if l, ok := f.KeyLines["anchor"]; ok {
			line = l
		}
		out = append(out, verr("H1", "Feature", anchor, f.File, line,
			"Feature ancorada em PEND — PEND nunca ancora Feature",
			"troque anchor para a RN que formalizou a pendência (ex.: anchor: RN-NNN)"))
	}
	return out
}
