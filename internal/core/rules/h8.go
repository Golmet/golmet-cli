package rules

import (
	"github.com/Golmet/golmet-cli/internal/core"
	"github.com/Golmet/golmet-cli/internal/core/indexstate"
)

// H8 — ADR aceito é imutável: hash de seção ≠ índice = erro (§7.8/§9).
// Requer golmet.index.json com secao_hash gravada (sem índice, o v0 pula).
// Hash = SHA-256 das linhas originais da seção (RawLines) — normalização
// canônica em indexstate.SectionHash, a MESMA que o writer do índice usa.
func H8(ctx *Context) []core.GolmetError {
	if ctx.Index == nil {
		return nil
	}
	var out []core.GolmetError
	for _, s := range ctx.ADRs {
		st := stateOf(s, "proposed")
		if st != "accepted" && st != "superseded" {
			continue
		}
		stored := ctx.Index.SecaoHashOf("adr", s.ID)
		if stored == "" {
			continue
		}
		cur := indexstate.SectionHash(s.RawLines)
		if cur != stored {
			out = append(out, verr("H8", "ADR", s.ID, s.File, s.Line,
				"conteúdo de ADR aceito foi modificado (hash de seção diverge do índice)",
				"restaure o conteúdo original ou registre um ADR novo — append-only (§7.8)"))
		}
	}
	return out
}
