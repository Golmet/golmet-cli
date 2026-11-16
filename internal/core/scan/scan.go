// Package scan é o motor do validate: dado um workspace → inventário
// de entidades (FORMAT §7) + aplicação das regras H1–H10/S1–S5 (§9).
package scan

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Golmet/golmet-cli/internal/adapters/fsutil"
	"github.com/Golmet/golmet-cli/internal/adapters/git"
	"github.com/Golmet/golmet-cli/internal/core"
	"github.com/Golmet/golmet-cli/internal/core/entities"
	"github.com/Golmet/golmet-cli/internal/core/evidence"
	"github.com/Golmet/golmet-cli/internal/core/frontmatter"
	"github.com/Golmet/golmet-cli/internal/core/indexstate"
	"github.com/Golmet/golmet-cli/internal/core/rules"
)

// Options controla a execução do scan.
type Options struct {
	Tolerant   bool // H4/H7 degradam para aviso com worklist
	WriteIndex bool // kit escreve golmet.index.json (semântica D2 §8)
	ForceIndex bool // --fix: recompõe o índice mesmo com erros hard (pós-merge)
}

// Contextos do validate (§9 — M21): a detecção é pelo golmet.yaml do dir
// dado. Container valida a DECLARAÇÃO da org (container.go); workspace-
// produto roda a gramática integral. Sem golmet.yaml = erro claro (exit 2).
const (
	ContextContainer = "container"
	ContextWorkspace = "workspace-produto"
)

// Inventory é o retrato parseado da prosa — insumo dos comandos de
// consulta (status) sem reexecutar o parse em paralelo ao motor.
type Inventory struct {
	Charter  *entities.Charter
	RNs      []entities.Section
	PENDs    []entities.Section
	CRs      []entities.Section
	ADRs     []entities.Section
	CANDs    []entities.Section
	Specs    []entities.Spec
	Features []entities.Feature
}

// Result agrega os findings do validate.
type Result struct {
	Workspace string
	Context   string // "container" | "workspace-produto" (§9)
	Files     int
	Counts    map[string]int
	Errors    []core.GolmetError
	Warnings  []core.GolmetError
	// Container só em contexto container — o retrato da declaração da org.
	Container    *ContainerReport
	Inventory    *Inventory
	Index        *indexstate.Index // índice recomputado da prosa (§8)
	IndexWritten bool              // índice foi (re)escrito nesta execução
}

// Run executa o scan completo. Erro de retorno = uso/config (exit 2);
// violações hard saem em Result.Errors (exit 1 no CLI).
func Run(dir string, opt Options) (*Result, error) {
	root, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("workspace inválido: %w", err)
	}
	if st, serr := os.Stat(root); serr != nil || !st.IsDir() {
		return nil, fmt.Errorf("workspace não encontrado: %s", root)
	}

	// golmet.yaml (§3) — presente e com format: 1; a ausência NÃO é
	// "workspace não inicializado" genérico: o dir pode ser um produto
	// dentro de um container — mensagem com os dois caminhos de init.
	cfgPath := filepath.Join(root, "golmet.yaml")
	cfgData, cfgErr := os.ReadFile(cfgPath)
	if cfgErr != nil {
		return nil, fmt.Errorf("golmet.yaml ausente em %s — não é workspace nem container (rode 'golmet init' no dir, ou 'golmet init <produto> --into <container>' para criar dentro de um container existente)", root)
	}
	var pre []core.GolmetError
	cfgFields, perr := frontmatter.ParseBlock(string(cfgData))
	if perr != nil {
		pre = append(pre, core.GolmetError{Rule: "H5", Entity: "Workspace", File: "golmet.yaml",
			Line: 1, Severity: core.SeverityError,
			Message: "golmet.yaml ilegível: " + perr.Error(),
			Fix:     "corrija o golmet.yaml conforme o subset §6.1"})
	} else {
		fv, ok := cfgFields["format"]
		if !ok || fv.Str == "" {
			pre = append(pre, core.GolmetError{Rule: "H5", Entity: "Workspace", File: "golmet.yaml",
				Line: 1, Severity: core.SeverityError,
				Message: "golmet.yaml sem format (DEVE ser 1)",
				Fix:     "adicione 'format: 1' ao golmet.yaml"})
		} else if fv.Str != "1" {
			return nil, fmt.Errorf("formato %s não suportado — este kit suporta format: 1 (§12)", fv.Str)
		}
	}

	// M21 — contexto container (§2.3): valida a DECLARAÇÃO da org e
	// encerra — a gramática de workspace-produto não se aplica (o container
	// não roda o pipeline de RNs; produtos não-adotados são estado legítimo).
	if perr == nil && cfgFields["container"].Str == "true" {
		crep, findings := validateContainer(root, cfgFields)
		return &Result{
			Workspace: root, Context: ContextContainer,
			Container: crep, Counts: map[string]int{}, Errors: findings,
		}, nil
	}

	files, err := fsutil.Walk(root)
	if err != nil {
		return nil, fmt.Errorf("falha ao caminhar o workspace: %w", err)
	}
	byRel := map[string]*fsutil.FileDoc{}
	for i := range files {
		byRel[files[i].Rel] = &files[i]
	}

	ctx := &rules.Context{Root: root, Files: files, Tolerant: opt.Tolerant}

	ctx.CharterDoc = byRel["PROJECT_CHARTER.md"]
	if ctx.CharterDoc != nil {
		ctx.Charter = entities.ParseCharter(ctx.CharterDoc, root)
	}

	if req := byRel["REQUIREMENTS.md"]; req != nil {
		ctx.RNs = entities.ParseSections(req.Rel, req.Plain, req.Lines,
			func(id string, lvl int) bool {
				return lvl == 2 && strings.HasPrefix(id, "RN-") && !entities.IsTaskID(id)
			})
		ctx.PENDs = entities.ParseSections(req.Rel, req.Plain, req.Lines,
			func(id string, lvl int) bool { return lvl == 2 && strings.HasPrefix(id, "PEND-") })
	}
	if cr := byRel["CHANGE_REQUESTS.md"]; cr != nil {
		ctx.CRs = entities.ParseSections(cr.Rel, cr.Plain, cr.Lines,
			func(id string, lvl int) bool { return lvl == 2 && strings.HasPrefix(id, "CR-") })
	}
	if dec := byRel["DECISIONS.md"]; dec != nil {
		ctx.ADRs = entities.ParseSections(dec.Rel, dec.Plain, dec.Lines,
			func(id string, lvl int) bool { return lvl == 2 && strings.HasPrefix(id, "ADR-") })
	}
	if cand := byRel["reverse/CANDIDATES.md"]; cand != nil {
		ctx.CANDs = entities.ParseSections(cand.Rel, cand.Plain, cand.Lines,
			func(id string, lvl int) bool { return lvl == 3 && strings.HasPrefix(id, "CAND-") })
	}
	for i := range files {
		d := &files[i]
		if entities.IsFeatureFile(d.Rel) {
			ctx.Features = append(ctx.Features, *entities.ParseFeature(d))
		}
		if strings.HasPrefix(d.Rel, "analysis/") {
			// Spec só quando a pasta herda RN (README/ajenos não são entidade)
			if sp := entities.ParseSpec(d); sp.ID != "" {
				ctx.Specs = append(ctx.Specs, *sp)
			}
		}
	}
	for i := range files {
		ctx.EvidenceAll = append(ctx.EvidenceAll, evidence.Scan(files[i].Rel, files[i].Plain)...)
	}
	if ctx.Charter != nil {
		ctx.Repos = ctx.Charter.Repos
	}
	ctx.Git = git.New()

	if idxData, ierr := os.ReadFile(filepath.Join(root, "golmet.index.json")); ierr == nil {
		idx, iperr := indexstate.Parse(idxData)
		if iperr != nil {
			pre = append(pre, core.GolmetError{Rule: "H6", Entity: "Index", File: "golmet.index.json",
				Severity: core.SeverityWarning,
				Message:  "golmet.index.json ilegível (" + iperr.Error() + ") — comparação prosa↔índice ignorada",
				Fix:      "recompose o índice (persistência do kit — bloco 2)"})
		} else {
			ctx.Index = idx
		}
	}

	res := &Result{Workspace: root, Context: ContextWorkspace, Files: len(files), Counts: map[string]int{}}
	for _, r := range rules.All() {
		for _, e := range r.Fn(ctx) {
			if e.Severity == core.SeverityError {
				res.Errors = append(res.Errors, e)
			} else {
				res.Warnings = append(res.Warnings, e)
			}
		}
	}
	res.Errors = append(res.Errors, pre...)
	sortFindings(res.Errors)
	sortFindings(res.Warnings)

	res.Counts["rn"] = len(ctx.RNs)
	res.Counts["pend"] = len(ctx.PENDs)
	res.Counts["cr"] = len(ctx.CRs)
	res.Counts["adr"] = len(ctx.ADRs)
	res.Counts["cand"] = len(ctx.CANDs)
	res.Counts["feature"] = len(ctx.Features)
	tasks := 0
	for _, f := range ctx.Features {
		tasks += len(f.Tasks)
	}
	res.Counts["task"] = tasks
	res.Counts["spec"] = len(ctx.Specs)
	res.Counts["file"] = len(files)

	res.Inventory = &Inventory{
		Charter: ctx.Charter, RNs: ctx.RNs, PENDs: ctx.PENDs, CRs: ctx.CRs,
		ADRs: ctx.ADRs, CANDs: ctx.CANDs, Specs: ctx.Specs, Features: ctx.Features,
	}

	// Índice de Estados (§8, D2) — o kit é o único escritor: compõe sempre;
	// escreve quando não há erro hard, ou quando --fix força (pós-merge).
	idx := buildIndex(ctx)
	res.Index = idx
	if opt.WriteIndex && (len(res.Errors) == 0 || opt.ForceIndex) {
		written, werr := indexstate.WriteIfChanged(filepath.Join(root, "golmet.index.json"), idx)
		if werr != nil {
			res.Warnings = append(res.Warnings, core.GolmetError{Rule: "H6", Entity: "Index",
				File: "golmet.index.json", Severity: core.SeverityWarning,
				Message: "falha ao escrever o índice: " + werr.Error(),
				Fix:     "verifique permissões do workspace"})
			sortFindings(res.Warnings)
		} else {
			res.IndexWritten = written
		}
	}
	return res, nil
}

// buildIndex compõe o retrato §8 a partir do inventário — mesma resolução
// de estado (entities.StateOf/Default) e a MESMA normalização de hash de
// seção (indexstate.SectionHash) que H8 usa.
func buildIndex(ctx *rules.Context) *indexstate.Index {
	var snap indexstate.Snapshot
	if ctx.Charter != nil {
		snap.CharterFound = true
		if v, ok := ctx.Charter.FM["status"]; ok {
			snap.CharterStatus = v.String()
		}
	}
	addSections := func(ns, kind string, secs []entities.Section) {
		for _, s := range secs {
			snap.Sections = append(snap.Sections, indexstate.SectionSnap{
				NS:        ns,
				ID:        s.ID,
				Estado:    entities.StateOf(s, entities.Default(kind)),
				RawLines:  s.RawLines,
				Evidencia: sectionEvidence(ctx, s),
			})
		}
	}
	addSections("rn", "RN", ctx.RNs)
	addSections("pend", "PEND", ctx.PENDs)
	addSections("cr", "CR", ctx.CRs)
	addSections("adr", "ADR", ctx.ADRs)
	addSections("cand", "CAND", ctx.CANDs)
	for _, sp := range ctx.Specs {
		est := entities.Default("Spec")
		if sp.Dir != nil {
			est = strings.ToLower(sp.Dir.State)
		}
		snap.Specs = append(snap.Specs, indexstate.SpecSnap{ID: sp.ID, Estado: est})
	}
	for i := range ctx.Features {
		f := &ctx.Features[i]
		tasks := map[string]string{}
		for _, t := range f.Tasks {
			tasks[t.ID] = entities.StateOf(t, entities.Default("Task"))
		}
		snap.Features = append(snap.Features, indexstate.FeatureSnap{
			Key:    entities.FeatureKey(f),
			Status: strings.ToLower(strings.TrimSpace(f.FMString("status"))),
			Anchor: strings.TrimSpace(f.FMString("anchor")),
			Tasks:  tasks,
		})
	}
	return indexstate.Build(filepath.Base(ctx.Root), snap)
}

// sectionEvidence coleta as evidências canônicas (§5) dentro do corpo da
// seção, com head_ok resolvido contra o HEAD do repo único (charter §1) —
// hash não resolvido em repo ou git indisponível = head_ok false.
func sectionEvidence(ctx *rules.Context, s entities.Section) []indexstate.EvidenceRef {
	var out []indexstate.EvidenceRef
	for _, f := range ctx.EvidenceAll {
		if f.Kind != evidence.KindCanonical || f.File != s.File {
			continue
		}
		if f.Line <= s.Line || f.Line > s.Line+len(s.RawLines) {
			continue
		}
		ref := indexstate.EvidenceRef{Path: f.Ev.Path, Line: f.Ev.Line, Hash: f.Ev.Hash}
		if n, repo := ctx.Resolver().Resolve(f.Ev.Hash); n == 1 && ctx.Git != nil {
			if head, err := ctx.Git.Head(repo); err == nil && strings.HasPrefix(head, f.Ev.Hash) {
				ref.HeadOK = true
			}
		}
		out = append(out, ref)
	}
	return out
}

func sortFindings(xs []core.GolmetError) {
	sort.SliceStable(xs, func(i, j int) bool {
		a, b := xs[i], xs[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Rule < b.Rule
	})
}
