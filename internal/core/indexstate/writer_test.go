package indexstate

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSectionHashMatchesManualSha256(t *testing.T) {
	lines := []string{"> estado: accepted", "Decisão: Postgres como banco principal."}
	want := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	if got := SectionHash(lines); got != hex.EncodeToString(want[:]) {
		t.Errorf("SectionHash diverge da normalização SHA-256(join \\n): %s", got)
	}
	// conteúdo diferente → hash diferente
	other := []string{"> estado: accepted", "Decisão: Postgres como banco principal", ""}
	if SectionHash(lines) == SectionHash(other) {
		t.Errorf("conteúdos distintos não deveriam colidir")
	}
}

func TestSecaoHashOfVersusEstado(t *testing.T) {
	idx := &Index{Entities: map[string]map[string]EntityState{
		"adr": {"ADR-001": {Estado: "accepted", SecaoHash: "abc123"}},
	}}
	if got := idx.SecaoHashOf("adr", "ADR-001"); got != "abc123" {
		t.Errorf("SecaoHashOf = %q, want abc123", got)
	}
	if got := idx.SecaoHashOf("adr", "ADR-999"); got != "" {
		t.Errorf("SecaoHashOf inexistente = %q, want vazio", got)
	}
	var nilIdx *Index
	if nilIdx.SecaoHashOf("adr", "ADR-001") != "" {
		t.Errorf("SecaoHashOf em índice nil deveria ser vazio")
	}
}

func TestBuildShapeAndCounts(t *testing.T) {
	idx := Build("Loja", Snapshot{
		CharterFound:  true,
		CharterStatus: "confirmed",
		Sections: []SectionSnap{
			{NS: "rn", ID: "RN-001", Estado: "active", RawLines: []string{"> estado: active", "corpo"}},
			{NS: "rn", ID: "RN-002", Estado: "draft"},
			{NS: "pend", ID: "PEND-001", Estado: "open"},
			{NS: "pend", ID: "PEND-002", Estado: "deferred"},
			{NS: "cr", ID: "CR-001", Estado: "pending"},
			{NS: "adr", ID: "ADR-001", Estado: "accepted", RawLines: []string{"linha"}},
			{NS: "cand", ID: "CAND-001", Estado: "pending"},
		},
		Specs: []SpecSnap{{ID: "RN-001", Estado: "current"}},
		Features: []FeatureSnap{{
			Key: "RN-001", Status: "in_progress", Anchor: "RN-001",
			Tasks: map[string]string{"RN-001-1": "done", "RN-001-2": "todo"},
		}},
	})
	// todas as namespaces existem (shape estável)
	for _, ns := range []string{"rn", "pend", "cr", "adr", "cand", "feature", "spec", "charter"} {
		if _, ok := idx.Entities[ns]; !ok {
			t.Errorf("namespace %s ausente do índice", ns)
		}
	}
	if idx.Format != 1 || idx.Workspace != "Loja" || idx.GeneratedAt == "" {
		t.Errorf("cabeçalho do índice: %+v", idx)
	}
	// counts por estado + aliases §8
	want := map[string]int{
		"rn": 2, "rn_active": 1, "rn_draft": 1,
		"pend": 2, "pend_open": 1, "pend_deferred": 1,
		"cr": 1, "cr_pending": 1,
		"adr": 1, "adr_accepted": 1,
		"cand": 1, "cand_pending": 1,
		"feature": 1, "feature_in_progress": 1,
		"task": 2, "task_done": 1, "task_todo": 1,
		"spec": 1, "spec_current": 1,
		"features_done": 0,
	}
	for k, v := range want {
		if got := idx.Counts[k]; got != v {
			t.Errorf("counts[%s] = %d, want %d", k, got, v)
		}
	}
	// secao_hash gravada para entidades seccionais
	if idx.Entities["adr"]["ADR-001"].SecaoHash != SectionHash([]string{"linha"}) {
		t.Errorf("secao_hash do ADR diverge da normalização canônica")
	}
	if idx.Entities["rn"]["RN-002"].SecaoHash != SectionHash(nil) {
		t.Errorf("secao_hash de seção sem corpo deveria ser hash de nil")
	}
	// charter registrado
	if st := idx.Entities["charter"]["PROJECT_CHARTER.md"].Status; st != "confirmed" {
		t.Errorf("charter status = %q, want confirmed", st)
	}
}

func TestBuildBaselineEmpty(t *testing.T) {
	idx := Build("Vazio", Snapshot{})
	if idx.Counts["rn"] != 0 || idx.Counts["pend_open"] != 0 || idx.Counts["features_done"] != 0 {
		t.Errorf("counts de baseline deveriam ser zero: %+v", idx.Counts)
	}
	for ns, m := range idx.Entities {
		if len(m) != 0 {
			t.Errorf("namespace %s deveria estar vazia no baseline", ns)
		}
	}
}

func TestWriteIfChanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "golmet.index.json")

	idxA := Build("Ws", Snapshot{Sections: []SectionSnap{{NS: "rn", ID: "RN-001", Estado: "draft"}}})
	written, err := WriteIfChanged(path, idxA)
	if err != nil || !written {
		t.Fatalf("primeira escrita: written=%v err=%v", written, err)
	}

	// mesmo estado semântico + generated_at diferente → NÃO reescreve
	idxA2 := Build("Ws", Snapshot{Sections: []SectionSnap{{NS: "rn", ID: "RN-001", Estado: "draft"}}})
	data1, _ := os.ReadFile(path)
	written, err = WriteIfChanged(path, idxA2)
	if err != nil || written {
		t.Fatalf("estado idêntico não deveria reescrever: written=%v err=%v", written, err)
	}
	data2, _ := os.ReadFile(path)
	if string(data1) != string(data2) {
		t.Errorf("arquivo mudou sem mudança semântica")
	}

	// transição de estado → reescreve com o novo estado
	idxB := Build("Ws", Snapshot{Sections: []SectionSnap{{NS: "rn", ID: "RN-001", Estado: "active"}}})
	written, err = WriteIfChanged(path, idxB)
	if err != nil || !written {
		t.Fatalf("transição deveria reescrever: written=%v err=%v", written, err)
	}
	parsed, perr := Parse(mustRead(t, path))
	if perr != nil || parsed.Estado("rn", "RN-001") != "active" {
		t.Errorf("índice reescrito deveria registrar active: %+v err=%v", parsed, perr)
	}

	// índice ilegível no disco → kit recompõe (ele é o único escritor)
	os.WriteFile(path, []byte("{não json"), 0o644)
	written, err = WriteIfChanged(path, idxB)
	if err != nil || !written {
		t.Fatalf("índice ilegível deveria ser recomposto: written=%v err=%v", written, err)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
