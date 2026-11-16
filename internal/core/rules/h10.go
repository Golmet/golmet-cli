package rules

import (
	"regexp"

	"github.com/Golmet/golmet-cli/internal/core"
	"github.com/Golmet/golmet-cli/internal/core/entities"
)

// H10 — Referências cruzadas resolvem (FORMAT §9): todo RN/PEND/CR/ADR/
// CAND citado existe no workspace. IMP é namespace global do workflow
// (§4) — não resolvível no escopo do workspace, fica fora do v0.

var refRe = regexp.MustCompile(`([A-Z]{2,6})-(\d{1,4})`)

var knownNS = map[string]bool{"RN": true, "PEND": true, "CR": true, "ADR": true, "CAND": true}

var headingAnyH10 = regexp.MustCompile(`^(#{1,6})\s+`)

func H10(ctx *Context) []core.GolmetError {
	defined := map[string]bool{}
	add := func(secs []entities.Section) {
		for _, s := range secs {
			defined[s.ID] = true
		}
	}
	add(ctx.RNs)
	add(ctx.PENDs)
	add(ctx.CRs)
	add(ctx.ADRs)
	add(ctx.CANDs)

	var out []core.GolmetError
	seen := map[string]bool{}
	for _, doc := range ctx.Files {
		for i, line := range doc.Plain {
			if headingAnyH10.MatchString(line) {
				continue // linha de definição não é referência
			}
			for _, loc := range refRe.FindAllStringSubmatchIndex(line, -1) {
				ns := line[loc[2]:loc[3]]
				if !knownNS[ns] {
					continue
				}
				id := line[loc[0]:loc[1]]
				if !boundaryOK(line, loc[0], loc[1]) {
					continue // ex.: RN-001-2 (Task) não é referência a RN-001
				}
				if defined[id] {
					continue
				}
				key := id + "|" + doc.Rel + "|" + itoa(i+1)
				if seen[key] {
					continue
				}
				seen[key] = true
				out = append(out, verr("H10", ns, id, doc.Rel, i+1,
					"referência quebrada — "+id+" não existe no workspace",
					"crie a entidade citada ou corrija o ID"))
			}
		}
	}
	return out
}

// boundaryOK exige fronteira de palavra e exclui IDs de Task (RN-NNN-M).
func boundaryOK(line string, start, end int) bool {
	if start > 0 {
		if c := line[start-1]; c == '-' || isAlnumByte(c) {
			return false
		}
	}
	if end < len(line) {
		c := line[end]
		if c == '-' && end+1 < len(line) && isDigitByte(line[end+1]) {
			return false
		}
	}
	return true
}

func isAlnumByte(c byte) bool {
	return isDigitByte(c) || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isDigitByte(c byte) bool { return c >= '0' && c <= '9' }
