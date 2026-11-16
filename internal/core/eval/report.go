// report.go — armazenamento canônico dos relatórios de eval (decisão de
// produto: relatórios são ESCRITOS PELO KIT em estado de máquina do kit —
// ~/.golmet/eval-reports/, mesma convenção de ~/.golmet/packs e
// ~/.golmet/sources do source sync). O RUNTIME nunca gerencia a pasta:
// roda gm_eval, recebe report_path no payload e lê o arquivo quando
// quiser o relatório humano. O workspace não recebe relatório.
package eval

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// EvalReportRetention — retenção: mantém os N relatórios mais recentes
// (por mtime) em ~/.golmet/eval-reports/.
const EvalReportRetention = 50

// ReportDirDefault devolve <home>/.golmet/eval-reports.
func ReportDirDefault(home string) string {
	return filepath.Join(home, ".golmet", "eval-reports")
}

// SkillFixture deriva (skill, fixture) do caminho da fixture pela
// convenção do corpus: <fixtures-root>/<skill>/<fixture>.
func SkillFixture(fixtureDir string) (string, string) {
	abs, err := filepath.Abs(fixtureDir)
	if err != nil {
		abs = fixtureDir
	}
	return filepath.Base(filepath.Dir(abs)), filepath.Base(abs)
}

// WriteReport renderiza o relatório markdown da fixture e grava em
// reportsDir ("" = default ~/.golmet/eval-reports via UserHomeDir — o
// home é injetável para testes). Mantém os EvalReportRetention relatórios
// mais recentes; erro de prune é best-effort e não falha o eval.
// skill/fixture são componentes de nome (sem separadores) — derivados via
// SkillFixture/SplitFixtureID pelos callers.
func WriteReport(skill, fixture, workspace string, res *FixtureResult, reportsDir string) (string, error) {
	if res == nil {
		return "", fmt.Errorf("resultado da fixture ausente")
	}
	dir := reportsDir
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("sem UserHomeDir e reportsDir vazio: %w", err)
		}
		dir = ReportDirDefault(home)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("não criou o diretório de relatórios: %w", err)
	}
	when := time.Now().UTC()
	name := fmt.Sprintf("eval-%s-%s-%s.md", skill, fixture, when.Format("20060102-150405"))
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(renderReport(skill, fixture, workspace, res, when)), 0o644); err != nil {
		return "", fmt.Errorf("não gravou o relatório: %w", err)
	}
	_ = pruneReports(dir, EvalReportRetention) // best-effort
	return path, nil
}

// renderReport — markdown humano: título, data UTC RFC3339, fixture,
// workspace, status e tabela de checks (check/type/severity/status/detail).
func renderReport(skill, fixture, workspace string, res *FixtureResult, when time.Time) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Eval Report — %s/%s\n\n", skill, fixture)
	fmt.Fprintf(&b, "- data (UTC): %s\n", when.Format(time.RFC3339))
	fmt.Fprintf(&b, "- fixture: %s/%s\n", skill, fixture)
	fmt.Fprintf(&b, "- workspace: %s\n", workspace)
	fmt.Fprintf(&b, "- status: %s\n", strings.ToUpper(res.Status))
	if res.Status == StatusSkipped && res.SkippedRsn != "" {
		fmt.Fprintf(&b, "- skipped: %s\n", res.SkippedRsn)
	}
	fmt.Fprintf(&b, "- resumo: %d PASS, %d FAIL, %d WARN, %d SKIPPED\n\n", res.Passed, res.Failed, res.Warned, res.Skipped)
	b.WriteString("| Check | Tipo | Severidade | Status | Detalhe |\n|---|---|---|---|---|\n")
	for _, c := range res.Checks {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
			mdCell(c.Check), mdCell(c.Type), mdCell(c.Severity), mdCell(c.Status), mdCell(c.Detail))
	}
	return b.String()
}

// mdCell escapa pipes e quebras de linha para a célula da tabela.
func mdCell(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	return strings.ReplaceAll(s, "\n", " ")
}

// pruneReports apaga os relatórios mais antigos além do limite (por mtime).
func pruneReports(dir string, keep int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	type stamped struct {
		name string
		mod  time.Time
	}
	var files []stamped
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, ierr := e.Info()
		if ierr != nil {
			continue
		}
		files = append(files, stamped{e.Name(), info.ModTime()})
	}
	if len(files) <= keep {
		return nil
	}
	sort.Slice(files, func(i, j int) bool { return files[i].mod.After(files[j].mod) })
	var firstErr error
	for _, f := range files[keep:] {
		if rerr := os.Remove(filepath.Join(dir, f.name)); rerr != nil && firstErr == nil {
			firstErr = rerr
		}
	}
	return firstErr
}

// ReportEntry é um relatório listado por ListReports (payload do
// gm_eval_reports). File é o path absoluto — o runtime lê direto.
type ReportEntry struct {
	File     string    `json:"file"`
	Skill    string    `json:"skill"`
	Fixture  string    `json:"fixture"`
	Modified time.Time `json:"modified"`
}

// evalReportNameRe — padrão do nome: eval-<skill>-<fixture>-<ts>.md com
// ts = 20060102-150405 (o ts é a âncora da direita; skill/fixture podem
// conter hífen).
var evalReportNameRe = regexp.MustCompile(`^eval-(.+)-(\d{8}-\d{6})\.md$`)

// parseReportName extrai (skill, fixture) do nome — a componente fixture
// inicia por "fixture-" (convenção do corpus). Fora do padrão → ok=false
// (o arquivo continua listado, com skill/fixture vazios).
func parseReportName(name string) (skill, fixture string, ok bool) {
	m := evalReportNameRe.FindStringSubmatch(name)
	if m == nil {
		return "", "", false
	}
	i := strings.Index(m[1], "-fixture")
	if i <= 0 || i+1 >= len(m[1]) {
		return "", "", false
	}
	return m[1][:i], m[1][i+1:], true
}

// ListReports lista os relatórios em dir ordenados desc por mtime, cap
// EvalReportRetention. Diretório ausente/vazio = lista vazia (não erro).
func ListReports(dir string) ([]ReportEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []ReportEntry{}, nil
		}
		return nil, fmt.Errorf("não listou o diretório de relatórios: %w", err)
	}
	out := []ReportEntry{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, ierr := e.Info()
		if ierr != nil {
			continue
		}
		entry := ReportEntry{File: filepath.Join(dir, e.Name()), Modified: info.ModTime()}
		if skill, fixture, ok := parseReportName(e.Name()); ok {
			entry.Skill, entry.Fixture = skill, fixture
		}
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Modified.After(out[j].Modified) })
	if len(out) > EvalReportRetention {
		out = out[:EvalReportRetention]
	}
	return out, nil
}
