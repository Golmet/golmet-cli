package cli

import (
	"bufio"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/Golmet/golmet-cli/internal/core/uninstall"
	"github.com/spf13/cobra"
)

// newUninstallCmd — golmet uninstall: remoção completa da estrutura global
// do Golmet na máquina com seleção por camadas. Tudo vem SELECIONADO por
// padrão MENOS o PACK — e o pack apaga APENAS o working copy local: o
// repositório remoto (ex.: github.com/TechMinds-Group/TM.PACK.RDD) NÃO é
// tocado — delete-o manualmente no GitHub se quiser remover de vez.
func newUninstallCmd() *cobra.Command {
	var only, home, workspace, engine string
	var dryRun, jsonOut, yes bool
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove a estrutura global do Golmet da máquina, por camadas (pack: só working copy local — repo remoto é manual)",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			plan, err := uninstall.Discover(uninstall.Options{
				EngineID: engine, Home: home,
			})
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "golmet: %s\n", err.Error())
				return exitError(2)
			}
			if dryRun {
				// dry-run: só o plano — apaga nada (a descoberta é read-only)
				fmt.Fprint(out, plan.Human())
				return nil
			}

			// seleção
			var selected []string
			if only != "" {
				if !yes {
					fmt.Fprintln(cmd.ErrOrStderr(), "golmet: modo não-interativo (--only) exige --yes — a remoção é destrutiva")
					return exitError(2)
				}
				for _, part := range strings.Split(only, ",") {
					p := strings.TrimSpace(part)
					if p == "" {
						continue
					}
					if !slices.Contains(uninstall.ValidLayerIDs(), p) {
						fmt.Fprintf(cmd.ErrOrStderr(), "golmet: camada %q não existe — válidas: %s\n",
							p, strings.Join(uninstall.ValidLayerIDs(), ", "))
						return exitError(2)
					}
					if !slices.Contains(selected, p) {
						selected = append(selected, p)
					}
				}
			} else {
				sel, err := uninstallWizardSelect(out, plan)
				if err != nil {
					return err
				}
				selected = sel
				// confirmação final (com aviso do pack, se presente)
				if !confirmUninstall(out, plan, selected) {
					fmt.Fprintln(out, "uninstall cancelado — nada foi apagado")
					return nil
				}
			}

			rep, err := uninstall.Run(plan, selected)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "golmet: %s\n", err.Error())
				return exitError(2)
			}
			if jsonOut {
				if encErr := newJSONEncoder(out).Encode(rep); encErr != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "golmet: falha ao serializar JSON: %s\n", encErr.Error())
				}
				return nil
			}
			fmt.Fprint(out, rep.Human())
			return nil
		},
	}
	cmd.Flags().StringVar(&engine, "engine", "opencode", "engine alvo (opencode nesta versão)")
	cmd.Flags().StringVar(&home, "home", "", "home do usuário (default: UserHomeDir — override p/ testes)")
	cmd.Flags().StringVar(&workspace, "workspace", ".", "workspace de contexto (reservado)")
	cmd.Flags().StringVar(&only, "only", "", "não-interativo: CSV das camadas a remover (ferramental,estado,packs,pessoal,nao-geridos,engine) — exige --yes")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "lista o que apagaria — apaga nada (a descoberta é sempre read-only; a bandeira é declarativa)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "saída estruturada (report do uninstall)")
	cmd.Flags().BoolVar(&yes, "yes", false, "não-interativo: assume sim na confirmação (obrigatório com --only)")
	return cmd
}

// uninstallWizardSelect — seleção numerada (padrão pickMCPs do install):
// todas as camadas com contagem real; seleção default = tudo MENOS packs
// (o pack vem DESELECIONADO e com o aviso do repo remoto).
func uninstallWizardSelect(out interface{ Write([]byte) (int, error) }, plan *uninstall.Plan) ([]string, error) {
	fmt.Fprintf(out, "\nCamadas do Golmet nesta máquina (%s) — ENTER aceita o default (pack fica FORA):\n\n", plan.ConfigRoot)
	var catalog []uninstall.Layer
	for _, l := range plan.Layers {
		catalog = append(catalog, l)
		mark := "[ ]"
		if l.Default {
			mark = "[x]"
		}
		fmt.Fprintf(out, "  %s %d. %s — %d alvo(s)\n      %s\n", mark, len(catalog), l.Label, l.Count, l.Hint)
		if l.ID == "packs" {
			fmt.Fprintf(out, "      ⚠️ APAGA APENAS O WORKING COPY LOCAL (packs/<id>/). O repositório remoto\n")
			fmt.Fprintf(out, "         NÃO é tocado — delete-o manualmente no GitHub se quiser remover de vez.\n")
		}
	}
	defaultList := uninstall.SelectedDefaults()
	fmt.Fprintf(out, "\nNúmeros das camadas a remover [default: %s]: ", strings.Join(defaultList, ", "))

	reader := bufio.NewReader(os.Stdin)
	answer, rerr := reader.ReadString('\n')
	if rerr != nil && answer == "" {
		fmt.Fprintf(out, "\nstdin fechado (EOF) — use --only <camadas> --yes para modo não-interativo\n")
		return nil, exitError(2)
	}
	answer = strings.TrimSpace(answer)
	var selected []string
	if answer == "" {
		selected = defaultList
	} else {
		for _, part := range strings.Split(answer, ",") {
			p := strings.TrimSpace(part)
			n := 0
			for _, ch := range p {
				if ch < '0' || ch > '9' {
					n = -1
					break
				}
				n = n*10 + int(ch-'0')
			}
			if n >= 1 && n <= len(catalog) {
				id := catalog[n-1].ID
				if !slices.Contains(selected, id) {
					selected = append(selected, id)
				}
			}
		}
		if len(selected) == 0 {
			fmt.Fprintf(out, "nenhuma camada válida selecionada — nada será apagado\n")
		}
	}
	return selected, nil
}

// confirmUninstall — confirmação final: resumo por camada + aviso do pack +
// "confirma a remoção?" (s/N). false = cancela sem apagar nada.
func confirmUninstall(out interface{ Write([]byte) (int, error) }, plan *uninstall.Plan, selected []string) bool {
	total := 0
	fmt.Fprintf(out, "\nVai remover:\n")
	for _, l := range plan.Layers {
		if !slices.Contains(selected, l.ID) {
			continue
		}
		total += l.Count
		fmt.Fprintf(out, "  • %s — %d alvo(s)\n", l.Label, l.Count)
		if l.ID == "packs" {
			fmt.Fprintf(out, "    ⚠️ Apaga APENAS o working copy local (packs/<id>/). O repositório remoto\n")
			fmt.Fprintf(out, "       (ex.: github.com/TechMinds-Group/TM.PACK.RDD) NÃO é tocado — delete-o\n")
			fmt.Fprintf(out, "       manualmente no GitHub se quiser remover de vez.\n")
		}
	}
	fmt.Fprintf(out, "\nTotal: %d alvo(s). Confirma a remoção? (s/N): ", total)
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "s" || answer == "sim" || answer == "y" || answer == "yes"
}
