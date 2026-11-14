package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Regra EOL-tolerante na zona .golmet/ do workspace: skill com o mesmo
// texto em CRLF no disco não é "modificada localmente" — o update cura
// rematerializando em LF; update --check não reporta modificação.
func crlfify(b []byte) []byte { return []byte(strings.ReplaceAll(string(b), "\n", "\r\n")) }

func TestInstallCuresEOLAndCheckClean(t *testing.T) {
	ws := t.TempDir()
	// 1. BASE instalada (LF canônico)
	if _, err := Install(ws); err != nil {
		t.Fatalf("install 1: %v", err)
	}
	skills, skillErrs, err := EmbeddedSkills()
	if err != nil || len(skillErrs) > 0 {
		t.Fatalf("EmbeddedSkills: %v %v", err, skillErrs)
	}

	// 2. semeia CRLF (mesmo texto) em duas skills
	mutated := map[string]bool{}
	for _, sk := range skills {
		if sk.Dir != "gm-charter" && sk.Dir != "gm-reverse" {
			continue
		}
		target := filepath.Join(ws, ".golmet", filepath.FromSlash(sk.File))
		data, rerr := os.ReadFile(target)
		if rerr != nil {
			t.Fatal(rerr)
		}
		if err := os.WriteFile(target, crlfify(data), 0o644); err != nil {
			t.Fatal(err)
		}
		mutated[sk.Name] = true
	}

	// 3. --check NÃO reporta modificação (EOL não é customização)
	check, err := Check(ws)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if len(check.Modificados) != 0 {
		t.Errorf("check deveria ser EOL-tolerante, modificados=%v", check.Modificados)
	}

	// 4. update cura: nada preservado, skills reescritas em LF
	rep, err := Install(ws)
	if err != nil {
		t.Fatalf("install 2: %v", err)
	}
	if len(rep.Preserved) != 0 {
		t.Errorf("EOL-only não é customização — preserved deveria ser vazio: %v", rep.Preserved)
	}
	if rep.Installed == 0 {
		t.Errorf("update deveria reescrever as skills curadas, installed=%d", rep.Installed)
	}
	for _, sk := range skills {
		if !mutated[sk.Name] {
			continue
		}
		data, rerr := os.ReadFile(filepath.Join(ws, ".golmet", filepath.FromSlash(sk.File)))
		if rerr != nil {
			t.Fatal(rerr)
		}
		if strings.Contains(string(data), "\r") {
			t.Errorf("%s deveria ter sido curada para LF", sk.Name)
		}
	}

	// 5. terceira execução: idempotente — 0 escritas novas
	rep3, err := Install(ws)
	if err != nil {
		t.Fatalf("install 3: %v", err)
	}
	if rep3.Installed != 0 || len(rep3.Preserved) != 0 {
		t.Errorf("re-run deveria ser idempotente: installed=%d preserved=%v", rep3.Installed, rep3.Preserved)
	}

	// 6. regressão: customização REAL continua preservada
	target := filepath.Join(ws, ".golmet", "skills", "gm-charter", "SKILL.md")
	os.WriteFile(target, []byte("---\nname: gm-charter\n---\n# adaptação local real\n"), 0o644)
	rep4, err := Install(ws)
	if err != nil {
		t.Fatalf("install 4: %v", err)
	}
	if len(rep4.Preserved) != 1 || rep4.Preserved[0] != "gm-charter" {
		t.Errorf("customização real deveria preservar gm-charter: %v", rep4.Preserved)
	}
}
