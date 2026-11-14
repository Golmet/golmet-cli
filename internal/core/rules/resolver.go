package rules

import (
	"github.com/Golmet/golmet-cli/internal/adapters/git"
	"github.com/Golmet/golmet-cli/internal/core/entities"
)

// hashResolver resolve "o hash existe em quais repos do charter §1"
// com cache por hash (H7: ambiguidade = erro; S2: worklist de refresh).
type hashResolver struct {
	g     *git.Adapter
	repos []entities.RepoRef
	cache map[string]hashRes
}

type hashRes struct {
	n    int
	repo string // Abs do repo quando n==1
}

func newHashResolver(g *git.Adapter, repos []entities.RepoRef) *hashResolver {
	return &hashResolver{g: g, repos: repos, cache: map[string]hashRes{}}
}

// Resolve devolve (nº de repos que contêm o hash, repo único se n==1).
func (r *hashResolver) Resolve(hash string) (int, string) {
	if r == nil || r.g == nil || !r.g.Available() {
		return 0, ""
	}
	if res, ok := r.cache[hash]; ok {
		return res.n, res.repo
	}
	seen := map[string]bool{}
	n := 0
	match := ""
	for _, rp := range r.repos {
		if seen[rp.Abs] {
			continue
		}
		seen[rp.Abs] = true
		if r.g.HasObject(rp.Abs, hash) {
			n++
			match = rp.Abs
		}
	}
	r.cache[hash] = hashRes{n: n, repo: match}
	return n, match
}
