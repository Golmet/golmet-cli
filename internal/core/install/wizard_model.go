// wizard_model.go — modelo PURO do wizard interativo do `golmet install`:
// estado de seleção (engine único, MCPs multi-toggle), transições e plano
// final — sem render, sem raw mode, sem term. O render TTY
// (internal/cli/wizard_tty.go) é camada fina que mapeia teclas para os
// mutadores daqui; os testes cobrem o modelo, não o desenho.
package install

import (
	"fmt"
	"strings"

	"github.com/Golmet/golmet-cli/internal/core/indexstate"
)

// WizardStep são os passos do wizard na ordem de execução. O perfil vem
// PRIMEIRO: é a decisão macro (que experiência — kit do repositório vs
// experiência completa na máquina); o engine vem depois e, com perfil
// completo em engine sem skills nativas, o passo engine exibe a nota de
// entrega via zona do workspace.
type WizardStep int

const (
	StepProfile WizardStep = iota
	StepEngine
	StepMCPs
	StepPlan
)

// LimitOfVisibleSkills — quantos nomes aparecem truncados nos detalhes.
const truncNames = 6

// WizardModel é o estado do wizard interativo. Campos públicos só os
// deriváveis; mutação por métodos (toggle/move/confirm) para os invariantes
// ficarem num lugar (golmet imutável, engine único, etc.).
type WizardModel struct {
	Engines   []EngineInfo // ordem do registry
	MCPs      []MCPDef     // ordem do catálogo (kit primeiro)
	Workspace string
	Profile   string
	Home      string

	Step          WizardStep
	EngineCursor  int    // índice do engine sob o cursor
	EngineChosen  string // "" até confirmar o passo engine
	ProfileCursor int    // índica a opção de perfil sob o cursor (0 = essencial)
	MCPCursor     int
	Checked       map[string]bool   // nome → marcado (golmet sempre true)
	ParamValues   map[string]string // valores de Param capturados nos prompts
	Notice        string            // aviso inline da última ação inválida (ex.: desmarcar golmet)
	Cancelled     bool
	ProfileNotice string

	// Registered — MCPs já registrados no engine escolhido (nome → status
	// textual do engine, ex. "connected"). Detectado UMA vez no
	// EngineConfirm (cache — 1 chamada de list por wizard); nil/indisponível
	// = sem badges, nada quebra. Já-registrados vêm DESMARCADOS por default
	// (nada a fazer — remarcar é opt-in); golmet permanece marcado sempre.
	Registered map[string]string

	allEngines     []EngineInfo // lista completa do registry (restaure ao voltar p/ essencial)
	registeredDone bool         // cache da detecção (1 chamada por wizard)
}

// NewWizardModel monta o modelo com engines detectados (home injetável),
// catálogo em ordem de exibição e golmet pré-marcado. codegraph NÃO vem
// pré-marcado (o download de 318MB não pode ser surpresa).
func NewWizardModel(home, workspace string) *WizardModel {
	m := &WizardModel{
		Engines:     DetectEngines(home, workspace),
		MCPs:        Catalog(),
		Workspace:   workspace,
		Profile:     ProfileEssencial,
		Step:        StepProfile,
		Checked:     map[string]bool{"golmet": true},
		ParamValues: map[string]string{},
		Home:        home,
	}
	m.allEngines = m.Engines
	return m
}

// ---- passo ENGINE (seleção única) ----

// EngineMove move o cursor com clamp [0, len-1].
func (m *WizardModel) EngineMove(delta int) {
	if len(m.Engines) == 0 {
		return
	}
	m.EngineCursor += delta
	if m.EngineCursor < 0 {
		m.EngineCursor = 0
	}
	if m.EngineCursor >= len(m.Engines) {
		m.EngineCursor = len(m.Engines) - 1
	}
	m.Notice = ""
}

// EngineConfirm trava o engine escolhido e avança para o passo MCPs (o
// perfil — decisão macro — já foi escolhido no passo anterior). Na
// PRIMEIRA confirmação roda a detecção de MCPs já registrados (1
// chamada de list por wizard — cache na struct) e aplica os defaults:
// já-registrados desmarcados (nada a fazer — remarcar é opt-in), golmet
// permanece marcado sempre.
func (m *WizardModel) EngineConfirm() {
	if len(m.Engines) == 0 {
		m.Notice = "nenhum engine disponível"
		return
	}
	m.EngineChosen = m.Engines[m.EngineCursor].ID
	if !m.registeredDone {
		m.registeredDone = true
		m.Registered = installedMCPsFn(m.Home, m.Workspace, m.EngineChosen)
		for name := range m.Registered {
			if name == "golmet" {
				continue // obrigatório: permanece marcado sempre
			}
			m.Checked[name] = false
		}
	}
	m.Step = StepMCPs
	m.Notice = ""
}

// EngineDescription é a linha de 1 linha mostrada por `?` — o que o
// install faz nesse engine (skills/MCP/remote). Fonte: adapter registry.
func (m *WizardModel) EngineDescription(id string) string {
	adapter, ok := LookupEngine(id)
	if !ok {
		return ""
	}
	if d, ok := adapter.(interface{ Description() string }); ok {
		return d.Description()
	}
	return ""
}

// ---- passo PERFIL (primeira decisão: que experiência) ----

// ProfileChoice é uma opção do passo perfil com a copy canônica de 2
// linhas: o que é + limitação principal. Fonte única usada pelo wizard VT
// (modelo), pelo numerado (CLI) e pelo blurb de `?`.
type ProfileChoice struct {
	ID         string // essencial | completo
	What       string // linha 1 — o que é
	For        string // linha 2 — indicado para quem/cenário
	Limitation string // linha 3 — limitação principal
}

// ProfileChoices devolve as duas opções na ordem de exibição. A copy é
// ENGINE-AGNÓSTICA (o perfil é escolhido antes do engine): com completo, o
// próprio ProfileConfirm FILTRA a lista de engines para os compatíveis
// (skills nativas) — a lista curta fala por si.
func ProfileChoices() []ProfileChoice {
	return []ProfileChoice{
		{
			ID:         ProfileEssencial,
			What:       "kit acoplado ao repositório — 16 skills gm-* na zona .golmet/, versionadas com o repo: quem clona o projeto herda o kit",
			For:        "equipes que querem o workflow versionado e compartilhado — funciona em qualquer engine agêntico, sem instalar nada na máquina",
			Limitation: "sem agents nativos (nenhum subagent spawnável) — apenas o núcleo de documentos",
		},
		{
			ID:         ProfileCompleto,
			What:       "experiência completa na sua máquina — 125 componentes (agents + skills + rules) no formato nativo do engine, no config global (pessoal, não versionado)",
			For:        "quem adota o workflow como padrão pessoal de trabalho — exige engine com skills nativas (opencode ou claude-code); o wizard filtra a lista",
			Limitation: "compatível apenas com opencode e claude-code (derive via SkillsDir) — em engines sem skills nativas, use o perfil essencial",
		},
	}
}

// ProfileOption é a visão derivada para o render VT (id + blurb de 1 linha).
type ProfileOption struct {
	ID    string // essencial | completo
	Blurb string // o que é + limitação (o que é aqui · rodapé do render)
}

// ProfileOptions devolve as opções derivadas da copy canônica.
func (m *WizardModel) ProfileOptions() []ProfileOption {
	var out []ProfileOption
	for _, c := range ProfileChoices() {
		out = append(out, ProfileOption{ID: c.ID, Blurb: c.What + " — indicado para: " + c.For + " — limitação: " + c.Limitation})
	}
	return out
}

// ProfileMove move o cursor entre as opções com clamp [0, len-1].
func (m *WizardModel) ProfileMove(delta int) {
	n := len(ProfileChoices())
	m.ProfileCursor += delta
	if m.ProfileCursor < 0 {
		m.ProfileCursor = 0
	}
	if m.ProfileCursor >= n {
		m.ProfileCursor = n - 1
	}
	m.ProfileNotice = ""
}

// ProfileToggle alterna para a próxima opção (espaço) — cicla (com 2
// opções, espaço alterna essencial ↔ completo).
func (m *WizardModel) ProfileToggle() {
	n := len(ProfileChoices())
	if n == 0 {
		return
	}
	m.ProfileCursor = (m.ProfileCursor + 1) % n
	m.ProfileNotice = ""
}

// ProfileConfirm trava a escolha sob o cursor e avança para o passo
// engine. Regra de produto: o perfil completo exige engine com skills
// nativas — a lista de engines é FILTRADA para os compatíveis (derive via
// SkillsDir, nunca hardcode) e o cursor reseta; com essencial, a lista
// completa do registry é restaurada (voltar e re-escolher funciona).
func (m *WizardModel) ProfileConfirm() {
	choices := ProfileChoices()
	if len(choices) == 0 {
		m.ProfileNotice = "nenhum perfil disponível"
		return
	}
	if m.ProfileCursor >= len(choices) {
		m.ProfileCursor = 0
	}
	m.Profile = choices[m.ProfileCursor].ID
	if m.Profile == ProfileCompleto {
		m.Engines = compatibleEngines(m.allEngines, m.Home)
	} else {
		m.Engines = m.allEngines
	}
	m.EngineCursor = 0
	m.Step = StepEngine
	m.Notice = ""
	m.ProfileNotice = ""
}

// compatibleEngines devolve os engines com skills nativas (critério do
// perfil completo — derive via SkillsDir, nunca hardcode da lista).
func compatibleEngines(all []EngineInfo, home string) []EngineInfo {
	var out []EngineInfo
	for _, e := range all {
		if EngineSupportsCompleto(e.ID, home) {
			out = append(out, e)
		}
	}
	return out
}

// SetProfile posiciona a escolha por id ("essencial" | "completo" —
// case-insensitive) — usado por testes e transports não-interativos.
// id desconhecido = false com ProfileNotice (nunca sobrescreve a escolha
// válida anterior).
func (m *WizardModel) SetProfile(id string) bool {
	id = strings.ToLower(strings.TrimSpace(id))
	for i, c := range ProfileChoices() {
		if c.ID == id {
			m.ProfileCursor = i
			m.Profile = id
			m.ProfileNotice = ""
			return true
		}
	}
	m.ProfileNotice = "perfil desconhecido: " + id
	return false
}

// ---- passo MCPs (multi-toggle) ----

// MCPToggle alterna o MCP sob o cursor. golmet (obrigatório) NÃO desmarca —
// vira Notice inline; o retorno indica se alternou.
func (m *WizardModel) MCPToggle() bool {
	if len(m.MCPs) == 0 {
		return false
	}
	def := m.MCPs[m.MCPCursor]
	if def.Required && m.Checked[def.Name] {
		m.Notice = def.Name + " é obrigatório e não pode ser desmarcado"
		return false
	}
	m.Checked[def.Name] = !m.Checked[def.Name]
	m.Notice = ""
	return true
}

// MCPToggleAll alterna todos os MCPs NÃO-obrigatórios e NÃO-registrados
// de uma vez (tecla `a`): se algum está desmarcado, marca todos; senão
// desmarca. golmet permanece marcado; já-registrados ficam como estão
// (desmarcados por default — alternar o que já existe é opt-in via
// espaço, MCP por MCP).
func (m *WizardModel) MCPToggleAll() {
	var anyUnchecked bool
	for _, def := range m.MCPs {
		if def.Required {
			continue
		}
		if _, registered := m.Registered[def.Name]; registered {
			continue // já-registrados: fora do toggle-all
		}
		if !m.Checked[def.Name] {
			anyUnchecked = true
			break
		}
	}
	for _, def := range m.MCPs {
		if def.Required {
			continue
		}
		if _, registered := m.Registered[def.Name]; registered {
			continue
		}
		m.Checked[def.Name] = anyUnchecked
	}
	m.Notice = ""
}

// MCPMove move o cursor de MCP com clamp.
func (m *WizardModel) MCPMove(delta int) {
	if len(m.MCPs) == 0 {
		return
	}
	m.MCPCursor += delta
	if m.MCPCursor < 0 {
		m.MCPCursor = 0
	}
	if m.MCPCursor >= len(m.MCPs) {
		m.MCPCursor = len(m.MCPs) - 1
	}
}

// CheckedNames devolve os nomes marcados na ordem do catálogo.
func (m *WizardModel) CheckedNames() []string {
	var out []string
	for _, def := range m.MCPs {
		if m.Checked[def.Name] {
			out = append(out, def.Name)
		}
	}
	return out
}

// MCPConfirm valida e avança para o plano. Entradas com Param e valor
// ainda ausente são normalizadas como "nome=" — o Run classifica como
// PENDÊNCIA no relatório (config quebrada nunca; o wizard não trava o
// fluxo inteiro por um valor que o usuário preenche depois).
func (m *WizardModel) MCPConfirm() error {
	var entries []string
	for _, name := range m.CheckedNames() {
		def, ok := Lookup(name)
		if ok && def.Param != "" && !m.hasParamValue(name) {
			entries = append(entries, name+"=") // valor vazio → pendência no Run
			continue
		}
		entries = append(entries, name)
	}
	if _, err := ParseMCPSelection(entries); err != nil {
		m.Notice = err.Error()
		return err
	}
	m.Step = StepPlan
	m.Notice = ""
	return nil
}

// hasParamValue reporta se o usuário já informou o valor do param no
// wizard (conjunto via SetParamValue no prompt do render TTY).
func (m *WizardModel) hasParamValue(name string) bool {
	return m.ParamValues[name] != ""
}

// SetParamValue registra o valor do parâmetro capturado pelo prompt do
// render TTY (ex.: ORG da azure-devops).
func (m *WizardModel) SetParamValue(name, value string) {
	m.ParamValues[name] = strings.TrimSpace(value)
}

// ---- passo PLAN / cancelamento ----

// Plan é o resumo confirmável antes de executar.
type Plan struct {
	Engine          string   `json:"engine"`
	Profile         string   `json:"profile"`
	SkillsDir       string   `json:"skills_dir,omitempty"`
	SkillsSupported bool     `json:"skills_supported"`
	MCPs            []string `json:"mcps"`
	Workspace       string   `json:"workspace"`
	Version         string   `json:"version"`
}

// Plan gera o resumo do passo final (determinístico; nada executa aqui).
func (m *WizardModel) Plan() Plan {
	engine := m.EngineChosen
	skillsDir := ""
	supported := false
	if adapter, ok := LookupEngine(engine); ok {
		skillsDir = adapter.SkillsDir(m.Home)
		supported = skillsDir != ""
	}
	return Plan{
		Engine:          engine,
		Profile:         m.Profile,
		SkillsDir:       skillsDir,
		SkillsSupported: supported,
		MCPs:            m.CheckedNames(),
		Workspace:       m.Workspace,
		Version:         indexstate.KitVersion,
	}
}

// ActiveDescription devolve a descrição do item sob o cursor (rodapé de
// contexto do render TTY): description do engine no passo Engine, blurb da
// opção de perfil no passo Perfil, description do MCP no passo MCPs; ""
// no passo sem item ativo (Plano).
func (m *WizardModel) ActiveDescription() string {
	switch m.Step {
	case StepEngine:
		if len(m.Engines) == 0 {
			return ""
		}
		return m.EngineDescription(m.Engines[m.EngineCursor].ID)
	case StepProfile:
		opts := m.ProfileOptions()
		if m.ProfileCursor >= len(opts) {
			return ""
		}
		return opts[m.ProfileCursor].Blurb
	case StepMCPs:
		if len(m.MCPs) == 0 || m.MCPCursor >= len(m.MCPs) {
			return ""
		}
		return m.MCPs[m.MCPCursor].Description
	}
	return ""
}

// TruncateWidth corta a string na largura máxima, marcando o corte com
// "…" (a marca conta na largura — a linha resultante nunca passa de width).
func TruncateWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	if width == 1 {
		return "…"
	}
	return string(r[:width-1]) + "…"
}

// Cancel marca o cancelamento limpo (tecla q/esc em qualquer passo) —
// nada é escrito e o CLI sai 0 com mensagem.
func (m *WizardModel) Cancel() { m.Cancelled = true }

// Back volta um passo (tecla backspace/DEL). O estado NUNCA é zerado ao
// voltar: perfil escolhido, engine escolhido, MCPs marcados e valores de
// param ficam intactos — re-confirmar o passo reproduz o estado. No passo
// Perfil é no-op (primeiro passo não tem trás).
func (m *WizardModel) Back() {
	switch m.Step {
	case StepProfile:
		// primeiro passo — no-op
	case StepEngine:
		m.Step = StepProfile
	case StepMCPs:
		m.Step = StepEngine
	case StepPlan:
		m.Step = StepMCPs
	}
	m.Notice = ""
}

// Advance confirma o perfil sob o cursor e fecha o passo (enter) — o
// passo perfil é uma ESCOLHA essencial | completo; o TTY chama
// ProfileConfirm, que carrega a mesma semântica.
func (m *WizardModel) Advance() {
	if m.Step == StepProfile {
		m.ProfileConfirm()
	}
}

// ToOptions converte o plano aprovado em Options para o install.Run —
// entradas MCP no formato canônico "nome" ou "nome=param" (valor vazio
// quando o usuário ainda não o informou → pendência no Run).
func (m *WizardModel) ToOptions() Options {
	plan := m.Plan()
	entries := make([]string, 0, len(plan.MCPs))
	for _, name := range plan.MCPs {
		def, ok := Lookup(name)
		if ok && def.Param != "" {
			entries = append(entries, name+"="+m.ParamValues[name])
			continue
		}
		entries = append(entries, name)
	}
	return Options{
		EngineID:  plan.Engine,
		Profile:   plan.Profile,
		MCPs:      entries,
		Workspace: m.Workspace,
		Home:      m.Home,
	}
}

// ---- não-TTY (scriptabilidade) ----

// NonTTYError é o erro estruturado quando o wizard interativo é pedido em
// stdin/stdout não-TTY (pipe/CI) sem flags suficientes — traz o comando
// exato de flags no fix (§11-style).
type NonTTYError struct {
	Engine string // engine default sugerido no fix (opencode)
}

func (e *NonTTYError) Error() string {
	return "stdin/stdout não é um terminal (pipe/CI) — o wizard interativo precisa de TTY"
}

// Fix devolve a remediação §11-style: o comando exato de flags.
func (e *NonTTYError) Fix() string {
	engine := e.Engine
	if engine == "" {
		engine = "opencode"
	}
	return fmt.Sprintf("rode com flags: golmet install --engine %s --mcp golmet --workspace . --yes (listas: --list-engines, --list-mcps)", engine)
}

// CheckInteractive decide o caminho do wizard: TTY presente → interativo
// de teclas; --plain → modo numerado; não-TTY sem flags → NonTTYError.
// engineID == "" significa que as flags não bastam.
func CheckInteractive(engineID string, tty, plain bool) error {
	if engineID != "" {
		return nil // flags suficientes — não-interativo
	}
	if !tty && !plain {
		return &NonTTYError{}
	}
	return nil // TTY → raw-mode; --plain → numerado
}

// NonTTYFix devolve o fix do NonTTYError quando err é desse tipo ("" caso
// contrário) — evita type-assert no CLI.
func NonTTYFix(err error) string {
	if nte, ok := err.(*NonTTYError); ok {
		return nte.Fix()
	}
	return ""
}

// TruncateNames formata a lista de nomes para a linha agregada/detalhe:
// até limit nomes, "… +N" quando passa (usado pelo render TTY).
func TruncateNames(names []string, limit int) string {
	if len(names) <= limit {
		return strings.Join(names, ", ")
	}
	return strings.Join(names[:limit], ", ") + fmt.Sprintf(", … +%d", len(names)-limit)
}
