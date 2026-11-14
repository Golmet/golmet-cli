package pack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseSpec(t *testing.T) {
	s, err := ParseSpec([]byte(`id: community-accessibility
version: 1.2.0
name: Accessibility helpers
requires-core: ">=0.1"
no-override: true
provides:
  - skills/gm-a11y/SKILL.md
  - agents/gm-a11y-reviewer.md
`))
	if err != nil {
		t.Fatal(err)
	}
	if s.ID != "community-accessibility" || s.Version != "1.2.0" || !s.NoOverride || len(s.Provides) != 2 {
		t.Fatalf("spec inesperada: %+v", s)
	}
}

func TestParseSpecRejectsTraversal(t *testing.T) {
	for _, path := range []string{"../escape.md", "skills/./escape.md", "skills/a/../escape.md", `skills\..\escape.md`} {
		if _, err := ParseSpec([]byte("id: bad\nprovides: [" + path + "]\n")); err == nil {
			t.Fatalf("provides inseguro deveria falhar: %q", path)
		}
	}
}

func TestParseSpecRequiresVersionAndName(t *testing.T) {
	if _, err := ParseSpec([]byte("id: sem-version\nprovides: [skills/x/SKILL.md]\n")); err == nil || !strings.Contains(err.Error(), "version") {
		t.Fatalf("pack sem version deveria falhar: %v", err)
	}
	if _, err := ParseSpec([]byte("id: sem-name\nversion: 1.0.0\nprovides: [skills/x/SKILL.md]\n")); err == nil || !strings.Contains(err.Error(), "name") {
		t.Fatalf("pack sem name deveria falhar: %v", err)
	}
}

func TestAddRejectsIncompatibleRequiresCore(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "skills", "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pack.yaml"), []byte("id: old\nversion: 1.0.0\nname: Old pack\nrequires-core: >=9.0\nprovides: [skills/x/SKILL.md]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "skills", "x", "SKILL.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := AddWithOptions(t.TempDir(), t.TempDir(), "opencode", root, false, true)
	if err == nil || !strings.Contains(err.Error(), "requires-core") {
		t.Fatalf("requires-core incompatível deveria falhar: %v", err)
	}
	if err := ValidateRequiresCore(">=0.1", "0.3.0"); err != nil {
		t.Fatalf("requisito compatível rejeitado: %v", err)
	}
	if err := ValidateRequiresCore("mystery", "0.3.0"); err == nil {
		t.Fatal("requisito desconhecido deveria falhar fechado")
	}
}

func TestAddRejectsSymlinkEscape(t *testing.T) {
	packRoot := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret.md")
	if err := os.WriteFile(outside, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(packRoot, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packRoot, "pack.yaml"), []byte("id: escape\nversion: 1.0.0\nname: Escape pack\nprovides: [skills/link.md]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(packRoot, "skills", "link.md")); err != nil {
		t.Skipf("symlink indisponível neste ambiente: %v", err)
	}
	_, err := AddWithOptions(t.TempDir(), t.TempDir(), "opencode", packRoot, false, true)
	if err == nil || !strings.Contains(err.Error(), "escapa") {
		t.Fatalf("symlink escape deveria falhar: %v", err)
	}
}
