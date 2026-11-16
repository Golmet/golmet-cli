package container

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Golmet/golmet-cli/internal/core/frontmatter"
)

// VersioningOf — org-repo declarado; ausente/outro = produtos-repo (o
// default retroativo dos containers anteriores a M29).
func TestVersioningOf(t *testing.T) {
	fields, err := frontmatter.ParseBlock("format: 1\ncontainer: true\nversioning: org-repo\n")
	if err != nil {
		t.Fatal(err)
	}
	if got := VersioningOf(fields); got != VersioningOrgRepo {
		t.Errorf("org-repo declarado = %q, want org-repo", got)
	}
	fields2, _ := frontmatter.ParseBlock("format: 1\ncontainer: true\n")
	if got := VersioningOf(fields2); got != VersioningProductsRepo {
		t.Errorf("sem o campo = %q, want produtos-repo (default retroativo)", got)
	}
	fields3, _ := frontmatter.ParseBlock("format: 1\ncontainer: true\nversioning: produtos-repo\n")
	if got := VersioningOf(fields3); got != VersioningProductsRepo {
		t.Errorf("produtos-repo declarado = %q", got)
	}
}

// O .gitignore da org: a zona de trabalho é `.golmet/*` (o CONTEÚDO — a
// negação git não re-inclui sob um dir excluído) + SO/editor.
func TestOrgGitignoreShape(t *testing.T) {
	c := OrgGitignore()
	if !strings.Contains(c, ".golmet/*") {
		t.Errorf("o gitignore da org deveria usar .golmet/* (conteúdo):\n%s", c)
	}
	if strings.Contains(c, "!.golmet/skills/") {
		t.Errorf("SEM update a org não tem zona de kit — a exceção não deveria estar:\n%s", c)
	}
	for _, basic := range []string{"Thumbs.db", "desktop.ini", "*.swp", ".idea/"} {
		if !strings.Contains(c, basic) {
			t.Errorf("básico de SO/editor ausente: %q", basic)
		}
	}
}

// EnsureKitZoneException — ausente → cria com a exceção; presente →
// idempotente; dir-level `.golmet/` (containers pré-M29) é convertido
// cirurgicamente para `.golmet/*`.
func TestEnsureKitZoneException(t *testing.T) {
	// sem .gitignore → cria com conteúdo padrão + exceção
	root := t.TempDir()
	if err := EnsureKitZoneException(root); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(root, ".gitignore"))
	c := string(data)
	if !strings.Contains(c, "!.golmet/skills/") || !strings.Contains(c, ".golmet/*") {
		t.Errorf("gitignore criado sem o conteúdo completo:\n%s", c)
	}

	// já presente → no-op (o conteúdo do operador intacto)
	custom := "# meus ignorados\n*.tmp\n!.golmet/skills/\n"
	os.WriteFile(filepath.Join(root, ".gitignore"), []byte(custom), 0o644)
	if err := EnsureKitZoneException(root); err != nil {
		t.Fatalf("ensure idempotente: %v", err)
	}
	if got, _ := os.ReadFile(filepath.Join(root, ".gitignore")); string(got) != custom {
		t.Errorf("gitignore existente foi alterado:\n%s", got)
	}

	// dir-level `.golmet/` (pré-M29) → convertido para .golmet/* + exceção
	old := "# meu gitignore\n.golmet/\n*.log\n"
	os.WriteFile(filepath.Join(root, ".gitignore"), []byte(old), 0o644)
	if err := EnsureKitZoneException(root); err != nil {
		t.Fatalf("ensure com dir-level: %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(root, ".gitignore"))
	g := string(got)
	if strings.Contains(g, "\n.golmet/\n") || strings.HasSuffix(g, ".golmet/\n") {
		t.Errorf("o dir-level .golmet/ deveria ter sido convertido:\n%s", g)
	}
	if !strings.Contains(g, ".golmet/*") || !strings.Contains(g, "!.golmet/skills/") {
		t.Errorf("conversão incompleta:\n%s", g)
	}
	// o resto do gitignore do operador preservado
	if !strings.Contains(g, "# meu gitignore") || !strings.Contains(g, "*.log") {
		t.Errorf("conteúdo do operador perdido na conversão:\n%s", g)
	}
}
