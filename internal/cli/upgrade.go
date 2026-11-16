package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/Golmet/golmet-cli/internal/core/upgrade"
	"github.com/spf13/cobra"
)

// Version é a versão do BINÁRIO, injetada via ldflags no release
// (-X github.com/Golmet/golmet-cli/internal/cli.Version=vX.Y.Z — o
// workflow .github/workflows/release.yml injeta a tag). Build local sem
// ldflags fica com "dev" — não-semver, o upgrade reporta update disponível
// sem crash. A versão do CONTEÚDO servido (banner/doctor/mcp) segue
// indexstate.KitVersion — identidades independentes por design.
var Version = "dev"

// o gm_upgrade no MCP precisa da MESMA versão do binário — o package
// mcpserve não pode importar cli (ciclo), então o upgrade.BinaryVersion
// recebe a versão injetada aqui, no load do package cli.
func init() {
	upgrade.BinaryVersion = Version
}

// newUpgradeCmd — golmet upgrade [--check]: self-update do binário a
// partir da última release do GitHub (assets assinados por checksums.txt).
func newUpgradeCmd() *cobra.Command {
	var checkOnly bool
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Self-update do binário a partir da última release do GitHub",
		Long: strings.Join([]string{
			"Self-update do binário golmet: consulta a última release de",
			"github.com/Golmet/golmet-cli, baixa o asset da plataforma corrente,",
			"valida o SHA-256 contra o checksums.txt da release (checksum ausente",
			"ou divergente ABORTA sem tocar no binário) e substitui o binário.",
			"",
			"Rollback: o binário anterior fica em <bin>.old — restaure renomeando",
			"de volta (o .old é substituído a cada upgrade, copie-o para um lugar",
			"seguro se quiser preservá-lo).",
			"",
			"Após atualizar o binário, re-materialize o conteúdo para o engine",
			"acompanhar o kit novo: golmet install --engine <id> --profile completo --yes",
			"(ou MCP gm_update com confirm=true).",
			"",
			"Respeita HTTP_PROXY/HTTPS_PROXY; GITHUB_TOKEN/GH_TOKEN é usado SE",
			"presente (rate limit da API) — nunca obrigatório.",
		}, "\n"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			res, err := upgrade.Run(context.Background(), upgrade.Options{
				CheckOnly: checkOnly,
				Out:       out,
			}, Version)
			if res != nil {
				if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
					if encErr := newJSONEncoder(out).Encode(res.Check); encErr != nil {
						fmt.Fprintln(cmd.ErrOrStderr(), "golmet: falha ao serializar JSON: "+encErr.Error())
					}
				}
			}
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "golmet: "+err.Error())
				return exitError(2)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&checkOnly, "check", false, "read-only: reporta versão local vs latest sem baixar nada")
	cmd.Flags().Bool("json", false, "saída estruturada do check (versão local, latest, has_update)")
	return cmd
}
