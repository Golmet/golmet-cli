package eval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---- helpers ----

func writeFixture(t *testing.T, ws, rel, content string) {
	t.Helper()
	p := filepath.Join(ws, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func boolPtr(b bool) *bool { return &b }

func runOne(t *testing.T, ws string, chk Check) CheckResult {
	t.Helper()
	return runOneAt(t, ws, ws, chk)
}

func runOneAt(t *testing.T, ws, fixtureDir string, chk Check) CheckResult {
	t.Helper()
	criteria := &Criteria{Checks: []Check{chk}}
	if chk.Severity == "" {
		criteria.Checks[0].Severity = SeverityError
	}
	status, detail, fix := runCheck(ws, fixtureDir, &criteria.Checks[0])
	return CheckResult{Check: chk.Type + ":" + chk.Path, Type: chk.Type,
		Path: chk.Path, Severity: criteria.Checks[0].Severity,
		Status: status, Detail: detail, Fix: fix}
}

const dmDoc = `---
owner: spec
---
# DATA_MODEL

## 1. Diagrama Entidade-Relacionamento

` + "```mermaid" + `
erDiagram
  PRODUTOR ||--o{ PROPRIEDADE : tem
  NOTE "algo {com chaves} aqui" {
    text: linha
  }
` + "```" + `

## 2. Tabelas

| Tabela | Campos | FK |
|---|---|---|
| PRODUTOR | id, nome | |
| PROPRIEDADE | id, produtor_id | PRODUTOR.id |

## 3. Decisão de armazenamento

Postgres. Referencia RN-001 e RN-002.
`

const reqDoc = `# Requisitos

## RN-001 — Produtores
## RN-002 — Propriedades
`

// ---- testes por tipo ----

func TestFileExistsAndNotFileExists(t *testing.T) {
	ws := t.TempDir()
	writeFixture(t, ws, "specs/DATA_MODEL.md", dmDoc)

	if r := runOne(t, ws, Check{Type: "file_exists", Path: "specs/DATA_MODEL.md"}); r.Status != StatusPass {
		t.Errorf("file_exists pass = %s (%s)", r.Status, r.Detail)
	}
	if r := runOne(t, ws, Check{Type: "file_exists", Path: "specs/AUSENTE.md"}); r.Status != StatusFail || r.Fix == "" {
		t.Errorf("file_exists ausente = %s (%s)", r.Status, r.Detail)
	}
	if r := runOne(t, ws, Check{Type: "file_exists", Path: "specs/AUSENTE.md", Severity: SeverityWarning}); r.Status != StatusWarn {
		t.Errorf("file_exists warning = %s", r.Status)
	}
	if r := runOne(t, ws, Check{Type: "not_file_exists", Path: "specs/AUSENTE.md"}); r.Status != StatusPass {
		t.Errorf("not_file_exists ausente = %s", r.Status)
	}
	if r := runOne(t, ws, Check{Type: "not_file_exists", Path: "specs/DATA_MODEL.md"}); r.Status != StatusFail {
		t.Errorf("not_file_exists existente = %s (%s)", r.Status, r.Detail)
	}
}

func TestContainsNotContainsNoTodos(t *testing.T) {
	ws := t.TempDir()
	writeFixture(t, ws, "specs/DATA_MODEL.md", dmDoc)

	if r := runOne(t, ws, Check{Type: "contains", Path: "specs/DATA_MODEL.md", Pattern: "PRODUTOR"}); r.Status != StatusPass {
		t.Errorf("contains = %s", r.Status)
	}
	if r := runOne(t, ws, Check{Type: "contains", Path: "specs/DATA_MODEL.md", Pattern: "INEXISTENTE"}); r.Status != StatusFail {
		t.Errorf("contains miss = %s", r.Status)
	}
	// arquivo ausente: contains falha, not_contains passa (contrato Read-Doc)
	if r := runOne(t, ws, Check{Type: "contains", Path: "specs/NOPE.md", Pattern: "x"}); r.Status != StatusFail {
		t.Errorf("contains em ausente = %s", r.Status)
	}
	if r := runOne(t, ws, Check{Type: "not_contains", Path: "specs/NOPE.md", Pattern: "x"}); r.Status != StatusPass {
		t.Errorf("not_contains em ausente = %s", r.Status)
	}
	if r := runOne(t, ws, Check{Type: "not_contains", Path: "specs/DATA_MODEL.md", Pattern: "INEXISTENTE"}); r.Status != StatusPass {
		t.Errorf("not_contains ok = %s", r.Status)
	}
	if r := runOne(t, ws, Check{Type: "no_todos", Path: "specs/DATA_MODEL.md", Patterns: []string{"TODO", "FIXME"}}); r.Status != StatusPass {
		t.Errorf("no_todos limpo = %s", r.Status)
	}
	writeFixture(t, ws, "specs/TODO.md", "todo: FIXME pendente\n")
	if r := runOne(t, ws, Check{Type: "no_todos", Path: "specs/TODO.md", Patterns: []string{"TODO", "FIXME"}}); r.Status != StatusFail ||
		!strings.Contains(r.Detail, "FIXME") {
		t.Errorf("no_todos com marcador = %s (%s)", r.Status, r.Detail)
	}
	// contains em DIRETÓRIO varre *.md recursivo
	if r := runOne(t, ws, Check{Type: "contains", Path: "specs/", Pattern: "PROPRIEDADE"}); r.Status != StatusPass {
		t.Errorf("contains em dir = %s", r.Status)
	}
}

func TestSectionExistsAndOrder(t *testing.T) {
	ws := t.TempDir()
	writeFixture(t, ws, "specs/DATA_MODEL.md", dmDoc)

	for _, s := range []string{"## 1. Diagrama Entidade-Relacionamento", "## 2. Tabelas", "## 3. Decisão de armazenamento"} {
		if r := runOne(t, ws, Check{Type: "section_exists", Path: "specs/DATA_MODEL.md", Section: s}); r.Status != StatusPass {
			t.Errorf("section_exists %q = %s", s, r.Status)
		}
	}
	if r := runOne(t, ws, Check{Type: "section_exists", Path: "specs/DATA_MODEL.md", Section: "## 9. Nada"}); r.Status != StatusFail {
		t.Errorf("section_exists ausente = %s", r.Status)
	}
	if r := runOne(t, ws, Check{Type: "section_order", Path: "specs/DATA_MODEL.md",
		Sections: []string{"## 1. Diagrama Entidade-Relacionamento", "## 2. Tabelas", "## 3. Decisão de armazenamento"}}); r.Status != StatusPass {
		t.Errorf("section_order ok = %s (%s)", r.Status, r.Detail)
	}
	// ordem invertida → fail
	reversed := []string{"## 3. Decisão de armazenamento", "## 1. Diagrama Entidade-Relacionamento"}
	if r := runOne(t, ws, Check{Type: "section_order", Path: "specs/DATA_MODEL.md", Sections: reversed}); r.Status != StatusFail {
		t.Errorf("section_order invertida = %s", r.Status)
	}
	// seção faltante na ordem → fail com detail citando
	if r := runOne(t, ws, Check{Type: "section_order", Path: "specs/DATA_MODEL.md",
		Sections: []string{"## 2. Tabelas", "## 4. Falta"}}); r.Status != StatusFail || !strings.Contains(r.Detail, "ausentes") {
		t.Errorf("section_order faltante = %s (%s)", r.Status, r.Detail)
	}
}

func TestLineCountRange(t *testing.T) {
	ws := t.TempDir()
	writeFixture(t, ws, "specs/DATA_MODEL.md", dmDoc) // ~25 linhas
	min, max := 10, 40
	if r := runOne(t, ws, Check{Type: "line_count_range", Path: "specs/DATA_MODEL.md", Min: &min, Max: &max}); r.Status != StatusPass {
		t.Errorf("line_count na faixa = %s (%s)", r.Status, r.Detail)
	}
	min, max = 100, 200
	if r := runOne(t, ws, Check{Type: "line_count_range", Path: "specs/DATA_MODEL.md", Min: &min, Max: &max, Severity: SeverityWarning}); r.Status != StatusWarn {
		t.Errorf("line_count abaixo (warning) = %s", r.Status)
	}
	// diretório soma os *.md recursivos
	writeFixture(t, ws, "specs/OUTRO.md", strings.Repeat("linha\n", 50))
	min, max = 60, 120
	if r := runOne(t, ws, Check{Type: "line_count_range", Path: "specs/", Min: &min, Max: &max}); r.Status != StatusPass {
		t.Errorf("line_count em dir = %s (%s)", r.Status, r.Detail)
	}
}

func TestIDSequenceAndCrossRef(t *testing.T) {
	ws := t.TempDir()
	writeFixture(t, ws, "REQUIREMENTS.md", reqDoc)
	writeFixture(t, ws, "specs/DATA_MODEL.md", dmDoc) // RN-001, RN-002 em ordem

	if r := runOne(t, ws, Check{Type: "id_sequence", Path: "specs/DATA_MODEL.md", Pattern: `RN-(\d+)`, AllowGaps: boolPtr(true)}); r.Status != StatusPass {
		t.Errorf("id_sequence allow_gaps = %s (%s)", r.Status, r.Detail)
	}
	if r := runOne(t, ws, Check{Type: "id_sequence", Path: "specs/DATA_MODEL.md", Pattern: `RN-(\d+)`}); r.Status != StatusPass {
		t.Errorf("id_sequence sem gaps (1,2 consecutivos) = %s (%s)", r.Status, r.Detail)
	}
	// fora de ordem → fail
	writeFixture(t, ws, "specs/DESORDENADO.md", "cite RN-003 antes RN-001 depois RN-002\n")
	if r := runOne(t, ws, Check{Type: "id_sequence", Path: "specs/DESORDENADO.md", Pattern: `RN-(\d+)`, AllowGaps: boolPtr(true)}); r.Status != StatusFail {
		t.Errorf("id_sequence desordenado = %s (%s)", r.Status, r.Detail)
	}

	// cross_ref: RN-001/RN-002 existem em REQUIREMENTS → pass
	xref := Check{Type: "cross_ref", Path: "specs/DATA_MODEL.md", Pattern: `RN-(\d+)`, Sources: []string{"REQUIREMENTS.md"}}
	if r := runOne(t, ws, xref); r.Status != StatusPass {
		t.Errorf("cross_ref ok = %s (%s)", r.Status, r.Detail)
	}
	// RN alucinada → fail citando
	writeFixture(t, ws, "specs/ALUCINADO.md", "referencia RN-999 e RN-001\n")
	aluc := Check{Type: "cross_ref", Path: "specs/ALUCINADO.md", Pattern: `RN-(\d+)`, Sources: []string{"REQUIREMENTS.md"}}
	if r := runOne(t, ws, aluc); r.Status != StatusFail || !strings.Contains(r.Detail, "999") {
		t.Errorf("cross_ref alucinado = %s (%s)", r.Status, r.Detail)
	}
}

func TestFrontmatterValidAndDirHasFiles(t *testing.T) {
	ws := t.TempDir()
	writeFixture(t, ws, "specs/DATA_MODEL.md", dmDoc) // começa com ---
	writeFixture(t, ws, "meetings/2026-01-01-01.md", "# checkpoint\n")

	if r := runOne(t, ws, Check{Type: "frontmatter_valid", Path: "specs/DATA_MODEL.md"}); r.Status != StatusPass {
		t.Errorf("frontmatter_valid ok = %s", r.Status)
	}
	writeFixture(t, ws, "specs/SEMFM.md", "# sem front matter\n")
	if r := runOne(t, ws, Check{Type: "frontmatter_valid", Path: "specs/SEMFM.md"}); r.Status != StatusFail {
		t.Errorf("frontmatter inválido = %s", r.Status)
	}
	if r := runOne(t, ws, Check{Type: "dir_has_files", Path: "meetings/"}); r.Status != StatusPass {
		t.Errorf("dir_has_files com arquivo = %s", r.Status)
	}
	os.MkdirAll(filepath.Join(ws, "vazio"), 0o755)
	if r := runOne(t, ws, Check{Type: "dir_has_files", Path: "vazio/"}); r.Status != StatusFail {
		t.Errorf("dir_has_files vazio = %s", r.Status)
	}
}

func TestMermaidValid(t *testing.T) {
	ws := t.TempDir()
	writeFixture(t, ws, "specs/DATA_MODEL.md", dmDoc)
	if r := runOne(t, ws, Check{Type: "mermaid_valid", Path: "specs/DATA_MODEL.md"}); r.Status != StatusPass {
		t.Errorf("mermaid ok = %s (%s)", r.Status, r.Detail)
	}
	// chaves desbalanceadas (linhas com -- excluídas da contagem)
	bad := "# x\n\n```mermaid\nerDiagram\n  A ||--o{ B : tem\n  N { campo sem fechar\n```\n"
	writeFixture(t, ws, "specs/BAD.md", bad)
	if r := runOne(t, ws, Check{Type: "mermaid_valid", Path: "specs/BAD.md"}); r.Status != StatusFail {
		t.Errorf("mermaid desbalanceado = %s (%s)", r.Status, r.Detail)
	}
	// sem bloco mermaid → fail
	writeFixture(t, ws, "specs/NOMERMAID.md", "# sem diagrama\n")
	if r := runOne(t, ws, Check{Type: "mermaid_valid", Path: "specs/NOMERMAID.md"}); r.Status != StatusFail {
		t.Errorf("mermaid ausente = %s", r.Status)
	}
	// tipo inválido na 1ª linha
	writeFixture(t, ws, "specs/TIPOERRADO.md", "```mermaid\nbanana\n```\n")
	if r := runOne(t, ws, Check{Type: "mermaid_valid", Path: "specs/TIPOERRADO.md"}); r.Status != StatusFail {
		t.Errorf("mermaid tipo inválido = %s", r.Status)
	}
}

func TestTableHasColumns(t *testing.T) {
	ws := t.TempDir()
	writeFixture(t, ws, "specs/DATA_MODEL.md", dmDoc)
	if r := runOne(t, ws, Check{Type: "table_has_columns", Path: "specs/DATA_MODEL.md", Columns: []string{"Tabela", "FK"}}); r.Status != StatusPass {
		t.Errorf("table_has_columns ok = %s", r.Status)
	}
	if r := runOne(t, ws, Check{Type: "table_has_columns", Path: "specs/DATA_MODEL.md", Columns: []string{"ColunaInexistente"}}); r.Status != StatusFail {
		t.Errorf("table_has_columns miss = %s", r.Status)
	}
}

func TestReferenceMatch(t *testing.T) {
	// fixture ≠ workspace (como no uso real: baseline vive na fixture)
	ws := t.TempDir()
	fx := t.TempDir()
	body := "# Doc\n\nAtualizado em 2026-01-02 às 09:30.\n\n| a | b |\n"
	writeFixture(t, ws, "specs/DOC.md", body)
	// baseline com data/hora diferentes + blank lines extras + trailing spaces → normaliza igual
	writeFixture(t, fx, "output-expected/DOC.md", "# Doc\n\n\nAtualizado em 2025-12-31 às 23:59.   \n\n\n\n| a | b |\n\n")
	if r := runOneAt(t, ws, fx, Check{Type: "reference_match", Path: "specs/DOC.md", Expected: "output-expected/DOC.md"}); r.Status != StatusPass {
		t.Errorf("reference_match tolerante = %s (%s)", r.Status, r.Detail)
	}
	// divergência real → fail
	writeFixture(t, ws, "specs/DOC.md", "# Doc DIFERENTE\n")
	if r := runOneAt(t, ws, fx, Check{Type: "reference_match", Path: "specs/DOC.md", Expected: "output-expected/DOC.md"}); r.Status != StatusFail {
		t.Errorf("reference_match divergente = %s", r.Status)
	}
	// baseline ausente na fixture E no workspace → fail "baseline ausente"
	ws2 := t.TempDir()
	if r := runOne(t, ws2, Check{Type: "reference_match", Path: "specs/DOC.md", Expected: "output-expected/NOPE.md"}); r.Status != StatusFail ||
		!strings.Contains(r.Detail, "baseline ausente") {
		t.Errorf("baseline ausente = %s (%s)", r.Status, r.Detail)
	}
	// fallback baseline-mode do runner: expected ausente na fixture, mas o
	// path existe no workspace → compara consigo mesmo (PASS trivial)
	ws3 := t.TempDir()
	writeFixture(t, ws3, "specs/DOC.md", body)
	if r := runOne(t, ws3, Check{Type: "reference_match", Path: "specs/DOC.md", Expected: "output-expected/NOPE.md"}); r.Status != StatusPass {
		t.Errorf("fallback baseline-mode = %s (%s)", r.Status, r.Detail)
	}
}

func TestExpectFailInversion(t *testing.T) {
	ws := t.TempDir()
	writeFixture(t, ws, "specs/QUEBRADO.md", "# sem nada esperado\n")
	// IMP-100: expect fail + check que falha = PASS (prato pego)
	if r := runOne(t, ws, Check{Type: "contains", Path: "specs/QUEBRADO.md", Pattern: "IMPOSSIVEL", Expect: "fail"}); r.Status != StatusPass {
		t.Errorf("expect:fail pegou o prato = %s", r.Status)
	}
	// expect fail + check que PASSA = FAIL (runner não discriminou)
	if r := runOne(t, ws, Check{Type: "contains", Path: "specs/QUEBRADO.md", Pattern: "# sem nada esperado", Expect: "fail"}); r.Status != StatusFail {
		t.Errorf("expect:fail não discriminado = %s", r.Status)
	}
}

func TestUnknownTypeSkipped(t *testing.T) {
	ws := t.TempDir()
	if r := runOne(t, ws, Check{Type: "check_quimico", Path: "x.md"}); r.Status != StatusSkipped {
		t.Errorf("tipo desconhecido = %s, want SKIPPED", r.Status)
	}
}

// ---- parser ----

func TestParseCriteriaFullSchema(t *testing.T) {
	y := `---
name: "Modelo de dados"
description: >
  REQUIREMENTS.md com RN-001 e RN-002.
  Verifica saída completa.
inputs_required: [REQUIREMENTS.md]

checks:
  - type: file_exists
    path: "specs/DATA_MODEL.md"
    severity: error
  - type: contains
    path: "specs/DATA_MODEL.md"
    pattern: "RN-001"
  - type: no_todos
    path: "specs/DATA_MODEL.md"
    patterns: ["TODO", "FIXME"]
    severity: warning
  - type: id_sequence
    path: "specs/DATA_MODEL.md"
    pattern: "RN-(\\d+)"
    allow_gaps: true
  - type: cross_ref
    path: "specs/DATA_MODEL.md"
    pattern: "RN-(\\d+)"
    sources: ["REQUIREMENTS.md"]
  - type: line_count_range
    path: "specs/DATA_MODEL.md"
    min: 10
    max: 400
  - type: section_order
    path: "specs/DATA_MODEL.md"
    sections:
      - "## 1. Diagrama"
      - "## 2. Tabelas"
  - type: reference_match
    path: "specs/DATA_MODEL.md"
    expected: "output-expected/DATA_MODEL.md"
  - type: expect_fail_demo
    path: "specs/DATA_MODEL.md"
    expect: fail
`
	c, err := ParseCriteria([]byte(y))
	if err != nil {
		t.Fatalf("ParseCriteria: %v", err)
	}
	if c.Name != "Modelo de dados" || len(c.InputsRequired) != 1 || len(c.Checks) != 9 {
		t.Errorf("envelope: %+v", c)
	}
	if !strings.Contains(c.Description, "RN-001 e RN-002") {
		t.Errorf("description multiline perdida: %q", c.Description)
	}
	// severity default = error
	if c.Checks[1].Severity != SeverityError {
		t.Errorf("severity default = %q", c.Checks[1].Severity)
	}
	if c.Checks[2].Severity != SeverityWarning {
		t.Errorf("severity explícito = %q", c.Checks[2].Severity)
	}
	if c.Checks[3].AllowGaps == nil || !*c.Checks[3].AllowGaps {
		t.Errorf("allow_gaps não parseado")
	}
	if c.Checks[5].Min == nil || *c.Checks[5].Min != 10 || *c.Checks[5].Max != 400 {
		t.Errorf("min/max não parseados")
	}
	if c.Checks[6].Sections[1] != "## 2. Tabelas" {
		t.Errorf("sections não parseadas")
	}
}

func TestParseCriteriaErrors(t *testing.T) {
	cases := []struct {
		name, y, wantSub string
	}{
		{"yaml quebrado", "checks: [a:\n  - {b", "ilegível"},
		{"sem checks", "name: x\n", "sem checks"},
		{"check sem type", "checks:\n  - path: a.md\n", "sem type"},
		{"contains sem pattern", "checks:\n  - type: contains\n    path: a.md\n", "pattern"},
		{"no_todos sem patterns", "checks:\n  - type: no_todos\n    path: a.md\n", "patterns"},
		{"line_range sem min", "checks:\n  - type: line_count_range\n    path: a.md\n    max: 5\n", "min e max"},
		{"reference_match sem expected", "checks:\n  - type: reference_match\n    path: a.md\n", "expected"},
		{"section_order sem sections", "checks:\n  - type: section_order\n    path: a.md\n", "sections"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseCriteria([]byte(tc.y))
			if err == nil || !strings.Contains(err.Error(), tc.wantSub) {
				t.Errorf("erro = %v, want contendo %q", err, tc.wantSub)
			}
		})
	}
}

// ---- suite ----

func TestRunFixtureAndAll(t *testing.T) {
	base := t.TempDir()
	fx := filepath.Join(base, "fixtures", "gm-datamodel", "fixture-001")
	ws := filepath.Join(base, "workspaces", "gm-datamodel", "fixture-001")
	os.MkdirAll(fx, 0o755)
	os.MkdirAll(ws, 0o755)
	writeFixture(t, ws, "specs/DATA_MODEL.md", dmDoc)
	writeFixture(t, ws, "REQUIREMENTS.md", reqDoc)
	writeFixture(t, fx, "criteria.yaml", "checks:\n  - type: file_exists\n    path: \"specs/DATA_MODEL.md\"\n  - type: contains\n    path: \"specs/DATA_MODEL.md\"\n    pattern: \"PRODUTOR\"\n  - type: contains\n    path: \"specs/DATA_MODEL.md\"\n    pattern: \"FALTA\"\n")

	res, err := RunFixture(fx, ws)
	if err != nil {
		t.Fatalf("RunFixture: %v", err)
	}
	if res.Status != StatusFail || res.Passed != 2 || res.Failed != 1 {
		t.Errorf("fixture result = %+v", res)
	}

	// --all espelhado
	suite, err := RunAll(filepath.Join(base, "fixtures"), filepath.Join(base, "workspaces"))
	if err != nil {
		t.Fatalf("RunAll: %v", err)
	}
	if len(suite.Fixtures) != 1 || suite.FailedFixtures != 1 || suite.Failed != 1 {
		t.Errorf("suite = %+v", suite)
	}

	// workspace espelhado ausente → skipped
	os.RemoveAll(filepath.Join(base, "workspaces", "gm-datamodel"))
	suite2, err := RunAll(filepath.Join(base, "fixtures"), filepath.Join(base, "workspaces"))
	if err != nil {
		t.Fatalf("RunAll 2: %v", err)
	}
	if len(suite2.Fixtures) != 1 || suite2.Fixtures[0].Status != StatusSkipped {
		t.Errorf("suite com workspace ausente = %+v", suite2.Fixtures)
	}
}

func TestRunFixtureSkippedWithoutCriteria(t *testing.T) {
	fx := t.TempDir()
	res, err := RunFixture(fx, t.TempDir())
	if err != nil {
		t.Fatalf("RunFixture: %v", err)
	}
	if res.Status != StatusSkipped || res.SkippedRsn == "" {
		t.Errorf("fixture sem criteria = %+v", res)
	}
}
