package cli

import (
	"fmt"
	"path/filepath"

	"github.com/Golmet/golmet-cli/internal/core/status"
	"github.com/spf13/cobra"
)

func absCwd() (string, error) {
	return filepath.Abs(".")
}

// newStatusCmd — golmet status [workspace] [--json] (somente-leitura)
func newStatusCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "status [workspace]",
		Short: "Dashboard de estados das entidades do workspace (somente-leitura)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws := "."
			if len(args) == 1 {
				ws = args[0]
			}
			rep, err := status.Run(ws, status.Options{})
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "golmet: "+err.Error())
				return exitError(2)
			}
			if jsonOut {
				if err := newJSONEncoder(cmd.OutOrStdout()).Encode(rep); err != nil {
					fmt.Fprintln(cmd.ErrOrStderr(), "golmet: falha ao serializar JSON: "+err.Error())
					return exitError(2)
				}
				return nil
			}
			fmt.Fprint(cmd.OutOrStdout(), rep.Human())
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "saída estruturada (dashboard em JSON)")
	return cmd
}
