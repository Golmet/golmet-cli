package eval

import (
	"strings"
	"testing"
)

// TestCheckDetailsReflectOutcome — o detail do relatório deve refletir o
// resultado real: detail de sucesso quando o check passa, detail de falha
// quando falha. Regressão do bug em que o detail de falha hardcoded
// aparecia também em checks PASS (relatório contradizendo o status).
func TestCheckDetailsReflectOutcome(t *testing.T) {
	ws := t.TempDir()
	writeFixture(t, ws, "doc.md", "---\ntitulo: \"X: Y\"\n---\n\n## Seção Alvo\n\nalpha beta\n")

	// contains passando → detail de sucesso
	r := runOne(t, ws, Check{Type: "contains", Path: "doc.md", Pattern: "alpha beta"})
	if r.Status != StatusPass || !strings.Contains(r.Detail, "encontrado") || strings.Contains(r.Detail, "não encontrado") {
		t.Errorf("contains pass = %+v", r)
	}
	// contains falhando → detail de falha (texto intacto)
	r = runOne(t, ws, Check{Type: "contains", Path: "doc.md", Pattern: "INEXISTENTE"})
	if r.Status != StatusFail || !strings.Contains(r.Detail, "não encontrado") {
		t.Errorf("contains fail = %+v", r)
	}
	// not_contains passando (padrão ausente) → detail de sucesso
	r = runOne(t, ws, Check{Type: "not_contains", Path: "doc.md", Pattern: "INEXISTENTE"})
	if r.Status != StatusPass || !strings.Contains(r.Detail, "não encontrado") {
		t.Errorf("not_contains pass = %+v", r)
	}
	// not_contains falhando (padrão presente) → detail de falha
	r = runOne(t, ws, Check{Type: "not_contains", Path: "doc.md", Pattern: "alpha beta"})
	if r.Status != StatusFail || !strings.Contains(r.Detail, "proibido") || !strings.Contains(r.Detail, "encontrado em") {
		t.Errorf("not_contains fail = %+v", r)
	}
	// file_exists passando → detail de sucesso
	r = runOne(t, ws, Check{Type: "file_exists", Path: "doc.md"})
	if r.Status != StatusPass || !strings.Contains(r.Detail, "presente") {
		t.Errorf("file_exists pass = %+v", r)
	}
	// section_exists passando → detail de sucesso
	r = runOne(t, ws, Check{Type: "section_exists", Path: "doc.md", Section: "## Seção Alvo"})
	if r.Status != StatusPass || !strings.Contains(r.Detail, "presente") {
		t.Errorf("section_exists pass = %+v", r)
	}
	// section_order passando → detail de sucesso
	r = runOne(t, ws, Check{Type: "section_order", Path: "doc.md", Sections: []string{"## Seção Alvo"}})
	if r.Status != StatusPass || !strings.Contains(r.Detail, "ordem esperada") {
		t.Errorf("section_order pass = %+v", r)
	}
	// no_todos passando → detail de sucesso
	r = runOne(t, ws, Check{Type: "no_todos", Path: "doc.md", Patterns: []string{"TODO", "FIXME"}})
	if r.Status != StatusPass || !strings.Contains(r.Detail, "nenhum marcador") {
		t.Errorf("no_todos pass = %+v", r)
	}
}
