// Package status implementa o `golmet status` — dashboard SOMENTE-LEITURA
// do workspace (FORMAT §7/§8): entidades por estado, PENDs abertas, CRs
// não-implementados, Features em andamento com tasks pendentes, specs
// stale (propagação devedora — S1) e versão do formato. NUNCA escreve.
package status

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Golmet/golmet-cli/internal/core/entities"
	"github.com/Golmet/golmet-cli/internal/core/frontmatter"
	"github.com/Golmet/golmet-cli/internal/core/scan"
)

// Options reservado (status não tem flags além de --json no CLI).
type Options struct{}

// PendAberta é uma PEND com estado open.
type PendAberta struct {
	ID     string `json:"id"`
	Titulo string `json:"titulo"`
}

// CRAberto é um CR pending/approved (não-implementado).
type CRAberto struct {
	ID     string `json:"id"`
	Estado string `json:"estado"`
	Titulo string `json:"titulo"`
}

// FeatureAndamento é uma Feature in_progress com tasks não-done.
type FeatureAndamento struct {
	Anchor         string   `json:"anchor"`
	File           string   `json:"file"`
	TasksPendentes []string `json:"tasks_pendentes"`
}

// SpecStale é uma spec com estado stale (§7.7).
type SpecStale struct {
	ID     string `json:"id"`
	File   string `json:"file"`
	Reason string `json:"reason,omitempty"`
}

// FormatInfo compara o format do workspace com o suportado pelo kit (§12).
type FormatInfo struct {
	Atual     int  `json:"atual"`
	Suportado bool `json:"suportado"`
}

// Report é o dashboard completo.
type Report struct {
	Workspace           string             `json:"workspace"`
	Format              FormatInfo         `json:"format"`
	Counts              map[string]int     `json:"counts"`
	PendsAbertas        []PendAberta       `json:"pends_abertas"`
	CRsAbertos          []CRAberto         `json:"crs_abertos"`
	FeaturesEmAndamento []FeatureAndamento `json:"features_em_andamento"`
	SpecsStale          []SpecStale        `json:"specs_stale"`
	PropagacaoDevedora  bool               `json:"propagacao_devedora"`
}

// Run executa o dashboard. Erro de retorno = workspace inválido (CLI sai 2).
// scan.Run roda SEM WriteIndex — o status jamais escreve.
func Run(dir string, _ Options) (*Report, error) {
	root, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("workspace inválido: %w", err)
	}
	if st, serr := os.Stat(root); serr != nil || !st.IsDir() {
		return nil, fmt.Errorf("workspace não encontrado: %s", root)
	}

	// format do workspace vs suportado pelo kit (§12 — kit 1 suporta 1..1)
	cfgData, err := os.ReadFile(filepath.Join(root, "golmet.yaml"))
	if err != nil {
		return nil, fmt.Errorf("golmet.yaml ausente em %s — workspace não inicializado (rode 'golmet init')", root)
	}
	fields, perr := frontmatter.ParseBlock(string(cfgData))
	format := FormatInfo{Suportado: false}
	if perr != nil {
		return nil, fmt.Errorf("golmet.yaml ilegível: %v", perr)
	}
	if fv, ok := fields["format"]; ok && fv.Str == "1" {
		format = FormatInfo{Atual: 1, Suportado: true}
	} else if !ok || fv.Str == "" {
		return nil, fmt.Errorf("golmet.yaml sem format (DEVE ser 1)")
	} else {
		return nil, fmt.Errorf("formato %s não suportado — este kit suporta format: 1 (§12)", fv.Str)
	}

	res, err := scan.Run(dir, scan.Options{}) // WriteIndex=false — read-only
	if err != nil {
		return nil, err
	}
	inv := res.Inventory

	rep := &Report{
		Workspace: filepath.Base(root),
		Format:    format,
		Counts:    res.Index.Counts,
	}
	for _, p := range inv.PENDs {
		if entities.StateOf(p, entities.Default("PEND")) != "open" {
			continue
		}
		rep.PendsAbertas = append(rep.PendsAbertas, PendAberta{ID: p.ID, Titulo: p.Title})
	}
	implemented := false
	for _, cr := range inv.CRs {
		est := entities.StateOf(cr, entities.Default("CR"))
		if est == "implemented" {
			implemented = true
		}
		if est == "pending" || est == "approved" {
			rep.CRsAbertos = append(rep.CRsAbertos, CRAberto{ID: cr.ID, Estado: est, Titulo: cr.Title})
		}
	}
	for i := range inv.Features {
		f := &inv.Features[i]
		st := strings.ToLower(strings.TrimSpace(f.FMString("status")))
		if st != "in_progress" {
			continue
		}
		fa := FeatureAndamento{Anchor: entities.FeatureKey(f), File: f.File}
		for _, t := range f.Tasks {
			if ts := entities.StateOf(t, entities.Default("Task")); ts != "done" {
				fa.TasksPendentes = append(fa.TasksPendentes, t.ID)
			}
		}
		rep.FeaturesEmAndamento = append(rep.FeaturesEmAndamento, fa)
	}
	for _, sp := range inv.Specs {
		est := entities.Default("Spec")
		if sp.Dir != nil {
			est = strings.ToLower(sp.Dir.State)
		}
		if est != "stale" {
			continue
		}
		ss := SpecStale{ID: sp.ID, File: sp.File}
		if sp.Dir != nil {
			ss.Reason = sp.Dir.Reason
		}
		rep.SpecsStale = append(rep.SpecsStale, ss)
	}
	rep.PropagacaoDevedora = implemented && len(rep.SpecsStale) > 0 // S1

	sortSections(rep)
	return rep, nil
}

func sortSections(rep *Report) {
	sort.Slice(rep.PendsAbertas, func(i, j int) bool { return rep.PendsAbertas[i].ID < rep.PendsAbertas[j].ID })
	sort.Slice(rep.CRsAbertos, func(i, j int) bool { return rep.CRsAbertos[i].ID < rep.CRsAbertos[j].ID })
	sort.Slice(rep.FeaturesEmAndamento, func(i, j int) bool {
		return rep.FeaturesEmAndamento[i].Anchor < rep.FeaturesEmAndamento[j].Anchor
	})
	sort.Slice(rep.SpecsStale, func(i, j int) bool { return rep.SpecsStale[i].ID < rep.SpecsStale[j].ID })
}

// Human renderiza o dashboard em texto (CLI sem --json).
func (r *Report) Human() string {
	var b strings.Builder
	fmt.Fprintf(&b, "workspace: %s — format: %d", r.Workspace, r.Format.Atual)
	if !r.Format.Suportado {
		b.WriteString(" (NÃO suportado por este kit)")
	}
	b.WriteString("\n\n")

	// entidades por estado, a partir dos counts (§8)
	type nsDef struct{ ns, label string }
	for _, d := range []nsDef{
		{"rn", "RN"}, {"pend", "PEND"}, {"cr", "CR"}, {"adr", "ADR"},
		{"cand", "CAND"}, {"feature", "Feature"}, {"task", "Task"}, {"spec", "Spec"},
	} {
		total, ok := r.Counts[d.ns]
		if !ok || total == 0 {
			fmt.Fprintf(&b, "%s 0\n", d.label)
			continue
		}
		var states []string
		prefix := d.ns + "_"
		for k, v := range r.Counts {
			if strings.HasPrefix(k, prefix) && k != "features_done" {
				states = append(states, fmt.Sprintf("%s %d", strings.TrimPrefix(k, prefix), v))
			}
		}
		sort.Strings(states)
		fmt.Fprintf(&b, "%s %d", d.label, total)
		if len(states) > 0 {
			fmt.Fprintf(&b, " (%s)", strings.Join(states, ", "))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")

	section := func(title string, items []string) {
		if len(items) == 0 {
			return
		}
		fmt.Fprintf(&b, "%s:\n", title)
		for _, it := range items {
			fmt.Fprintf(&b, "  - %s\n", it)
		}
	}
	pends := make([]string, 0, len(r.PendsAbertas))
	for _, p := range r.PendsAbertas {
		pends = append(pends, p.ID+" — "+p.Titulo)
	}
	section("PENDs abertas", pends)
	crs := make([]string, 0, len(r.CRsAbertos))
	for _, c := range r.CRsAbertos {
		crs = append(crs, c.ID+" ("+c.Estado+") — "+c.Titulo)
	}
	section("CRs não-implementados", crs)
	feats := make([]string, 0, len(r.FeaturesEmAndamento))
	for _, f := range r.FeaturesEmAndamento {
		line := f.Anchor
		if len(f.TasksPendentes) > 0 {
			line += " — tasks pendentes: " + strings.Join(f.TasksPendentes, ", ")
		}
		feats = append(feats, line)
	}
	section("Features in_progress", feats)
	specs := make([]string, 0, len(r.SpecsStale))
	for _, s := range r.SpecsStale {
		line := s.ID + " (" + s.File
		if s.Reason != "" {
			line += " · reason: " + s.Reason
		}
		line += ")"
		specs = append(specs, line)
	}
	section("Specs stale", specs)
	if r.PropagacaoDevedora {
		b.WriteString("propagação devedora: SIM — CR implemented com spec stale (S1)\n")
	}
	return b.String()
}
