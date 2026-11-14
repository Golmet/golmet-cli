package cli

import (
	"fmt"

	"github.com/Golmet/golmet-cli/internal/core/scaffold"
	"github.com/spf13/cobra"
)

// newInitCmd — golmet init [produto] [--org] [--into <container>]
func newInitCmd() *cobra.Command {
	var org bool
	var into string
	cmd := &cobra.Command{
		Use:   "init [produto]",
		Short: "Inicializa um workspace (golmet.yaml, charter, zonas)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) == 1 {
				name = args[0]
			}
			base, err := absCwd()
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "golmet: "+err.Error())
				return exitError(2)
			}
			rep, err := scaffold.Run(base, name, scaffold.Options{Org: org, Into: into})
			if err != nil {
				if rep != nil {
					printScaffold(cmd, rep)
				}
				fmt.Fprintln(cmd.ErrOrStderr(), "golmet: "+err.Error())
				return exitError(2)
			}
			printScaffold(cmd, rep)
			return nil
		},
	}
	cmd.Flags().BoolVar(&org, "org", false, "cria um container de organização (container: true — §2)")
	cmd.Flags().StringVar(&into, "into", "", "cria o produto dentro de um container existente")
	return cmd
}

func printScaffold(cmd *cobra.Command, rep *scaffold.Report) {
	out := cmd.OutOrStdout()
	for _, f := range rep.Files {
		switch {
		case f.Note != "":
			fmt.Fprintf(out, "  pulou (%s): %s\n", f.Note, f.Path)
		case f.Skipped:
			fmt.Fprintf(out, "  pulou (já existe): %s\n", f.Path)
		default:
			fmt.Fprintf(out, "  criou: %s\n", f.Path)
		}
	}
	label := "workspace"
	if rep.ContainerUpdated {
		label = "produto"
	}
	fmt.Fprintf(out, "%s %s inicializado em %s — %d criado(s), %d pulado(s)\n",
		label, rep.Name, rep.Root, rep.Created, rep.Skipped)
	if rep.ContainerUpdated {
		fmt.Fprintf(out, "container atualizado: products: [%s]\n", rep.ContainerProducts)
	}
}
