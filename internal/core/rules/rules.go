// Package rules implementa o enforcement do FORMATO v1 §9 — uma regra =
// um arquivo/função com nome mapeado ao FORMATO (H1..H10 hard, S1..S5 soft).
package rules

import (
	"strconv"

	"github.com/Golmet/golmet-cli/internal/adapters/fsutil"
	"github.com/Golmet/golmet-cli/internal/adapters/git"
	"github.com/Golmet/golmet-cli/internal/core"
	"github.com/Golmet/golmet-cli/internal/core/entities"
	"github.com/Golmet/golmet-cli/internal/core/evidence"
	"github.com/Golmet/golmet-cli/internal/core/indexstate"
)

// Context agrega o inventário do workspace para as regras.
type Context struct {
	Root        string
	Files       []fsutil.FileDoc
	Charter     *entities.Charter
	CharterDoc  *fsutil.FileDoc
	RNs         []entities.Section
	PENDs       []entities.Section
	CRs         []entities.Section
	ADRs        []entities.Section
	CANDs       []entities.Section
	Specs       []entities.Spec
	Features    []entities.Feature
	EvidenceAll []evidence.Finding
	Repos       []entities.RepoRef
	Git         *git.Adapter
	Index       *indexstate.Index
	Tolerant    bool

	resolver *hashResolver
}

// Resolver devolve o cache de resolução de hashes (H7/S2).
func (c *Context) Resolver() *hashResolver {
	if c.resolver == nil {
		c.resolver = newHashResolver(c.Git, c.Repos)
	}
	return c.resolver
}

// Rule é uma regra §9 com nome mapeado ao FORMATO.
type Rule struct {
	ID   string
	Name string
	Fn   func(*Context) []core.GolmetError
}

// All devolve as regras na ordem normativa H1..H10, S1..S5.
func All() []Rule {
	return []Rule{
		{"H1", "PEND nunca ancora Feature", H1},
		{"H2", "IDs únicos e imutáveis", H2},
		{"H3", "Feature ready/done exige anchor RN active", H3},
		{"H4", "RN active exige ≥1 evidência parseável", H4},
		{"H5", "Front matter obrigatório presente e parseável", H5},
		{"H6", "Estados no vocabulário; só transições legais", H6},
		{"H7", "Evidência parseável; hash em exatamente 1 repo", H7},
		{"H8", "ADR aceito é imutável (hash de seção)", H8},
		{"H9", "Spec stale exige reason: CR-XXX", H9},
		{"H10", "Referências cruzadas resolvem", H10},
		{"S1", "CR implemented com Spec stale", S1},
		{"S2", "Hash de evidência ≠ HEAD (worklist)", S2},
		{"S3", "requires de skill não satisfeitas pelo engine", S3},
		{"S4", "Evidência legada sem hash", S4},
		{"S5", "Capacidade kit.* declarada e ausente", S5},
	}
}

// ---- helpers compartilhados ----

func verr(rule, entity, id, file string, line int, msg, fix string) core.GolmetError {
	return core.GolmetError{Rule: rule, Entity: entity, ID: id, File: file, Line: line,
		Severity: core.SeverityError, Message: msg, Fix: fix}
}

func warn(rule, entity, id, file string, line int, msg, fix string) core.GolmetError {
	return core.GolmetError{Rule: rule, Entity: entity, ID: id, File: file, Line: line,
		Severity: core.SeverityWarning, Message: msg, Fix: fix}
}

// stateOf devolve o estado corrente da seção — delega para
// entities.StateOf (fonte única compartilhada com status/índice).
func stateOf(s entities.Section, def string) string {
	return entities.StateOf(s, def)
}

func itoa(n int) string { return strconv.Itoa(n) }

func shortHash(h string) string {
	if len(h) > 7 {
		return h[:7]
	}
	return h
}
