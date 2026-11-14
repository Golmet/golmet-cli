package eval

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fixturesRootRel devolve a raiz do corpus versionado no repo.
func fixturesRootRel(t *testing.T) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "evals")
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		t.Skipf("corpus ausente: %s", root)
	}
	return root
}

// Smoke de carga do corpus: TODA fixture versionada carrega o criteria
// sem erro de parse (os checks resolvem sem crash contra um workspace
// vazio — status aqui não importa, só a robustez).
func TestCorpusLoads(t *testing.T) {
	root := fixturesRootRel(t)
	skills, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, s := range skills {
		if !s.IsDir() || !strings.HasPrefix(s.Name(), "gm-") {
			continue
		}
		fixtures, ferr := os.ReadDir(filepath.Join(root, s.Name()))
		if ferr != nil {
			continue
		}
		for _, fx := range fixtures {
			if !fx.IsDir() || !strings.HasPrefix(fx.Name(), "fixture-") {
				continue
			}
			total++
			data, rerr := os.ReadFile(filepath.Join(root, s.Name(), fx.Name(), "criteria.yaml"))
			if rerr != nil {
				t.Errorf("%s/%s sem criteria.yaml: %v", s.Name(), fx.Name(), rerr)
				continue
			}
			if _, perr := ParseCriteria(data); perr != nil {
				t.Errorf("%s/%s criteria inválido: %v", s.Name(), fx.Name(), perr)
			}
		}
	}
	if total < 22 {
		t.Errorf("corpus com %d fixtures — esperado ≥22 (1 por skill gm)", total)
	}
	// as skills gm do corpus têm ≥1 fixture (piso de cobertura, estilo
	// minSelfScoreFixtures — o corpus cresce a cada pack/skill nova)
	skillCount := 0
	for _, s := range skills {
		if !s.IsDir() || !strings.HasPrefix(s.Name(), "gm-") {
			continue
		}
		fixtures, _ := os.ReadDir(filepath.Join(root, s.Name()))
		n := 0
		for _, fx := range fixtures {
			if fx.IsDir() && strings.HasPrefix(fx.Name(), "fixture-") {
				n++
			}
		}
		if n == 0 {
			t.Errorf("skill %s sem fixture", s.Name())
		}
		skillCount++
	}
	if skillCount < 22 {
		t.Errorf("corpus com %d skills gm — esperado ≥22 (1 fixture por skill)", skillCount)
	}
}
