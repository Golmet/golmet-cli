package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Golmet/golmet-cli/internal/core/install"
	"github.com/spf13/cobra"
)

// newInstallCmd — golmet install: onboarding em um comando (engine +
// skills gm-* + MCPs + doctor). Sem flags = wizard interativo hand-rolled
// (stdin, sem lib de TUI); com flags = scriptable/CI.
func newInstallCmd() *cobra.Command {
	var engineID, profile, mcpsCSV, workspace string
	var yes, jsonOut, force, full, plain, listEngines, listMcps, noBanner bool
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Onboarding do kit: engine + skills gm-* + MCPs + doctor (um comando)",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			// listagens saem antes de qualquer execução (sem banner — queries limpas)
			if listEngines || listMcps {
				return runInstallList(cmd, listEngines, listMcps)
			}

			req := install.Options{Force: force, Workspace: workspace}
			req.Full = full
			// dispatch do wizard: TTY+VT → seleção de teclas; --plain ou VT
			// indisponível (conhost legado) → modo numerado; não-TTY sem
			// flags → erro com o comando de flags
			interactive := !yes && engineID == ""
			mode := ""
			if interactive {
				mode = ttyWizardMode(isTTY(), plain, enableVT())
			}

			// banner do onboarding (install only; --json/listagens nunca):
			// layout Copilot CLI (wordmark âmbar + golem + tagline/versão,
			// ~12 rows fixas — sem cascata de altura). TTY+VT: a composição
			// vai EMBUTIDA no frame do wizard — pre-renderizada UMA vez e o
			// full-clear redesenha banner+passo a cada tecla. Pipe/plain/
			// plain-auto: impressa UMA vez (sem clear no raw — persiste).
			var banner string
			if !jsonOut && !noBanner {
				banner = composeBanner(isTTY() && enableVT() && os.Getenv("NO_COLOR") == "" && !plain, terminalWidth())
				if mode != "tty" {
					fmt.Fprint(out, banner)
					banner = ""
				}
			}

			if interactive {
				switch mode {
				case "tty":
					// wizard de teclas (raw mode) — modelo puro + render fino
					model := install.NewWizardModel(mustHome(), workspace)
					model, cancelled, err := runInteractiveTTY(model, banner)
					if err != nil {
						fmt.Fprintln(cmd.ErrOrStderr(), "golmet: "+err.Error())
						return exitError(2)
					}
					if cancelled {
						fmt.Fprintln(out, "cancelado — nada foi aplicado")
						return nil
					}
					req = model.ToOptions()
					req.Force = force
					req.Full = full
				default:
					if mode == "plain-auto" {
						// conhost legado sem VT: o clear ANSI não seria
						// interpretado (frames empilham) — degradação elegante
						fmt.Fprintln(out, "ℹ️ terminal sem suporte a VT — modo plain (use Windows Terminal para o wizard completo)")
					}
					// não-TTY sem flags suficientes → erro com o comando exato
					// (--plain força o modo numerado, que também lê stdin)
					if nerr := install.CheckInteractive(engineID, isTTY(), plain); nerr != nil {
						fmt.Fprintln(cmd.ErrOrStderr(), "golmet: "+nerr.Error())
						if fix := install.NonTTYFix(nerr); fix != "" {
							fmt.Fprintln(cmd.ErrOrStderr(), "     fix: "+fix)
						}
						return exitError(2)
					}
					// modo numerado (--plain): reusa o picker clássico; a
					// detecção de já-registrados alimenta o badge da lista.
					// Ordem: perfil (decisão macro) → engine → MCPs. Com
					// perfil completo, a lista de engines já vem FILTRADA
					// para os compatíveis (skills nativas).
					w := &installWizard{out: out, in: bufio.NewReader(os.Stdin), tty: isTTY()}
					var werr error
					if req.Profile, werr = w.pickProfile(); werr != nil {
						return installExit(cmd, werr)
					}
					if req.EngineID, werr = w.pick("engine", engineChoices(req.Profile, w.home()), func(id string) string {
						for _, info := range install.DetectEngines(w.home(), ".") {
							if info.ID == id {
								return "[detectado]"
							}
						}
						return "[ausente  ]"
					}); werr != nil {
						return installExit(cmd, werr)
					}
					if req.MCPs, werr = w.pickMCPs(install.InstalledMCPs(w.home(), req.Workspace, req.EngineID)); werr != nil {
						return installExit(cmd, werr)
					}
					if req.Workspace == "" {
						req.Workspace = "."
					}
					ok, werr := w.confirm(req)
					if werr != nil {
						return installExit(cmd, werr)
					}
					if !ok {
						fmt.Fprintln(out, "cancelado — nada foi aplicado")
						return nil
					}
				}
			} else {
				req.EngineID = engineID
				req.Profile = profile
				req.Full = full
				if mcpsCSV != "" {
					req.MCPs = strings.Split(mcpsCSV, ",")
				}
			}

			// A normalização do perfil (essencial/completo + aliases base/
			// full e completo ≡ Full) é do install.Run — o wizard (numerado
			// ou TTY) e as flags passam pelo MESMO canal, sem alias local.

			// execução progressiva: TTY (spinner braille por passo, estilo
			// npm/cargo) ou texto plano (pipe/CI — mesmas linhas sem ANSI);
			// --json é o caminho da máquina: sem progresso, relatório intacto
			ttyExec := !jsonOut && !plain && isTTY()
			if ttyExec {
				fmt.Fprintln(out, "◆ Instalando...")
			}
			prog := &installProgress{out: out, tty: ttyExec, colors: newTTYColors(ttyExec)}
			if !jsonOut {
				req.Progress = prog.consume
			}

			res, err := install.Run(req)
			if err != nil {
				prog.abort() // spinner ativo não disputa a linha com o erro
				fmt.Fprintln(cmd.ErrOrStderr(), "golmet: "+err.Error())
				return exitError(2)
			}
			if jsonOut {
				if err := newJSONEncoder(out).Encode(res.Report); err != nil {
					fmt.Fprintln(cmd.ErrOrStderr(), "golmet: falha ao serializar JSON: "+err.Error())
					return exitError(2)
				}
			} else if ttyExec {
				printInstallSummaryTTY(out, res)
			} else {
				printInstallReport(out, res.Report)
			}
			if res.Partial {
				return exitError(1)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&engineID, "engine", "", "engine de destino: opencode | claude-code | vscode-copilot | cursor | gemini-cli | codex | copilot-cli")
	cmd.Flags().StringVar(&profile, "profile", "", "perfil do kit: essencial (default — kit do repositório) | completo (experiência completa na máquina; atalho: --full)")
	cmd.Flags().StringVar(&mcpsCSV, "mcp", "", "MCPs a registrar, separados por vírgula (golmet é sempre incluído)")
	cmd.Flags().StringVar(&workspace, "workspace", "", "workspace para o doctor final (default: atual)")
	cmd.Flags().BoolVar(&yes, "yes", false, "não-interativo: executa direto com as flags informadas")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "saída estruturada (relatório do install)")
	cmd.Flags().BoolVar(&force, "force", false, "sobrescreve skills/blocos divergentes no engine")
	cmd.Flags().BoolVar(&full, "full", false, "materializa o workflow FULL (127 componentes embarcados) no engine")
	cmd.Flags().BoolVar(&plain, "plain", false, "wizard em modo numerado (fallback sem teclas de seta)")
	cmd.Flags().BoolVar(&listEngines, "list-engines", false, "lista os engines suportados (detectado/ausente) e sai")
	cmd.Flags().BoolVar(&listMcps, "list-mcps", false, "lista o catálogo de MCPs e sai")
	cmd.Flags().BoolVar(&noBanner, "no-banner", false, "pula o banner visual do onboarding")
	return cmd
}

func installExit(cmd *cobra.Command, err error) error {
	if errors.Is(err, errWizardCancelled) {
		// ESC/CTRL+C/EOF no TTY = cancelamento limpo (o padrão único de
		// wizard_read.go) — NUNCA a mensagem de script/CI aqui
		fmt.Fprintln(cmd.OutOrStdout(), "cancelado — nada foi aplicado")
		return nil
	}
	fmt.Fprintln(cmd.ErrOrStderr(), "golmet: "+err.Error())
	return exitError(2)
}

// ttyWizardMode decide o modo do dispatch interativo a partir de
// (terminal, --plain, VT processing):
//   - "tty": terminal + VT → wizard de teclas (setas/full-clear)
//   - "plain-auto": terminal SEM VT (conhost legado) → modo numerado
//     automático com a linha ℹ️ — o clear ANSI não seria interpretado
//   - "plain": --plain ou não-TTY (guard CheckInteractive decide)
func ttyWizardMode(tty, plain, vt bool) string {
	if !tty || plain {
		return "plain"
	}
	if vt {
		return "tty"
	}
	return "plain-auto"
}

// engineChoices devolve a lista de engines do passo engine do wizard
// numerado: com perfil completo, só os compatíveis (skills nativas —
// derive via SkillsDir, nunca hardcode); com essencial, todos os 7.
func engineChoices(profile, home string) []string {
	ids := install.Engines()
	if profile != install.ProfileCompleto {
		return ids
	}
	var out []string
	for _, id := range ids {
		if install.EngineSupportsCompleto(id, home) {
			out = append(out, id)
		}
	}
	return out
}

// printInstallSummaryTTY — fechamento do human path TTY (estilo npm/cargo):
// resumo de contagens + pendências/falhas com a instrução completa (o passo
// resumiu). O relatório detalhado por passo segue o padrão nas rotas
// --json (intacto) e texto plano (printInstallReport).
func printInstallSummaryTTY(out interface{ Write([]byte) (int, error) }, res *install.Result) {
	rep := res.Report
	w := terminalWidth() - 8
	if w < 40 {
		w = 40
	}
	oks := 0
	for _, m := range rep.MCPs {
		if m.Status == install.OutcomeRegistered || m.Status == install.OutcomeUnchanged {
			oks++
		}
	}
	fmt.Fprintf(out, "✅ Instalação concluída — %d ok · %d pendente(s)\n", oks, len(rep.Pending))
	// destino do FULL declarado no fechamento (fallback zona do workspace)
	if rep.FullZone == install.FullZoneWorkspace {
		fmt.Fprintf(out, "   ℹ workflow FULL na zona do workspace (%s) — engine %s não tem skills nativas\n", rep.FullZonePath, rep.Engine)
	}
	// falhas e instruções manuais com o texto completo
	for _, m := range rep.MCPs {
		switch m.Status {
		case install.OutcomeManual:
			fmt.Fprintf(out, "   ℹ mcp %s — %s\n", m.Name, truncateLine(m.Detail, w))
		case install.OutcomeDiverged, install.OutcomeError, install.OutcomeUnsupported:
			fmt.Fprintf(out, "   ❌ mcp %s — %s\n", m.Name, truncateLine(m.Detail, w))
			if m.Fix != "" {
				fmt.Fprintf(out, "     fix: %s\n", truncateLine(m.Fix, w))
			}
		}
	}
	// pendências (env/param/binário) com o fix
	for _, p := range rep.Pending {
		d := p.Fix
		if d == "" {
			d = p.Detail
		}
		fmt.Fprintf(out, "   ⚠ %s — %s\n", p.Name, truncateLine(d, w))
	}
	// avisos/falhas do doctor merecem o texto completo (ok limpo não repete)
	if d := rep.Doctor; d != nil && (d.Warns > 0 || d.Fails > 0) {
		fmt.Fprintf(out, "\n%s", d.Human)
	}
	fmt.Fprintf(out, "\ninstall concluído — passo seguinte: rode 'golmet validate' no workspace após o primeiro uso\n")
}

func runInstallList(cmd *cobra.Command, engines, mcps bool) error {
	out := cmd.OutOrStdout()
	if engines {
		fmt.Fprintln(out, "engines suportados:")
		for _, info := range install.DetectEngines(mustHome(), ".") {
			mark := "[ausente  ]"
			if info.Detected {
				mark = "[detectado]"
			}
			fmt.Fprintf(out, "  %s %s\n", mark, info.ID)
		}
	}
	if mcps {
		fmt.Fprintln(out, "catálogo de MCPs:")
		for _, def := range install.Catalog() {
			tag := "opcional   "
			if def.Required {
				tag = "obrigatório"
			} else if def.Recommended {
				tag = "recomendado"
			}
			envInfo := ""
			if len(def.EnvRequired) > 0 {
				envInfo = " · env: " + strings.Join(def.EnvRequired, ",")
			}
			if len(def.EnvOptional) > 0 {
				envInfo += " · env opcional: " + strings.Join(def.EnvOptional, ",")
			}
			fmt.Fprintf(out, "  %s %-10s %-10s%s\n", tag, def.Type, def.Name, envInfo)
		}
	}
	return nil
}

func mustHome() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return h
}

// printInstallReport — progresso por passo + relatório final.
func printInstallReport(out interface{ Write([]byte) (int, error) }, rep *install.Report) {
	fmt.Fprintf(out, "golmet %s — install [%s · perfil %s]\n", rep.Version, rep.Engine, rep.Profile)

	if rep.SkillsSupported {
		if len(rep.SkillsWritten) > 0 {
			fmt.Fprintf(out, "✅ skills gm-* — %d escrita(s) em %s\n", len(rep.SkillsWritten), rep.SkillsDir)
		}
		for _, p := range rep.SkillsPreserved {
			fmt.Fprintf(out, "⚠️  skill %s preservada (modificada localmente — use --force p/ sobrescrever)\n", p)
		}
	} else {
		fmt.Fprintf(out, "ℹ️  engine %s não tem sistema de skills — skills ficam no workspace golmet (zona .golmet/), não no engine\n", rep.Engine)
	}

	// destino do FULL declarado: engine sem skills nativas recebe o FULL na
	// zona do workspace — o usuário não pode achar que foi para o config global
	if rep.FullZone == install.FullZoneWorkspace {
		fmt.Fprintf(out, "ℹ️  workflow FULL materializado na zona do workspace (%s) — engine %s não tem skills nativas\n", rep.FullZonePath, rep.Engine)
	}

	for _, m := range rep.MCPs {
		switch m.Status {
		case install.OutcomeRegistered:
			fmt.Fprintf(out, "✅ mcp %s — %s\n", m.Name, m.Detail)
		case install.OutcomeUnchanged:
			fmt.Fprintf(out, "✅ mcp %s — %s\n", m.Name, m.Detail)
		case install.OutcomeManual:
			fmt.Fprintf(out, "ℹ️  mcp %s — registro manual necessário: %s\n", m.Name, m.Detail)
		case install.OutcomeDiverged, install.OutcomeError, install.OutcomeUnsupported, install.OutcomeSkipped:
			fmt.Fprintf(out, "❌ mcp %s — %s\n", m.Name, m.Detail)
			if m.Fix != "" {
				fmt.Fprintf(out, "     fix: %s\n", m.Fix)
			}
		}
	}
	for _, p := range rep.Pending {
		fmt.Fprintf(out, "⚠️  pendente: %s — %s\n", p.Name, p.Detail)
		if p.Fix != "" {
			fmt.Fprintf(out, "     fix: %s\n", p.Fix)
		}
	}

	if rep.Doctor != nil {
		fmt.Fprintf(out, "\n%s", rep.Doctor.Human)
	}
	fmt.Fprintf(out, "\ninstall concluído — passo seguinte: rode 'golmet validate' no workspace após o primeiro uso\n")
}

// installWizard — prompts numerados lendo stdin (hand-rolled). tty é o
// contexto do wizard — o cancelamento segue o padrão único de
// wizard_read.go (ESC/CTRL+C/EOF no TTY = limpo; mensagem de script só
// fora de TTY).
type installWizard struct {
	out interface{ Write([]byte) (int, error) }
	in  *bufio.Reader
	tty bool
}

func (w *installWizard) home() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return h
}

func (w *installWizard) prompt(question string) (string, error) {
	fmt.Fprint(w.out, question)
	line, cancelled, serr := readWizardLine(w.in, w.tty,
		"use --engine/--mcp/--yes para não-interativo")
	switch {
	case cancelled:
		fmt.Fprintln(w.out) // quebra a linha do prompt antes da saída
		return "", errWizardCancelled
	case serr != nil:
		return "", serr
	}
	return line, nil
}

func (w *installWizard) pick(label string, options []string, decorate func(string) string) (string, error) {
	if len(options) == 0 {
		return "", fmt.Errorf("nenhum %s disponível", label)
	}
	fmt.Fprintf(w.out, "\nEscolha o %s:\n\n", label)
	for i, opt := range options {
		if i > 0 {
			fmt.Fprint(w.out, "\n") // respiro entre as opções
		}
		fmt.Fprintf(w.out, "  %d. %s %s\n", i+1, decorate(opt), opt)
	}
	for {
		answer, perr := w.prompt(fmt.Sprintf("número [1-%d]: ", len(options)))
		if perr != nil {
			return "", perr
		}
		n, err := strconv.Atoi(strings.TrimSpace(answer))
		if err == nil && n >= 1 && n <= len(options) {
			return options[n-1], nil
		}
		fmt.Fprintf(w.out, "  resposta inválida (%q) — informe um número da lista\n", answer)
	}
}

// pickProfile — PRIMEIRO passo do wizard numerado: a decisão macro (que
// experiência). Copy de 2 linhas por opção (o que é + limitação principal)
// da fonte canônica install.ProfileChoices — a mesma do wizard VT. ENTER
// vazio fica no default (essencial).
func (w *installWizard) pickProfile() (string, error) {
	fmt.Fprintf(w.out, "\n%s\n", install.Blurb())
	fmt.Fprintf(w.out, "Escolha o perfil:\n")
	choices := install.ProfileChoices()
	for i, ch := range choices {
		if i > 0 {
			fmt.Fprint(w.out, "\n") // respiro entre as opções
		}
		fmt.Fprintf(w.out, "  %d. %s\n", i+1, ch.ID)
		fmt.Fprintf(w.out, "     %s\n", ch.What)
		fmt.Fprintf(w.out, "     indicado para: %s\n", ch.For)
		fmt.Fprintf(w.out, "     limitação: %s\n", ch.Limitation)
	}
	for {
		answer, perr := w.prompt("número [1-2] (default: 1): ")
		if perr != nil {
			return "", perr
		}
		if answer == "" {
			return choices[0].ID, nil
		}
		n, err := strconv.Atoi(strings.TrimSpace(answer))
		if err == nil && n >= 1 && n <= len(choices) {
			return choices[n-1].ID, nil
		}
		fmt.Fprintf(w.out, "  resposta inválida (%q) — informe um número da lista\n", answer)
	}
}

func (w *installWizard) pickMCPs(installed map[string]string) ([]string, error) {
	fmt.Fprintf(w.out, "\nMCPs do catálogo (o golmet é obrigatório e já está incluído):\n\n")
	var catalog []string
	i := 0
	for _, def := range install.Catalog() {
		if i > 0 {
			fmt.Fprint(w.out, "\n") // respiro entre as opções
		}
		i++
		tag := ""
		switch {
		case def.Required:
			tag = "[obrigatório]"
		case def.Recommended:
			tag = "[recomendado]"
		default:
			tag = "[opcional  ]"
		}
		// badge de status: já registrado no engine → substitui os atributos
		if status, reg := installed[def.Name]; reg {
			fmt.Fprintf(w.out, "  %s %s — já registrado (%s)\n", tag, def.Name, status)
			catalog = append(catalog, def.Name)
			continue
		}
		extra := ""
		if def.Param != "" {
			extra = " · exige parâmetro " + def.Param
		}
		if len(def.EnvRequired) > 0 {
			extra += " · precisa env: " + strings.Join(def.EnvRequired, ",")
		}
		if len(def.EnvOptional) > 0 {
			extra += " · env opcional: " + strings.Join(def.EnvOptional, ",")
		}
		if def.Type == "remote" {
			extra += " · remote"
		}
		fmt.Fprintf(w.out, "  %s %s — %s%s\n", tag, def.Name, def.Description, extra)
		catalog = append(catalog, def.Name)
	}
	answer, perr := w.prompt("\nNúmeros dos MCPs a registrar (ex.: 1,2,4) [default: 1 (golmet)]: ")
	if perr != nil {
		return nil, perr
	}
	var selected []string
	for _, part := range strings.Split(strings.TrimSpace(answer), ",") {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err == nil && n >= 1 && n <= len(catalog) {
			selected = append(selected, catalog[n-1])
		}
	}
	// prompts de parâmetro obrigatório (ex.: azure-devops → ORG) — o wizard
	// serializa no formato canônico "nome=valor" (mesmo parser das flags)
	for _, name := range selected {
		def, ok := install.Lookup(name)
		if !ok || def.Param == "" {
			continue
		}
		value, verr := w.prompt(fmt.Sprintf("Valor de %s para o %s (ex.: MinhaOrg): ", def.Param, def.Name))
		if verr != nil {
			return nil, verr
		}
		for i, s := range selected {
			if s == name {
				selected[i] = name + "=" + strings.TrimSpace(value)
			}
		}
	}
	names, serr := install.ParseMCPSelection(selected)
	if serr != nil {
		return nil, serr
	}
	var out []string
	for _, s := range names {
		out = append(out, s.Name+"="+s.ParamValue)
	}
	return out, nil
}

func (w *installWizard) confirm(req install.Options) (bool, error) {
	fmt.Fprintf(w.out, "\nResumo:\n\n  engine:   %s\n  perfil:   %s\n  mcps:     %s\n  workspace (doctor final): %s\n\n",
		req.EngineID, req.Profile, strings.Join(req.MCPs, ", "), req.Workspace)
	// o frame ◆ isolado do resumo (o padrão visual do install — M28):
	// ENTER confirma, n/esc cancelam (o install não tem passo de edição)
	fmt.Fprintf(w.out, "%s◆ Confirmar e instalar?%s %s(ENTER confirma · n/esc cancelam)%s\n\n",
		wizardANSI("\x1b[1m"), wizardANSI("\x1b[0m"), wizardANSI("\x1b[90m"), wizardANSI("\x1b[0m"))
	answer, perr := w.prompt("> ")
	if perr != nil {
		return false, perr
	}
	switch {
	case answer == "":
		return true, nil // ENTER confirma (o default)
	case strings.EqualFold(answer, "s"):
		return true, nil // alias do ENTER (compat)
	}
	return false, nil // n (ou qualquer outra coisa) cancela
}
