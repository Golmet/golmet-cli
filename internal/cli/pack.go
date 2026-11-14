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
			"Comandos: add, update, publish, list, search.\n\n" +
			"Arquétipos (kind do pack.yaml): overlay (padrão — cache somente-leitura,\n" +
			"provides em agents//skills/ sincronizados por overlay unidirecional) e\n" +
			"state (working copy Git bidirecional em <config>/packs/<id>: skills/ e\n" +
			"agents/ são materializadas; knowledge/, accumulation/ e rules/ vivem NO\n" +
			"clone e sobem via `pack publish`).\n\n" +
			"Convenção da comunidade: packs públicos do Golmet publicam no GitHub com o\n" +
			"tópico \"golmet-pack\" — não há registry central. `golmet pack search <termo>`\n" +
			"consulta a Search API do GitHub e encontra esses repositórios; instale com:\n" +
			"golmet pack add https://github.com/<owner>/<repo> --engine <opencode|claude-code>",
	}
	add := newPackAddCmd()
	update := newPackAddCmd()
	update.Use = "update <git-url-ou-diretorio>"
	update.Short = "Atualiza um pack sincronizando novamente sua fonte (state: commit do WIP + rebase)"
	cmd.AddCommand(add, update, newPackPublishCmd(), newPackListCmd(), newPackSearchCmd())
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
		Short: "Sincroniza e aplica um pack no config do engine (overlay ou state)",
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
			fmt.Fprintf(cmd.OutOrStdout(), "pack %s %s (%s) — %d arquivo(s) escrito(s), %d conflito(s) preservado(s)\n", rep.Spec.ID, rep.Spec.Version, rep.Spec.Kind, len(rep.Written), len(rep.Conflicts))
			for _, p := range rep.Conflicts {
				fmt.Fprintf(cmd.OutOrStdout(), "  conflito: %s\n", p)
			}
			printStateSync(cmd, rep)
			return nil
		},
	}
	cmd.Flags().StringVar(&engine, "engine", "", "engine destino: opencode | claude-code")
	cmd.Flags().StringVar(&home, "home", "", "home alternativo (testes)")
	cmd.Flags().BoolVar(&force, "force", false, "sobrescreve arquivos divergentes")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "saída estruturada")
	return cmd
}

// printStateSync — detalhe do working copy quando o pack é state (arquétipo
// bidirecional: onde está o clone, o que o sync commitou).
func printStateSync(cmd *cobra.Command, rep *pack.Report) {
	if rep.StateSync == nil {
		return
	}
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "  working copy (state): %s (branch %s)\n", rep.StateSync.Dir, rep.StateSync.Branch)
	if rep.StateSync.AutoCommit {
		fmt.Fprintf(out, "  acumulação local commitada (%d entrada(s) no status) — nada se perdeu\n", rep.StateSync.Committed)
	}
	if rep.StateSync.NoRemote {
		fmt.Fprintln(out, "  sem origin no working copy — sync apenas local")
	}
	if rep.StateSync.DryRun {
		fmt.Fprintln(out, "  dry-run: working copy não foi modificado")
	}
}

// packRootsForEngines — raízes packs/ de cada engine com config global
// (para operações de state pack sem --engine: publish e list).
func packRootsForEngines(home string) []string {
	var roots []string
	seen := map[string]bool{}
	for _, id := range coreinstall.Engines() {
		root, ok := coreinstall.EngineConfigRootFor(home, id)
		if !ok {
			continue
		}
		packsRoot := filepath.Join(root, "packs")
		if !seen[packsRoot] {
			seen[packsRoot] = true
			roots = append(roots, packsRoot)
		}
	}
	return roots
}

func newPackPublishCmd() *cobra.Command {
	var home, message string
	var jsonOut bool
	cmd := &cobra.Command{
		Use: "publish <git-url-ou-diretorio>", Args: cobra.ExactArgs(1),
		Short: "Sobe um state pack: commita o WIP do working copy e faz push para a origem",
		Long: "Sobe um state pack (arquétipo kind: state): commita as mudanças não-\n" +
			"commitadas do working copy (\"publish: <mensagem>\") e faz git push origin\n" +
			"<branch>. A fonte pode ser a URL origem do pack ou o caminho do próprio\n" +
			"working copy (<config>/packs/<id>). Push é MANUAL por design — o sync\n" +
			"(pack update) nunca publica por conta própria.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if home == "" {
				var err error
				home, err = os.UserHomeDir()
				if err != nil {
					return err
				}
			}
			rep, err := pack.PublishState(args[0], message, packRootsForEngines(home))
			if err != nil {
				return err
			}
			if jsonOut {
				return encodeJSON(cmd, rep)
			}
			if rep.Committed {
				fmt.Fprintf(cmd.OutOrStdout(), "pack %s — WIP commitado (publish: %s)\n", rep.Pack, message)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "pack %s — sem WIP a commitar\n", rep.Pack)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "push origin/%s ok (%s)\n", rep.Branch, rep.Dir)
			return nil
		},
	}
	cmd.Flags().StringVar(&home, "home", "", "home alternativo (testes)")
	cmd.Flags().StringVar(&message, "message", "acumulação local", "mensagem do commit de publicação")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "saída estruturada")
	return cmd
}

func newPackListCmd() *cobra.Command {
	var home string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Lista packs sincronizados (overlay no cache, state no working copy)",
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
			if err != nil && !os.IsNotExist(err) {
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
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", spec.ID, spec.Version, spec.Kind, e.Name())
				found++
			}
			stateRoots := packRootsForEngines(home)
			statePacks, err := pack.ListState(stateRoots)
			if err != nil {
				return err
			}
			for _, s := range statePacks {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s (working copy)\n", s.Spec.ID, s.Spec.Version, s.Spec.Kind, s.Path)
				found++
			}
			if found == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "nenhum pack sincronizado")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&home, "home", "", "home alternativo (testes)")
	return cmd
}
