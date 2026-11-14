// Package cli define os comandos cobra do golmet. Comandos são THIN —
// toda a lógica vive em internal/core.
package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/Golmet/golmet-cli/internal/core"
	"github.com/Golmet/golmet-cli/internal/core/scan"
	"github.com/spf13/cobra"
)

// exitError carrega o código de saída até o main (0 ok, 1 violações
// hard, 2 erro de uso/config).
type exitError int

func (e exitError) Error() string { return fmt.Sprintf("exit code %d", int(e)) }

// ExitCodeOf extrai o código de saída de um error do CLI.
func ExitCodeOf(err error) (int, bool) {
	var e exitError
	if errors.As(err, &e) {
		return int(e), true
	}
	return 0, false
}

// newRootCmd monta a árvore de comandos (exposta para os testes do CLI).
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "golmet",
		Short:   "The stone guardian of your product's documented truth",
		Long:    "golmet valida workspaces contra o FORMATO v1: requisitos, specs e evidência congruentes com o código.",
		Version: Version, // binVersion — ldflags no release; "dev" no build local
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(
		newValidateCmd(),
		newInitCmd(),
		newAdoptCmd(),
		newStatusCmd(),
		newDoctorCmd(),
		newInstallCmd(),
		newUpdateCmd(),
		newUpgradeCmd(),
		newPackCmd(),
		newToolsCmd(),
		newMcpCmd(),
		newIndexCmd(),
		newNextCmd(),
		newEvalCmd(),
		newDBCmd(),
	)
	return root
}

// Execute roda o comando raiz.
func Execute() error {
	return newRootCmd().Execute()
}

// newValidateCmd — golmet validate [workspace] [--json] [--tolerant] [--strict] [--fix]
func newValidateCmd() *cobra.Command {
	var jsonOut, tolerant, strict, fix bool
	cmd := &cobra.Command{
		Use:   "validate [workspace]",
		Short: "Valida o workspace contra o FORMATO v1 (H1–H10, S1–S5)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if tolerant && strict && cmd.Flags().Changed("strict") {
				fmt.Fprintln(cmd.ErrOrStderr(), "golmet: flags --tolerant e --strict são exclusivas")
				return exitError(2)
			}
			ws := "."
			if len(args) == 1 {
				ws = args[0]
			}
			res, err := scan.Run(ws, scan.Options{Tolerant: tolerant, WriteIndex: true, ForceIndex: fix})
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "golmet: "+err.Error())
				return exitError(2)
			}
			out := cmd.OutOrStdout()
			if jsonOut {
				items := append(append([]core.GolmetError{}, res.Errors...), res.Warnings...)
				enc := newJSONEncoder(out)
				if err := enc.Encode(items); err != nil {
					fmt.Fprintln(cmd.ErrOrStderr(), "golmet: falha ao serializar JSON: "+err.Error())
					return exitError(2)
				}
			} else {
				printHuman(out, res)
				if res.IndexWritten {
					fmt.Fprintln(out, "índice atualizado (golmet.index.json)")
				}
			}
			if len(res.Errors) > 0 {
				return exitError(1)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "saída estruturada §11 (array de erros)")
	cmd.Flags().BoolVar(&tolerant, "tolerant", false, "H4/H7 degradam para aviso com worklist")
	cmd.Flags().BoolVar(&strict, "strict", true, "bloqueia H4/H7 (default; affirmativo)")
	cmd.Flags().BoolVar(&fix, "fix", false, "recompõe o golmet.index.json mesmo com erros (recuperação pós-merge — §8)")
	return cmd
}

func printHuman(w io.Writer, res *scan.Result) {
	for _, e := range res.Errors {
		printFinding(w, e, "")
	}
	for _, wn := range res.Warnings {
		printFinding(w, wn, "AVISO ")
	}
	if len(res.Errors) == 0 && len(res.Warnings) == 0 {
		fmt.Fprintln(w, "ok — workspace conforme (FORMATO v1)")
	}
	fmt.Fprintf(w, "%d erro(s), %d aviso(s) — %d arquivo(s), RN %d, PEND %d, CR %d, ADR %d, CAND %d, Feature %d, Task %d, Spec %d\n",
		len(res.Errors), len(res.Warnings), res.Files,
		res.Counts["rn"], res.Counts["pend"], res.Counts["cr"], res.Counts["adr"],
		res.Counts["cand"], res.Counts["feature"], res.Counts["task"], res.Counts["spec"])
}

func printFinding(w io.Writer, e core.GolmetError, prefix string) {
	loc := e.File
	if e.Line > 0 {
		loc += ":" + itoa(e.Line)
	}
	id := ""
	if e.ID != "" {
		id = " " + e.ID
	}
	fmt.Fprintf(w, "%s[%s] %s%s [%s] %s\n", prefix, loc, e.Entity, id, e.Rule, e.Message)
	if e.Fix != "" {
		fmt.Fprintf(w, "  fix: %s\n", e.Fix)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
