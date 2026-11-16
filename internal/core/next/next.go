// Package next implementa o `golmet next` — recomendação determinística
// da próxima ação sobre o workspace (FORMAT §7/§8/§9). Consulta pura:
// lê prosa + índice + manifest e produz lista ORDENADA por prioridade
// (p1 = mais urgente); nunca escreve.
package next

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Golmet/golmet-cli/internal/adapters/fsutil"
	"github.com/Golmet/golmet-cli/internal/core/entities"
	"github.com/Golmet/golmet-cli/internal/core/scan"
	"github.com/Golmet/golmet-cli/internal/core/update"
)

// Recommendation é uma ação sugerida, com a regra que a ancora.
type Recommendation struct {
	Priority int    `json:"priority"`
	Action   string `json:"action"`
	Reason   string `json:"reason"`
	Rule     string `json:"rule,omitempty"`
}

// Report agrega as recomendações ordenadas.
type Report struct {
	Workspace       string           `json:"workspace"`
	Recommendations []Recommendation `json:"recommendations"`
}

// Prioridades (menor = mais urgente) — fonte única para o CLI e o MCP.
const (
	PValidationError = 1  // erros hard de validate bloqueiam
	PCharter         = 2  // charter ausente — nada parte sem ele
	PUpdate          = 3  // subset BASE não instalado (gm-*)
	PIndexStale      = 4  // prosa mudou após a última validação boa
	PPendOpen        = 5  // PENDs abertas esperam decisão/promoção
	PCRNotDone       = 6  // CRs pending/approved aguardam propagação
	PSpecStale       = 7  // specs stale (S1 — propagação devedora)
	PFeatureTask     = 8  // feature in_progress com tasks pendentes
	PFeatureReady    = 9  // feature ready não iniciada
	PAllDone         = 99 // nada pendente
)

// Run executa o diagnóstico. Erro = workspace inválido (CLI exit 2).
func Run(dir string) (*Report, error) {
	root, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("workspace inválido: %w", err)
	}
	if st, serr := os.Stat(root); serr != nil || !st.IsDir() {
		return nil, fmt.Errorf("workspace não encontrado: %s", root)
	}
	res, err := scan.Run(dir, scan.Options{})
	if err != nil {
		return nil, err
	}
	rep := &Report{Workspace: filepath.Base(root)}

	// p1 — erros hard de validate (bloqueiam CI/done)
	for _, e := range res.Errors {
		rep.Recommendations = append(rep.Recommendations, Recommendation{
			Priority: PValidationError,
			Action:   "corrija " + e.Rule + " e rode 'golmet validate'",
			Reason:   e.Message,
			Rule:     e.Rule,
		})
	}

	inv := res.Inventory
	// p2 — charter ausente
	if inv == nil || inv.Charter == nil {
		rep.Recommendations = append(rep.Recommendations, Recommendation{
			Priority: PCharter,
			Action:   "crie o charter com gm-charter (ou 'golmet init' para o esqueleto)",
			Reason:   "PROJECT_CHARTER.md ausente — o workspace não tem identidade",
			Rule:     "H5",
		})
	}

	// p3 — subset BASE não instalado
	mf, merr := update.ParseManifest(root)
	if merr != nil || mf == nil || len(mf.Skills) == 0 {
		rep.Recommendations = append(rep.Recommendations, Recommendation{
			Priority: PUpdate,
			Action:   "rode 'golmet update' para instalar o subset BASE de skills gm-*",
			Reason:   "zona .golmet/ sem manifest com skills — engines não têm roteamento BASE",
		})
	}

	// p4 — índice stale (ausente, ilegível, ou prosa mais nova que a
	// última validação boa — generated_at vem do ARQUIVO, não do rebuild)
	idxFile := filepath.Join(root, "golmet.index.json")
	if idxData, idxErr := os.ReadFile(idxFile); idxErr != nil {
		rep.Recommendations = append(rep.Recommendations, Recommendation{
			Priority: PIndexStale,
			Action:   "rode 'golmet validate' para criar o índice (baseline)",
			Reason:   "golmet.index.json ausente — H6/H8 sem história",
			Rule:     "§8",
		})
	} else if gen := indexGeneratedAt(idxData); gen == "" {
		rep.Recommendations = append(rep.Recommendations, Recommendation{
			Priority: PIndexStale,
			Action:   "rode 'golmet index' para recompor o índice",
			Reason:   "golmet.index.json ilegível — cache inválido",
			Rule:     "§8",
		})
	} else if t, terr := time.Parse(time.RFC3339, gen); terr == nil {
		if newest, nerr := fsutil.NewestMDMod(root); nerr == nil && newest.Sub(t) > 2*time.Second {
			rep.Recommendations = append(rep.Recommendations, Recommendation{
				Priority: PIndexStale,
				Action:   "rode 'golmet validate' para reconciliar o índice",
				Reason:   "prosa modificada após a última validação boa (" + gen + ")",
				Rule:     "§8",
			})
		}
	}

	// p5 — PENDs abertas
	for _, p := range inv.PENDs {
		if entities.StateOf(p, entities.Default("PEND")) != "open" {
			continue
		}
		rep.Recommendations = append(rep.Recommendations, Recommendation{
			Priority: PPendOpen,
			Action:   "decida/promova " + p.ID + " com gm-clarify + gm-requirements",
			Reason:   "PEND aberta: " + p.Title + " — PEND nunca ancora feature (H1)",
			Rule:     "H1",
		})
	}

	// p6 — CRs pending/approved não-implementados
	for _, cr := range inv.CRs {
		est := entities.StateOf(cr, entities.Default("CR"))
		if est != "pending" && est != "approved" {
			continue
		}
		rep.Recommendations = append(rep.Recommendations, Recommendation{
			Priority: PCRNotDone,
			Action:   "propague/implemente " + cr.ID + " (" + est + ")",
			Reason:   "CR não-implementado: " + cr.Title,
		})
	}

	// p7 — specs stale (propagação devedora)
	for _, sp := range inv.Specs {
		est := entities.Default("Spec")
		if sp.Dir != nil {
			est = strings.ToLower(sp.Dir.State)
		}
		if est != "stale" {
			continue
		}
		reason := sp.Dir.Reason
		rep.Recommendations = append(rep.Recommendations, Recommendation{
			Priority: PSpecStale,
			Action:   "atualize a spec " + sp.ID + " (" + sp.File + ") e marque current",
			Reason:   "spec stale" + mapBool(reason != "", " · reason: "+reason, ""),
			Rule:     "S1",
		})
	}

	// p8 — features in_progress com tasks pendentes
	for i := range inv.Features {
		f := &inv.Features[i]
		if strings.ToLower(strings.TrimSpace(f.FMString("status"))) != "in_progress" {
			continue
		}
		var pend []string
		for _, t := range f.Tasks {
			if entities.StateOf(t, entities.Default("Task")) != "done" {
				pend = append(pend, t.ID)
			}
		}
		if len(pend) == 0 {
			continue
		}
		rep.Recommendations = append(rep.Recommendations, Recommendation{
			Priority: PFeatureTask,
			Action:   "trabalhe a task " + pend[0] + " (" + strings.Join(pend[1:], ", ") + ") na feature " + entities.FeatureKey(f),
			Reason:   fmt.Sprintf("%d task(s) pendente(s) na feature in_progress", len(pend)),
		})
	}

	// p9 — features ready não iniciadas
	for i := range inv.Features {
		f := &inv.Features[i]
		if strings.ToLower(strings.TrimSpace(f.FMString("status"))) != "ready" {
			continue
		}
		rep.Recommendations = append(rep.Recommendations, Recommendation{
			Priority: PFeatureReady,
			Action:   "inicie a feature " + entities.FeatureKey(f) + " (ready → in_progress)",
			Reason:   "feature pronta sem início",
		})
	}

	sort.SliceStable(rep.Recommendations, func(i, j int) bool {
		return rep.Recommendations[i].Priority < rep.Recommendations[j].Priority
	})
	if len(rep.Recommendations) == 0 {
		rep.Recommendations = append(rep.Recommendations, Recommendation{
			Priority: PAllDone,
			Action:   "nada pendente — workspace em dia",
			Reason:   "validate limpo, sem PENDs/CRs abertos, specs e índice em dia",
		})
	}
	return rep, nil
}

// indexGeneratedAt lê o generated_at do arquivo de índice ("" = ilegível).
func indexGeneratedAt(data []byte) string {
	var probe struct {
		GeneratedAt string `json:"generated_at"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return ""
	}
	return probe.GeneratedAt
}

func mapBool(cond bool, ifTrue, ifFalse string) string {
	if cond {
		return ifTrue
	}
	return ifFalse
}

// Human renderiza as recomendações (CLI sem --json).
func (r *Report) Human() string {
	var b strings.Builder
	fmt.Fprintf(&b, "workspace: %s — próxima ação\n", r.Workspace)
	for _, rec := range r.Recommendations {
		fmt.Fprintf(&b, "p%-2d %s\n     %s\n", rec.Priority, rec.Action, rec.Reason)
		if rec.Rule != "" {
			fmt.Fprintf(&b, "     regra: %s\n", rec.Rule)
		}
	}
	return b.String()
}
