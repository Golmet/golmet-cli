package rules

import (
	"path/filepath"
	"strings"

	"github.com/Golmet/golmet-cli/internal/core"
	"github.com/Golmet/golmet-cli/internal/core/entities"
)

// H5 — Front matter obrigatório presente e parseável (FORMATO §9):
// Charter e Feature. golmet.yaml é tratado no scan.Run (gate de config).

var charterStates = map[string]bool{"draft": true, "confirmed": true}

var featureStates = map[string]bool{
	"draft": true, "ready": true, "in_progress": true, "done": true, "cancelled": true,
}

func H5(ctx *Context) []core.GolmetError {
	var out []core.GolmetError
	if ctx.Charter == nil {
		out = append(out, verr("H5", "Charter", "", "PROJECT_CHARTER.md", 1,
			"PROJECT_CHARTER.md ausente",
			"crie o charter com front matter status: draft|confirmed"))
	} else {
		c := ctx.Charter
		switch {
		case c.FMError != nil:
			out = append(out, verr("H5", "Charter", "", c.File, 1,
				"front matter ilegível: "+c.FMError.Error(),
				"corrija o front matter conforme o subset §6.1"))
		case !c.HasFM:
			out = append(out, verr("H5", "Charter", "", c.File, 1,
				"front matter ausente (obrigatório)",
				"adicione '---\\nstatus: draft\\n---' no topo do arquivo"))
		default:
			st := strings.ToLower(c.FM["status"].String())
			if !charterStates[st] {
				out = append(out, verr("H5", "Charter", "", c.File, 1,
					"status do charter inválido: \""+st+"\" (draft|confirmed)",
					"use status: draft ou status: confirmed"))
			}
		}
	}
	for i := range ctx.Features {
		f := &ctx.Features[i]
		id := featureID(f)
		switch {
		case f.FMError != nil:
			out = append(out, verr("H5", "Feature", id, f.File, 1,
				"front matter ilegível: "+f.FMError.Error(),
				"corrija o front matter conforme o subset §6.1"))
		case !f.HasFM:
			out = append(out, verr("H5", "Feature", id, f.File, 1,
				"front matter ausente (status e anchor são obrigatórios)",
				"adicione '---\\nstatus: draft\\nanchor: RN-NNN\\n---' no topo do arquivo"))
		default:
			st := strings.ToLower(f.FM["status"].String())
			if !featureStates[st] {
				out = append(out, verr("H5", "Feature", id, f.File, 1,
					"status inválido: \""+st+"\" (draft|ready|in_progress|done|cancelled)",
					"use um status do vocabulário da Feature (§7.5)"))
			}
			anchor := strings.TrimSpace(f.FM["anchor"].String())
			switch {
			case anchor == "":
				out = append(out, verr("H5", "Feature", id, f.File, 1,
					"anchor ausente (obrigatório)",
					"adicione anchor: RN-NNN ao front matter"))
			case !strings.HasPrefix(anchor, "RN-") && !strings.HasPrefix(strings.ToUpper(anchor), "PEND-"):
				out = append(out, verr("H5", "Feature", id, f.File, 1,
					"anchor malformado: \""+anchor+"\"",
					"use anchor: RN-NNN"))
			}
		}
	}
	return out
}

func featureID(f *entities.Feature) string {
	if a := strings.TrimSpace(f.FMString("anchor")); a != "" {
		return a
	}
	return filepath.Base(f.File)
}
