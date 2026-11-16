package cli

import (
	"fmt"

	"github.com/Golmet/golmet-cli/internal/core/adopt"
	"github.com/spf13/cobra"
)

// newAdoptCmd — golmet adopt [workspace] [--dry-run] [--apply]
// [--charter-status <s>] [--keep-legacy-layouts] [--json]
// Papel do kit no handshake da skill gm-adopt: dry-run = leitura pura
// (safe default); apply = transform determinístico das escolhas
// declaradas + validate tolerant interno (worklist, não bloqueia).
func newAdoptCmd() *cobra.Command {
	var dryRun, apply, jsonOut, keepLegacy bool
	var charterStatus string
	cmd := &cobra.Command{
		Use:   "adopt [workspace]",
		Short: "Adota o FORMAT v1 num workspace existente (dry-run → apply; skill gm-adopt)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRun && apply {
				fmt.Fprintln(cmd.ErrOrStderr(), "golmet: flags --dry-run e --apply são exclusivas (default é dry-run)")
				return exitError(2)
			}
			ws := "."
			if len(args) == 1 {
				ws = args[0]
			}
			rep, err := adopt.Run(ws, adopt.Options{
				Apply:             apply,
				CharterStatus:     charterStatus,
				KeepLegacyLayouts: keepLegacy,
			})
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
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "relatório de conformidade sem escrita (default)")
	cmd.Flags().BoolVar(&apply, "apply", false, "executa o transform determinístico das escolhas declaradas")
	cmd.Flags().StringVar(&charterStatus, "charter-status", "", "status do charter no apply: draft (default) | confirmed (§7.1)")
	cmd.Flags().BoolVar(&keepLegacy, "keep-legacy-layouts", false, "registra as variantes legadas de layout como mantidas por escolha")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "saída estruturada (relatório em JSON)")
	return cmd
}
