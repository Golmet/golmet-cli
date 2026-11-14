package cli

import (
	"fmt"

	"github.com/Golmet/golmet-cli/internal/core/doctor"
	"github.com/spf13/cobra"
)

// newDoctorCmd — golmet doctor [workspace] [--json] [--network]
func newDoctorCmd() *cobra.Command {
	var jsonOut, network, noNetwork bool
	cmd := &cobra.Command{
		Use:   "doctor [workspace]",
		Short: "Verificações soft (S1–S5) e saúde do ambiente",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws := "."
			if len(args) == 1 {
				ws = args[0]
			}
			rep, err := doctor.Run(ws, doctor.Options{Network: network, NoNetwork: noNetwork})
			if rep != nil {
				if jsonOut {
					if encErr := newJSONEncoder(cmd.OutOrStdout()).Encode(rep.Checks); encErr != nil {
						fmt.Fprintln(cmd.ErrOrStderr(), "golmet: falha ao serializar JSON: "+encErr.Error())
					}
				} else {
					fmt.Fprint(cmd.OutOrStdout(), rep.Human())
				}
			}
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "golmet: "+err.Error())
				return exitError(2)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "saída estruturada (array de checks)")
	cmd.Flags().BoolVar(&network, "network", false, "alias de compatibilidade — o smoke do kit.jev já roda por default com key")
	cmd.Flags().BoolVar(&noNetwork, "no-network", false, "pula o smoke do kit.jev (CI/offline — só checa a key)")
	return cmd
}
