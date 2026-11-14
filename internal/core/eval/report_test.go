package eval

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func sampleResult(status string) *FixtureResult {
	return &FixtureResult{
		Fixture: "fixture-001", Status: status, Passed: 2, Failed: 1, Checks: []CheckResult{
			{Check: "file_exists:specs/DATA_MODEL.md", Type: "file_exists", Severity: SeverityError, Status: StatusPass},
			{Check: "contains:specs/DATA_MODEL.md", Type: "contains", Severity: SeverityError, Status: StatusPass},
			{Check: "contains:specs/DATA_MODEL.md", Type: "contains", Severity: SeverityError, Status: StatusFail,
				Detail: `pattern "FALTA" não encontrado em specs/DATA_MODEL.md | pipe`, Fix: "gere o conteúdo"},
		},
	}
}

// (a) WriteReport cria o arquivo com nome no padrão e conteúdo esperado
// (título, data, status, tabela com checks).
func TestWriteReportContentAndName(t *testing.T) {
	dir := t.TempDir() // home injetável via reportsDir — nunca a home real
	path, err := WriteReport("gm-datamodel", "fixture-001", "C:/ws/produto", sampleResult(StatusPass), dir)
	if err != nil {
		t.Fatalf("WriteReport: %v", err)
	}
	name := filepath.Base(path)
	if !regexp.MustCompile(`^eval-gm-datamodel-fixture-001-\d{8}-\d{6}\.md$`).MatchString(name) {
		t.Errorf("nome fora do padrão: %s", name)
	}
	content := readFile(t, path)
	for _, want := range []string{
		"# Eval Report — gm-datamodel/fixture-001",
		"- data (UTC): ",
		"- fixture: gm-datamodel/fixture-001",
		"- workspace: C:/ws/produto",
		"- status: PASS",
		"| Check | Tipo | Severidade | Status | Detalhe |",
		"| file_exists:specs/DATA_MODEL.md | file_exists | error | PASS |  |",
		`| contains:specs/DATA_MODEL.md | contains | error | FAIL | pattern "FALTA" não encontrado em specs/DATA_MODEL.md \| pipe |`,
	} {
		if !strings.Contains(content, want) {
			t.Errorf("relatório sem %q:\n%s", want, content)
		}
	}
	// data em RFC3339 UTC
	m := regexp.MustCompile(`- data \(UTC\): (\S+)`).FindStringSubmatch(content)
	if m == nil {
		t.Fatalf("linha de data ausente:\n%s", content)
	}
	if _, err := time.Parse(time.RFC3339, m[1]); err != nil {
		t.Errorf("data fora do RFC3339: %q (%v)", m[1], err)
	}
	if !strings.HasSuffix(m[1], "Z") {
		t.Errorf("data deveria ser UTC: %q", m[1])
	}

	// status FAIL também é reportado
	path2, err := WriteReport("gm-datamodel", "fixture-002", "ws", sampleResult(StatusFail), dir)
	if err != nil {
		t.Fatalf("WriteReport FAIL: %v", err)
	}
	if c := readFile(t, path2); !strings.Contains(c, "- status: FAIL") {
		t.Errorf("status FAIL ausente:\n%s", c)
	}
}

// (b) retenção: 55 relatórios semeados + 1 novo → restam 50 e os mais
// antigos sumiram.
func TestWriteReportRetention(t *testing.T) {
	dir := t.TempDir()
	base := time.Now().Add(-2 * time.Hour)
	var oldest, newestKept string
	for i := 0; i < 55; i++ {
		name := filepath.Join(dir, "eval-antigo-fixture-"+strings.Repeat("x", i+1)+"-20250101-000000.md")
		if err := os.WriteFile(name, []byte("old"), 0o644); err != nil {
			t.Fatal(err)
		}
		mod := base.Add(time.Duration(i) * time.Minute)
		if err := os.Chtimes(name, mod, mod); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			oldest = name
		}
		if i == 6 {
			newestKept = name // fronteira: 55 sementes + 1 novo = 56 → os 6 mais antigos caem
		}
	}
	if _, err := WriteReport("gm-datamodel", "fixture-001", "ws", sampleResult(StatusPass), dir); err != nil {
		t.Fatalf("WriteReport: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != EvalReportRetention {
		t.Fatalf("restaram %d relatórios, want %d", len(entries), EvalReportRetention)
	}
	if _, err := os.Stat(oldest); !os.IsNotExist(err) {
		t.Errorf("o relatório mais antigo deveria ter sido podado (err=%v)", err)
	}
	if _, err := os.Stat(newestKept); err != nil {
		t.Errorf("relatório na fronteira da retenção deveria permanecer: %v", err)
	}
}

func TestWriteReportNilResultAndDefaultDirErrors(t *testing.T) {
	if _, err := WriteReport("s", "f", "ws", nil, t.TempDir()); err == nil {
		t.Errorf("res nil deveria errar")
	}
}

// (c/d base) ListReports lista desc por mtime com parse skill/fixture;
// arquivos fora do padrão aparecem com skill/fixture vazios; cap 50;
// diretório ausente = lista vazia sem erro.
func TestListReports(t *testing.T) {
	dir := t.TempDir()
	p1, err := WriteReport("gm-datamodel", "fixture-001", "ws", sampleResult(StatusPass), dir)
	if err != nil {
		t.Fatal(err)
	}
	p2, err := WriteReport("gm-datamodel", "fixture-002", "ws", sampleResult(StatusPass), dir)
	if err != nil {
		t.Fatal(err)
	}
	outro := filepath.Join(dir, "notas-soltas.md")
	if err := os.WriteFile(outro, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// mtime determinístico (mesmo segundo → ordem de criação não é garantia)
	base := time.Now()
	modOut := base.Add(-3 * time.Minute)
	mod1 := base.Add(-2 * time.Minute)
	mod2 := base.Add(-1 * time.Minute)
	os.Chtimes(outro, modOut, modOut)
	os.Chtimes(p1, mod1, mod1)
	os.Chtimes(p2, mod2, mod2)

	list, err := ListReports(dir)
	if err != nil {
		t.Fatalf("ListReports: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("listou %d, want 3: %+v", len(list), list)
	}
	if list[0].File != p2 || list[1].File != p1 || list[2].File != outro {
		t.Errorf("ordem desc por mtime incorreta: %+v", list)
	}
	if list[0].Skill != "gm-datamodel" || list[0].Fixture != "fixture-002" {
		t.Errorf("parse skill/fixture = %q/%q", list[0].Skill, list[0].Fixture)
	}
	if list[2].Skill != "" || list[2].Fixture != "" {
		t.Errorf("arquivo fora do padrão deveria vir vazio: %+v", list[2])
	}
	if list[0].Modified.IsZero() {
		t.Errorf("modified zerado: %+v", list[0])
	}

	// cap 50
	for i := 0; i < 55; i++ {
		os.WriteFile(filepath.Join(dir, "eval-lote-fixture-00"+strings.Repeat("y", i+1)+"-20250102-000000.md"), []byte("x"), 0o644)
	}
	list2, err := ListReports(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(list2) != EvalReportRetention {
		t.Errorf("cap = %d, want %d", len(list2), EvalReportRetention)
	}

	// diretório ausente → vazia, sem erro
	vazia, err := ListReports(filepath.Join(t.TempDir(), "nada"))
	if err != nil || len(vazia) != 0 {
		t.Errorf("dir ausente = (%v, %d), want (nil, 0)", err, len(vazia))
	}
}

func TestSkillFixture(t *testing.T) {
	skill, fixture := SkillFixture(filepath.Join("raiz", "gm-datamodel", "fixture-001"))
	if skill != "gm-datamodel" || fixture != "fixture-001" {
		t.Errorf("SkillFixture = %q/%q", skill, fixture)
	}
}

func TestParseReportName(t *testing.T) {
	cases := []struct {
		name      string
		skill, fx string
		ok        bool
	}{
		{"eval-gm-datamodel-fixture-001-20260114-153000.md", "gm-datamodel", "fixture-001", true},
		{"eval-gm-pack-update-fixture-001-20260114-153000.md", "gm-pack-update", "fixture-001", true},
		{"notas-soltas.md", "", "", false},
		{"eval-sem-ts.md", "", "", false},
		{"eval-so-skill-20260114-153000.md", "", "", false},
	}
	for _, tc := range cases {
		skill, fx, ok := parseReportName(tc.name)
		if ok != tc.ok || skill != tc.skill || fx != tc.fx {
			t.Errorf("parseReportName(%q) = %q/%q/%v, want %q/%q/%v", tc.name, skill, fx, ok, tc.skill, tc.fx, tc.ok)
		}
	}
}
