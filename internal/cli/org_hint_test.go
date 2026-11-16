package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Golmet/golmet-cli/internal/core/orgs"
)

// ---- M33 — o hint do versionamento na saída do init --org ----

// Criação nova de org: a saída orienta o passo Git do operador (o kit nunca
// roda git init por conta própria).
func TestInitOrgVersioningHintOnCreation(t *testing.T) {
	home := isolatedHome(t)
	base := inTempDir(t)

	out, err := runCmd(t, "init", "MinhaOrg", "--org")
	if err != nil {
		t.Fatalf("init --org: %v\n%s", err, out)
	}
	if !strings.Contains(out, "repositório de versionamento") ||
		!strings.Contains(out, "git init") ||
		!strings.Contains(out, ".gitignore já protege a zona de trabalho") {
		t.Errorf("hint do versionamento ausente da saída:\n%s", out)
	}
	// a org foi registrada no catálogo (o fluxo normal segue)
	c, _ := orgs.Load(home)
	if len(c.Organizations) != 1 || c.Organizations[0].Name != "MinhaOrg" {
		t.Errorf("catálogo após o init: %+v", c.Organizations)
	}
	if _, serr := os.Stat(filepath.Join(base, "MinhaOrg", ".gitignore")); serr != nil {
		t.Errorf(".gitignore da org ausente: %v", serr)
	}
}

// ADOÇÃO (golmet.yaml existente): o hint NÃO aparece — a org já existe e
// pode já estar versionada.
func TestInitOrgVersioningHintNotOnAdoption(t *testing.T) {
	isolatedHome(t)
	base := inTempDir(t)

	// container existente (a adoção do tutorial)
	existing := filepath.Join(base, "Adotada")
	if err := os.MkdirAll(existing, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(existing, "golmet.yaml"),
		[]byte("format: 1\ncontainer: true\nversioning: org-repo\nproducts: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, "init", "Adotada", "--org")
	if err != nil {
		t.Fatalf("init --org adoção: %v\n%s", err, out)
	}
	if strings.Contains(out, "repositório de versionamento") {
		t.Errorf("o hint de criação não deveria aparecer na adoção:\n%s", out)
	}
}
