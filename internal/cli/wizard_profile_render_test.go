package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Golmet/golmet-cli/internal/core/install"
)

// newProfileTestModel monta o modelo em home virtual. A lista de engines é
// sempre os 7 do registry (deteção só muda o badge) — a escolha por índice
// é determinística; a detecção de MCPs pode sondar CLIs reais (mesmo padrão
// de TestMCPFrameUniformPrefix) mas nenhum assertion aqui depende dela.
func newProfileTestModel(t *testing.T) *install.WizardModel {
	t.Helper()
	home := t.TempDir()
	return install.NewWizardModel(home, filepath.Join(home, "ws"))
}

// dispatch de teclas do TTY: perfil (PRIMEIRO) → escolhe completo (setas +
// espaço) → engine → MCPs; backspace volta preservando a escolha — o mesmo
// percurso das setas no terminal, sem console real.
func TestWizardTTYDispatchNavigatesProfileFirstChoosesCompleto(t *testing.T) {
	m := newProfileTestModel(t)
	step := 0

	if m.Step != install.StepProfile || step != 0 {
		t.Fatalf("entrada: step=%d model=%v, want 0/StepProfile", step, m.Step)
	}

	// ↑ clamp e ↓ até completo
	applyWizardKey(m, &step, keyUp)
	if m.ProfileCursor != 0 {
		t.Errorf("clamp do ↑ no topo: %d", m.ProfileCursor)
	}
	applyWizardKey(m, &step, keyDown)
	applyWizardKey(m, &step, keyDown)
	if m.ProfileCursor != 1 {
		t.Fatalf("setas não chegaram ao completo: %d", m.ProfileCursor)
	}

	// espaço alterna (completo → essencial) e volta (essencial → completo)
	applyWizardKey(m, &step, keySpace)
	if m.ProfileCursor != 0 {
		t.Errorf("espaço não alternou para essencial: %d", m.ProfileCursor)
	}
	applyWizardKey(m, &step, keySpace)
	if m.ProfileCursor != 1 {
		t.Errorf("espaço não alternou para completo: %d", m.ProfileCursor)
	}

	// enter confirma completo → passo ENGINE (não MCPs)
	if applyWizardKey(m, &step, keyEnter) {
		t.Fatalf("enter no perfil não deveria sair")
	}
	if m.Profile != install.ProfileCompleto || step != 1 || m.Step != install.StepEngine {
		t.Fatalf("completo não confirmado: profile=%q step=%d model=%v", m.Profile, step, m.Step)
	}

	// no passo engine, `?` mostra a description do engine sob o cursor
	applyWizardKey(m, &step, "?")
	if m.Notice == "" {
		t.Errorf("`?` no engine deveria mostrar a description")
	}

	// enter confirma o engine → MCPs
	if applyWizardKey(m, &step, keyEnter) {
		t.Fatalf("enter no engine não deveria sair")
	}
	if step != 2 || m.Step != install.StepMCPs {
		t.Fatalf("engine não confirmado: step=%d model=%v", step, m.Step)
	}

	// backspace volta ao ENGINE; de novo → PERFIL com a escolha preservada
	applyWizardKey(m, &step, keyBack)
	if step != 1 {
		t.Fatalf("back de MCPs: step=%d, want 1 (engine)", step)
	}
	applyWizardKey(m, &step, keyBack)
	if step != 0 || m.Profile != install.ProfileCompleto || m.ProfileCursor != 1 {
		t.Fatalf("back ao perfil: step=%d profile=%q cursor=%d", step, m.Profile, m.ProfileCursor)
	}

	// backspace no primeiro passo = no-op com notice
	applyWizardKey(m, &step, keyBack)
	if step != 0 || !strings.Contains(m.ProfileNotice, "primeiro passo") {
		t.Errorf("back no primeiro passo: step=%d notice=%q", step, m.ProfileNotice)
	}

	// re-confirma — mesmo estado; `?` traz o blurb da opção sob o cursor
	applyWizardKey(m, &step, keyEnter)
	if m.Step != install.StepEngine || m.Profile != install.ProfileCompleto {
		t.Fatalf("re-confirm divergente: %q %v", m.Profile, m.Step)
	}
	applyWizardKey(m, &step, keyBack) // volta ao perfil
	applyWizardKey(m, &step, "?")
	if !strings.Contains(m.ProfileNotice, "125 componentes") {
		t.Errorf("`?` não trouxe o blurb do completo: %q", m.ProfileNotice)
	}
	applyWizardKey(m, &step, keyUp)
	applyWizardKey(m, &step, "?")
	if !strings.Contains(m.ProfileNotice, "16 skills") {
		t.Errorf("`?` não trouxe o blurb do essencial: %q", m.ProfileNotice)
	}

	// cancelamento (q) sai do loop e marca o modelo
	if !applyWizardKey(m, &step, keyCancel) {
		t.Fatalf("q deveria sinalizar saída")
	}
	if !m.Cancelled {
		t.Errorf("q deveria cancelar o wizard")
	}
}

// frame do passo perfil (primeiro passo): as duas opções com copy de 3
// linhas — o que é + indicado para + limitação — e checkbox na opção ativa.
func TestWizardTTYDrawProfileStep(t *testing.T) {
	m := newProfileTestModel(t)
	m.ProfileMove(1) // cursor no completo

	var buf strings.Builder
	c := newTTYColors(false)
	drawProfileStep(&rawWriter{alt: &buf, c: c}, m, c, 200)
	out := buf.String()

	for _, want := range []string{
		"Perfil",
		"essencial",
		"completo",
		"kit acoplado ao repositório",
		"quem clona o projeto herda o kit",
		"indicado para: equipes que querem o workflow versionado",
		"limitação: sem agents nativos (nenhum subagent spawnável)",
		"125 componentes (agents + skills + rules)",
		"formato nativo do engine",
		"indicado para: quem adota o workflow como padrão pessoal",
		"compatível apenas com opencode e claude-code",
		"use o perfil essencial",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("frame do perfil sem %q:\n%s", want, out)
		}
	}
	// checkbox marcado na linha do completo (opção sob o cursor)
	if line := lineOf(out, "[x]"); !strings.Contains(line, "completo") {
		t.Errorf("[x] na linha errada: %q", line)
	}
	// coluna do checkbox uniforme entre as duas linhas
	colE := strings.Index(lineOf(out, "essencial"), "[")
	colC := strings.Index(lineOf(out, "completo"), "[")
	if colE != colC {
		t.Errorf("coluna do checkbox difere: essencial=%d completo=%d", colE, colC)
	}
}

// passo engine com perfil completo: o modelo FILTRA a lista para os
// compatíveis (opencode, claude-code) — o frame mostra só esses dois e
// nunca uma nota de zona (engine fraco não existe mais na lista).
func TestWizardTTYDrawEngineStepFilteredForCompleto(t *testing.T) {
	m := newProfileTestModel(t)
	m.SetProfile(install.ProfileCompleto)
	m.ProfileConfirm()
	if len(m.Engines) != 2 {
		t.Fatalf("lista pós-completo = %d, want 2", len(m.Engines))
	}
	var buf strings.Builder
	c := newTTYColors(false)
	drawEngineStep(&rawWriter{alt: &buf, c: c}, m, c, 100)
	out := buf.String()
	if !strings.Contains(out, "opencode") || !strings.Contains(out, "claude-code") {
		t.Errorf("frame sem os engines compatíveis:\n%s", out)
	}
	for _, weak := range []string{"cursor", "codex", "gemini-cli", "vscode-copilot", "copilot-cli"} {
		if lineOf(out, weak) != "" {
			t.Errorf("engine %s não deveria aparecer com perfil completo:\n%s", weak, out)
		}
	}
	if strings.Contains(out, "zona do workspace") {
		t.Errorf("nota de zona não deveria mais existir no fluxo completo:\n%s", out)
	}
}

// plano mostra o perfil escolhido (o usuário vê o que confirmou).
func TestWizardTTYPlanShowsProfile(t *testing.T) {
	m := newProfileTestModel(t)
	m.SetProfile(install.ProfileCompleto)
	m.ProfileConfirm()
	m.EngineCursor = 1
	m.EngineConfirm()
	if err := m.MCPConfirm(); err != nil {
		t.Fatalf("MCPConfirm: %v", err)
	}
	var buf strings.Builder
	c := newTTYColors(false)
	drawPlanStep(&rawWriter{alt: &buf, c: c}, m, c)
	if !strings.Contains(buf.String(), "perfil:    completo") {
		t.Errorf("plano sem a linha do perfil:\n%s", buf.String())
	}
}

func lineOf(s, needle string) string {
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(line, needle) {
			return line
		}
	}
	return ""
}
