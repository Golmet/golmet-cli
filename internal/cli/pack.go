package cli

import (
	"fmt"
	"os"
	"path/filepath"

	coreinstall "github.com/Golmet/golmet-cli/internal/core/install"
	"github.com/Golmet/golmet-cli/internal/core/pack"
	"github.com/spf13/cobra"
)

func newPackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pack",
		Short: "Gerencia packs de skills e agents (camada L2)",
		Long: "Gerencia packs de skills e agents (camada L2).\n\n" +
			"Comandos: add, update, list, search.\n\n" +
			"Convenção da comunidade: packs públicos do Golmet publicam no GitHub com o\n" +
			"tópico \"golmet-pack\" — não há registry central. `golmet pack search <termo>`\n" +
			"consulta a Search API do GitHub e encontra esses repositórios; instale com:\n" +
			"golmet pack add https://github.com/<owner>/<repo> --engine <opencode|claude-code>",
	}
	add := newPackAddCmd()
	update := newPackAddCmd()
	update.Use = "update <git-url-ou-diretorio>"
	update.Short = "Atualiza um pack sincronizando novamente sua fonte"
	cmd.AddCommand(add, update, newPackListCmd(), newPackSearchCmd())
	return cmd
}

func newPackSearchCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use: "search [termo]", Args: cobra.MaximumNArgs(1),
		Short: "Busca packs públicos da comunidade no tópico GitHub golmet-pack",
		RunE: func(cmd *cobra.Command, args []string) error {
			term := ""
			if len(args) > 0 {
				term = args[0]
			}
			results, err := pack.Search(term, pack.PackSearchOptions{})
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "golmet: "+err.Error())
				return exitError(2)
			}
			if jsonOut {
				return encodeJSON(cmd, results)
			}
			pack.RenderSearchResults(cmd.OutOrStdout(), results)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "saída estruturada (array de resultados)")
	return cmd
}

func newPackAddCmd() *cobra.Command {
	var engine, home string
	var force, jsonOut bool
	cmd := &cobra.Command{
		Use: "add <git-url-ou-diretorio>", Args: cobra.ExactArgs(1),
		Short: "Sincroniza e aplica um pack no config do engine",
		RunE: func(cmd *cobra.Command, args []string) error {
			if home == "" {
				var err error
				home, err = os.UserHomeDir()
				if err != nil {
					return err
				}
			}
			if engine == "" {
				return fmt.Errorf("--engine é obrigatório (opencode ou claude-code)")
			}
			adapter, ok := coreinstall.LookupEngine(engine)
			if !ok || adapter.SkillsDir(home) == "" {
				return fmt.Errorf("engine %q não possui suporte de conteúdo", engine)
			}
			rep, err := pack.Add(home, filepath.Dir(adapter.SkillsDir(home)), engine, args[0], force)
			if err != nil {
				return err
			}
			if jsonOut {
				return encodeJSON(cmd, rep)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "pack %s %s — %d arquivo(s) escrito(s), %d conflito(s) preservado(s)\n", rep.Spec.ID, rep.Spec.Version, len(rep.Written), len(rep.Conflicts))
			for _, p := range rep.Conflicts {
				fmt.Fprintf(cmd.OutOrStdout(), "  conflito: %s\n", p)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&engine, "engine", "", "engine destino: opencode | claude-code")
	cmd.Flags().StringVar(&home, "home", "", "home alternativo (testes)")
	cmd.Flags().BoolVar(&force, "force", false, "sobrescreve arquivos divergentes")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "saída estruturada")
	return cmd
}

func newPackListCmd() *cobra.Command {
	var home string
	return &cobra.Command{
		Use: "list", Short: "Lista packs sincronizados no cache local",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if home == "" {
				var err error
				home, err = os.UserHomeDir()
				if err != nil {
					return err
				}
			}
			root := filepath.Join(home, ".golmet", "packs")
			entries, err := os.ReadDir(root)
			if os.IsNotExist(err) {
				fmt.Fprintln(cmd.OutOrStdout(), "nenhum pack sincronizado")
				return nil
			}
			if err != nil {
				return err
			}
			found := 0
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				data, err := os.ReadFile(filepath.Join(root, e.Name(), "pack.yaml"))
				if err != nil {
					continue
				}
				spec, err := pack.ParseSpec(data)
				if err != nil {
					continue
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", spec.ID, spec.Version, e.Name())
				found++
			}
			if found == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "nenhum pack sincronizado")
			}
			return nil
		},
	}
}
