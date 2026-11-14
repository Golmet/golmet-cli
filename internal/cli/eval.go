package cli

import (
	"fmt"

	"github.com/Golmet/golmet-cli/internal/core/eval"
	"github.com/spf13/cobra"
)

// newEvalCmd — golmet eval <fixture-dir> --workspace <dir> [--json]
// golmet eval --all <fixtures-root> --workspaces-root <dir> [--json]
// Checker determinístico do corpus de evals (DR-017): o kit pontua os
// artefatos produzidos pelo engine (seeding/execução NÃO é papel do kit).
func newEvalCmd() *cobra.Command {
	var workspace, workspacesRoot string
	var all, jsonOut bool
	cmd := &cobra.Command{
		Use:   "eval [fixture-dir | fixtures-root]",
		Short: "Pontua criteria.yaml de fixtures de eval contra o workspace (checker determinístico — DR-017)",
		Args:  cobra.MaximumNArgs(1),
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
					return encodeJSON(cmd, suite)
				}
				fmt.Fprint(out, suite.Human())
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
			if jsonOut {
				return encodeJSON(cmd, res)
			}
			fmt.Fprint(out, res.Human())
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
