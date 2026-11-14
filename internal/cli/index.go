package cli

import (
	"fmt"

	"github.com/Golmet/golmet-cli/internal/core/scan"
	"github.com/spf13/cobra"
)

// newIndexCmd — golmet index [workspace]: recomposição explícita do
// golmet.index.json a partir da prosa (atalho de validate --fix — §8 D2).
func newIndexCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "index [workspace]",
		Short: "Recompõe o golmet.index.json a partir da prosa",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws := "."
			if len(args) == 1 {
				ws = args[0]
			}
			res, err := scan.Run(ws, scan.Options{WriteIndex: true, ForceIndex: true})
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "golmet: "+err.Error())
				return exitError(2)
			}
			if res.IndexWritten {
				fmt.Fprintln(cmd.OutOrStdout(), "índice recomposto (golmet.index.json)")
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "índice já conforme (golmet.index.json) — sem mudanças")
			}
			c := res.Index.Counts
			fmt.Fprintf(cmd.OutOrStdout(), "RN %d, PEND %d, CR %d, ADR %d, CAND %d, Feature %d, Task %d, Spec %d\n",
				c["rn"], c["pend"], c["cr"], c["adr"], c["cand"], c["feature"], c["task"], c["spec"])
			if len(res.Errors) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "%d erro(s) hard permanece(m) na prosa — rode 'golmet validate' para o diagnóstico\n", len(res.Errors))
			}
			return nil
		},
	}
}
