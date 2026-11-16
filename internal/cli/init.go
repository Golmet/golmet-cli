package cli

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/Golmet/golmet-cli/internal/core/container"
	"github.com/Golmet/golmet-cli/internal/core/orgs"
	"github.com/Golmet/golmet-cli/internal/core/scaffold"
	"github.com/spf13/cobra"
)

// newInitCmd — golmet init [produto] [--org] [--into <container>] [--yes]
// [--plain]. SEM flags e SEM arg (init bare): wizard interativo em TTY —
// pergunta o que inicializar (Organização / workspace em Organização /
// raiz livre) e confirma antes de criar (M22 — FORMAT §2); nunca mais
// cria no cwd silenciosamente. Sem TTY e sem --plain: fail-closed com o
// comando a rodar (scripts/CI exigem --org ou --into + --yes).
func newInitCmd() *cobra.Command {
	var org bool
	var into string
	var yes, plain bool
	cmd := &cobra.Command{
		Use:   "init [produto]",
		Short: "Inicializa um workspace (golmet.yaml, charter, zonas) — sem flags = wizard interativo",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			name := ""
			if len(args) == 1 {
				name = args[0]
			}
			explicit := org || into != "" || name != ""
			// init bare (sem flags e sem arg): o destino não é conhecido —
			// o wizard pergunta; script/CI fail-closed com a remediação.
			if !explicit {
				if !isTTY() && !plain {
					fmt.Fprintln(cmd.ErrOrStderr(), "golmet: 'golmet init' sem flags não cria nada em script/CI — nada foi escrito")
					fmt.Fprintln(cmd.ErrOrStderr(), "     fix: rode com --org ou --into <container> e --yes; ou em terminal interativo (o wizard pergunta antes de criar); --plain ativa o wizard numerado em stdin")
					return exitError(2)
				}
				base, berr := absCwd()
				if berr != nil {
					fmt.Fprintln(cmd.ErrOrStderr(), "golmet: "+berr.Error())
					return exitError(2)
				}
				// escada do install (ttyWizardMode): TTY+VT → wizard VISUAL
				// com setas; TTY sem VT (conhost legado) → numerado com
				// degradação elegante; sem TTY + --plain → numerado.
				visual := isTTY() && enableVT()
				if isTTY() && !visual {
					fmt.Fprintln(out, "ℹ️ terminal sem suporte a VT — modo numerado (use Windows Terminal para o wizard visual)")
				}
				w := &initWizard{out: out, in: stdinReader(), cwd: base, tty: isTTY(), raw: visual}
				opt, baseDir, wname, werr := runInitWizard(w, yes, visual)
				if werr != nil {
					if errors.Is(werr, errWizardCancelled) {
						fmt.Fprintln(out, "cancelado — nada foi criado")
						return nil
					}
					fmt.Fprintln(cmd.ErrOrStderr(), "golmet: "+werr.Error())
					return exitError(2)
				}
				if opt == nil {
					fmt.Fprintln(out, "cancelado — nada foi criado")
					return nil
				}
				rep, err := scaffold.Run(baseDir, wname, *opt)
				return finishInit(cmd, opt.Org, rep, err)
			}
			base, err := absCwd()
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "golmet: "+err.Error())
				return exitError(2)
			}
			// --org: pré-checagem do nome no catálogo global — o kit nunca
			// cria organização que não possa registrar (nome duplicado com
			// path diferente = erro ANTES de criar qualquer coisa).
			if org {
				root := base
				if name != "" {
					root = filepath.Join(base, name)
				}
				if cerr := checkOrgCatalogName(root); cerr != nil {
					fmt.Fprintln(cmd.ErrOrStderr(), "golmet: "+cerr.Error())
					return exitError(2)
				}
			}
			rep, err := scaffold.Run(base, name, scaffold.Options{Org: org, Into: into})
			return finishInit(cmd, org, rep, err)
		},
	}
	cmd.Flags().BoolVar(&org, "org", false, "cria um container de organização (container: true — §2)")
	cmd.Flags().StringVar(&into, "into", "", "cria o produto dentro de um container existente")
	cmd.Flags().BoolVar(&yes, "yes", false, "pula a confirmação do wizard (script/CI — requer --org ou --into)")
	cmd.Flags().BoolVar(&plain, "plain", false, "wizard em modo numerado lendo stdin (sem TTY)")
	return cmd
}

// finishInit centraliza a saída do init (wizard e flags usam o mesmo
// printScaffold e o mesmo tratamento de erro — exit 2 com o relatório
// parcial quando houver) e REGISTRA a organização no catálogo global
// quando o scaffold criou/adotou um container (--org).
func finishInit(cmd *cobra.Command, isOrg bool, rep *scaffold.Report, err error) error {
	if err != nil {
		if rep != nil {
			printScaffold(cmd, rep)
		}
		fmt.Fprintln(cmd.ErrOrStderr(), "golmet: "+err.Error())
		return exitError(2)
	}
	printScaffold(cmd, rep)
	if isOrg && rep != nil {
		msg, regErr := registerOrgCatalog(rep)
		if regErr != nil {
			fmt.Fprintln(cmd.ErrOrStderr(), "golmet: "+regErr.Error())
			return exitError(2)
		}
		if msg != "" {
			fmt.Fprintln(cmd.OutOrStdout(), msg)
		}
		// M33 — a Organização (org-repo) é o repositório de versionamento:
		// o hint orienta o passo de Git do operador (o kit NUNCA roda
		// git init por conta própria)
		if !adoptedOrg(rep) {
			fmt.Fprintln(cmd.OutOrStdout(), "Organização criada. Ela foi feita para ser o seu repositório de versionamento: rode `git init` na raiz e faça o primeiro commit (o .gitignore já protege a zona de trabalho).")
		}
	}
	return nil
}

// registerOrgCatalog registra a organização no catálogo global
// (~/.golmet/organizations.json) após o scaffold --org bem-sucedido —
// criação nova OU adoção (golmet.yaml existente íntegro; a idempotência
// do scaffold é intacta — nada é sobrescrito). O registro exige container
// íntegro (a fonte da verdade da org é o golmet.yaml); declaração
// ausente/ilegível/não-container → o init segue SEM registrar com a nota
// explicando (o catálogo não adentra estados ambíguos).
func registerOrgCatalog(rep *scaffold.Report) (string, error) {
	home, herr := userHomeFn()
	if herr != nil {
		return "", fmt.Errorf("home do operador indisponível — o catálogo global de organizações não pôde ser atualizado")
	}
	fields, isContainer, cerr := container.Config(rep.Root)
	if cerr != nil || !isContainer || fields["format"].Str != "1" {
		return fmt.Sprintf("⚠️ golmet.yaml de %s não declara Organização íntegra (container: true, format: 1) — catálogo global NÃO atualizado", rep.Root), nil
	}
	if err := orgs.Register(home, rep.Name, rep.Root); err != nil {
		return "", err
	}
	if adoptedOrg(rep) {
		return fmt.Sprintf("Organização %s registrada no catálogo global (adoção — nada foi sobrescrito)", rep.Name), nil
	}
	return fmt.Sprintf("Organização %s registrada no catálogo global (~/.golmet/organizations.json)", rep.Name), nil
}

// adoptedOrg — o golmet.yaml já existia (scaffold pulou) = adoção de
// container nascido fora do kit.
func adoptedOrg(rep *scaffold.Report) bool {
	for _, f := range rep.Files {
		if f.Path == "golmet.yaml" && f.Skipped {
			return true
		}
	}
	return false
}

// printScaffold exibe o relatório do scaffold (arquivos criados/pulados).
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
	if rep.VersioningModel != "" {
		fmt.Fprintf(out, "modelo de versionamento da organização: %s\n", rep.VersioningModel)
		if rep.ProductZoneSaved {
			fmt.Fprintf(out, "produto SEM .golmet/ própria — a zona do kit é a da organização (org-repo)\n")
		}
	}
}
