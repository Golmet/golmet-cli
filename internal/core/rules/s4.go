package rules

import (
	"github.com/Golmet/golmet-cli/internal/core"
	"github.com/Golmet/golmet-cli/internal/core/evidence"
)

// S4 — Evidência legada sem hash (FORMATO §5/§9) — AVISO permanente
// até o refresh para a forma canônica [caminho:linha @hash].
func S4(ctx *Context) []core.GolmetError {
	var out []core.GolmetError
	for _, f := range ctx.EvidenceAll {
		if f.Kind != evidence.KindLegacy {
			continue
		}
		out = append(out, warn("S4", "Evidence", "", f.File, f.Line,
			"evidência legada sem hash: ["+f.Ev.Path+":"+itoa(f.Ev.Line)+"]",
			"refresh para [caminho:linha @hash]"))
	}
	return out
}
