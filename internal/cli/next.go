package cli

import (
	"fmt"

	"github.com/Golmet/golmet-cli/internal/core/next"
	"github.com/spf13/cobra"
)

// newNextCmd — golmet next [workspace] [--json] (consulta pura)
func newNextCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "next [workspace]",
		Short: "Recomenda a próxima ação sobre o workspace (prioridades p1..p9)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws := "."
			if len(args) == 1 {
				ws = args[0]
			}
			rep, err := next.Run(ws)
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "golmet: "+err.Error())
				return exitError(2)
			}
			if jsonOut {
				return encodeJSON(cmd, rep)
			}
			fmt.Fprint(cmd.OutOrStdout(), rep.Human())
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "saída estruturada (recomendações em JSON)")
	return cmd
}
