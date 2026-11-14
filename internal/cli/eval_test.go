package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Golmet/golmet-cli/internal/core/eval"
)

// evalFixture monta uma fixture mínima + workspace com o artefato.
func evalFixture(t *testing.T) (string, string) {
	t.Helper()
	base := inTempDir(t)
	fx := filepath.Join(base, "fx", "gm-datamodel", "fixture-001")
	ws := filepath.Join(base, "ws", "gm-datamodel", "fixture-001")
	if err := os.MkdirAll(filepath.Join(fx), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(ws, "specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fx, "criteria.yaml"),
		[]byte("checks:\n  - type: file_exists\n    path: \"specs/DATA_MODEL.md\"\n  - type: contains\n    path: \"specs/DATA_MODEL.md\"\n    pattern: \"PRODUTOR\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws, "specs", "DATA_MODEL.md"),
		[]byte("# DM\n\nPRODUTOR aqui.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return fx, ws
}

func TestEvalCmdSinglePassJSON(t *testing.T) {
	fx, ws := evalFixture(t)
	out, err := runCmd(t, "eval", fx, "--workspace", ws, "--json")
	if err != nil {
		t.Fatalf("eval: %v\n%s", err, out)
	}
	var res map[string]any
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("json inválido: %v\n%s", err, out)
	}
	if res["status"] != eval.StatusPass {
		t.Errorf("status = %v (saída: %s)", res["status"], out)
	}
}

func TestEvalCmdSingleFailExit1(t *testing.T) {
	fx, ws := evalFixture(t)
	os.Remove(filepath.Join(ws, "specs", "DATA_MODEL.md")) // file_exists quebra
	_, err := runCmd(t, "eval", fx, "--workspace", ws)
	if code, ok := ExitCodeOf(err); !ok || code != 1 {
		t.Errorf("eval com falha deveria sair 1, veio %d (ok=%v)", code, ok)
	}
}

func TestEvalCmdUsageErrors(t *testing.T) {
	// sem --workspace → 2
	_, err := runCmd(t, "eval", filepath.Join(t.TempDir(), "fx"))
	if code, ok := ExitCodeOf(err); !ok || code != 2 {
		t.Errorf("eval sem --workspace deveria sair 2, veio %d (ok=%v)", code, ok)
	}
	// --all sem workspaces-root → 2
	_, err = runCmd(t, "eval", "--all", t.TempDir())
	if code, ok := ExitCodeOf(err); !ok || code != 2 {
		t.Errorf("--all sem --workspaces-root deveria sair 2, veio %d (ok=%v)", code, ok)
	}
	// criteria inválido → 2 (erro estruturado de parse)
	base := t.TempDir()
	fx := filepath.Join(base, "fx")
	os.MkdirAll(fx, 0o755)
	os.WriteFile(filepath.Join(fx, "criteria.yaml"), []byte("checks:\n  - type: contains\n    path: a.md\n"), 0o644)
	_, err = runCmd(t, "eval", fx, "--workspace", base)
	if code, ok := ExitCodeOf(err); !ok || code != 2 {
		t.Errorf("criteria inválido deveria sair 2, veio %d (ok=%v)", code, ok)
	}
}

func TestEvalCmdAllSuite(t *testing.T) {
	fx, ws := evalFixture(t)
	base := filepath.Dir(filepath.Dir(filepath.Dir(fx))) // <base>
	wsRoot := filepath.Dir(filepath.Dir(filepath.Dir(ws)))
	out, err := runCmd(t, "eval", "--all", filepath.Join(base, "fx"), "--workspaces-root", wsRoot)
	if err != nil {
		t.Fatalf("eval --all: %v\n%s", err, out)
	}
	if !strings.Contains(out, "gm-datamodel/fixture-001") || !strings.Contains(out, "geral:") {
		t.Errorf("saída da suite inesperada:\n%s", out)
	}
	// --json da suite
	out2, err := runCmd(t, "eval", "--all", filepath.Join(base, "fx"), "--workspaces-root", wsRoot, "--json")
	if err != nil {
		t.Fatalf("eval --all --json: %v\n%s", err, out2)
	}
	var suite map[string]any
	if err := json.Unmarshal([]byte(out2), &suite); err != nil {
		t.Fatalf("suite json inválida: %v", err)
	}
}

// corpus do repo passa no checker via single-fixture quando os workspaces
// são semeados (smoke estrutural com uma fixture real versionada).
func TestEvalCmdCorpusFixtureSmoke(t *testing.T) {
	base := inTempDir(t)
	src := filepath.Join("..", "..", "evals", "gm-datamodel", "fixture-001")
	if _, err := os.Stat(src); err != nil {
		t.Skip("corpus não acessível a partir do cli package")
	}
	ws := filepath.Join(base, "ws")
	os.MkdirAll(filepath.Join(ws, "specs"), 0o755)
	os.WriteFile(filepath.Join(ws, "specs", "DATA_MODEL.md"), []byte("# DM\nRN-001\n"), 0o644)
	out, err := runCmd(t, "eval", src, "--workspace", ws, "--json")
	if err != nil {
		t.Fatalf("eval de fixture versionada: %v\n%s", err, out)
	}
	if !strings.Contains(out, `"fixture"`) {
		t.Errorf("json sem fixture: %s", out)
	}
}
