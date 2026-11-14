package rules

import (
	"strings"

	"github.com/Golmet/golmet-cli/internal/core"
	"github.com/Golmet/golmet-cli/internal/core/entities"
)

// H6 — Estados no vocabulário da entidade; só transições legais
// (FORMATO §7/§9 — o índice é a história). Sem golmet.index.json o
// v0 valida vocabulário e destinos obrigatórios; com índice, valida
// também a transição a partir do estado anterior registrado.

var vocab = map[string]map[string]bool{
	"RN":      {"draft": true, "active": true, "superseded": true, "dropped": true},
	"PEND":    {"open": true, "deferred": true, "resolved": true, "dropped": true},
	"CR":      {"pending": true, "approved": true, "rejected": true, "implemented": true},
	"Feature": {"draft": true, "ready": true, "in_progress": true, "done": true, "cancelled": true},
	"Task":    {"todo": true, "in_progress": true, "done": true, "blocked": true},
	"Spec":    {"current": true, "stale": true},
	"ADR":     {"proposed": true, "accepted": true, "superseded": true},
	"CAND":    {"pending": true, "approved": true, "rejected": true, "promoted": true},
}

// defaults — via entities.Default (§6/§7 — fonte única compartilhada com
// status e o writer do índice).

// transitions — mapa de legalidade (de → para). Estados finais não têm
// saída; Task transita livre entre não-finais (§7.6).
var transitions = map[string]map[string][]string{
	"RN":      {"draft": {"active"}, "active": {"superseded", "dropped"}},
	"PEND":    {"open": {"resolved", "deferred", "dropped"}},
	"CR":      {"pending": {"approved", "rejected"}, "approved": {"implemented"}},
	"Feature": {"draft": {"ready", "cancelled"}, "ready": {"in_progress", "cancelled"}, "in_progress": {"done", "cancelled"}},
	"Task":    {"todo": {"in_progress", "done", "blocked"}, "in_progress": {"todo", "done", "blocked"}, "blocked": {"todo", "in_progress", "done"}},
	"Spec":    {"current": {"stale"}, "stale": {"current"}},
	"ADR":     {"proposed": {"accepted"}, "accepted": {"superseded"}},
	"CAND":    {"pending": {"approved", "rejected", "promoted"}, "approved": {"promoted"}},
}

// destNeed — estados que exigem destino com namespace específico (§7.2/§7.3/§7.8).
var destNeed = map[string]string{
	"RN:superseded":  "RN",
	"PEND:resolved":  "RN|CR|ADR",
	"ADR:superseded": "ADR",
}

// indexNS — namespace da entidade no golmet.index.json (§8).
var indexNS = map[string]string{
	"RN": "rn", "PEND": "pend", "CR": "cr", "ADR": "adr",
	"CAND": "cand", "Spec": "spec", "Feature": "feature",
}

func legal(kind, from, to string) bool {
	t, ok := transitions[kind]
	if !ok {
		return false
	}
	for _, to2 := range t[from] {
		if to2 == to {
			return true
		}
	}
	return false
}

func matchNS(allowed, dest string) bool {
	for _, ns := range strings.Split(allowed, "|") {
		if strings.HasPrefix(dest, ns+"-") {
			return true
		}
	}
	return false
}

// stateEntry agrupa o estado resolvido de uma entidade seccional.
type stateEntry struct {
	kind, id, file string
	line           int
	state, dest    string
	dirLine        int // 0 quando o estado veio de default (sem diretiva)
}

func H6(ctx *Context) []core.GolmetError {
	var out []core.GolmetError

	secCheck := func(kind string, entries []stateEntry) {
		for _, e := range entries {
			st := e.state
			if !vocab[kind][st] {
				out = append(out, verr("H6", kind, e.id, e.file, e.line,
					"estado \""+st+"\" fora do vocabulário de "+kind,
					"use um estado do vocabulário de "+kind+" (§7)"))
				continue
			}
			if need, ok := destNeed[kind+":"+st]; ok && e.dirLine > 0 && !matchNS(need, e.dest) {
				out = append(out, verr("H6", kind, e.id, e.file, e.dirLine,
					kind+" "+st+" exige destino ("+need+")",
					"use 'estado: "+st+" → "+strings.ReplaceAll(need, "|", "/")+"-NNN'"))
			}
			if ctx.Index != nil {
				old := ctx.Index.Estado(indexNS[kind], e.id)
				if old != "" && old != st && !legal(kind, old, st) {
					out = append(out, verr("H6", kind, e.id, e.file, e.line,
						"transição ilegal "+old+" → "+st+" (índice registra "+old+")",
						"siga as transições legais de "+kind+" (§7)"))
				}
			}
		}
	}

	entries := func(kind string, secs []entities.Section) []stateEntry {
		var es []stateEntry
		for _, s := range secs {
			e := stateEntry{kind: kind, id: s.ID, file: s.File, line: s.Line,
				state: stateOf(s, entities.Default(kind))}
			if s.State != nil {
				e.dest = s.State.Dest
				e.dirLine = s.State.Line
			}
			es = append(es, e)
		}
		return es
	}

	secCheck("RN", entries("RN", ctx.RNs))
	secCheck("PEND", entries("PEND", ctx.PENDs))
	secCheck("CR", entries("CR", ctx.CRs))
	secCheck("ADR", entries("ADR", ctx.ADRs))
	secCheck("CAND", entries("CAND", ctx.CANDs))

	// Spec — estado por arquivo (§7.7)
	for _, sp := range ctx.Specs {
		e := stateEntry{kind: "Spec", id: sp.ID, file: sp.File, line: 1, state: entities.Default("Spec")}
		if sp.Dir != nil {
			e.state = strings.ToLower(sp.Dir.State)
			e.dirLine = sp.Dir.Line
			e.line = sp.Dir.Line
		}
		secCheck("Spec", []stateEntry{e})
	}

	// Feature + Tasks
	for i := range ctx.Features {
		f := &ctx.Features[i]
		fid := entities.FeatureKey(f)
		if st := strings.ToLower(strings.TrimSpace(f.FMString("status"))); st != "" {
			if !vocab["Feature"][st] {
				out = append(out, verr("H6", "Feature", fid, f.File, 1,
					"status \""+st+"\" fora do vocabulário de Feature",
					"use draft|ready|in_progress|done|cancelled (§7.5)"))
			} else if ctx.Index != nil {
				if old := ctx.Index.Estado("feature", fid); old != "" && old != st && !legal("Feature", old, st) {
					out = append(out, verr("H6", "Feature", fid, f.File, 1,
						"transição ilegal "+old+" → "+st+" (índice registra "+old+")",
						"siga as transições sequenciais da Feature (§7.5)"))
				}
			}
		}
		for _, t := range f.Tasks {
			st := stateOf(t, entities.Default("Task"))
			if !vocab["Task"][st] {
				out = append(out, verr("H6", "Task", t.ID, t.File, t.Line,
					"estado \""+st+"\" fora do vocabulário de Task",
					"use todo|in_progress|done|blocked (§7.6)"))
				continue
			}
			if ctx.Index != nil && fid != "" {
				if old := ctx.Index.Task(fid, t.ID); old != "" && old != st && !legal("Task", old, st) {
					out = append(out, verr("H6", "Task", t.ID, t.File, t.Line,
						"transição ilegal "+old+" → "+st+" (índice registra "+old+")",
						"tasks transitam livre entre não-finais; done é final (§7.6)"))
				}
			}
		}
	}
	return out
}
