package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Golmet/golmet-cli/internal/core/update"
)

// ---- M29 — os 2 modelos de versionamento de organização ----

// init --org: golmet.yaml com versioning: org-repo (o default do init
// --org — o caso da TechMinds) + o .gitignore da organização criado.
func TestInitOrgArtifacts(t *testing.T) {
	base := t.TempDir()
	org := filepath.Join(base, "Org")
	if _, err := Run(base, "Org", Options{Org: true}); err != nil {
		t.Fatalf("Run org: %v", err)
	}
	cfg := readFile(t, filepath.Join(org, "golmet.yaml"))
	if !strings.Contains(cfg, "versioning: org-repo") {
		t.Errorf("golmet.yaml do container sem versioning: org-repo:\n%s", cfg)
	}
	gi := readFile(t, filepath.Join(org, ".gitignore"))
	if !strings.Contains(gi, ".golmet/*") {
		t.Errorf(".gitignore da org sem a zona de trabalho:\n%s", gi)
	}
	if strings.Contains(gi, "!.golmet/skills/") {
		t.Errorf("sem update a org não tem zona de kit — exceção não deveria estar:\n%s", gi)
	}
}

// init --org idempotente: .gitignore existente NÃO-vazio nunca é
// sobrescrito (o padrão do scaffold).
func TestInitOrgGitignoreIdempotent(t *testing.T) {
	base := t.TempDir()
	org := filepath.Join(base, "Org")
	os.MkdirAll(org, 0o755)
	sentinel := "# gitignore do operador\n*.segredo\n"
	os.WriteFile(filepath.Join(org, ".gitignore"), []byte(sentinel), 0o644)
	if _, err := Run(base, "Org", Options{Org: true}); err != nil {
		t.Fatalf("Run org: %v", err)
	}
	if got := readFile(t, filepath.Join(org, ".gitignore")); got != sentinel {
		t.Errorf(".gitignore do operador sobrescrito:\n%s", got)
	}
}

// Container org-repo → o produto nasce SEM .golmet/ própria (a zona do
// kit é a da ORG) — e SEM .gitignore próprio; o golmet.index.json segue.
func TestInitIntoOrgRepoNoProductZone(t *testing.T) {
	tmp := t.TempDir()
	org := filepath.Join(tmp, "Org")
	if _, err := Run(tmp, "Org", Options{Org: true}); err != nil {
		t.Fatalf("Run org: %v", err)
	}
	rep, err := Run(org, "MeuApp", Options{Into: org})
	if err != nil {
		t.Fatalf("Run into: %v", err)
	}
	if rep.VersioningModel != "org-repo" || !rep.ProductZoneSaved {
		t.Errorf("report = modelo %q zoneSaved %v", rep.VersioningModel, rep.ProductZoneSaved)
	}
	if _, serr := os.Stat(filepath.Join(org, "MeuApp", ".golmet")); !os.IsNotExist(serr) {
		t.Errorf("produto org-repo NÃO deveria ter .golmet/ própria")
	}
	if _, serr := os.Stat(filepath.Join(org, "MeuApp", ".gitignore")); !os.IsNotExist(serr) {
		t.Errorf("produto org-repo NÃO deveria ter .gitignore próprio (o da org vale)")
	}
	if _, serr := os.Stat(filepath.Join(org, "MeuApp", "golmet.index.json")); serr != nil {
		t.Errorf("o golmet.index.json é do workspace — deveria existir: %v", serr)
	}
	// o essencial do produto segue (charter, entidades)
	if !fileExists(filepath.Join(org, "MeuApp", "PROJECT_CHARTER.md")) {
		t.Errorf("charter ausente no produto org-repo")
	}
}

// Container produtos-repo → como HOJE (regressão): produto com
// .golmet/manifest.json (o kit do repo DELE).
func TestInitIntoProductsRepoKeepsZone(t *testing.T) {
	tmp := t.TempDir()
	org := filepath.Join(tmp, "Org")
	if _, err := Run(tmp, "Org", Options{Org: true}); err != nil {
		t.Fatalf("Run org: %v", err)
	}
	// o operador reclassifica o modelo no golmet.yaml (produtos-repo)
	p := filepath.Join(org, "golmet.yaml")
	data, _ := os.ReadFile(p)
	os.WriteFile(p, []byte(strings.Replace(string(data), "versioning: org-repo", "versioning: produtos-repo", 1)), 0o644)

	rep, err := Run(org, "MeuApp", Options{Into: org})
	if err != nil {
		t.Fatalf("Run into: %v", err)
	}
	if rep.VersioningModel != "produtos-repo" || rep.ProductZoneSaved {
		t.Errorf("report = modelo %q zoneSaved %v", rep.VersioningModel, rep.ProductZoneSaved)
	}
	if _, serr := os.Stat(filepath.Join(org, "MeuApp", ".golmet", "manifest.json")); serr != nil {
		t.Errorf("produto produtos-repo deveria ter a .golmet/ própria: %v", serr)
	}
}

// Container SEM o campo (criado antes da M29) → produtos-repo (default
// retroativo).
func TestInitIntoLegacyContainerDefaultsToProductsRepo(t *testing.T) {
	tmp := t.TempDir()
	org := filepath.Join(tmp, "Org")
	os.MkdirAll(org, 0o755)
	os.WriteFile(filepath.Join(org, "golmet.yaml"), []byte("format: 1\ncontainer: true\nproducts: []\n"), 0o644)
	rep, err := Run(org, "MeuApp", Options{Into: org})
	if err != nil {
		t.Fatalf("Run into: %v", err)
	}
	if rep.VersioningModel != "produtos-repo" {
		t.Errorf("container sem o campo = %q, want produtos-repo (default retroativo)", rep.VersioningModel)
	}
	if _, serr := os.Stat(filepath.Join(org, "MeuApp", ".golmet", "manifest.json")); serr != nil {
		t.Errorf("produto do container legado deveria ter a .golmet/: %v", serr)
	}
}

// O update essencial na raiz da org (org-repo) instala .golmet/skills/ e
// o .gitignore ganha a EXCEÇÃO !.golmet/skills/.
func TestUpdateOnOrgRepoAddsKitZoneException(t *testing.T) {
	tmp := t.TempDir()
	org := filepath.Join(tmp, "Org")
	if _, err := Run(tmp, "Org", Options{Org: true}); err != nil {
		t.Fatalf("Run org: %v", err)
	}
	rep, err := update.Install(org)
	if err != nil {
		t.Fatalf("update essencial na org: %v", err)
	}
	if !rep.GitignoreException {
		t.Errorf("o update deveria reportar a exceção do gitignore: %+v", rep)
	}
	gi := readFile(t, filepath.Join(org, ".gitignore"))
	if !strings.Contains(gi, "!.golmet/skills/") {
		t.Errorf(".gitignore da org sem a exceção da zona de kit:\n%s", gi)
	}
	if _, serr := os.Stat(filepath.Join(org, ".golmet", "skills", "gm-charter", "SKILL.md")); serr != nil {
		t.Errorf("zona de kit da org não instalada: %v", serr)
	}
	// re-run idempotente (a exceção já presente = intacta)
	before := readFile(t, filepath.Join(org, ".gitignore"))
	if _, err := update.Install(org); err != nil {
		t.Fatalf("update re-run: %v", err)
	}
	if got := readFile(t, filepath.Join(org, ".gitignore")); got != before {
		t.Errorf("re-run alterou o .gitignore:\nantes:\n%s\ndepois:\n%s", before, got)
	}
}
