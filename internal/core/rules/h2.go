package rules

import (
	"strings"

	"github.com/Golmet/golmet-cli/internal/core"
	"github.com/Golmet/golmet-cli/internal/core/entities"
)

// H2 — IDs únicos e imutáveis (FORMAT §4/§9). A verificação v0 cobre
// duplicados; a detecção de renome via índice fica completa no bloco 2
// (junto à persistência do golmet.index.json).
func H2(ctx *Context) []core.GolmetError {
	var out []core.GolmetError
	dup := func(kind string, secs []entities.Section) {
		seen := map[string]entities.Section{}
		for _, s := range secs {
			if prev, ok := seen[s.ID]; ok {
				out = append(out, verr("H2", kind, s.ID, s.File, s.Line,
					"ID duplicado — definido antes em "+prev.File+":"+itoa(prev.Line),
					"IDs são únicos e imutáveis — remova ou renumere a seção nova"))
				continue
			}
			seen[s.ID] = s
		}
	}
	dup("RN", ctx.RNs)
	dup("PEND", ctx.PENDs)
	dup("CR", ctx.CRs)
	dup("ADR", ctx.ADRs)
	dup("CAND", ctx.CANDs)

	seenF := map[string]string{}
	for _, f := range ctx.Features {
		a := strings.TrimSpace(f.FMString("anchor"))
		if a == "" {
			continue
		}
		if prev, ok := seenF[a]; ok {
			out = append(out, verr("H2", "Feature", a, f.File, 1,
				"Feature duplicada para o anchor — já existe em "+prev,
				"mantenha uma Feature por RN"))
			continue
		}
		seenF[a] = f.File
	}
	for _, f := range ctx.Features {
		seen := map[string]bool{}
		for _, t := range f.Tasks {
			if seen[t.ID] {
				out = append(out, verr("H2", "Task", t.ID, t.File, t.Line,
					"Task duplicada na Feature",
					"mantenha uma seção por Task (RN-NNN-M)"))
				continue
			}
			seen[t.ID] = true
		}
	}
	return out
}
