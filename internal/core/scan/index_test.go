package scan

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Golmet/golmet-cli/internal/core/indexstate"
)

// writeIndexFixture grava um golmet.index.json mínimo com o estado dado.
func writeIndexFixture(t *testing.T, wsPath, ns, id, estado string) {
	t.Helper()
	idx := indexstate.Build(filepath.Base(wsPath), indexstate.Snapshot{
		Sections: []indexstate.SectionSnap{{NS: ns, ID: id, Estado: estado}},
	})
	data, err := idx.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wsPath, "golmet.index.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func readIndexFile(t *testing.T, ws string) *indexstate.Index {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(ws, "golmet.index.json"))
	if err != nil {
		t.Fatalf("golmet.index.json ausente: %v", err)
	}
	idx, perr := indexstate.Parse(data)
	if perr != nil {
		t.Fatalf("índice ilegível: %v", perr)
	}
	return idx
}

// Primeira execução sem índice = cria baseline (H6/H8 não disparam —
// não há história anterior).
func TestIndexBaselineFirstRun(t *testing.T) {
	tmp := t.TempDir()
	ws := filepath.Join(tmp, "ws")
	copyDir(t, filepath.Join(testdata, "valid-workspace"), ws)

	res, err := Run(ws, Options{WriteIndex: true, Tolerant: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !res.IndexWritten {
		t.Errorf("primeira execução deveria criar o baseline")
	}
	idx := readIndexFile(t, ws)
	if got := idx.Estado("rn", "RN-001"); got != "active" {
		t.Errorf("baseline RN-001 = %q, want active", got)
	}
	if idx.Counts["rn"] != 1 || idx.Counts["pend_open"] != 1 {
		t.Errorf("counts do baseline = %+v", idx.Counts)
	}
	if len(idx.Entities["feature"]) != 1 {
		t.Errorf("baseline deveria conter 1 feature: %+v", idx.Entities["feature"])
	}
}

// Transição legal → validate sem erros hard REESCREVE o índice.
func TestIndexUpdatesOnLegalTransition(t *testing.T) {
	tmp := t.TempDir()
	ws := filepath.Join(tmp, "ws")
	copyDir(t, filepath.Join(testdata, "valid-workspace"), ws)
	writeIndexFixture(t, ws, "rn", "RN-001", "draft") // draft→active é legal

	res, err := Run(ws, Options{WriteIndex: true, Tolerant: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Errors) != 0 {
		t.Fatalf("transição legal não deveria gerar erro: %+v", res.Errors)
	}
	if !res.IndexWritten {
		t.Errorf("índice deveria ser reescrito após transição legal")
	}
	if got := readIndexFile(t, ws).Estado("rn", "RN-001"); got != "active" {
		t.Errorf("índice deveria registrar active, veio %q", got)
	}
}

// Erro hard → índice NÃO é escrito (história preservada para H6/H8).
func TestIndexPreservedOnHardError(t *testing.T) {
	tmp := t.TempDir()
	ws := filepath.Join(tmp, "ws")
	copyDir(t, filepath.Join(testdata, "valid-workspace"), ws)
	writeIndexFixture(t, ws, "rn", "RN-001", "dropped") // dropped é final; active é ilegal

	res, err := Run(ws, Options{WriteIndex: true, Tolerant: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !rulesIn(res, severityError)["H6"] {
		t.Fatalf("esperava H6 (transição ilegal); erros=%+v", summary(res.Errors))
	}
	if res.IndexWritten {
		t.Errorf("com erro hard o índice NÃO deveria ser escrito")
	}
	if got := readIndexFile(t, ws).Estado("rn", "RN-001"); got != "dropped" {
		t.Errorf("história deveria permanecer dropped, veio %q", got)
	}
}

// --fix força a recomposição ignorando a divergência (pós-merge).
func TestIndexFixForcesRewrite(t *testing.T) {
	tmp := t.TempDir()
	ws := filepath.Join(tmp, "ws")
	copyDir(t, filepath.Join(testdata, "valid-workspace"), ws)
	writeIndexFixture(t, ws, "rn", "RN-001", "dropped")

	res, err := Run(ws, Options{WriteIndex: true, ForceIndex: true, Tolerant: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !rulesIn(res, severityError)["H6"] {
		t.Fatalf("H6 permanece no diagnóstico mesmo com --fix; erros=%+v", summary(res.Errors))
	}
	if !res.IndexWritten {
		t.Errorf("--fix deveria forçar a reescrita")
	}
	if got := readIndexFile(t, ws).Estado("rn", "RN-001"); got != "active" {
		t.Errorf("--fix deveria recompor para active, veio %q", got)
	}
}

// Sem WriteIndex (status/doctor/default) — NUNCA cria nem altera o índice.
func TestIndexNotWrittenWithoutOptIn(t *testing.T) {
	tmp := t.TempDir()
	ws := filepath.Join(tmp, "ws")
	copyDir(t, filepath.Join(testdata, "valid-workspace"), ws)

	if _, err := Run(ws, Options{Tolerant: true}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ws, "golmet.index.json")); !os.IsNotExist(err) {
		t.Errorf("Run sem WriteIndex não deveria criar golmet.index.json")
	}

	// com índice pré-existente, conteúdo preservado byte a byte
	writeIndexFixture(t, ws, "rn", "RN-001", "draft")
	before := mustReadRaw(t, filepath.Join(ws, "golmet.index.json"))
	if _, err := Run(ws, Options{Tolerant: true}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	after := mustReadRaw(t, filepath.Join(ws, "golmet.index.json"))
	if string(before) != string(after) {
		t.Errorf("Run sem WriteIndex alterou o índice existente")
	}
}

// secao_hash do ADR usa a MESMA normalização do H8 (linhas originais).
func TestIndexSecaoHashMatchesH8(t *testing.T) {
	tmp := t.TempDir()
	ws := filepath.Join(tmp, "ws")
	copyDir(t, filepath.Join(testdata, "violations", "h8"), ws) // ADR-001 accepted, hash divergente

	// validate SEM escrita: H8 permanece (índice antigo intacto)
	res, err := Run(ws, Options{WriteIndex: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !rulesIn(res, severityError)["H8"] {
		t.Fatalf("esperava H8 com secao_hash divergente; erros=%+v", summary(res.Errors))
	}
	if res.IndexWritten {
		t.Errorf("H8 é erro hard — índice não deveria ser reescrito")
	}

	// recomposição (--fix) grava secao_hash correta → H8 silencia
	if _, err := Run(ws, Options{WriteIndex: true, ForceIndex: true}); err != nil {
		t.Fatalf("Run fix: %v", err)
	}
	res2, err := Run(ws, Options{WriteIndex: true})
	if err != nil {
		t.Fatalf("Run pós-fix: %v", err)
	}
	if rulesIn(res2, severityError)["H8"] {
		t.Errorf("após recomposição H8 deveria silenciar; erros=%+v", summary(res2.Errors))
	}
	idx := readIndexFile(t, ws)
	stored := idx.SecaoHashOf("adr", "ADR-001")
	if stored == "" || stored == "nomatch" {
		t.Errorf("secao_hash recomposta inválida: %q", stored)
	}
}

// Feature SEM tasks: o round-trip JSON (omitempty) lê Tasks nil; o
// rebuild produz mapa vazio — sem a normalização, cada validate reescreveria
// o índice (churn eterno). Regressão do bug descoberto no bloco 4.
func TestIndexTasklessFeatureNoChurn(t *testing.T) {
	tmp := t.TempDir()
	ws := filepath.Join(tmp, "ws")
	copyDir(t, filepath.Join(testdata, "valid-workspace"), ws)
	// feature SEM seções de task
	feat := filepath.Join(ws, "delivery", "2026-11", "RN-001 — Agendamento — Feature.md")
	os.WriteFile(feat, []byte("---\nstatus: ready\nanchor: RN-001\n---\n# Feature sem tasks\n"), 0o644)

	res1, err := Run(ws, Options{WriteIndex: true, Tolerant: true})
	if err != nil {
		t.Fatalf("Run 1: %v", err)
	}
	idxAfter1 := mustReadRaw(t, filepath.Join(ws, "golmet.index.json"))
	_ = res1

	res2, err := Run(ws, Options{WriteIndex: true, Tolerant: true})
	if err != nil {
		t.Fatalf("Run 2: %v", err)
	}
	if res2.IndexWritten {
		t.Errorf("segundo validate reescreveu o índice de feature sem tasks (churn de round-trip nil-vs-vazio)")
	}
	idxAfter2 := mustReadRaw(t, filepath.Join(ws, "golmet.index.json"))
	if string(idxAfter1) != string(idxAfter2) {
		t.Errorf("índice mudou sem mudança semântica")
	}
}

func mustReadRaw(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
