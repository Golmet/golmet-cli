package rules

import (
	"github.com/Golmet/golmet-cli/internal/core"
	"github.com/Golmet/golmet-cli/internal/core/evidence"
)

// H7 — Toda evidência DEVE parsear; hash resolve em exatamente 1 repo
// do charter §1 (FORMAT §5/§9). Com --tolerant degrada para aviso com
// worklist; sem git disponível degrada com aviso S-level.
func H7(ctx *Context) []core.GolmetError {
	var out []core.GolmetError
	emit := func(f evidence.Finding, hash, msg, fix string) {
		sev := core.SeverityError
		if ctx.Tolerant {
			sev = core.SeverityWarning
			msg += " (worklist — --tolerant)"
		}
		out = append(out, core.GolmetError{Rule: "H7", Entity: "Evidence", ID: hash,
			File: f.File, Line: f.Line, Severity: sev, Message: msg, Fix: fix})
	}
	for _, f := range ctx.EvidenceAll {
		switch f.Kind {
		case evidence.KindMalformed:
			emit(f, "",
				"evidência não parseável: ["+f.Ev.Path+":"+itoa(f.Ev.Line)+" @"+f.Ev.Hash+"]",
				"use a forma canônica [caminho:linha @hash] — 7–40 hex minúsculos (§5)")
		case evidence.KindCanonical:
			if ctx.Git == nil || !ctx.Git.Available() {
				emit(f, f.Ev.Hash,
					"git indisponível — resolução do hash "+f.Ev.Hash+" degradada",
					"instale o git para validação de evidência")
				continue
			}
			n, _ := ctx.Resolver().Resolve(f.Ev.Hash)
			switch {
			case len(ctx.Repos) == 0:
				emit(f, f.Ev.Hash,
					"sem repos no charter §1 — hash "+f.Ev.Hash+" não resolvível",
					"registre os repos do produto no §1 do PROJECT_CHARTER.md")
			case n == 0:
				emit(f, f.Ev.Hash,
					"hash "+f.Ev.Hash+" não resolve em nenhum repo do charter §1",
					"corrija o hash ou registre o repo no charter §1")
			case n > 1:
				emit(f, f.Ev.Hash,
					"hash "+f.Ev.Hash+" resolve em mais de 1 repo (ambiguidade)",
					"use o hash completo (40 hex) do repo correto")
			}
		}
	}
	return out
}
