// install.go — orquestração do `golmet install`: perfil BASE (único no
// 0.1), skills do embed no engine escolhido, MCPs do catálogo com env
// resolvida (config quebrada nunca — env ausente = pendente), codegraph
// com download verificado quando ausente, doctor final no workspace.
package install

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	golmet "github.com/Golmet/golmet-cli"
	"github.com/Golmet/golmet-cli/internal/core/content"
	"github.com/Golmet/golmet-cli/internal/core/doctor"
	"github.com/Golmet/golmet-cli/internal/core/indexstate"
	"github.com/Golmet/golmet-cli/internal/core/source"
	"github.com/Golmet/golmet-cli/internal/core/tools"
	"github.com/Golmet/golmet-cli/internal/core/update"
)

// Perfis do CLI — semântica de CONTRATO (não de tamanho):
//   - essencial: kit do repositório — 16 skills gm-* na zona .golmet/ do
//     workspace, versionadas com o repo (a equipe inteira herda); rodam em
//     qualquer engine agêntico. Sem agents nativos (só o núcleo de docs).
//   - completo: experiência completa na sua máquina — 121 componentes
//     (agents + skills + rules) no formato do engine, no config global (pessoal,
//     não versionado). Agents nativos com delegação real só em opencode e
//     claude-code; nos demais os arquivos vão para a zona do workspace
//     como referência (não spawnáveis nativamente).
//
// base/full são ALIASES internos aceitos na normalização do Run (compat de
// flags) — nunca aparecem no help nem no relatório.
const (
	ProfileEssencial = "essencial"
	ProfileCompleto  = "completo"
)

const (
	profileAliasBase = "base"
	profileAliasFull = "full"
)

const profileBlurb = "Perfis: essencial (kit do repositório — 16 skills gm-* na zona .golmet/ do workspace, versionadas com o repo · qualquer engine) · completo (experiência completa na máquina — 121 componentes no formato do engine · agents nativos só em opencode e claude-code)."

// Blurb devolve o resumo dos perfis para superfícies que precisam da
// visão geral em uma linha.
func Blurb() string { return profileBlurb }

// Options do install (flags do CLI; o wizard interativo preenche os
// mesmos campos). Home vazio = os.UserHomeDir.
type Options struct {
	EngineID     string
	Profile      string
	MCPs         []string // entradas "nome" ou "nome=param" (ex.: azure-devops=MinhaOrg)
	Workspace    string   // doctor final
	Force        bool
	Full         bool   // materializa o workflow FULL embarcado (L1) no engine
	Home         string // override p/ testes (default os.UserHomeDir)
	LocalAppData string // override p/ testes (default env LOCALAPPDATA ou home/AppData/Local)

	// InstallCodegraphFn injetável (testes); default = tools.InstallCodegraph
	InstallCodegraphFn func(ctx context.Context) error
	// InstallSqliteFn injetável (testes); default = tools.InstallSqlite
	InstallSqliteFn func(ctx context.Context) error
	// DoctorFn injetável (testes); default = doctor.Run
	DoctorFn func(workspace string) (*doctor.Report, error)

	// Progress (opcional) — eventos de passo do install para UX
	// progressiva (o CLI humano consome com spinner por passo; JSON e
	// texto plano não registram consumidor). Nil = silencioso.
	Progress func(StepEvent)
}

// MCPResult é o resultado do registro de um MCP.
type MCPResult struct {
	Name   string `json:"name"`
	Status string `json:"status"` // registered | unchanged | diverged | manual | pending-env | skipped | unsupported | error
	Detail string `json:"detail,omitempty"`
	Fix    string `json:"fix,omitempty"`
}

// DoctorSummary é o resumo do doctor final (fechamento do install).
type DoctorSummary struct {
	Workspace string `json:"workspace"`
	OKs       int    `json:"ok"`
	Warns     int    `json:"warns"`
	Fails     int    `json:"fails"`
	Human     string `json:"-"`
}

// Report é o relatório final do install.
type Report struct {
	Engine          string         `json:"engine"`
	Profile         string         `json:"profile"`
	Version         string         `json:"version"`
	SkillsDir       string         `json:"skills_dir,omitempty"`
	SkillsSupported bool           `json:"skills_supported"`
	SkillsWritten   []string       `json:"skills_written,omitempty"`
	SkillsPreserved []string       `json:"skills_preserved,omitempty"`
	FullWritten     []string       `json:"full_written,omitempty"`
	FullPreserved   []string       `json:"full_preserved,omitempty"`
	FullConflicts   []string       `json:"full_conflicts,omitempty"`
	FullTotal       int            `json:"full_total,omitempty"`
	FullZone        string         `json:"full_zone,omitempty"`      // engine-config | workspace-zone (fallback)
	FullZonePath    string         `json:"full_zone_path,omitempty"` // raiz onde o FULL foi materializado
	MCPs            []MCPResult    `json:"mcps,omitempty"`
	Pending         []MCPResult    `json:"pending,omitempty"` // env ausente / param ausente / binário ausente
	Doctor          *DoctorSummary `json:"doctor,omitempty"`
}

// Result agrega tudo para o CLI decidir o exit. Partial = algum MCP
// escolhido não registrou (pending-env, diverged, manual, unsupported,
// error, skipped) → CLI sai 1.
type Result struct {
	Report  *Report
	Partial bool
}

// StepEvent é um evento de progresso do Run (start/done por passo). O par
// start→done é garantido: cada passo que EXECUTA emite exatamente um de
// cada, em ordem (skills → codegraph-download → mcp:<nome>… → doctor);
// pendências sem execução (env/param/binário ausente) não emitem evento —
// aparecem no relatório final.
type StepEvent struct {
	Kind    string // "start" | "done"
	Step    string // skills | mcp:<nome> | codegraph-download | doctor
	Label   string // label humano do passo em curso (só em start)
	Detail  string // resultado em uma linha (só em done)
	Outcome string // "ok" | "warn" | "fail" (só em done)
}

// Zonas de destino do FULL (relatório `full_zone`): engines com skills
// nativas recebem o FULL no config root do engine; engines sem sistema de
// skills (copilot-cli, vscode-copilot, cursor, gemini-cli, codex) recebem o
// FULL materializado na zona .golmet/ do workspace — mesmo modelo DR-008 do
// BASE.
const (
	FullZoneEngineConfig = "engine-config"  // config root do engine (skills nativas)
	FullZoneWorkspace    = "workspace-zone" // zona .golmet/ do workspace (fallback DR-008)
)

// workspaceContentManifestName separa o manifest de conteúdo do manifest
// BASE quando o FULL cai na zona do workspace: <ws>/.golmet/manifest.json é
// do update.Install (schema Manifest/Skills) e o conteúdo usa nome próprio —
// BASE e FULL coexistem na mesma zona sem corromper um ao outro.
const workspaceContentManifestName = "manifest-content.json"

// FullPlacement descreve onde o FULL foi materializado (relatório JSON, UX e
// provas de coexistência BASE/FULL na mesma zona).
type FullPlacement struct {
	Zone       string // FullZoneEngineConfig | FullZoneWorkspace
	ConfigRoot string // raiz onde agents/ e skills/ foram escritos
	Manifest   string // manifest de conteúdo usado (na zona do workspace: manifest-content.json)
}

// resolveFullTarget escolhe o destino do FULL: config root do engine quando
// ele tem skills nativas; zona .golmet/ do workspace caso contrário. Erro
// APENAS quando não há skills nativas E nenhum workspace foi informado
// (impossível determinar o destino).
func resolveFullTarget(adapter EngineAdapter, home, workspace string) (FullPlacement, error) {
	if skillsDir := adapter.SkillsDir(home); skillsDir != "" {
		root := filepath.Dir(skillsDir)
		return FullPlacement{Zone: FullZoneEngineConfig, ConfigRoot: root,
			Manifest: content.ContentManifestPath(root, "")}, nil
	}
	if workspace == "" {
		return FullPlacement{}, fmt.Errorf("engine %q não possui diretório de skills e nenhum workspace foi informado — impossível determinar o destino do FULL (informe --workspace)", adapter.ID())
	}
	root := filepath.Join(workspace, ".golmet")
	return FullPlacement{Zone: FullZoneWorkspace, ConfigRoot: root,
		Manifest: filepath.Join(root, workspaceContentManifestName)}, nil
}

// MaterializeFull projeta o FULL embarcado (L1) no config do engine — ou, em
// engines sem sistema de skills, na zona .golmet/ do workspace (fallback
// DR-008; workspace vazio = erro). É exportado para `install --full` e
// `update --full`, mantendo um único caminho de 3-way e de proveniência.
func MaterializeFull(home, engineID, workspace string, force bool) (*content.MaterializeReport, *FullPlacement, error) {
	rep, place, _, err := materializeFullWithOptions(home, engineID, workspace, force, false)
	return rep, place, err
}

// MaterializeFullWithOptions é a variante usada por transportes que precisam
// de planejar a operação sem tocar o runtime.
func MaterializeFullWithOptions(home, engineID, workspace string, force, dryRun bool) (*content.MaterializeReport, *FullPlacement, error) {
	rep, place, _, err := materializeFullWithOptions(home, engineID, workspace, force, dryRun)
	return rep, place, err
}

func materializeFullWithOptions(home, engineID, workspace string, force, dryRun bool) (*content.MaterializeReport, *FullPlacement, *source.SyncReport, error) {
	adapter, ok := LookupEngine(engineID)
	if !ok {
		return nil, nil, nil, fmt.Errorf("engine %q não suportado", engineID)
	}
	place, terr := resolveFullTarget(adapter, home, workspace)
	if terr != nil {
		return nil, nil, nil, terr
	}
	coreFS, err := fs.Sub(golmet.CoreDataFS(), "coredata")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("FULL embarcado ilegível: %w", err)
	}
	registryData, err := fs.ReadFile(coreFS, "registry.yaml")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("registry FULL ilegível: %w", err)
	}
	reg := content.ParseRegistry(string(registryData))
	skills, skillErrs, err := update.EmbeddedSkills()
	if err != nil {
		return nil, nil, nil, err
	}
	if len(skillErrs) > 0 {
		return nil, nil, nil, fmt.Errorf("BASE embutida inválida: %d erro(s)", len(skillErrs))
	}
	baseHashes := map[string]string{}
	for _, sk := range skills {
		baseHashes[filepath.ToSlash(filepath.Join("skills", sk.Dir, "SKILL.md"))] = sk.SHA256
	}
	rep, err := content.Materialize(coreFS, reg, content.MaterializeOptions{
		ConfigRoot: place.ConfigRoot, Engine: engineID,
		KitVersion: indexstate.KitVersion, Force: force,
		ManagedHashes: baseHashes, DryRun: dryRun,
		ManifestPath: place.Manifest,
	})
	if err != nil {
		return nil, nil, nil, err
	}
	return rep, &place, nil, nil
}

// MaterializeExternalCore sincroniza um core Git/local e renderiza o mesmo
// registry do FULL embarcado. O cache fica em ~/.golmet/sources/core e o
// manifest registra a origem; a semântica de overlay/3-way é a mesma.
func MaterializeExternalCore(home, engineID, sourceURL string, force bool) (*content.MaterializeReport, *source.SyncReport, error) {
	return MaterializeExternalCoreWithOptions(home, engineID, sourceURL, force, false)
}

// MaterializeExternalCoreWithOptions mantém o cache Git intocado em dry-run.
func MaterializeExternalCoreWithOptions(home, engineID, sourceURL string, force, dryRun bool) (*content.MaterializeReport, *source.SyncReport, error) {
	adapter, ok := LookupEngine(engineID)
	if !ok {
		return nil, nil, fmt.Errorf("engine %q não suportado", engineID)
	}
	skillsDir := adapter.SkillsDir(home)
	if skillsDir == "" {
		return nil, nil, fmt.Errorf("engine %q não possui diretório de skills para o core", engineID)
	}
	sync, err := source.SyncWithOptions(sourceURL, filepath.Join(home, ".golmet", "sources", "core"), source.SyncOptions{DryRun: dryRun})
	if err != nil {
		return nil, nil, err
	}
	coreFS, err := content.SafeDirFS(sync.Path)
	if err != nil {
		return nil, sync, err
	}
	registryData, err := fs.ReadFile(coreFS, "registry.yaml")
	if err != nil {
		return nil, sync, fmt.Errorf("registry externo ilegível: %w", err)
	}
	reg := content.ParseRegistry(string(registryData))
	rep, err := content.Materialize(coreFS, reg, content.MaterializeOptions{
		ConfigRoot: filepath.Dir(skillsDir), Engine: engineID,
		KitVersion: indexstate.KitVersion, Force: force,
		Source: source.Redact(sourceURL) + "@" + sync.SHA, DryRun: dryRun,
	})
	if err != nil {
		return nil, sync, err
	}
	return rep, sync, nil
}

// emitProgress — nil-safe (sem consumidor = silencioso).
func (opt *Options) emitProgress(ev StepEvent) {
	if opt.Progress != nil {
		opt.Progress(ev)
	}
}

// resolveHome — override injetável p/ testes (os.UserHomeDir é o default;
// no Windows lê USERPROFILE, no Unix HOME).
func resolveHome(opt *Options) (string, error) {
	if opt.Home != "" {
		return opt.Home, nil
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("sem HOME para o install: %w", err)
	}
	return h, nil
}

// Run executa o install. Erro = uso/config inválido (CLI 2); falha
// parcial de registro vai em Result.Partial (CLI 1).
func Run(opt Options) (*Result, error) {
	// normalização do perfil: essencial (default) | completo — semântica de
	// contrato, não de tamanho. base/full são aliases aceitos (compat);
	// desconhecido falha listando os válidos. completo ≡ Full (a camada de
	// 121 componentes materializa) — e --full com --profile essencial vira
	// completo (o relatório nunca cita base/full/base+full).
	switch p := strings.ToLower(strings.TrimSpace(opt.Profile)); p {
	case "", profileAliasBase, ProfileEssencial:
		opt.Profile = ProfileEssencial
	case profileAliasFull, ProfileCompleto:
		opt.Profile = ProfileCompleto
		opt.Full = true
	default:
		return nil, fmt.Errorf("perfil %q não existe — válidos: essencial, completo (aliases aceitos: base, full)", opt.Profile)
	}
	if opt.Full {
		opt.Profile = ProfileCompleto
	}
	adapter, ok := LookupEngine(opt.EngineID)
	if !ok {
		return nil, fmt.Errorf("engine %q não existe — disponíveis: %s", opt.EngineID, strings.Join(Engines(), ", "))
	}
	home, err := resolveHome(&opt)
	if err != nil {
		return nil, err
	}
	// fail-closed: o perfil completo exige engine com skills nativas — uso
	// não-interativo NÃO degrada silenciosamente para a zona do workspace
	// (a infra de zona segue no pipeline interno, mas não é exposta aqui).
	if opt.Full && adapter.SkillsDir(home) == "" {
		return nil, fmt.Errorf("o perfil completo exige engine com skills nativas — compatíveis: %s; para %s, use o perfil essencial",
			strings.Join(CompatibleEngineIDs(home), ", "), adapter.ID())
	}
	res := &Result{Report: &Report{
		Engine: adapter.ID(), Profile: opt.Profile, Version: indexstate.KitVersion,
	}}

	// passo 1 — perfil completo: materializa o registry canônico (L1)
	// ANTES de qualquer passo BASE (o FULL é superconjunto no mesmo
	// destino — rodar o BASE antes compararia formatos diferentes e
	// sinalizaria falsas "modificações locais"). Nativo no engine com
	// skills; zona .golmet/ do workspace nos demais (DR-008).
	if opt.Full {
		if ferr := opt.runFull(home, res); ferr != nil {
			return nil, ferr
		}
	}

	// passo 2 — skills BASE do embed (somente perfil essencial: com FULL
	// o passo é redundante — o FULL já materializou superconjunto no
	// mesmo destino e o BASE sinalizaria falsas "modificações locais").
	skillsDir := adapter.SkillsDir(home)
	if skillsDir == "" {
		res.Report.SkillsSupported = false
		// O perfil essencial em engine sem skills nativas é o caminho
		// válido: skills ficam na zona .golmet/ do workspace (DR-008) e o
		// registro de MCPs segue normalmente.
	} else if !opt.Full {
		res.Report.SkillsSupported = true
		res.Report.SkillsDir = skillsDir
		opt.emitProgress(StepEvent{Kind: "start", Step: "skills", Label: "skills gm-* — " + skillsDir})
		skills, skillErrs, serr := update.EmbeddedSkills()
		if serr != nil {
			return nil, fmt.Errorf("subset BASE embutido inválido: %w", serr)
		}
		if len(skillErrs) > 0 {
			return nil, fmt.Errorf("subset BASE embutido inválido — não instalado (%d skill(es) com front matter ilegível)", len(skillErrs))
		}
		written, preserved, werr := writeEngineSkills(skillsDir, skills, opt.Force)
		if werr != nil {
			return nil, werr
		}
		res.Report.SkillsWritten = written
		res.Report.SkillsPreserved = preserved
		detail := fmt.Sprintf("skills gm-* — %d escrita(s)", len(written))
		outcome := "ok"
		if len(preserved) > 0 {
			outcome = "warn"
			detail = fmt.Sprintf("skills gm-* — %d escrita(s), %d preservada(s) (modificadas localmente)", len(written), len(preserved))
		}
		opt.emitProgress(StepEvent{Kind: "done", Step: "skills", Detail: detail, Outcome: outcome})
	} else {
		res.Report.SkillsSupported = true
		res.Report.SkillsDir = skillsDir
	}

	// passo 3 — MCPs selecionados (golmet obrigatório sempre)
	selected, err := ParseMCPSelection(opt.MCPs)
	if err != nil {
		return nil, err
	}
	selfExe, eerr := os.Executable()
	if eerr != nil {
		return nil, fmt.Errorf("não resolvi o executável do kit: %w", eerr)
	}
	for _, sel := range selected {
		def, _ := Lookup(sel.Name)
		// Param obrigatório (§10 — hoje só azure-devops): valor ausente =
		// pendência com a sintaxe (o kit nunca adivinha a organização)
		if def.Param != "" && strings.TrimSpace(sel.ParamValue) == "" {
			mr := MCPResult{Name: def.Name, Status: OutcomeSkipped,
				Detail: "parâmetro obrigatório ausente: " + def.Param,
				Fix:    fmt.Sprintf("sintaxe: golmet install --mcp %s=<%s> --engine %s (ex.: --mcp %s=MinhaOrg)", def.Name, def.Param, opt.EngineID, def.Name),
			}
			res.Partial = true
			res.Report.Pending = append(res.Report.Pending, mr)
			res.Report.MCPs = append(res.Report.MCPs, mr)
			continue
		}
		// placeholders (@self/@codegraph/@sqlite) resolvidos ANTES de qualquer
		// adapter — nunca cruzam a fronteira para a CLI do engine
		resolved := resolveDef(&opt, def, home, selfExe, res, sel.ParamValue)
		if resolved == nil {
			res.Partial = true
			continue // pendência registrada no relatório (env/binário/param)
		}
		opt.emitProgress(StepEvent{Kind: "start", Step: "mcp:" + def.Name, Label: def.Name + " (mcp add)"})
		mr := registerMCP(adapter, home, *resolved, &opt)
		if mr.Status != OutcomeRegistered && mr.Status != OutcomeUnchanged {
			res.Partial = true
		}
		if mr.Status == OutcomeSkipped && strings.HasPrefix(mr.Detail, "env obrigatória ausente") {
			res.Report.Pending = append(res.Report.Pending, mr)
		}
		res.Report.MCPs = append(res.Report.MCPs, mr)
		detail, outcome := mcpStepDetail(mr)
		opt.emitProgress(StepEvent{Kind: "done", Step: "mcp:" + def.Name, Detail: detail, Outcome: outcome})
	}

	// fechamento — doctor no workspace escolhido
	if opt.Workspace != "" {
		opt.emitProgress(StepEvent{Kind: "start", Step: "doctor", Label: "doctor — verificando workspace..."})
		sum, derr := runDoctor(opt.Workspace, &opt)
		res.Report.Doctor = sum
		if derr != nil {
			opt.emitProgress(StepEvent{Kind: "done", Step: "doctor", Detail: "doctor — não pôde rodar no workspace", Outcome: "warn"})
		} else {
			outcome := "ok"
			detail := fmt.Sprintf("doctor — %d ok, %d aviso(s)", sum.OKs, sum.Warns)
			switch {
			case sum.Fails > 0:
				outcome = "fail"
				detail = fmt.Sprintf("doctor — %d ok, %d aviso(s), %d falha(s)", sum.OKs, sum.Warns, sum.Fails)
			case sum.Warns > 0:
				outcome = "warn"
			}
			opt.emitProgress(StepEvent{Kind: "done", Step: "doctor", Detail: detail, Outcome: outcome})
		}
	}
	return res, nil
}

// runFull materializa o FULL embarcado e registra o resultado no relatório
// e no progresso — único caminho para ambos os destinos (config root do
// engine com skills nativas; zona .golmet/ do workspace nos demais).
// Workspace vazio = "." (default do flag --workspace; a semântica do doctor
// não muda — ele continua rodando apenas quando o workspace foi informado).
func (opt *Options) runFull(home string, res *Result) error {
	ws := opt.Workspace
	if ws == "" {
		ws = "."
	}
	adapter, ok := LookupEngine(opt.EngineID) // já validado no Run
	if !ok {
		return fmt.Errorf("engine %q não existe", opt.EngineID)
	}
	place, terr := resolveFullTarget(adapter, home, ws)
	if terr != nil {
		return terr
	}
	opt.emitProgress(StepEvent{Kind: "start", Step: "full", Label: "workflow FULL gm-* → " + place.ConfigRoot})
	fullRep, _, _, merr := materializeFullWithOptions(home, opt.EngineID, ws, opt.Force, false)
	if merr != nil {
		return fmt.Errorf("FULL não instalado: %w", merr)
	}
	res.Report.FullWritten = fullRep.Written
	res.Report.FullPreserved = fullRep.Preserved
	res.Report.FullConflicts = fullRep.Conflicts
	res.Report.FullTotal = fullRep.Total
	res.Report.FullZone = place.Zone
	res.Report.FullZonePath = place.ConfigRoot
	outcome := "ok"
	detail := fmt.Sprintf("workflow FULL — %d escrita(s) em %s (%s)", len(fullRep.Written), place.ConfigRoot, place.Zone)
	if len(fullRep.Conflicts) > 0 {
		outcome = "warn"
		detail = fmt.Sprintf("workflow FULL — %d escrita(s), %d conflito(s) preservado(s) em %s (%s)", len(fullRep.Written), len(fullRep.Conflicts), place.ConfigRoot, place.Zone)
	}
	opt.emitProgress(StepEvent{Kind: "done", Step: "full", Detail: detail, Outcome: outcome})
	return nil
}

// resolveDef devolve uma CÓPIA do def com placeholders e parâmetros
// resolvidos (@self → exe corrente; @codegraph → binário do bloco 3, com
// download verificado quando ausente; @sqlite → binário local em
// %LOCALAPPDATA%; <Param> → valor informado pelo usuário).
// nil = pendência registrada no res (env ausente / binário indisponível) —
// config quebrada nunca.
func resolveDef(opt *Options, def MCPDef, home, selfExe string, res *Result, paramValue string) *MCPDef {
	// param obrigatório interpolado no Command/Args (token "<Param>")
	if def.Param != "" {
		token := "<" + def.Param + ">"
		value := strings.TrimSpace(paramValue)
		def.Command = strings.ReplaceAll(def.Command, token, value)
		for i, a := range def.Args {
			def.Args[i] = strings.ReplaceAll(a, token, value)
		}
	}
	if def.Type == MCPTypeRemote || (def.Command != "@self" && def.Command != "@codegraph" && def.Command != "@sqlite") {
		return &def
	}
	switch def.Command {
	case "@self":
		def.Command = selfExe
		return &def
	case "@codegraph":
		bin := codegraphBinary(home)
		if fileExists(bin) {
			def.Command = bin
			return &def
		}
		installFn := opt.InstallCodegraphFn
		if installFn == nil {
			installFn = func(ctx context.Context) error {
				_, err := tools.InstallCodegraph(ctx, tools.Options{
					DestDir: filepath.Join(home, ".golmet", "tools", "codegraph"),
				})
				return err
			}
		}
		opt.emitProgress(StepEvent{Kind: "start", Step: "codegraph-download",
			Label: "codegraph — baixando binário verificado (pode levar um minuto)"})
		if err := installFn(context.Background()); err != nil {
			opt.emitProgress(StepEvent{Kind: "done", Step: "codegraph-download",
				Detail: "codegraph — download falhou: " + err.Error(), Outcome: "fail"})
			res.Report.MCPs = append(res.Report.MCPs, MCPResult{
				Name: def.Name, Status: OutcomeSkipped,
				Detail: "binário do codegraph ausente e o download falhou (" + err.Error() + ")",
				Fix:    "verifique a conectividade ou rode 'golmet tools install codegraph' manualmente; depois rode golmet install --mcp codegraph --engine " + opt.EngineID,
			})
			return nil
		}
		if !fileExists(bin) {
			opt.emitProgress(StepEvent{Kind: "done", Step: "codegraph-download",
				Detail: "codegraph — download concluiu mas o binário não está em " + bin, Outcome: "fail"})
			res.Report.MCPs = append(res.Report.MCPs, MCPResult{
				Name: def.Name, Status: OutcomeSkipped,
				Detail: "download concluiu mas o binário não está em " + bin,
				Fix:    "rode 'golmet tools install codegraph' manualmente",
			})
			return nil
		}
		opt.emitProgress(StepEvent{Kind: "done", Step: "codegraph-download",
			Detail: "codegraph — instalado (SHA-256 + attestation)", Outcome: "ok"})
		def.Command = bin
		return &def
	case "@sqlite":
		// rusq/mcp-sqlite — auto-install com SHA-256 do checksums.txt da
		// release v0.0.3 (procedimento da skill gm-mcp-sqlite-setup; o
		// release NÃO tem attestation SLSA — aviso declarado no tools)
		bin := sqliteBinary(opt.LocalAppData)
		if fileExists(bin) {
			def.Command = bin
			return &def
		}
		installFn := opt.InstallSqliteFn
		if installFn == nil {
			installFn = func(ctx context.Context) error {
				_, err := tools.InstallSqlite(ctx, tools.SqliteOptions{
					DestDir:      filepath.Dir(bin),
					LocalAppData: opt.LocalAppData,
				})
				return err
			}
		}
		opt.emitProgress(StepEvent{Kind: "start", Step: "sqlite-download",
			Label: "sqlite — baixando binário verificado (pode levar um minuto)"})
		if err := installFn(context.Background()); err != nil {
			opt.emitProgress(StepEvent{Kind: "done", Step: "sqlite-download",
				Detail: "sqlite — download falhou: " + err.Error(), Outcome: "fail"})
			res.Report.Pending = append(res.Report.Pending, MCPResult{
				Name: def.Name, Status: OutcomeSkipped,
				Detail: "binário do sqlite ausente e o download falhou (" + err.Error() + ")",
				Fix:    "baixe a release " + tools.SqliteVersion + " de github.com/rusq/mcp-sqlite (verifique o checksum), extraia o binário em " + filepath.Dir(bin) + " e rode golmet install --mcp sqlite --engine " + opt.EngineID,
			})
			return nil
		}
		if !fileExists(bin) {
			opt.emitProgress(StepEvent{Kind: "done", Step: "sqlite-download",
				Detail: "sqlite — download concluiu mas o binário não está em " + bin, Outcome: "fail"})
			res.Report.Pending = append(res.Report.Pending, MCPResult{
				Name: def.Name, Status: OutcomeSkipped,
				Detail: "download concluiu mas o binário não está em " + filepath.Dir(bin),
				Fix:    "rode 'golmet tools install sqlite' manualmente",
			})
			return nil
		}
		opt.emitProgress(StepEvent{Kind: "done", Step: "sqlite-download",
			Detail: "sqlite — instalado (SHA-256 do checksums.txt)", Outcome: "ok"})
		def.Command = bin
		return &def
	}
	return &def
}

// sqliteBinary devolve o caminho esperado do binário rusq/mcp-sqlite
// (%LOCALAPPDATA%\Programs\mcp-sqlite\ — localAppData injetável p/ testes;
// default: env LOCALAPPDATA ou home/AppData/Local).
func sqliteBinary(localAppData string) string {
	if localAppData == "" {
		localAppData = os.Getenv("LOCALAPPDATA")
	}
	if localAppData == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			localAppData = filepath.Join(home, "AppData", "Local")
		}
	}
	name := "mcp-sqlite"
	if runtime.GOOS == "windows" {
		name = "mcp-sqlite.exe"
	}
	return filepath.Join(localAppData, "Programs", "mcp-sqlite", name)
}

// registerMCP registra UM MCP no engine. O def chega com placeholders
// JÁ resolvidos (resolveDef) — o adapter só aplica o transporte/formato.
func registerMCP(adapter EngineAdapter, home string, def MCPDef, opt *Options) MCPResult {
	mr := MCPResult{Name: def.Name}

	// env obrigatória ausente → NÃO registra (config quebrada nunca)
	if len(def.EnvRequired) > 0 && !hasAllEnv(def.EnvRequired) {
		mr.Status = OutcomeSkipped
		mr.Detail = "env obrigatória ausente: " + strings.Join(def.EnvRequired, ", ")
		mr.Fix = fmt.Sprintf("exporte %s e rode golmet install --mcp %s --engine %s",
			strings.Join(def.EnvRequired, ", "), def.Name, adapter.ID())
		return mr
	}

	// entrada de config no formato dos engines de patch (claude fallback,
	// vscode, cursor, gemini). opencode/codex usam CLI/TOML — env v0.1:
	// o engine lê do ambiente herdado no boot do processo filho.
	entry := map[string]any{}
	if def.Type == MCPTypeRemote {
		entry["type"] = "http"
		entry["url"] = def.URL
	} else {
		entry["command"] = def.Command
		if len(def.Args) > 0 {
			entry["args"] = def.Args
		}
		if len(def.EnvRequired) > 0 {
			env := map[string]string{}
			for _, k := range def.EnvRequired {
				env[k] = os.Getenv(k) // VALOR literal — decision v0.1 (nunca placeholder; nunca log)
			}
			entry["env"] = env
		}
	}

	// Adapters que registram no WORKSPACE (vscode-copilot) recebem
	// opt.Workspace via hook opcional; os demais recebem a home global.
	var outcome, detail string
	var rerr error
	if wr, ok := adapter.(WorkspaceMCPRegistrar); ok {
		outcome, detail, rerr = wr.RegisterMCPWorkspace(opt.Workspace, entry, def, opt.Force)
	} else {
		outcome, detail, rerr = adapter.RegisterMCP(home, entry, def, opt.Force)
	}
	mr.Status, mr.Detail = outcome, detail
	if rerr != nil {
		mr.Status = OutcomeError
		mr.Detail = rerr.Error()
	}
	return mr
}

func codegraphBinary(home string) string {
	name := "codebase-memory-mcp"
	if runtime.GOOS == "windows" {
		name = "codebase-memory-mcp.exe"
	}
	return filepath.Join(home, ".golmet", "tools", "codegraph", name)
}

func hasAllEnv(keys []string) bool {
	for _, k := range keys {
		if os.Getenv(k) == "" {
			return false
		}
	}
	return true
}

// runDoctor — fechamento do install (o doctor carrega a versão do kit).
// Err != nil = o doctor não pôde rodar no workspace (o install continua —
// o resumo reporta como aviso). O check de estado do engine-config roda
// contra a raiz do engine escolhido — o usuário vê logo após instalar o
// que o kit não reconstrói naquele config.
func runDoctor(workspace string, opt *Options) (*DoctorSummary, error) {
	sum := &DoctorSummary{Workspace: filepath.Base(workspace)}
	runner := opt.DoctorFn
	if runner == nil {
		dopt := doctor.Options{}
		if home, herr := resolveHome(opt); herr == nil {
			if root, applies := EngineConfigRootFor(home, opt.EngineID); applies {
				dopt.EngineConfigRoot = root
			}
		} // home indisponível → check não roda (degradação declarada na ausência da seção)
		runner = func(ws string) (*doctor.Report, error) {
			return doctor.Run(ws, dopt)
		}
	}
	rep, err := runner(workspace)
	if err != nil || rep == nil {
		reason := mapBool(err != nil, err.Error(), "sem resultado")
		sum.Human = "doctor não pôde rodar no workspace informado: " + reason
		return sum, errors.New(reason)
	}
	sum.OKs, sum.Warns, sum.Fails = rep.OKs, rep.Warns, rep.Fails
	sum.Human = rep.Human()
	return sum, nil
}

// mcpStepDetail resume o resultado de um MCP para a linha de passo do
// human path (o detail completo do MCPResult vai no relatório final).
func mcpStepDetail(mr MCPResult) (detail, outcome string) {
	base := ""
	switch mr.Status {
	case OutcomeRegistered:
		base, outcome = mr.Name+" — registrado", "ok"
	case OutcomeUnchanged:
		base, outcome = mr.Name+" — já registrado", "ok"
	case OutcomeManual:
		return mr.Name + " — registro manual necessário", "warn"
	case OutcomeSkipped:
		return mr.Name + " — pendente", "warn"
	case OutcomeDiverged:
		return mr.Name + " — divergiu do instalado", "fail"
	default: // error, unsupported
		return mr.Name + " — falhou", "fail"
	}
	if note := mcpHealthSnippet(mr.Detail); note != "" {
		return base + " · ⚠ " + note, outcome
	}
	return base, outcome
}

// mcpHealthSnippet extrai a nota curta de saúde do detail do adapter
// (marcadores canônicos das health notes do install — autenticação,
// conexão falha, desabilitado, sondagem pendente).
func mcpHealthSnippet(detail string) string {
	switch {
	case strings.Contains(detail, "autentique no engine"):
		return "autentique no engine"
	case strings.Contains(detail, "não conseguiu conectar"):
		return "engine não conectou — verifique o binário"
	case strings.Contains(detail, "desabilitado no engine"):
		return "desabilitado no engine"
	case strings.Contains(detail, "sondagem em curso"):
		return "sondagem em curso"
	}
	return ""
}

func mapBool(cond bool, ifTrue, ifFalse string) string {
	if cond {
		return ifTrue
	}
	return ifFalse
}
