package scan

import (
	"os"
	"os/exec"
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

func gitAvailable(t *testing.T) bool {
	t.Helper()
	return exec.Command("git", "--version").Run() == nil
}

// makeRepo cria um repo git com um commit e devolve o SHA do HEAD.
func makeRepo(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	os.MkdirAll(filepath.Join(dir, "src"), 0o755)
	if err := os.WriteFile(filepath.Join(dir, "src", "Agenda.cs"), []byte("class Agenda {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("init", "-b", "main")
	run("add", ".")
	run("-c", "user.email=kit@golmet.dev", "-c", "user.name=golmet", "commit", "-m", "init")
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

func rulesIn(res *Result, severity Severity) map[string]bool {
	set := map[string]bool{}
	var xs []GolmetError
	if severity == severityError {
		xs = res.Errors
	} else {
		xs = res.Warnings
	}
	for _, e := range xs {
		set[e.Rule] = true
	}
	return set
}

func TestValidWorkspaceTolerant(t *testing.T) {
	tmp := t.TempDir()
	ws := filepath.Join(tmp, "ws")
	copyDir(t, filepath.Join(testdata, "valid-workspace"), ws)
	res, err := Run(ws, Options{Tolerant: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// sem repo real, H7 degrada para aviso (--tolerant) — zero erros
	if len(res.Errors) != 0 {
		for _, e := range res.Errors {
			t.Errorf("erro inesperado: %+v", e)
		}
	}
	if res.Counts["rn"] != 1 || res.Counts["pend"] != 1 || res.Counts["feature"] != 1 || res.Counts["task"] != 1 {
		t.Errorf("counts = %+v", res.Counts)
	}
}

func TestValidWorkspaceStrictWithGit(t *testing.T) {
	if !gitAvailable(t) {
		t.Skip("git indisponível")
	}
	tmp := t.TempDir()
	ws := filepath.Join(tmp, "ws")
	repo := filepath.Join(tmp, "MeuApp-API")
	head := makeRepo(t, repo)
	copyDir(t, filepath.Join(testdata, "valid-workspace"), ws)
	// substitui o hash placeholder pelo hash real do repo criado
	err := filepath.WalkDir(ws, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		fixed := strings.ReplaceAll(string(data), "a1b2c3d", head[:7])
		return os.WriteFile(p, []byte(fixed), 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := Run(ws, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Errors) != 0 {
		for _, e := range res.Errors {
			t.Errorf("erro inesperado: %+v", e)
		}
	}
	if len(res.Warnings) != 0 {
		for _, e := range res.Warnings {
			t.Errorf("aviso inesperado: %+v", e)
		}
	}
}

func TestAmbiguousHashIsH7(t *testing.T) {
	if !gitAvailable(t) {
		t.Skip("git indisponível")
	}
	tmp := t.TempDir()
	ws := filepath.Join(tmp, "ws")
	headA := makeRepo(t, filepath.Join(tmp, "repoA"))
	makeRepo(t, filepath.Join(tmp, "repoB")) // commit idêntico → mesmo SHA
	copyDir(t, filepath.Join(testdata, "valid-workspace"), ws)
	err := filepath.WalkDir(ws, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		fixed := strings.ReplaceAll(string(data), "../MeuApp-API", "../repoA\n| ../repoB | main | "+headA[:7]+" |")
		return os.WriteFile(p, []byte(fixed), 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := Run(ws, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !rulesIn(res, severityError)["H7"] {
		t.Errorf("esperava H7 (ambiguidade), erros=%+v avisos=%+v", res.Errors, res.Warnings)
	}
}

func TestViolations(t *testing.T) {
	cases := []struct {
		name     string
		dir      string
		tolerant bool
		wantErr  []string
		wantWarn []string
	}{
		{"h1", "violations/h1", false, []string{"H1"}, nil},
		{"h2", "violations/h2", false, []string{"H2"}, nil},
		{"h3", "violations/h3", false, []string{"H3"}, nil},
		{"h4 strict", "violations/h4", false, []string{"H4"}, nil},
		{"h4 tolerant", "violations/h4", true, nil, []string{"H4"}},
		{"h5", "violations/h5", false, []string{"H5"}, nil},
		{"h6", "violations/h6", false, []string{"H6"}, nil},
		{"h7 strict", "violations/h7", false, []string{"H7"}, nil},
		{"h7 tolerant", "violations/h7", true, nil, []string{"H7"}},
		{"h8", "violations/h8", false, []string{"H8"}, nil},
		{"h9", "violations/h9", false, []string{"H9"}, nil},
		{"h10", "violations/h10", false, []string{"H10"}, nil},
		{"s1", "violations/s1", false, nil, []string{"S1"}},
		{"s4", "violations/s4", false, []string{"H4"}, []string{"S4"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			ws := filepath.Join(tmp, "ws")
			copyDir(t, filepath.Join(testdata, tc.dir), ws)
			res, err := Run(ws, Options{Tolerant: tc.tolerant})
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			errs := rulesIn(res, severityError)
			warns := rulesIn(res, severityWarning)
			for _, want := range tc.wantErr {
				if !errs[want] {
					t.Errorf("esperava erro %s; erros=%+v avisos=%+v", want, summary(res.Errors), summary(res.Warnings))
				}
			}
			for _, want := range tc.wantWarn {
				if !warns[want] {
					t.Errorf("esperava aviso %s; erros=%+v avisos=%+v", want, summary(res.Errors), summary(res.Warnings))
				}
			}
		})
	}
}

func TestMissingGolmetYamlIsConfigError(t *testing.T) {
	tmp := t.TempDir()
	if _, err := Run(tmp, Options{}); err == nil {
		t.Errorf("esperava erro de config (golmet.yaml ausente)")
	}
}

func TestUnsupportedFormatIsConfigError(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "golmet.yaml"), []byte("format: 2\n"), 0o644)
	_, err := Run(tmp, Options{})
	if err == nil || !strings.Contains(err.Error(), "não suportado") {
		t.Errorf("esperava erro de formato, veio %v", err)
	}
}

func TestBadGolmetYamlIsH5(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "golmet.yaml"), []byte("format: |\n  multiline\n"), 0o644)
	res, err := Run(tmp, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err) // parse falho vira H5, não config error
	}
	if !rulesIn(res, severityError)["H5"] {
		t.Errorf("esperava H5 para golmet.yaml ilegível; erros=%+v", summary(res.Errors))
	}
}
