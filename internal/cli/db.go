package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Golmet/golmet-cli/internal/core/db"
	"github.com/spf13/cobra"
)

// newDBCmd — golmet db [drift]: serviços de banco de dados (camada do gm-dba)
func newDBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Serviços de banco de dados (camada do gm-dba)",
	}
	cmd.AddCommand(newDBDriftCmd())
	return cmd
}

func newDBDriftCmd() *cobra.Command {
	var (
		repo    string
		dialect string
		asJSON  bool
		write   bool
	)
	cmd := &cobra.Command{
		Use:   "drift [workspace]",
		Short: "Compara DATA_MODEL ↔ migrations/snapshot e reporta divergências",
		Long: `Compara o DATA_MODEL.md documentado com o schema efetivo das migrations
(ModelSnapshot EF para dotnet) e reporta as divergências em três direções:
documentado sem schema · schema sem documentação · naming entidade↔tabela.

A interpretação e a correção são papel do gm-dba (regra 9: drift é
reportado, nunca auto-corrigido). Schema vivo: conecte via MCP postgres/sqlite.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws := "."
			if len(args) > 0 {
				ws = args[0]
			}
			if dialect != "dotnet" {
				fmt.Fprintf(cmd.ErrOrStderr(), "golmet: dialect %q ainda não suportado — disponível: dotnet\n", dialect)
				return exitError(2)
			}
			rep, err := db.Drift(ws, repo)
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "golmet: "+err.Error())
				return exitError(2)
			}
			out := cmd.OutOrStdout()
			if asJSON {
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(rep)
			}
			fmt.Fprint(out, rep.RenderMarkdown())
			if write {
				path := filepath.Join(ws, "specs", "DB_DRIFT.md")
				if err := os.WriteFile(path, []byte(rep.RenderMarkdown()), 0o644); err != nil {
					fmt.Fprintln(cmd.ErrOrStderr(), "golmet: erro escrevendo DB_DRIFT.md: "+err.Error())
					return exitError(2)
				}
				fmt.Fprintf(out, "\nDB_DRIFT.md escrito em %s\n", path)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&repo, "repo", "", "caminho do repo com as migrations (obrigatório)")
	cmd.Flags().StringVar(&dialect, "dialect", "dotnet", "dialect das migrations: dotnet")
	cmd.Flags().BoolVar(&asJSON, "json", false, "saída estruturada")
	cmd.Flags().BoolVar(&write, "write", false, "escreve o relatório em specs/DB_DRIFT.md do workspace")
	if err := cmd.MarkFlagRequired("repo"); err != nil {
		panic(err)
	}
	return cmd
}
