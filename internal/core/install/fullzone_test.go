package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Golmet/golmet-cli/internal/core/update"
)

// (a) PONTO DE ENTRADA Run: perfil completo + engine SEM skills nativas
// (cursor) = erro fail-closed citando os compatíveis — uso não-interativo
// nunca degrada silenciosamente para a zona do workspace.
func TestRunCompletoWeakEngineErrors(t *testing.T) {
	noEngineCLI(t)
	home := newHome(t)
	ws := t.TempDir()
	res, err := Run(Options{
		EngineID: "cursor", Profile: ProfileCompleto,
		Workspace: ws, Home: home,
	})
	if err == nil {
		t.Fatalf("completo+cursor deveria falhar no Run: %+v", res.Report)
	}
	if !strings.Contains(err.Error(), "perfil completo exige engine com skills nativas") ||
		!strings.Contains(err.Error(), "opencode") || !strings.Contains(err.Error(), "claude-code") ||
		!strings.Contains(err.Error(), "essencial") {
		t.Errorf("mensagem de erro incompleta: %v", err)
	}
	// nada foi escrito na zona nem na home
	if _, serr := os.Stat(filepath.Join(ws, ".golmet")); !os.IsNotExist(serr) {
		t.Errorf("Run inválido escreveu na zona: %v", serr)
	}
	// alias --full + engine fraco falha igual (mesma validação)
	if _, err := Run(Options{EngineID: "cursor", Full: true, Home: newHome(t)}); err == nil ||
		!strings.Contains(err.Error(), "perfil completo") {
		t.Errorf("Full=true + cursor deveria falhar igual: %v", err)
	}
}

// (b) Coexistência BASE + FULL na MESMA zona .golmet/ do workspace —
// NÍVEL DE LIB (a zona é comportamento interno do pipeline; os pontos de
// entrada não mais a expõem): o manifest do BASE (manifest.json, schema
// update.Manifest) e o manifest de conteúdo (manifest-content.json, schema
// ContentManifest) vivem lado a lado sem corromper um ao outro.
func TestFullWorkspaceZoneCoexistsWithBaseManifest(t *testing.T) {
	noEngineCLI(t)
	home := newHome(t)
	ws := t.TempDir()
	// 1. BASE na zona (update.Install — escritor canônico do manifest.json)
	if _, err := update.Install(ws); err != nil {
		t.Fatalf("BASE install: %v", err)
	}
	baseManifestPath := filepath.Join(ws, ".golmet", "manifest.json")
	sentinel := readFileT(t, baseManifestPath)

	// 2. FULL na mesma zona (cursor — pipeline interno de zona)
	res, _, err := MaterializeFullWithOptions(home, "cursor", ws, false, false)
	if err != nil {
		t.Fatalf("FULL após BASE: %v", err)
	}
	if res == nil || res.Total != 125 {
		t.Fatalf("FULL na zona: total=%v", res)
	}
	// manifest BASE intocado pelo FULL
	if got := readFileT(t, baseManifestPath); got != sentinel {
		t.Errorf("FULL sobrescreveu o manifest BASE (manifest.json)")
	}
	mf, err := update.ParseManifest(ws)
	if err != nil || mf == nil || len(mf.Skills) != 16 {
		t.Fatalf("manifest BASE ilegível após FULL: %v (%+v)", err, mf)
	}
	// manifest de conteúdo separado, schema próprio, 125 arquivos
	var cm struct {
		Version string `json:"version"`
		Files   []struct {
			File  string `json:"file"`
			Layer string `json:"layer"`
		} `json:"files"`
	}
	raw := readFileT(t, filepath.Join(ws, ".golmet", "manifest-content.json"))
	if err := json.Unmarshal([]byte(raw), &cm); err != nil {
		t.Fatalf("manifest-content.json inválido: %v", err)
	}
	if len(cm.Files) != 125 {
		t.Errorf("manifest-content com %d arquivos, want 125", len(cm.Files))
	}
	if _, err := os.Stat(filepath.Join(ws, ".golmet", "agents", "gm-dev.md")); err != nil {
		t.Errorf("agents do FULL ausente da zona: %v", err)
	}

	// 3. BASE re-instala na zona (kit é o único escritor) — FULL sobrevive
	if _, err := update.Install(ws); err != nil {
		t.Fatalf("BASE re-install: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ws, ".golmet", "agents", "gm-dev.md")); err != nil {
		t.Errorf("BASE apagou os agents do FULL: %v", err)
	}
	if mf, err = update.ParseManifest(ws); err != nil || mf == nil || len(mf.Skills) != 16 {
		t.Fatalf("manifest BASE corrompido pelo ciclo BASE→FULL→BASE: %v", err)
	}

	// 4. FULL re-materializa — 3-way na zona: no máximo conflitos em skills/
	// (BASE reescrito pode divergir do hash de conteúdo); agents nunca.
	res2, _, err := MaterializeFullWithOptions(home, "cursor", ws, false, false)
	if err != nil {
		t.Fatalf("FULL re-run: %v", err)
	}
	if res2.Total != 125 {
		t.Fatalf("re-run total=%d", res2.Total)
	}
	for _, c := range res2.Conflicts {
		if !strings.HasPrefix(c, "skills/") {
			t.Errorf("conflito fora de skills/ no re-run da zona: %s", c)
		}
	}
}

// (c) dry_run do fallback calcula o plano sem escrever NADA na zona.
func TestFullFallbackDryRunWritesNothing(t *testing.T) {
	noEngineCLI(t)
	home := newHome(t)
	ws := t.TempDir()
	rep, place, err := MaterializeFullWithOptions(home, "cursor", ws, false, true)
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if place == nil || place.Zone != FullZoneWorkspace {
		t.Fatalf("zona do dry-run = %+v, want workspace-zone", place)
	}
	if len(rep.Written) != rep.Total || rep.Total == 0 {
		t.Fatalf("plano dry-run: total=%d written=%d", rep.Total, len(rep.Written))
	}
	if _, err := os.Stat(filepath.Join(ws, ".golmet")); !os.IsNotExist(err) {
		t.Errorf("dry-run criou a zona .golmet/: %v", err)
	}
}

// (d) erro permanece APENAS quando engine não tem skills E nenhum workspace
// foi informado (impossível determinar o destino).
func TestFullFallbackWithoutWorkspaceErrors(t *testing.T) {
	noEngineCLI(t)
	home := newHome(t)
	rep, place, err := MaterializeFullWithOptions(home, "cursor", "", false, false)
	if err == nil {
		t.Fatalf("sem workspace deveria errar: %+v", rep)
	}
	if !strings.Contains(err.Error(), "workspace") {
		t.Errorf("erro deveria citar workspace: %v", err)
	}
	if rep != nil || place != nil {
		t.Errorf("erro deveria vir sem relatório: %+v %+v", rep, place)
	}
	// engines com skills nativas não dependem do workspace
	_, place, err = MaterializeFullWithOptions(home, "claude-code", "", false, true)
	if err != nil || place == nil || place.Zone != FullZoneEngineConfig {
		t.Errorf("FULL nativo sem workspace deveria ir ao config do engine: %v %+v", err, place)
	}
}
