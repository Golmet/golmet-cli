package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Golmet/golmet-cli/internal/mcpserve"
	"github.com/spf13/cobra"
)

// newMcpCmd — golmet mcp: servidor MCP (stdio) com validação, diagnóstico,
// atualização do FULL/core, packs e demais tools do kit. Processo long-running
// — exit codes de uso não se aplicam; fechamento limpo sai 0.
func newMcpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Serve o kit via MCP (stdio) — validação, update e packs",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			if err := mcpserve.Run(ctx); err != nil {
				// fechamento pelo cliente (stdin fechado) não é erro
				if ctx.Err() != nil {
					return nil
				}
				fmt.Fprintln(cmd.ErrOrStderr(), "golmet mcp: "+err.Error())
				return exitError(2)
			}
			return nil
		},
	}
}
