package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Golmet/golmet-cli/internal/core/tools"
	"github.com/spf13/cobra"
)

// newToolsCmd — golmet tools [install codegraph | list]
func newToolsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tools",
		Short: "Instalação de tools nativas do kit (grafo de código — DR-016)",
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "install codegraph",
			Short: "Baixa e instala o grafo DeusData em ~/.golmet/tools/codegraph (SHA-256 + attestation)",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				if args[0] != "codegraph" {
					fmt.Fprintf(cmd.ErrOrStderr(), "golmet: tool desconhecido %q — disponível: codegraph\n", args[0])
					return exitError(2)
				}
				ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
				defer stop()
				rep, err := tools.InstallCodegraph(ctx, tools.Options{})
				if err != nil {
					fmt.Fprintln(cmd.ErrOrStderr(), "golmet: "+err.Error())
					return exitError(2)
				}
				out := cmd.OutOrStdout()
				fmt.Fprintf(out, "codegraph %s instalado em %s\n", rep.Version, rep.Binary)
				fmt.Fprintf(out, "  sha256: %s (%d bytes)\n", rep.SHA256, rep.Bytes)
				fmt.Fprintf(out, "  attestation: %s\n", rep.Attestation)
				for _, w := range rep.Warnings {
					fmt.Fprintf(out, "  AVISO: %s\n", w)
				}
				return nil
			},
		},
		&cobra.Command{
			Use:   "list",
			Short: "Lista o estado dos tools geridos pelo kit",
			RunE: func(cmd *cobra.Command, args []string) error {
				for _, l := range tools.List() {
					fmt.Fprintln(cmd.OutOrStdout(), l)
				}
				return nil
			},
		},
	)
	return cmd
}
