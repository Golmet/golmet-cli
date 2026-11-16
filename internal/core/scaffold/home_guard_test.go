package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Guarda do caso real (M22 — FORMAT §2): o init NUNCA cria na RAIZ do
// home do operador — o ~/.golmet/ é a instalação global do kit (tools,
// packs) e um workspace ali mistura os dois mundos (o codegraph de 333 MB
// quase foi apagado junto). Subpastas do home são legítimas.
func TestScaffoldNeverCreatesInHome(t *testing.T) {
	oldHome := userHomeDir
	t.Cleanup(func() { userHomeDir = oldHome })
	home := t.TempDir()
	userHomeDir = func() (string, error) { return home, nil }

	// workspace NA RAIZ do home = bloqueado (com e sem --org)
	_, err := Run(home, "", Options{})
	if err == nil || !strings.Contains(err.Error(), "HOME do operador") {
		t.Fatalf("init na raiz do home deveria bloquear: %v", err)
	}
	if _, serr := os.Stat(filepath.Join(home, "golmet.yaml")); serr == nil {
		t.Errorf("home não deveria ganhar golmet.yaml")
	}
	if _, err := Run(home, "", Options{Org: true}); err == nil {
		t.Errorf("container na raiz do home idem bloqueado")
	}
	if _, serr := os.Stat(filepath.Join(home, "CONSTITUTION.md")); serr == nil {
		t.Errorf("home não deveria ganhar CONSTITUTION.md")
	}

	// SUBPASTA do home é legítima
	if _, err := Run(home, "Trabalho", Options{}); err != nil {
		t.Fatalf("subpasta do home deveria criar: %v", err)
	}
	if _, serr := os.Stat(filepath.Join(home, "Trabalho", "golmet.yaml")); serr != nil {
		t.Errorf("subpasta não criada: %v", serr)
	}
}
