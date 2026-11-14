package eval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// minSelfScoreFixtures — piso de cobertura do corpus pontuável: abaixo
// disso o corpus regrediu (fixture removida ou skill sem fixture).
const minSelfScoreFixtures = 22

// skipMarker — presença do arquivo na fixture pula a pontuação self-score
// (conteúdo = motivo em 1 linha). Gap conhecido é DECLARADO, nunca silêncio.
const skipMarker = ".self-score-skip"

// TestCorpusSelfScore — o corpus evals/ como regressão viva de CI, SEM
// engine externo: cada fixture monta um workspace temporário (cópia de
// input/ + cópia de output-expected/ quando existe; só input/ quando não
// existe — fixtures negativas pontuam contra o estado semeado) e é
// pontuada pela mesma RunFixture usada pelo MCP gmEval. O gate exige
// status PASS (0 checks com severity error quebrados). Fixture com gap
// conhecido (baseline ainda não capturada) declara o marcador
// .self-score-skip com motivo — pulada com o motivo logado; marcador sem
// motivo falha (gap não declarado).
func TestCorpusSelfScore(t *testing.T) {
	root := fixturesRootRel(t)
	skills, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	type fixtureRef struct{ skill, name string }
	var fixtures []fixtureRef
	for _, s := range skills {
		if !s.IsDir() || !strings.HasPrefix(s.Name(), "gm-") {
			continue
		}
		entries, ferr := os.ReadDir(filepath.Join(root, s.Name()))
		if ferr != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() || !strings.HasPrefix(e.Name(), "fixture-") {
				continue
			}
			fixtures = append(fixtures, fixtureRef{skill: s.Name(), name: e.Name()})
		}
	}
	if len(fixtures) < minSelfScoreFixtures {
		t.Fatalf("corpus com %d fixtures — self-score exige ≥%d", len(fixtures), minSelfScoreFixtures)
	}

	var (
		passed    int
		skipped   []string // "<skill>/<fixture>" com motivo declarado
		unmotived []string // marcador presente sem motivo — gap não declarado
	)
	for _, fx := range fixtures {
		t.Run(fx.skill+"/"+fx.name, func(t *testing.T) {
			dir := filepath.Join(root, fx.skill, fx.name)
			if reason, ok := selfScoreSkipReason(dir); ok {
				label := fx.skill + "/" + fx.name
				if reason == "" {
					unmotived = append(unmotived, label)
					t.Logf("marcador %s SEM motivo — registrado para falha", skipMarker)
					return
				}
				skipped = append(skipped, label)
				t.Logf("pulado: %s", reason)
				return
			}
			ws := t.TempDir()
			if err := copyTree(filepath.Join(dir, "input"), ws); err != nil {
				t.Fatalf("copiando input/: %v", err)
			}
			// output-expected é opcional: espelha o estado final do
			// workspace (sobrescreve o input em conflito); fixture sem
			// baseline pontua contra o input puro.
			if err := copyTree(filepath.Join(dir, "output-expected"), ws); err != nil && !os.IsNotExist(err) {
				t.Fatalf("copiando output-expected/: %v", err)
			}
			res, err := RunFixture(dir, ws)
			if err != nil {
				t.Fatalf("RunFixture: %v", err)
			}
			if res.Status != StatusPass {
				for _, c := range res.Checks {
					if c.Status == StatusFail {
						t.Errorf("check %s [%s] falhou — %s (fix: %s)", c.Check, c.Severity, c.Detail, c.Fix)
					}
				}
				if res.Status == StatusSkipped {
					t.Errorf("status %s — %s", res.Status, res.SkippedRsn)
				}
				return
			}
			passed++
		})
	}
	t.Logf("self-score: %d/%d fixtures PASS, %d pulada(s) com motivo", passed, len(fixtures), len(skipped))
	for _, s := range skipped {
		t.Logf("  skip: %s", s)
	}
	if len(unmotived) > 0 {
		t.Errorf("marcador %s sem motivo (preencha o gap em 1 linha) em: %s",
			skipMarker, strings.Join(unmotived, ", "))
	}
}

// copyTree copia recursivamente src → dst preservando a estrutura
// relativa (pastas via MkdirAll, arquivos byte a byte). Raiz ausente
// retorna erro — input/ é obrigatório, output-expected/ é filtrado
// pela chamada com os.IsNotExist.
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, rerr := filepath.Rel(src, p)
		if rerr != nil {
			return rerr
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		return os.WriteFile(target, data, 0o644)
	})
}

// selfScoreSkipReason lê o marcador de skip da fixture: (motivo, presente).
// Motivo vazio é devolvido como presente-e-vazio — o teste decide a falha.
func selfScoreSkipReason(fixtureDir string) (string, bool) {
	data, err := os.ReadFile(filepath.Join(fixtureDir, skipMarker))
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(data)), true
}
