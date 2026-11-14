package adopt

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Golmet/golmet-cli/internal/core/eval"
)

// fixturesRel devolve o corpus versionado (evals/ na raiz do repo).
func fixturesRel(t *testing.T) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "evals")
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		t.Skipf("corpus ausente: %s", root)
	}
	return root
}

// seedWorkspace copia input/ da fixture para um workspace novo.
func seedWorkspace(t *testing.T, fixtureDir string) string {
	t.Helper()
	ws := t.TempDir()
	err := filepath.WalkDir(filepath.Join(fixtureDir, "input"), func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(filepath.Join(fixtureDir, "input"), p)
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		target := filepath.Join(ws, rel)
		os.MkdirAll(filepath.Dir(target), 0o755)
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return ws
}

// A fixture gm-adopt/fixture-001 exercita o próprio kit: o "engine" é o
// comando adopt. Sequência do handshake (skill gm-adopt):
//  1. seed do workspace legado
//  2. `adopt --dry-run`  → leitura pura (zero escrita)
//  3. `adopt --apply`    → transform determinístico
//  4. `golmet eval`      → criteria da fixture pontua o estado final
func TestGmAdoptFixtureEndToEnd(t *testing.T) {
	fixtureDir := filepath.Join(fixturesRel(t), "gm-adopt", "fixture-001")
	ws := seedWorkspace(t, fixtureDir)
	before := map[string]string{}
	filepath.WalkDir(ws, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, _ := os.ReadFile(p)
		before[strings.ReplaceAll(p, ws, "")] = string(data)
		return nil
	})

	// passo 1 do handshake — dry-run NÃO escreve nada
	if _, err := Run(ws, Options{Apply: false}); err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	filepath.WalkDir(ws, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, _ := os.ReadFile(p)
		if before[strings.ReplaceAll(p, ws, "")] != string(data) {
			t.Errorf("dry-run alterou %s", p)
		}
		return nil
	})
	if _, err := os.Stat(filepath.Join(ws, "golmet.yaml")); !os.IsNotExist(err) {
		t.Errorf("dry-run criou golmet.yaml")
	}

	// passos 3+4 — gates "confirmados" e apply
	if _, err := Run(ws, Options{Apply: true, CharterStatus: "draft", KeepLegacyLayouts: true}); err != nil {
		t.Fatalf("apply: %v", err)
	}

	// o kit pontua com os criteria da fixture
	res, err := eval.RunFixture(fixtureDir, ws)
	if err != nil {
		t.Fatalf("eval.RunFixture: %v", err)
	}
	if res.Status != eval.StatusPass {
		for _, c := range res.Checks {
			if c.Status != eval.StatusPass {
				t.Errorf("check %s = %s (%s) — fix: %s", c.Check, c.Status, c.Detail, c.Fix)
			}
		}
		t.Fatalf("fixture gm-adopt deveria passar no próprio golmet eval")
	}
	if res.Failed != 0 || res.Passed < 10 {
		t.Errorf("gm-adopt: %d PASS, %d FAIL (esperado ≥10 PASS, 0 FAIL)", res.Passed, res.Failed)
	}
}
