package rules

import (
	"github.com/Golmet/golmet-cli/internal/core"
	"github.com/Golmet/golmet-cli/internal/core/evidence"
)

// S2 — Hash de evidência ≠ HEAD do repo → worklist de refresh
// (FORMATO §5/§9). Requer git disponível e repos no charter §1.
func S2(ctx *Context) []core.GolmetError {
	if ctx.Git == nil || !ctx.Git.Available() || len(ctx.Repos) == 0 {
		return nil
	}
	var out []core.GolmetError
	seen := map[string]bool{}
	for _, f := range ctx.EvidenceAll {
		if f.Kind != evidence.KindCanonical {
			continue
		}
		n, repo := ctx.Resolver().Resolve(f.Ev.Hash)
		if n != 1 {
			continue // H7 já cobre não-resolvidos/ambíguos
		}
		head, err := ctx.Git.Head(repo)
		if err != nil {
			continue
		}
		if len(head) >= len(f.Ev.Hash) && head[:len(f.Ev.Hash)] == f.Ev.Hash {
			continue
		}
		key := f.Ev.Hash + "|" + repo
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, warn("S2", "Evidence", f.Ev.Hash, f.File, f.Line,
			"hash "+f.Ev.Hash+" ≠ HEAD do repo ("+shortHash(head)+") — evidência stale",
			"refresh da evidência no hash atual de "+repo))
	}
	return out
}
