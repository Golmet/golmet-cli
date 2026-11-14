// suite.go — execução de uma fixture e da suite completa (--all).
// Convenção espelhada: <fixtures-root>/<skill>/fixture-N ↔
// <workspaces-root>/<skill>/fixture-N — o CI semeia o workspace e roda a
// skill; o kit pontua com os mesmos criteria nas duas rodas (paridade).
package eval

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CheckResult é o resultado de um check individual.
type CheckResult struct {
	Check      string `json:"check"` // label "<type>:<path>"
	Type       string `json:"type"`
	Path       string `json:"path,omitempty"`
	Severity   string `json:"severity"`
	Status     string `json:"status"` // PASS | FAIL | WARN | SKIPPED
	Detail     string `json:"detail,omitempty"`
	Fix        string `json:"fix,omitempty"`
	ExpectFail bool   `json:"expect_fail,omitempty"` // IMP-100
}

// FixtureResult é o resultado de uma fixture.
type FixtureResult struct {
	Fixture    string        `json:"fixture"` // "<skill>/<fixture-N>"
	Status     string        `json:"status"`  // pass | fail | skipped
	SkippedRsn string        `json:"skipped_reason,omitempty"`
	Checks     []CheckResult `json:"checks,omitempty"`
	Passed     int           `json:"passed"`
	Failed     int           `json:"failed"`
	Warned     int           `json:"warned"`
	Skipped    int           `json:"skipped"`
}

// SuiteResult agrega o --all.
type SuiteResult struct {
	FixturesRoot   string          `json:"fixtures_root"`
	WorkspacesRoot string          `json:"workspaces_root"`
	Fixtures       []FixtureResult `json:"fixtures"`
	Passed         int             `json:"passed"`
	Failed         int             `json:"failed"`
	Warned         int             `json:"warned"`
	Skipped        int             `json:"skipped"`
	FailedFixtures int             `json:"failed_fixtures"`
}

// RunFixture avalia os checks de fixtureDir contra o workspace.
// Fixture sem criteria.yaml = SKIPPED (não aborta a suite).
func RunFixture(fixtureDir, workspace string) (*FixtureResult, error) {
	abs, err := filepath.Abs(fixtureDir)
	if err != nil {
		return nil, fmt.Errorf("fixture inválida: %w", err)
	}
	if st, serr := os.Stat(abs); serr != nil || !st.IsDir() {
		return nil, fmt.Errorf("fixture não encontrada: %s", fixtureDir)
	}
	res := &FixtureResult{Fixture: filepath.Base(abs)}
	data, err := os.ReadFile(filepath.Join(abs, "criteria.yaml"))
	if err != nil {
		if os.IsNotExist(err) {
			res.Status = StatusSkipped
			res.SkippedRsn = "criteria.yaml ausente"
			res.Skipped = 1
			return res, nil
		}
		return nil, fmt.Errorf("criteria.yaml ilegível: %w", err)
	}
	criteria, cerr := ParseCriteria(data)
	if cerr != nil {
		return nil, cerr // erro de uso (CLI exit 2)
	}
	res.Status = StatusPass
	for i := range criteria.Checks {
		chk := &criteria.Checks[i]
		status, detail, fix := runCheck(workspace, abs, chk)
		cr := CheckResult{
			Check:      chk.Type + mapBool(chk.Path != "", ":"+chk.Path, ""),
			Type:       chk.Type,
			Path:       chk.Path,
			Severity:   chk.Severity,
			Status:     status,
			Detail:     detail,
			Fix:        fix,
			ExpectFail: chk.Expect == "fail",
		}
		switch status {
		case StatusPass:
			res.Passed++
		case StatusFail:
			res.Failed++
			res.Status = StatusFail
		case StatusWarn:
			res.Warned++
		case StatusSkipped:
			res.Skipped++
		}
		res.Checks = append(res.Checks, cr)
	}
	return res, nil
}

// RunAll varre <fixturesRoot>/<skill>/fixture-N/ e pontua cada uma contra
// o workspace espelhado <workspacesRoot>/<skill>/fixture-N/.
func RunAll(fixturesRoot, workspacesRoot string) (*SuiteResult, error) {
	for _, d := range []string{fixturesRoot, workspacesRoot} {
		if st, err := os.Stat(d); err != nil || !st.IsDir() {
			return nil, fmt.Errorf("raiz não encontrada: %s", d)
		}
	}
	suite := &SuiteResult{FixturesRoot: fixturesRoot, WorkspacesRoot: workspacesRoot}
	skills, err := os.ReadDir(fixturesRoot)
	if err != nil {
		return nil, fmt.Errorf("raiz de fixtures ilegível: %w", err)
	}
	var names []string
	for _, s := range skills {
		if s.IsDir() {
			names = append(names, s.Name())
		}
	}
	sort.Strings(names)
	for _, skill := range names {
		fixtures, ferr := os.ReadDir(filepath.Join(fixturesRoot, skill))
		if ferr != nil {
			continue
		}
		var fxNames []string
		for _, fx := range fixtures {
			if fx.IsDir() && strings.HasPrefix(fx.Name(), "fixture-") {
				fxNames = append(fxNames, fx.Name())
			}
		}
		sort.Strings(fxNames)
		for _, fx := range fxNames {
			fixtureDir := filepath.Join(fixturesRoot, skill, fx)
			workspace := filepath.Join(workspacesRoot, skill, fx)
			if _, err := os.Stat(workspace); err != nil {
				// workspace espelhado ausente — fixture skipped (CI não rodou)
				suite.Fixtures = append(suite.Fixtures, FixtureResult{
					Fixture: skill + "/" + fx, Status: StatusSkipped,
					SkippedRsn: "workspace espelhado ausente: " + skill + "/" + fx,
					Skipped:    1,
				})
				suite.Skipped++
				continue
			}
			res, rerr := RunFixture(fixtureDir, workspace)
			if rerr != nil {
				return nil, fmt.Errorf("%s/%s: %w", skill, fx, rerr)
			}
			res.Fixture = skill + "/" + fx
			suite.Fixtures = append(suite.Fixtures, *res)
			suite.Passed += res.Passed
			suite.Failed += res.Failed
			suite.Warned += res.Warned
			suite.Skipped += res.Skipped
			if res.Status == StatusFail {
				suite.FailedFixtures++
			}
		}
	}
	return suite, nil
}

// ---- render human ----

func markFor(status, severity string) string {
	switch status {
	case StatusPass:
		if severity == SeverityWarning {
			return "✅"
		}
		return "✅"
	case StatusFail:
		return "❌"
	case StatusWarn:
		return "⚠️ "
	case StatusSkipped:
		return "–"
	}
	return "?"
}

// Human renderiza o resultado human de uma fixture.
func (r *FixtureResult) Human() string {
	var b strings.Builder
	fmt.Fprintf(&b, "fixture: %s [%s]", r.Fixture, r.Mode())
	if r.Status == StatusSkipped {
		fmt.Fprintf(&b, " — skipped: %s\n", r.SkippedRsn)
		return b.String()
	}
	b.WriteString("\n")
	for _, c := range r.Checks {
		fmt.Fprintf(&b, "  %s %s [%s/%s]\n", markFor(c.Status, c.Severity), c.Check, c.Status, c.Severity)
		if c.Status != StatusPass && c.Detail != "" {
			fmt.Fprintf(&b, "      %s\n", c.Detail)
		}
		if c.Status != StatusPass && c.Fix != "" {
			fmt.Fprintf(&b, "      fix: %s\n", c.Fix)
		}
	}
	fmt.Fprintf(&b, "  resumo: %d PASS, %d FAIL, %d WARN, %d SKIPPED\n", r.Passed, r.Failed, r.Warned, r.Skipped)
	return b.String()
}

// Mode resume o estado geral da fixture (pass|fail|skipped).
func (r *FixtureResult) Mode() string { return r.Status }

// Human renderiza a suite completa.
func (s *SuiteResult) Human() string {
	var b strings.Builder
	fmt.Fprintf(&b, "suite de evals — fixtures: %s · workspaces: %s\n", s.FixturesRoot, s.WorkspacesRoot)
	for _, f := range s.Fixtures {
		switch f.Status {
		case StatusSkipped:
			fmt.Fprintf(&b, "  – %s — skipped (%s)\n", f.Fixture, f.SkippedRsn)
		default:
			fmt.Fprintf(&b, "  %s %s — %d PASS, %d FAIL, %d WARN, %d SKIPPED\n",
				mapBool(f.Status == StatusFail, "❌", "✅"), f.Fixture, f.Passed, f.Failed, f.Warned, f.Skipped)
			for _, c := range f.Checks {
				if c.Status == StatusFail || c.Status == StatusWarn {
					fmt.Fprintf(&b, "      %s %s [%s] %s", markFor(c.Status, c.Severity), c.Check, c.Severity, c.Detail)
					if c.Fix != "" {
						fmt.Fprintf(&b, " — fix: %s", c.Fix)
					}
					b.WriteString("\n")
				}
			}
		}
	}
	fmt.Fprintf(&b, "geral: %d fixture(s) — %d PASS, %d FAIL, %d WARN, %d SKIPPED · %d fixture(s) com falha\n",
		len(s.Fixtures), s.Passed, s.Failed, s.Warned, s.Skipped, s.FailedFixtures)
	return b.String()
}
