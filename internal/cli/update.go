package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Golmet/golmet-cli/internal/core/content"
	coreinstall "github.com/Golmet/golmet-cli/internal/core/install"
	"github.com/Golmet/golmet-cli/internal/core/source"
	"github.com/Golmet/golmet-cli/internal/core/update"
	"github.com/spf13/cobra"
)

// newUpdateCmd — golmet update [workspace] [--check] [--json]
// Instala o subset BASE de skills na zona .golmet/ (DR-008) ou
// compara embutido vs instalado com --check (consulta, sem escrita).
func newUpdateCmd() *cobra.Command {
	var check, jsonOut, full, force bool
	var engineID, coreSource, workspace string
	cmd := &cobra.Command{
		Use:   "update [workspace]",
		Short: "Instala o subset BASE de skills na zona .golmet/ (DR-008)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			if coreSource != "" && !full {
				return fmt.Errorf("--core exige --full; use 'golmet update --full --core <source> --engine %s'", engineID)
			}
			if full {
				if check {
					return fmt.Errorf("--check --full ainda não é suportado; use 'golmet update --full --engine %s' para materializar", engineID)
				}
				if engineID == "" {
					return fmt.Errorf("--full exige --engine (o perfil completo exige skills nativas — compatíveis: opencode, claude-code)")
				}
				home, err := os.UserHomeDir()
				if err != nil {
					return err
				}
				// fail-closed: o perfil completo exige engine com skills
				// nativas — sem degradação silenciosa para a zona do workspace
				if !coreinstall.EngineSupportsCompleto(engineID, home) {
					return installExit(cmd, fmt.Errorf("o perfil completo exige engine com skills nativas — compatíveis: %s; para %s, use o perfil essencial",
						strings.Join(coreinstall.CompatibleEngineIDs(home), ", "), engineID))
				}
				ws := workspace
				if ws == "" {
					ws = "."
				}
				var rep *content.MaterializeReport
				var syncReport *source.SyncReport
				if coreSource != "" {
					rep, syncReport, err = coreinstall.MaterializeExternalCore(home, engineID, coreSource, force)
				} else {
					// a zona (FullPlacement) é comportamento interno do
					// pipeline — via CLI o destino é sempre o config nativo
					rep, _, err = coreinstall.MaterializeFull(home, engineID, ws, force)
				}
				if err != nil {
					fmt.Fprintln(cmd.ErrOrStderr(), "golmet:", err)
					return exitError(2)
				}
				if jsonOut {
					return encodeJSON(cmd, rep)
				}
				if syncReport != nil {
					fmt.Fprintf(out, "core externo %s @ %s — %d arquivo(s) escrito(s), %d conflito(s) preservado(s)\n", syncReport.Source, syncReport.SHA, len(rep.Written), len(rep.Conflicts))
				} else {
					fmt.Fprintf(out, "FULL embarcado %s — %d arquivo(s) escrito(s), %d conflito(s) preservado(s)\n", engineID, len(rep.Written), len(rep.Conflicts))
				}
				for _, p := range rep.Conflicts {
					fmt.Fprintf(out, "  conflito preservado (customização local): %s\n", p)
				}
				return nil
			}
			ws := "."
			if len(args) == 1 {
				ws = args[0]
			}
			if check {
				rep, err := update.Check(ws)
				if err != nil {
					fmt.Fprintln(cmd.ErrOrStderr(), "golmet: "+err.Error())
					return exitError(2)
				}
				if jsonOut {
					return encodeJSON(cmd, rep)
				}
				printUpdateCheck(out, rep)
				return nil
			}
			rep, err := update.Install(ws)
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "golmet: "+err.Error())
				return exitError(2)
			}
			if jsonOut {
				return encodeJSON(cmd, rep)
			}
			fmt.Fprintf(out, "kit %s — %d skill(es) instalada(s) em %s/.golmet/skills (manifest atualizado)\n",
				rep.KitVersion, rep.Installed, rep.Workspace)
			for _, p := range rep.Preserved {
				fmt.Fprintf(out, "  preservada (modificada localmente — DR-008): %s\n", p)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "compara kit embutido vs instalado — sem escrita")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "saída estruturada (install report / check report)")
	cmd.Flags().BoolVar(&full, "full", false, "atualiza/materializa o workflow FULL no config do engine (perfil completo — exige skills nativas)")
	cmd.Flags().StringVar(&engineID, "engine", "", "engine do FULL (perfil completo — compatíveis: opencode | claude-code)")
	cmd.Flags().StringVar(&workspace, "workspace", "", "workspace do kit (contexto da materialização; default: atual)")
	cmd.Flags().StringVar(&coreSource, "core", "", "fonte Git ou diretório local do core externo (substitui o FULL embarcado)")
	cmd.Flags().BoolVar(&force, "force", false, "sobrescreve customizações locais no FULL")
	return cmd
}

func printUpdateCheck(w io.Writer, rep *update.CheckReport) {
	fmt.Fprintf(w, "kit embutido: %s", rep.KitVersion)
	if rep.NotInstalled {
		fmt.Fprintf(w, " — zona .golmet/ sem skills instaladas\n  fix: rode 'golmet update'\n")
		return
	}
	fmt.Fprintf(w, " | instalado: %s (%s)\n", rep.InstalledVersion, rep.InstalledAt)
	drift := false
	report := func(label string, xs []string) {
		if len(xs) > 0 {
			drift = true
			fmt.Fprintf(w, "  %s: %s\n", label, strings.Join(xs, ", "))
		}
	}
	report("faltantes", rep.Faltantes)
	report("extras (fora do subset)", rep.Extras)
	report("modificadas localmente", rep.Modificados)
	if drift {
		fmt.Fprintf(w, "  fix: rode 'golmet update' para reconciliar (modificadas são preservadas)\n")
	} else {
		fmt.Fprintf(w, "  zona conforme ao subset embutido\n")
	}
}

func encodeJSON(cmd *cobra.Command, v any) error {
	if err := newJSONEncoder(cmd.OutOrStdout()).Encode(v); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), "golmet: falha ao serializar JSON: "+err.Error())
		return exitError(2)
	}
	return nil
}
