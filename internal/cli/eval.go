package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Golmet/golmet-cli/internal/core/eval"
	"github.com/spf13/cobra"
)

// newEvalCmd — golmet eval <fixture-dir> --workspace <dir> [--json]
// golmet eval --all <fixtures-root> --workspaces-root <dir> [--json]
// Checker determinístico do corpus de evals (DR-017): o kit pontua os
// artefatos produzidos pelo engine (seeding/execução NÃO é papel do kit).
// Cada fixture avaliada gera um relatório markdown em ~/.golmet/eval-reports/
// (estado de máquina do kit, retenção de 50) — o path sai como
// "relatório: <path>" (stderr: stdout é canal de dados — humano ou JSON).
func newEvalCmd() *cobra.Command {
	var workspace, workspacesRoot string
	var all, jsonOut bool
	cmd := &cobra.Command{
		Use:   "eval [fixture-dir | fixtures-root]",
		Short: "Pontua criteria.yaml de fixtures de eval contra o workspace (checker determinístico — DR-017); grava relatório markdown em ~/.golmet/eval-reports",
		Long: `Pontua criteria.yaml de fixtures de eval contra o workspace (checker determinístico — DR-017).

Cada fixture avaliada gera um relatório markdown em ~/.golmet/eval-reports/
(estado de máquina do kit, mesma convenção de ~/.golmet/packs e
~/.golmet/sources; retenção dos 50 relatórios mais recentes por mtime).
O path do relatório é impresso como "relatório: <path>" — no stderr,
mantendo o stdout puro (humano ou --json).`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			if all {
				if len(args) != 1 {
					fmt.Fprintln(cmd.ErrOrStderr(), "golmet: --all exige a raiz das fixtures: golmet eval --all <fixtures-root> --workspaces-root <dir>")
					return exitError(2)
				}
				if workspacesRoot == "" {
					fmt.Fprintln(cmd.ErrOrStderr(), "golmet: --workspaces-root é obrigatória com --all")
					return exitError(2)
				}
				suite, err := eval.RunAll(args[0], workspacesRoot)
				if err != nil {
					fmt.Fprintln(cmd.ErrOrStderr(), "golmet: "+err.Error())
					return exitError(2)
				}
				if jsonOut {
					if err := encodeJSON(cmd, suite); err != nil {
						return err
					}
				} else {
					fmt.Fprint(out, suite.Human())
				}
				if err := writeSuiteReports(cmd, suite, workspacesRoot); err != nil {
					return err
				}
				if suite.FailedFixtures > 0 {
					return exitError(1)
				}
				return nil
			}
			if len(args) != 1 {
				fmt.Fprintln(cmd.ErrOrStderr(), "golmet: informe a fixture (eval <fixture-dir> --workspace <dir>) ou use --all")
				return exitError(2)
			}
			if workspace == "" {
				fmt.Fprintln(cmd.ErrOrStderr(), "golmet: --workspace é obrigatório no modo single-fixture")
				return exitError(2)
			}
			res, err := eval.RunFixture(args[0], workspace)
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "golmet: "+err.Error())
				return exitError(2)
			}
			path, rerr := writeFixtureReport(args[0], workspace, res)
			if rerr != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "golmet: "+rerr.Error())
				return exitError(2)
			}
			if jsonOut {
				if err := encodeJSON(cmd, res); err != nil {
					return err
				}
			} else {
				fmt.Fprint(out, res.Human())
			}
			// depois do output (JSON puro primeiro no stdout; relatório é
			// stderr) — stdout continua consumível por jq/consumidores
			printReportLine(cmd, path)
			if res.Status == eval.StatusFail {
				return exitError(1)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&workspace, "workspace", "", "workspace onde o engine produziu os artefatos (paths dos checks resolvem contra ele)")
	cmd.Flags().BoolVar(&all, "all", false, "pontua todo o corpus: <fixtures-root>/<skill>/fixture-N ↔ <workspaces-root>/<skill>/fixture-N")
	cmd.Flags().StringVar(&workspacesRoot, "workspaces-root", "", "raiz dos workspaces espelhados (obrigatória com --all)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "saída estruturada (resultado da fixture ou da suite)")
	return cmd
}

// writeFixtureReport grava o relatório canônico da fixture — skill/fx
// derivados do caminho pela convenção <fixtures-root>/<skill>/<fixture>.
// Sem flag para desligar: sempre grava (barato e útil).
func writeFixtureReport(fixtureDir, workspace string, res *eval.FixtureResult) (string, error) {
	skill, fixture := eval.SkillFixture(fixtureDir)
	return eval.WriteReport(skill, fixture, workspace, res, "")
}

// writeSuiteReports grava um relatório por fixture avaliada da suite
// (skipped não gera relatório — não houve execução).
func writeSuiteReports(cmd *cobra.Command, suite *eval.SuiteResult, workspacesRoot string) error {
	for i := range suite.Fixtures {
		f := &suite.Fixtures[i]
		if f.Status == eval.StatusSkipped {
			continue
		}
		skill, fixture := splitFixtureID(f.Fixture)
		path, err := eval.WriteReport(skill, fixture, filepath.Join(workspacesRoot, skill, fixture), f, "")
		if err != nil {
			fmt.Fprintln(cmd.ErrOrStderr(), "golmet: "+err.Error())
			return exitError(2)
		}
		printReportLine(cmd, path)
	}
	return nil
}

// splitFixtureID separa "skill/fixture-N" (id da FixtureResult da suite).
func splitFixtureID(id string) (string, string) {
	if i := strings.Index(id, "/"); i >= 0 {
		return id[:i], id[i+1:]
	}
	return id, id
}

// printReportLine — "relatório: <path>" no stderr (stdout é canal de dados).
func printReportLine(cmd *cobra.Command, path string) {
	fmt.Fprintln(cmd.ErrOrStderr(), "relatório: "+path)
}
