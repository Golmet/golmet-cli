package next

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Golmet/golmet-cli/internal/core/scaffold"
	"github.com/Golmet/golmet-cli/internal/core/update"
)

func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	err := filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
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
	if err != nil {
		t.Fatal(err)
	}
}

const fixture = "../../testdata/status-fixture"

func priorities(rep *Report) map[int]int {
	m := map[int]int{}
	for _, r := range rep.Recommendations {
		m[r.Priority]++
	}
	return m
}

func actionsFor(rep *Report, p int) []string {
	var out []string
	for _, r := range rep.Recommendations {
		if r.Priority == p {
			out = append(out, r.Action)
		}
	}
	return out
}

// Scaffold em dia exceto skills: charter presente, validate limpo,
// índice baseline criado → única pendência é o subset BASE.
func TestNextScaffoldedWorkspace(t *testing.T) {
	ws := filepath.Join(t.TempDir(), "ws")
	if _, err := scaffold.Run(filepath.Dir(ws), filepath.Base(ws), scaffold.Options{}); err != nil {
		t.Fatalf("scaffold: %v", err)
	}
	rep, err := Run(ws)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	m := priorities(rep)
	if m[PUpdate] != 1 {
		t.Errorf("esperava 1 recomendação de update, veio %v", m)
	}
	if len(rep.Recommendations) != 1 {
		t.Fatalf("recomendações inesperadas: %+v", rep.Recommendations)
	}
	// pós-update → tudo em dia
	if _, err := update.Install(ws); err != nil {
		t.Fatalf("update: %v", err)
	}
	rep2, err := Run(ws)
	if err != nil {
		t.Fatalf("Run 2: %v", err)
	}
	if len(rep2.Recommendations) != 1 || rep2.Recommendations[0].Priority != PAllDone {
		t.Errorf("pós-update esperava 'nada pendente': %+v", rep2.Recommendations)
	}
}

// Fixture rico: erro hard + índice ausente + PEND + CRs + spec stale +
// feature in_progress com task pendente.
func TestNextStatusFixture(t *testing.T) {
	ws := filepath.Join(t.TempDir(), "ws")
	copyDir(t, fixture, ws)

	rep, err := Run(ws)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	m := priorities(rep)

	// RN-001 active sem evidência = H4 (erro hard)
	if m[PValidationError] != 1 {
		t.Errorf("esperava 1 erro hard (H4), veio %v", m)
	}
	// skills não instaladas
	if m[PUpdate] != 1 {
		t.Errorf("esperava update, veio %v", m)
	}
	// índice ausente
	if m[PIndexStale] != 1 {
		t.Errorf("esperava índice ausente, veio %v", m)
	}
	// PEND-001 open (PEND-002 é deferred)
	if acts := actionsFor(rep, PPendOpen); len(acts) != 1 || !reflect.DeepEqual(acts, []string{"decida/promova PEND-001 com gm-clarify + gm-requirements"}) {
		t.Errorf("PENDs = %+v", rep.Recommendations)
	}
	// CR-001 approved + CR-002 pending (CR-003 implemented fora)
	if m[PCRNotDone] != 2 {
		t.Errorf("esperava 2 CRs abertos, veio %v", m)
	}
	// spec stale RN-003
	if m[PSpecStale] != 1 {
		t.Errorf("esperava 1 spec stale, veio %v", m)
	}
	// feature RN-001 in_progress com RN-001-2 pendente
	if m[PFeatureTask] != 1 {
		t.Errorf("esperava 1 feature em andamento, veio %v", m)
	}
	for _, r := range rep.Recommendations {
		if r.Priority == PFeatureTask && !contains(r.Action, "RN-001-2") {
			t.Errorf("task recomendada deveria ser RN-001-2: %s", r.Action)
		}
	}
	// ordenação: prioridades não-decrescentes
	for i := 1; i < len(rep.Recommendations); i++ {
		if rep.Recommendations[i].Priority < rep.Recommendations[i-1].Priority {
			t.Errorf("recomendações fora de ordem: %+v", rep.Recommendations)
		}
	}
	// determinismo
	rep2, _ := Run(ws)
	if !reflect.DeepEqual(rep, rep2) {
		t.Errorf("Run não é determinístico")
	}
}

// Feature ready sem start → recomendação de início (§7.5).
func TestNextFeatureReady(t *testing.T) {
	ws := filepath.Join(t.TempDir(), "ws")
	copyDir(t, filepath.Join(fixture, "..", "valid-workspace"), ws)
	// valid-workspace já tem feature ready (RN-001 anchor ativa)
	rep, err := Run(ws)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	found := false
	for _, r := range rep.Recommendations {
		if r.Priority == PFeatureReady {
			found = true
		}
	}
	if !found {
		t.Errorf("esperava recomendação de feature ready: %+v", rep.Recommendations)
	}
}

func TestNextInvalidWorkspace(t *testing.T) {
	if _, err := Run(filepath.Join(t.TempDir(), "nada")); err == nil {
		t.Errorf("workspace inexistente deveria falhar")
	}
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }
