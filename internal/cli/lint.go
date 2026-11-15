package cli

import (
	"fmt"
	"strings"

	"github.com/Golmet/golmet-cli/internal/core/install"
	"github.com/Golmet/golmet-cli/internal/core/lint"
	"github.com/spf13/cobra"
)

// newLintCmd — golmet lint [--workspace <dir>] [--engine <id>] [--home]
// [--json]. Varredura estrutural determinística (L1/L3/L4/L5/L6/L7 —
// read-only): findings com severidade, arquivo (+linha) e remediação
// sugerida. Exit 0 limpo; 2 com findings.
func newLintCmd() *cobra.Command {
	var jsonOut bool
	var engineID, home, workspace string
	cmd := &cobra.Command{
		Use:   "lint",
		Short: "Varredura estrutural determinística (refs mortas, órfãos, manifest stale, nomes, fantasmas, estrutura canônica)",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			if _, known := install.LookupEngine(engineID); !known {
				fmt.Fprintf(cmd.ErrOrStderr(), "golmet: engine %q não existe — disponíveis: %s\n",
					engineID, strings.Join(install.Engines(), ", "))
				return exitError(2)
			}
			if home == "" {
				home = mustHome()
			}
			rep, err := lint.Run(lint.Options{Workspace: workspace, EngineID: engineID, Home: home})
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "golmet: %s\n", err.Error())
				return exitError(2)
			}
			if jsonOut {
				if encErr := newJSONEncoder(out).Encode(rep); encErr != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "golmet: falha ao serializar JSON: %s\n", encErr.Error())
				}
			} else {
				fmt.Fprint(out, rep.Human())
			}
			if rep.Errors > 0 || rep.Warnings > 0 {
				return exitError(2)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&workspace, "workspace", ".", "dir alvo da varredura (kit repo para os checks L1/L5/L6/L7 de kit; default: atual)")
	cmd.Flags().StringVar(&engineID, "engine", "opencode", "engine para resolver o config root (opencode | claude-code | cursor | gemini-cli | codex; vscode-copilot é workspace-scoped)")
	cmd.Flags().StringVar(&home, "home", "", "home do usuário (default: UserHomeDir — override p/ testes)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "saída estruturada (report do lint)")
	return cmd
}
