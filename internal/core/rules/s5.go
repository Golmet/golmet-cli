package rules

import (
	"github.com/Golmet/golmet-cli/internal/core"
)

// S5 — Capacidade kit.jev/kit.graph declarada e ausente no ambiente
// (FORMAT §9/§10, DR-015/016). v0 nunca detecta essas capacidades —
// emite AVISO com remediação.
func S5(ctx *Context) []core.GolmetError {
	skills := readSkills(ctx.Root)
	if len(skills) == 0 {
		return nil
	}
	var out []core.GolmetError
	for _, sk := range skills {
		for _, r := range sk.requires {
			var fix string
			switch r {
			case "kit.jev":
				fix = "configure o runtime Typesafe/Jev (DR-015)"
			case "kit.graph":
				fix = "ative o grafo de código do kit (DR-016)"
			default:
				continue
			}
			out = append(out, warn("S5", "Skill", sk.name, sk.file, 1,
				"capacidade "+r+" declarada e ausente no ambiente", fix))
		}
	}
	return out
}
