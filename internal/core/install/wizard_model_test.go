package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestModel(t *testing.T) *WizardModel {
	t.Helper()
	home := newHome(t)
	// detect tudo: pastas dos engines + copilot por lookPath fake
	os.MkdirAll(filepath.Join(home, ".config", "opencode"), 0o755)
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	os.MkdirAll(filepath.Join(home, ".cursor"), 0o755)
	os.MkdirAll(filepath.Join(home, ".gemini"), 0o755)
	os.MkdirAll(filepath.Join(home, ".codex"), 0o755)
	os.Unsetenv("COPILOT_HOME")
	old := lookPathFn
	lookPathFn = func(string) (string, error) { return "copilot-fake", nil }
	t.Cleanup(func() { lookPathFn = old })
	// detecção de já-registrados DESLIGADA nos testes do modelo (nunca roda
	// a CLI real do operador; os comportamentos de registro têm testes
	// dedicados com mock em wizard_registered_test.go)
	oldDetect := installedMCPsFn
	installedMCPsFn = func(string, string, string) map[string]string { return nil }
	t.Cleanup(func() { installedMCPsFn = oldDetect })

	m := NewWizardModel(home, filepath.Join(home, "ws"))
	if m.Step != StepProfile {
		t.Fatalf("step inicial = %v, want StepProfile (perfil vem primeiro)", m.Step)
	}
	if len(m.Engines) != 7 {
		t.Fatalf("engines = %d, want 7", len(m.Engines))
	}
	// golmet pré-marcado; codegraph NÃO (download de 318MB não é surpresa)
	if !m.Checked["golmet"] {
		t.Errorf("golmet deveria vir pré-marcado")
	}
	if m.Checked["codegraph"] {
		t.Errorf("codegraph não deveria vir pré-marcado (download de 318MB não é surpresa)")
	}
	return m
}

func TestWizardEngineSelectAndAdvance(t *testing.T) {
	m := newTestModel(t)
	// perfil vem primeiro: enter confirma essencial e vai ao engine
	if m.Step != StepProfile {
		t.Fatalf("step inicial = %v, want StepProfile", m.Step)
	}
	m.Advance()
	if m.Step != StepEngine {
		t.Fatalf("Advance no perfil = %v, want StepEngine", m.Step)
	} // setas navegam com clamp
	m.EngineMove(-1)
	if m.EngineCursor != 0 {
		t.Errorf("clamp superior da home falhou: %d", m.EngineCursor)
	}
	m.EngineMove(2)
	if m.EngineCursor != 2 {
		t.Errorf("EngineMove = %d, want 2", m.EngineCursor)
	}
	chosen := m.Engines[m.EngineCursor].ID
	m.EngineConfirm()
	if m.EngineChosen != chosen || m.Step != StepMCPs {
		t.Errorf("EngineConfirm = %q step %v", m.EngineChosen, m.Step)
	}
	// descrição de 1 linha para o `?`
	desc := m.EngineDescription(chosen)
	if desc == "" {
		t.Errorf("engine %s sem description", chosen)
	}
}

func TestWizardEngineOnlyOneSelection(t *testing.T) {
	m := newTestModel(t)
	m.EngineCursor = 3
	m.EngineConfirm()
	if m.EngineChosen != m.Engines[3].ID {
		t.Errorf("engine único violado: escolhido %q", m.EngineChosen)
	}
	// confirmar de novo não muda (passo já fechado — chamada idempotente
	// do modelo re-confirma o cursor atual; o estado continua um engine só)
	m.EngineCursor = 0
	m.EngineConfirm()
	if strings.Count(strings.ToLower(m.EngineChosen), ",") != 0 {
		t.Errorf("múltiplos engines na seleção: %q", m.EngineChosen)
	}
}

func TestWizardMCPToggleAndGolmetLocked(t *testing.T) {
	m := newTestModel(t)
	m.EngineConfirm() // perfil (default essencial) já escolhido — direto ao engine → MCPs
	// cursor começa no golmet (índice 0) — desmarcar é proibido
	if m.MCPToggle() {
		t.Errorf("golmet desmarcado — obrigatório é travado")
	}
	if !m.Checked["golmet"] {
		t.Errorf("golmet perdeu a marca")
	}
	if !strings.Contains(m.Notice, "obrigatório") {
		t.Errorf("notice inline ausente: %q", m.Notice)
	}
	// marcar playwright (navegar até ele)
	m.MCPMove(1) // codegraph
	m.MCPMove(1) // azure-devops
	m.MCPMove(1) // github
	m.MCPMove(1) // supabase
	m.MCPMove(1) // cloudflare
	m.MCPMove(1) // aws
	m.MCPMove(1) // sqlite
	m.MCPMove(1) // figma
	m.MCPMove(1) // playwright
	if !m.MCPToggle() {
		t.Errorf("toggle do playwright falhou")
	}
	if !m.Checked["playwright"] {
		t.Errorf("playwright deveria estar marcado")
	}
}

func TestWizardToggleAll(t *testing.T) {
	m := newTestModel(t)
	m.EngineConfirm() // perfil (default essencial) já escolhido — direto ao engine → MCPs
	// estado inicial: só golmet marcado → `a` marca todos os não-obrigatórios
	m.MCPToggleAll()
	for _, def := range m.MCPs {
		if !m.Checked[def.Name] {
			t.Errorf("toggle-all deveria marcar %s", def.Name)
		}
	}
	if !m.Checked["golmet"] {
		t.Errorf("golmet deveria continuar marcado")
	}
	// segundo `a` → desmarca todos os não-obrigatórios, golmet fica
	m.MCPToggleAll()
	for _, def := range m.MCPs {
		if def.Required {
			continue
		}
		if m.Checked[def.Name] {
			t.Errorf("toggle-all deveria desmarcar %s", def.Name)
		}
	}
	if !m.Checked["golmet"] {
		t.Errorf("golmet não pode ser desmarcado pelo toggle-all")
	}
}

func TestWizardPlanAndCancel(t *testing.T) {
	home := newHome(t)
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	m := NewWizardModel(home, filepath.Join(home, "ws"))
	m.ProfileConfirm() // essencial (default)
	m.EngineMove(1)    // claude-code
	m.EngineConfirm()
	m.MCPToggleAll()
	if err := m.MCPConfirm(); err != nil {
		t.Fatalf("MCPConfirm: %v", err)
	}
	if m.Step != StepPlan {
		t.Fatalf("step = %v, want StepPlan", m.Step)
	}
	plan := m.Plan()
	if plan.Engine != "claude-code" || plan.Profile != ProfileEssencial {
		t.Errorf("plan header = %+v", plan)
	}
	// toggle-all marca os 12 não-obrigatórios + golmet (13 selecionados);
	// azure-devops entra no plano SEM valor (pendência classificada no Run)
	if len(plan.MCPs) != 13 {
		t.Errorf("plan mcps = %d, want 13 (golmet + toggle-all)", len(plan.MCPs))
	}
	if plan.Version == "" || plan.SkillsDir == "" || !plan.SkillsSupported {
		t.Errorf("plan incompleto: %+v", plan)
	}
	// ToOptions produz Options consistentes (com o formato canônico de
	// param: azure-devops= vazio → pendência no Run)
	opt := m.ToOptions()
	if opt.EngineID != plan.Engine || opt.Home != home {
		t.Errorf("ToOptions divergente do plano: %+v vs %+v", opt, plan)
	}
	joined := strings.Join(opt.MCPs, ",")
	if !strings.Contains(joined, "azure-devops=") {
		t.Errorf("ToOptions deveria serializar o azure com slot de param: %s", joined)
	}
	if strings.Contains(joined, "@") {
		t.Errorf("placeholder vazou no ToOptions: %s", joined)
	}
	// cancelamento limpo
	m.Cancel()
	if !m.Cancelled {
		t.Errorf("Cancel deveria marcar Cancelled")
	}
}

// mesmo forçando o mapa sem golmet, o Confirm revalida contra o catálogo
// (ParseMCPSelection re-inclui o obrigatório)
func TestWizardMCPConfirmWithoutGolmetRevalidates(t *testing.T) {
	m := newTestModel(t)
	m.EngineConfirm() // perfil (default essencial) já escolhido — direto ao engine → MCPs
	delete(m.Checked, "golmet")
	if err := m.MCPConfirm(); err != nil {
		t.Errorf("confirm sem golmet deveria re-validar sem erro: %v", err)
	}
	if _, err := ParseMCPSelection(m.CheckedNames()); err != nil {
		t.Errorf("nomes pós-confirm inválidos: %v", err)
	}
}

// ---- ActiveDescription (rodapé de contexto) + truncamento ----

// ActiveDescription = descrição do item ATIVO (rodapé do render TTY) —
// muda ao navegar; vazio nos passos sem item ativo (Perfil/Plano).
func TestWizardActiveDescription(t *testing.T) {
	m := newTestModel(t)
	// passo perfil (primeiro): blurb da opção sob o cursor
	p0 := m.ActiveDescription()
	if p0 == "" || !strings.Contains(p0, "16 skills") {
		t.Fatalf("perfil sob o cursor sem o blurb: %q", p0)
	}
	m.ProfileMove(1)
	if p1 := m.ActiveDescription(); p1 == p0 || !strings.Contains(p1, "121 componentes") {
		t.Errorf("blurb do perfil deveria mudar ao navegar: %q", p1)
	}
	// passo engine: descrição do engine sob o cursor
	m.ProfileConfirm()
	m.EngineCursor = 0
	d0 := m.ActiveDescription()
	if d0 == "" {
		t.Fatalf("engine sob o cursor sem descrição")
	}
	m.EngineMove(1)
	d1 := m.ActiveDescription()
	if d1 == d0 {
		t.Errorf("descrição deveria mudar ao navegar: %q", d1)
	}
	// passo MCPs: descrição do MCP sob o cursor
	m.EngineConfirm()
	m.MCPCursor = 0
	dm0 := m.ActiveDescription()
	if dm0 == "" {
		t.Fatalf("mcp sob o cursor sem descrição")
	}
	m.MCPMove(2)
	dm2 := m.ActiveDescription()
	if dm2 == dm0 {
		t.Errorf("descrição deveria mudar ao navegar MCPs: %q", dm2)
	}
	// passos sem item ativo → vazio
	m.MCPConfirm()
	if got := m.ActiveDescription(); got != "" {
		t.Errorf("ActiveDescription no plano = %q, want vazio", got)
	}
}

// Truncamento na largura exata (a marca … conta na largura).
func TestTruncateWidth(t *testing.T) {
	long := "Azure DevOps MCP — boards, repos, pipelines e PRs da organização"
	if got := TruncateWidth(long, 100); got != long {
		t.Errorf("linha curta não deveria truncar: %q", got)
	}
	got := TruncateWidth(long, 30)
	if n := len([]rune(got)); n != 30 {
		t.Errorf("truncada com %d runes, want 30: %q", n, got)
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("truncada sem marca …: %q", got)
	}
	if TruncateWidth(long, 1) != "…" {
		t.Errorf("width 1 deveria ser só a marca")
	}
	if TruncateWidth(long, 0) != "" {
		t.Errorf("width 0 deveria ser vazio")
	}
}

// TruncateNames (linha agregada do doctor) — até limit, … +N quando passa.
func TestTruncateNames(t *testing.T) {
	names := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"}
	if got := TruncateNames(names, 6); !strings.Contains(got, "… +3") || !strings.Contains(got, "a,") {
		t.Errorf("truncamento = %q", got)
	}
	if got := TruncateNames(names[:3], 6); strings.Contains(got, "…") {
		t.Errorf("lista curta não deveria truncar: %q", got)
	}
}

// non-TTY: sem flags suficientes → NonTTYError com fix de flags; TTY e
// --plain seguem os caminhos interativos.
func TestCheckInteractive(t *testing.T) {
	// flags suficientes → não-interativo sempre
	if err := CheckInteractive("opencode", false, false); err != nil {
		t.Errorf("com --engine não deveria pedir interativo: %v", err)
	}
	// non-TTY sem flags → NonTTYError com fix
	err := CheckInteractive("", false, false)
	if err == nil {
		t.Fatalf("non-TTY sem flags deveria virar NonTTYError")
	}
	var nte *NonTTYError
	if !asNonTTY(err, &nte) {
		t.Fatalf("erro = %T, want *NonTTYError", err)
	}
	if !strings.Contains(nte.Fix(), "--engine opencode") || !strings.Contains(nte.Fix(), "--yes") {
		t.Errorf("fix deveria citar o comando de flags: %q", nte.Fix())
	}
	// NonTTYFix (accessor usado pelo CLI) devolve o mesmo fix
	if fix := NonTTYFix(err); fix != nte.Fix() {
		t.Errorf("NonTTYFix diverge do Fix do erro: %q", fix)
	}
	if fix := NonTTYFix(nil); fix != "" {
		t.Errorf("NonTTYFix(nil) = %q, want vazio", fix)
	}
	// TTY → interativo; --plain → numerado
	if err := CheckInteractive("", true, false); err != nil {
		t.Errorf("TTY deveria permitir interativo: %v", err)
	}
	if err := CheckInteractive("", false, true); err != nil {
		t.Errorf("--plain deveria permitir numerado mesmo em non-TTY: %v", err)
	}
}

func asNonTTY(err error, target **NonTTYError) bool {
	if e, ok := err.(*NonTTYError); ok {
		*target = e
		return true
	}
	return false
}
