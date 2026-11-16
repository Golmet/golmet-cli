package rules

import (
	"strings"

	"github.com/Golmet/golmet-cli/internal/core"
	"github.com/Golmet/golmet-cli/internal/core/entities"
)

// H3 — Feature ready/done exige anchor RN active (FORMAT §7.5/§9);
// done exige, além disso, todas as Tasks done.
func H3(ctx *Context) []core.GolmetError {
	var out []core.GolmetError
	rnState := map[string]string{}
	for _, s := range ctx.RNs {
		rnState[s.ID] = stateOf(s, "draft")
	}
	for i := range ctx.Features {
		f := &ctx.Features[i]
		status := strings.ToLower(strings.TrimSpace(f.FMString("status")))
		if status != "ready" && status != "done" {
			continue
		}
		anchor := strings.TrimSpace(f.FMString("anchor"))
		if anchor == "" || !strings.HasPrefix(anchor, "RN-") {
			continue // H1/H5 cobrem anchor ausente/PEND/malformado
		}
		st, ok := rnState[anchor]
		if !ok {
			out = append(out, verr("H3", "Feature", anchor, f.File, 1,
				"anchor "+anchor+" não existe em REQUIREMENTS.md",
				"crie a RN ou corrija o anchor"))
			continue
		}
		if st != "active" {
			out = append(out, verr("H3", "Feature", anchor, f.File, 1,
				"Feature "+status+" exige anchor RN active — "+anchor+" está "+st,
				"mova "+anchor+" para active antes de "+status))
		}
		if status == "done" {
			for _, t := range f.Tasks {
				if ts := stateOf(t, entities.Default("Task")); ts != "done" {
					out = append(out, verr("H3", "Task", t.ID, t.File, t.Line,
						"Feature done exige Task done — "+t.ID+" está "+ts,
						"conclua a Task antes de marcar a Feature como done"))
				}
			}
		}
	}
	return out
}
