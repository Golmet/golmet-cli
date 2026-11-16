package cli

import (
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"

	golmet "github.com/Golmet/golmet-cli"
	"github.com/Golmet/golmet-cli/internal/core/content"
	"github.com/spf13/cobra"
)

// templateAssets — nomes aceitos pelo `golmet template` → path no embed
// (CoreDataFS). Template é etapa de construção: vive no KIT, não se instala
// (o config carrega só os definitivos — o materialize NÃO deposita
// templates; ver content.nonDepositTemplates).
var templateAssets = map[string]string{
	"agents":           "coredata/templates/AGENTS.template.md",
	"client-profile":   "coredata/rules/client-profile.template.md",
	"operator-profile": "coredata/rules/operator-profile.template.md",
}

// newTemplateCmd — golmet template <nome> [--out <caminho>]. Emite o
// conteúdo direto do embed (LF canônico, sem BOM) para stdout, ou grava em
// arquivo com --out. É o caminho oficial do gm-setup obter o template no
// repair — sem arquivo permanente no config.
func newTemplateCmd() *cobra.Command {
	var outFile string
	cmd := &cobra.Command{
		Use:   "template <agents|client-profile|operator-profile>",
		Short: "Emite um template do kit (AGENTS.md, perfis de perfil) direto do embed — template vive no kit, não se instala",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			name := strings.ToLower(strings.TrimSpace(args[0]))
			path, ok := templateAssets[name]
			if !ok {
				names := make([]string, 0, len(templateAssets))
				for n := range templateAssets {
					names = append(names, n)
				}
				sort.Strings(names)
				fmt.Fprintf(cmd.ErrOrStderr(), "golmet: template %q não existe — válidos: %s\n",
					args[0], strings.Join(names, ", "))
				return exitError(2)
			}
			data, err := fs.ReadFile(golmet.CoreDataFS(), path)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "golmet: template %q ilegível no embed: %v\n", name, err)
				return exitError(2)
			}
			data = content.Normalize(data)
			if outFile != "" {
				if werr := os.WriteFile(outFile, data, 0o644); werr != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "golmet: falha ao gravar %s: %v\n", outFile, werr)
					return exitError(2)
				}
				fmt.Fprintf(out, "template %s gravado em %s (%d bytes)\n", name, outFile, len(data))
				return nil
			}
			_, _ = out.Write(data)
			return nil
		},
	}
	cmd.Flags().StringVar(&outFile, "out", "", "grava o conteúdo em arquivo (opcional — sem ela emite no stdout)")
	return cmd
}
