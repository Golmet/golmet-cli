package status

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testdata = "../../testdata"

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

func TestStatusCountsAndLists(t *testing.T) {
	tmp := t.TempDir()
	ws := filepath.Join(tmp, "ws")
	copyDir(t, filepath.Join(testdata, "status-fixture"), ws)

	rep, err := Run(ws, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.Workspace != "ws" || !rep.Format.Suportado || rep.Format.Atual != 1 {
		t.Errorf("cabeçalho: %+v", rep)
	}
	c := rep.Counts
	if c["rn"] != 2 || c["rn_active"] != 1 || c["rn_draft"] != 1 {
		t.Errorf("counts RN = %+v", c)
	}
	if c["pend"] != 2 || c["pend_open"] != 1 || c["pend_deferred"] != 1 {
		t.Errorf("counts PEND = %+v", c)
	}
	if c["cr"] != 3 || c["cr_pending"] != 1 || c["cr_approved"] != 1 || c["cr_implemented"] != 1 {
		t.Errorf("counts CR = %+v", c)
	}
	if c["feature"] != 1 || c["feature_in_progress"] != 1 || c["features_done"] != 0 {
		t.Errorf("counts Feature = %+v", c)
	}
	if c["task"] != 2 || c["task_done"] != 1 || c["task_todo"] != 1 {
		t.Errorf("counts Task = %+v", c)
	}
	if c["spec"] != 1 || c["spec_stale"] != 1 {
		t.Errorf("counts Spec = %+v", c)
	}

	// PENDs abertas — só open (deferred não é aberta)
	if len(rep.PendsAbertas) != 1 || rep.PendsAbertas[0].ID != "PEND-001" ||
		rep.PendsAbertas[0].Titulo != "Política de frete" {
		t.Errorf("pends abertas = %+v", rep.PendsAbertas)
	}
	// CRs abertos — pending/approved, implemented fora
	if len(rep.CRsAbertos) != 2 {
		t.Fatalf("crs abertos = %+v", rep.CRsAbertos)
	}
	if rep.CRsAbertos[0].ID != "CR-001" || rep.CRsAbertos[0].Estado != "approved" ||
		rep.CRsAbertos[1].ID != "CR-002" || rep.CRsAbertos[1].Estado != "pending" {
		t.Errorf("crs abertos = %+v", rep.CRsAbertos)
	}
	// Feature in_progress com task pendente
	if len(rep.FeaturesEmAndamento) != 1 {
		t.Fatalf("features em andamento = %+v", rep.FeaturesEmAndamento)
	}
	fa := rep.FeaturesEmAndamento[0]
	if fa.Anchor != "RN-001" || len(fa.TasksPendentes) != 1 || fa.TasksPendentes[0] != "RN-001-2" {
		t.Errorf("feature em andamento = %+v", fa)
	}
	// specs stale + propagação devedora (S1 — CR implemented + stale)
	if len(rep.SpecsStale) != 1 || rep.SpecsStale[0].ID != "RN-003" || rep.SpecsStale[0].Reason != "CR-003" {
		t.Errorf("specs stale = %+v", rep.SpecsStale)
	}
	if !rep.PropagacaoDevedora {
		t.Errorf("propagação devedora deveria ser true (CR-003 implemented com spec stale)")
	}
	// saída human menciona os pontos-chave
	h := rep.Human()
	for _, want := range []string{
		"format: 1", "PEND-001 — Política de frete", "CR-003", "RN-001-2",
		"propagação devedora: SIM",
	} {
		if !strings.Contains(h, want) {
			t.Errorf("human output não contém %q:\n%s", want, h)
		}
	}
}

func TestStatusReadOnlyNeverWrites(t *testing.T) {
	tmp := t.TempDir()
	ws := filepath.Join(tmp, "ws")
	copyDir(t, filepath.Join(testdata, "status-fixture"), ws)

	// índice pré-existente com conteúdo sentinel — status não toca
	sentinel := "{\"format\":1,\"generated_at\":\"sentinel\",\"workspace\":\"ws\",\"entities\":{},\"counts\":{}}\n"
	if err := os.WriteFile(filepath.Join(ws, "golmet.index.json"), []byte(sentinel), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(ws, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(ws, "golmet.index.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != sentinel {
		t.Errorf("status alterou o golmet.index.json (NUNCA deve escrever):\n%s", string(data))
	}
}

func TestStatusInvalidWorkspace(t *testing.T) {
	// pasta inexistente
	if _, err := Run(filepath.Join(t.TempDir(), "nada"), Options{}); err == nil {
		t.Errorf("workspace inexistente deveria falhar")
	}
	// sem golmet.yaml
	empty := t.TempDir()
	if _, err := Run(empty, Options{}); err == nil {
		t.Errorf("workspace sem golmet.yaml deveria falhar")
	}
	// format não suportado
	ws := t.TempDir()
	os.WriteFile(filepath.Join(ws, "golmet.yaml"), []byte("format: 2\n"), 0o644)
	_, err := Run(ws, Options{})
	if err == nil || !strings.Contains(err.Error(), "não suportado") {
		t.Errorf("format 2 deveria falhar com 'não suportado', veio %v", err)
	}
}
