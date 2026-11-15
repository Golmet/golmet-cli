package install

import (
	"strings"
	"testing"
)

// passo perfil do wizard: PRIMEIRO passo (decisão macro — que experiência)
// com copy de contrato: essencial (kit do repositório) e completo
// (experiência completa na máquina). Modelo puro — o TTY só mapeia teclas.

// default essencial na entrada do wizard; cursor na primeira opção.
func TestWizardProfileDefaultsEssencial(t *testing.T) {
	m := newTestModel(t)
	if m.Step != StepProfile {
		t.Fatalf("step inicial = %v, want StepProfile (perfil vem primeiro)", m.Step)
	}
	if m.Profile != ProfileEssencial {
		t.Errorf("perfil default = %q, want %q", m.Profile, ProfileEssencial)
	}
	if m.ProfileCursor != 0 {
		t.Errorf("cursor default = %d, want 0 (essencial)", m.ProfileCursor)
	}
	opts := m.ProfileOptions()
	if len(opts) != 2 || opts[0].ID != ProfileEssencial || opts[1].ID != ProfileCompleto {
		t.Fatalf("opções = %+v, want [essencial completo]", opts)
	}
}

// copy de contrato (2 linhas por opção): o que é + limitação principal.
func TestWizardProfileChoicesCopy(t *testing.T) {
	choices := ProfileChoices()
	if len(choices) != 2 {
		t.Fatalf("choices = %d, want 2", len(choices))
	}
	if !strings.Contains(choices[0].What, "16 skills") || !strings.Contains(choices[0].What, "versionadas com o repo") {
		t.Errorf("essencial.What sem o kit do repositório: %q", choices[0].What)
	}
	if !strings.Contains(choices[0].Limitation, "sem agents nativos") {
		t.Errorf("essencial.Limitation sem a limitação: %q", choices[0].Limitation)
	}
	if !strings.Contains(choices[1].What, "125 componentes") || !strings.Contains(choices[1].What, "config global") {
		t.Errorf("completo.What sem a experiência completa: %q", choices[1].What)
	}
	if !strings.Contains(choices[1].Limitation, "compatível apenas com opencode e claude-code") ||
		!strings.Contains(choices[1].Limitation, "use o perfil essencial") {
		t.Errorf("completo.Limitation sem a restrição de engines: %q", choices[1].Limitation)
	}
}

// ProfileMove com clamp nas duas pontas; ProfileToggle cicla (espaço).
func TestWizardProfileMoveAndToggle(t *testing.T) {
	m := newTestModel(t)
	m.ProfileMove(-1)
	if m.ProfileCursor != 0 {
		t.Errorf("clamp superior: %d", m.ProfileCursor)
	}
	m.ProfileMove(1)
	if m.ProfileCursor != 1 {
		t.Errorf("move para completo: %d", m.ProfileCursor)
	}
	m.ProfileMove(1)
	if m.ProfileCursor != 1 {
		t.Errorf("clamp inferior: %d", m.ProfileCursor)
	}
	m.ProfileToggle() // ciclo: completo → essencial
	if m.ProfileCursor != 0 {
		t.Errorf("toggle do completo: %d, want 0", m.ProfileCursor)
	}
	m.ProfileToggle() // ciclo: essencial → completo
	if m.ProfileCursor != 1 {
		t.Errorf("toggle do essencial: %d, want 1", m.ProfileCursor)
	}
}

// Ordem: perfil → engine → MCPs. ProfileConfirm avança para o ENGINE (e
// trava a escolha); EngineConfirm avança para os MCPs.
func TestWizardProfileConfirmChoosesCompletoThenEngine(t *testing.T) {
	m := newTestModel(t)
	m.ProfileMove(1)
	m.ProfileConfirm()
	if m.Profile != ProfileCompleto {
		t.Errorf("perfil confirmado = %q, want %q", m.Profile, ProfileCompleto)
	}
	if m.Step != StepEngine {
		t.Fatalf("step pós-perfil = %v, want StepEngine", m.Step)
	}
	m.EngineCursor = 1 // claude-code
	m.EngineConfirm()
	if m.Step != StepMCPs {
		t.Fatalf("step pós-engine = %v, want StepMCPs", m.Step)
	}
	if plan := m.Plan(); plan.Profile != ProfileCompleto {
		t.Errorf("plan.Profile = %q, want completo", plan.Profile)
	}
	// ToOptions carrega a escolha (o Run normaliza completo ≡ Full)
	if opt := m.ToOptions(); opt.Profile != ProfileCompleto {
		t.Errorf("ToOptions.Profile = %q, want completo", opt.Profile)
	}
}

// SetProfile por id: válido posiciona; inválido preserva a escolha anterior
// e sinaliza no ProfileNotice.
func TestWizardProfileSetProfile(t *testing.T) {
	m := newTestModel(t)
	if !m.SetProfile("COMPLETO") { // case-insensitive/trim
		t.Fatalf("SetProfile(completo) = false")
	}
	if m.ProfileCursor != 1 || m.Profile != ProfileCompleto {
		t.Errorf("SetProfile(completo): cursor=%d profile=%q", m.ProfileCursor, m.Profile)
	}
	if m.SetProfile("enterprise") {
		t.Errorf("perfil desconhecido deveria falhar")
	}
	if m.Profile != ProfileCompleto || m.ProfileCursor != 1 {
		t.Errorf("falha do SetProfile alterou a escolha válida: %q cursor=%d", m.Profile, m.ProfileCursor)
	}
	if !strings.Contains(m.ProfileNotice, "enterprise") {
		t.Errorf("notice do perfil inválido ausente: %q", m.ProfileNotice)
	}
}

// regra de produto: perfil completo exige engine com skills nativas —
// ProfileConfirm FILTRA a lista (derive via SkillsDir, nunca hardcode) e
// reseta o cursor; essencial restaura a lista completa (voltar e
// re-escolher funciona).
func TestWizardProfileConfirmFiltersEngines(t *testing.T) {
	if got := CompatibleEngineIDs("home"); len(got) != 2 || got[0] != "opencode" || got[1] != "claude-code" {
		t.Fatalf("compatíveis = %v, want [opencode claude-code]", got)
	}
	if EngineSupportsCompleto("codex", "home") || EngineSupportsCompleto("warp", "home") {
		t.Errorf("engines sem SkillsDir não são compatíveis com completo")
	}

	m := newTestModel(t)
	if len(m.Engines) != 7 {
		t.Fatalf("lista inicial = %d engines, want 7 (essencial aceita todos)", len(m.Engines))
	}
	m.SetProfile(ProfileCompleto)
	m.ProfileConfirm()
	if len(m.Engines) != 2 || m.Engines[0].ID != "opencode" || m.Engines[1].ID != "claude-code" {
		t.Fatalf("lista pós-completo = %+v, want [opencode claude-code]", engineIDs(m.Engines))
	}
	if m.EngineCursor != 0 {
		t.Errorf("cursor deveria resetar para 0, veio %d", m.EngineCursor)
	}
	if m.Step != StepEngine {
		t.Errorf("step pós-confirm = %v, want StepEngine", m.Step)
	}

	// voltar ao perfil e re-escolher essencial restaura a lista completa
	m.Back()
	if m.Step != StepProfile {
		t.Fatalf("back = %v, want StepProfile", m.Step)
	}
	m.SetProfile(ProfileEssencial)
	m.ProfileConfirm()
	if len(m.Engines) != 7 {
		t.Fatalf("lista pós-essencial = %d engines, want 7", len(m.Engines))
	}
	if m.EngineCursor != 0 {
		t.Errorf("cursor deveria resetar para 0, veio %d", m.EngineCursor)
	}
}

func engineIDs(all []EngineInfo) []string {
	var out []string
	for _, e := range all {
		out = append(out, e.ID)
	}
	return out
}

// Back preserva o estado: MCPs → engine → perfil (escolha intacta);
// perfil é o primeiro passo — back nele é no-op.
func TestWizardProfileBackPreservesChoice(t *testing.T) {
	m := newTestModel(t)
	m.SetProfile(ProfileCompleto)
	m.ProfileConfirm() // → engine
	m.EngineCursor = 1
	m.EngineConfirm() // → MCPs
	m.Back()
	if m.Step != StepEngine {
		t.Fatalf("back de MCPs = %v, want StepEngine", m.Step)
	}
	m.Back()
	if m.Step != StepProfile {
		t.Fatalf("back de engine = %v, want StepProfile", m.Step)
	}
	if m.Profile != ProfileCompleto || m.ProfileCursor != 1 {
		t.Errorf("back perdeu a escolha: %q cursor=%d", m.Profile, m.ProfileCursor)
	}
	m.Back() // primeiro passo — no-op
	if m.Step != StepProfile {
		t.Errorf("back no primeiro passo deveria ser no-op: %v", m.Step)
	}
	m.ProfileConfirm() // re-confirma — mesmo estado
	if m.Profile != ProfileCompleto || m.Step != StepEngine {
		t.Errorf("re-confirm divergente: %q %v", m.Profile, m.Step)
	}
}
