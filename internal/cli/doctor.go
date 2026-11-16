package cli

import (
	"fmt"
	"strings"

	"github.com/Golmet/golmet-cli/internal/core/doctor"
	"github.com/Golmet/golmet-cli/internal/core/install"
	"github.com/spf13/cobra"
)

// newDoctorCmd — golmet doctor [workspace] [--json] [--network]
// [--engine opencode]. O --engine resolve a raiz do config do engine via
// install.EngineConfigRootFor e alimenta o check de estado (cache
// reconstrutível vs dado do usuário); vscode-copilot é workspace-scoped —
// check não se aplica (nota informada).
func newDoctorCmd() *cobra.Command {
	var jsonOut, network, noNetwork bool
	var engineID string
	cmd := &cobra.Command{
		Use:   "doctor [workspace]",
		Short: "Verificações soft (S1–S5) e saúde do ambiente",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			ws := "."
			if len(args) == 1 {
				ws = args[0]
			}
			opt := doctor.Options{Network: network, NoNetwork: noNetwork}
			if _, known := install.LookupEngine(engineID); !known {
				fmt.Fprintf(cmd.ErrOrStderr(), "golmet: engine %q não existe — disponíveis: %s\n",
					engineID, strings.Join(install.Engines(), ", "))
				return exitError(2)
			}
			if root, applies := install.EngineConfigRootFor(mustHome(), engineID); applies {
				opt.EngineConfigRoot = root
			} else {
				fmt.Fprintln(out, "ℹ️ engine "+engineID+" é workspace-scoped (.vscode/ do workspace) — check de engine-config não se aplica")
			}
			rep, err := doctor.Run(ws, opt)
			if rep != nil {
				if jsonOut {
					// relatório COMPLETO (checks + seção engine_config) —
					// mesma forma do payload do gm_doctor no MCP
					if encErr := newJSONEncoder(out).Encode(rep); encErr != nil {
						fmt.Fprintln(cmd.ErrOrStderr(), "golmet: falha ao serializar JSON: "+encErr.Error())
					}
				} else {
					fmt.Fprint(out, rep.Human())
				}
			}
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "golmet: "+err.Error())
				return exitError(2)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "saída estruturada (relatório do doctor)")
	cmd.Flags().BoolVar(&network, "network", false, "alias de compatibilidade — o smoke do kit.jev já roda por default com key")
	cmd.Flags().BoolVar(&noNetwork, "no-network", false, "pula o smoke do kit.jev (CI/offline — só checa a key)")
	cmd.Flags().StringVar(&engineID, "engine", "opencode", "engine para o check de estado do config: opencode | claude-code | vscode-copilot | cursor | gemini-cli | codex | copilot-cli")
	return cmd
}
