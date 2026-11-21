package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Golmet/golmet-cli/internal/core/graph"
	"github.com/spf13/cobra"
)

// newGraphCmd — golmet graph [--config-root] [--packs-root] [--projects-root]
// [--out] [--pack] [--scope] [--max-nodes] [--from-json] [--what] [--json]
// Motor determinístico do grafo de conhecimento (T1-T5) — ADR-019
// ("determinístico mora no CLI"). Strict SÓ no registry (exit 1); T1-T4
// ausente = aviso + grafo parcial; saída integral (3 artefatos ou nada).
func newGraphCmd() *cobra.Command {
	var (
		configRoot   string
		packsRoot    string
		projectsRoot string
		outDir       string
		pack         string
		scope        string
		maxNodes     int
		fromJSON     string
		what         string
		jsonOut      bool
	)
	cmd := &cobra.Command{
		Use:   "graph",
		Short: "Grafo de conhecimento T1-T5 (workflow/pack/docs) — graph.json + HTML self-contained + resumo MD (determinístico)",
		Long: `Gera o grafo navegável da base de conhecimento (stores T1-T5 do knowledge-routing).

Seletor --what: workflow (T5) · pack (T1+T2+T3) · docs (T4) — combináveis com
'+' (ex.: workflow+pack); default all. Arestas cruzadas só entram quando
origem e destino estão selecionados. Strict SÓ no registry (ausente/inválido
aborta exit 1); fontes T1-T4 ausentes viram aviso com grafo parcial. A saída
é integral: os 3 artefatos ou nada. Determinístico — a única variação entre
execuções é o gerado_em (precisão de minuto).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			if maxNodes < 0 {
				fmt.Fprintln(cmd.ErrOrStderr(), "golmet: --max-nodes deve ser >= 0")
				return exitError(2)
			}
			if fromJSON != "" {
				if _, err := os.Stat(fromJSON); err != nil {
					fmt.Fprintln(cmd.ErrOrStderr(), "golmet: --from-json ilegível: "+fromJSON)
					return exitError(2)
				}
			}
			sum, err := graph.Run(graph.Options{
				ConfigRoot:   configRoot,
				PacksRoot:    packsRoot,
				Pack:         pack,
				ProjectsRoot: projectsRoot,
				Out:          outDir,
				Scope:        scope,
				MaxNodes:     maxNodes,
				FromJSON:     fromJSON,
				What:         what,
				DispCfg:      configRoot,
				DispPacks:    packsRoot,
				DispProjects: projectsRoot,
			})
			if err != nil {
				if se, ok := err.(*graph.StrictError); ok {
					fmt.Fprintln(cmd.ErrOrStderr(), "golmet: "+se.Msg)
					return exitError(1)
				}
				fmt.Fprintln(cmd.ErrOrStderr(), "golmet: "+err.Error())
				return exitError(2)
			}
			if jsonOut {
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				if jerr := enc.Encode(sum); jerr != nil {
					fmt.Fprintln(cmd.ErrOrStderr(), "golmet: falha ao serializar JSON: "+jerr.Error())
					return exitError(2)
				}
				return nil
			}
			for _, line := range sum.ConsoleLines() {
				fmt.Fprintln(out, line)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&configRoot, "config-root", "", "raiz do config do engine (default: ~/.config/opencode)")
	cmd.Flags().StringVar(&packsRoot, "packs-root", "", "raiz onde vive packs/ (default: --config-root)")
	cmd.Flags().StringVar(&projectsRoot, "projects-root", "", "raiz dos módulos de produto (sem ela, T4 é pulado com aviso)")
	cmd.Flags().StringVar(&outDir, "out", "", "diretório de saída (default: <cwd>/knowledge-graph)")
	cmd.Flags().StringVar(&pack, "pack", "", "id do pack (default: único pack com domains/ em packs-root/packs)")
	cmd.Flags().StringVar(&scope, "scope", "", "delta: restringe docs (T4) ao módulo nomeado")
	cmd.Flags().IntVar(&maxNodes, "max-nodes", 3500, "teto de nós (poda determinística RN/PEND do global primeiro)")
	cmd.Flags().StringVar(&fromJSON, "from-json", "", "regenera os artefatos de um graph.json congelado (sem re-parseio)")
	cmd.Flags().StringVar(&what, "what", "all", "seletor de conteúdo: workflow|pack|docs|all — combinável com '+' (ex.: workflow+pack)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "saída estruturada (resumo da execução)")
	return cmd
}
