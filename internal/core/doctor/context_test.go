package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---- contexto engine-config (M21): o dir dado É o config root do engine ----

// O doctor sobre o config root do engine roda SÓ os checks de máquina +
// o engine-config DETALHADO (corpo principal) — os checks de workspace-
// produto não se aplicam (o config não é workspace; a falha inválida de
// "golmet.yaml ausente" some).
func TestDoctorEngineConfigContext(t *testing.T) {
	cfg := t.TempDir()
	// 2 geridos conforme (core + local), 1 não-reconstrutível, 1 pessoal
	writeContentManifestFixture(t, cfg, map[string]manifestEntryFixture{
		"agents/x.md":  {SHA: shaOf("v1\n"), Layer: "core"},
		"profile/p.md": {SHA: shaOf("perfil\n"), Layer: "local"},
	})
	writeFile(t, cfg, "agents/x.md", "v1\n")
	writeFile(t, cfg, "profile/p.md", "perfil\n")
	writeFile(t, cfg, "plugins/nativo.js", "código do usuário\n")
	writePersonalFixture(t, cfg, []string{"profile/client-profile.md"})
	writeFile(t, cfg, "profile/client-profile.md", "dado do cliente (gitignored por design)\n")

	rep, err := Run(cfg, Options{EngineConfigRoot: cfg, NoNetwork: true})
	if err != nil {
		t.Fatalf("doctor sobre o config root não deveria falhar: %v\n%s", err, rep.Human())
	}
	if rep.Context != ContextEngineConfig {
		t.Errorf("contexto = %q, want %q", rep.Context, ContextEngineConfig)
	}
	// sem o check inválido de golmet.yaml (o config não é workspace)
	for _, c := range rep.Checks {
		if c.Item == "golmet.yaml" {
			t.Errorf("check golmet.yaml não deveria existir no contexto engine-config: %+v", c)
		}
		if c.Item == "worklist-s2" || c.Item == "manifest" || c.Item == "capacidades" || c.Item == "indice" {
			t.Errorf("check de workspace-produto %q não deveria rodar sobre o config", c.Item)
		}
	}
	// checks de máquina presentes
	checkOf(t, rep, "git")
	checkOf(t, rep, "kit.jev")
	checkOf(t, rep, "kit.graph")
	// engine-config como corpo principal, com estado classificado
	checkOf(t, rep, "engine-config")
	if rep.EngineConfig == nil {
		t.Fatalf("seção engine_config ausente do relatório")
	}
	if rep.EngineConfig.ManagedOK != 2 {
		t.Errorf("geridos = %d, want 2", rep.EngineConfig.ManagedOK)
	}
	if len(rep.EngineConfig.ManagedLayers) != 2 {
		t.Errorf("camadas dos geridos = %v, want core+local", rep.EngineConfig.ManagedLayers)
	}
	if rep.EngineConfig.UntrackedTotal != 1 {
		t.Errorf("não-reconstrutíveis = %d, want 1 (plugins/nativo.js)", rep.EngineConfig.UntrackedTotal)
	}
	if rep.EngineConfig.PersonalTotal != 1 {
		t.Errorf("pessoal-por-design = %d, want 1", rep.EngineConfig.PersonalTotal)
	}
	h := rep.Human()
	if !strings.Contains(h, "contexto: engine-config") {
		t.Errorf("Human sem mensagem de contexto:\n%s", h)
	}
	if !strings.Contains(h, "relatório completo do engine-config") {
		t.Errorf("Human sem o cabeçalho do relatório detalhado:\n%s", h)
	}
	if !strings.Contains(h, "core: 1") || !strings.Contains(h, "local: 1") {
		t.Errorf("Human sem as camadas dos geridos:\n%s", h)
	}
}

// Dir inexistente == config root: o contexto engine-config trata com a
// mensagem própria de raiz inexistente (não é "workspace não encontrado").
func TestDoctorEngineConfigContextRootMissing(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "opencode")
	rep, err := Run(cfg, Options{EngineConfigRoot: cfg})
	if err != nil {
		t.Fatalf("config root inexistente é estado legítimo no contexto engine-config: %v", err)
	}
	if rep.Context != ContextEngineConfig {
		t.Errorf("contexto = %q, want %q", rep.Context, ContextEngineConfig)
	}
	if c := checkOf(t, rep, "engine-config"); !strings.Contains(c.Detail, "inexistente") {
		t.Errorf("check engine-config deveria reportar raiz inexistente: %+v", c)
	}
}

// Sem match com o config root, dir sem golmet.yaml continua erro claro
// de workspace não inicializado.
func TestDoctorNonWorkspaceWithoutMatch(t *testing.T) {
	ws := t.TempDir()
	_, err := Run(ws, Options{EngineConfigRoot: filepath.Join(os.TempDir(), "outro-config")})
	if err == nil || !strings.Contains(err.Error(), "golmet.yaml ausente") {
		t.Fatalf("workspace sem golmet.yaml deveria falhar: %v", err)
	}
}

// ---- contexto container e workspace-produto: mensagem de contexto ----

func TestDoctorContainerContextLabel(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "golmet.yaml"), []byte("format: 1\ncontainer: true\nproducts: [alfa]\n"), 0o644)
	os.MkdirAll(filepath.Join(root, "alfa"), 0o755)

	rep, err := Run(root, Options{NoNetwork: true})
	if err != nil {
		t.Fatalf("container íntegro não deveria falhar: %v", err)
	}
	if rep.Context != ContextContainer {
		t.Errorf("contexto = %q, want %q", rep.Context, ContextContainer)
	}
	checkOf(t, rep, "container-products") // checklist de hierarquia intacto
	if !strings.Contains(rep.Human(), "contexto: container") {
		t.Errorf("Human sem mensagem de contexto:\n%s", rep.Human())
	}
}

func TestDoctorWorkspaceContextLabel(t *testing.T) {
	ws := filepath.Join(t.TempDir(), "ws")
	os.MkdirAll(ws, 0o755)
	os.WriteFile(filepath.Join(ws, "golmet.yaml"), []byte("format: 1\n"), 0o644)

	rep, err := Run(ws, Options{NoNetwork: true})
	if err != nil {
		t.Fatalf("workspace mínimo não deveria falhar: %v", err)
	}
	if rep.Context != ContextWorkspaceProduct {
		t.Errorf("contexto = %q, want %q", rep.Context, ContextWorkspaceProduct)
	}
	if !strings.Contains(rep.Human(), "contexto: workspace-produto") {
		t.Errorf("Human sem mensagem de contexto:\n%s", rep.Human())
	}
}
