package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Golmet/golmet-cli/internal/core/indexstate"
	"github.com/Golmet/golmet-cli/internal/core/scan"
)

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestInitProductStructure(t *testing.T) {
	tmp := t.TempDir()
	rep, err := Run(tmp, "MeuApp", Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	root := filepath.Join(tmp, "MeuApp")
	if rep.Root != root || rep.Name != "MeuApp" {
		t.Errorf("rep.Root/Name = %s/%s", rep.Root, rep.Name)
	}

	// §2 — arquivos do layout
	for _, rel := range []string{
		"golmet.yaml", "PROJECT_CHARTER.md", "REQUIREMENTS.md",
		"CHANGE_REQUESTS.md", "DECISIONS.md", "reverse/CANDIDATES.md",
		"specs/README.md", ".golmet/manifest.json", "golmet.index.json",
	} {
		if !fileExists(filepath.Join(root, filepath.FromSlash(rel))) {
			t.Errorf("arquivo ausente: %s", rel)
		}
	}
	for _, dir := range []string{"analysis", "delivery", ".golmet/skills"} {
		if st, err := os.Stat(filepath.Join(root, dir)); err != nil || !st.IsDir() {
			t.Errorf("pasta ausente: %s", dir)
		}
	}
	// §3 — golmet.yaml com format: 1
	if got := readFile(t, filepath.Join(root, "golmet.yaml")); got != "format: 1\n" {
		t.Errorf("golmet.yaml = %q", got)
	}
	// §7.1 — charter draft com §1 de repos vazia
	charter := readFile(t, filepath.Join(root, "PROJECT_CHARTER.md"))
	if !strings.Contains(charter, "status: draft") || !strings.Contains(charter, "## 1. Repos") ||
		!strings.Contains(charter, "|---|---|---|") {
		t.Errorf("charter sem front matter draft ou tabela de repos vazia:\n%s", charter)
	}
	// zona do kit — manifest vazio válido
	manifest := readFile(t, filepath.Join(root, ".golmet", "manifest.json"))
	if !strings.Contains(manifest, "\"skills\": []") || !strings.Contains(manifest, indexstate.KitVersion) {
		t.Errorf("manifest inválido:\n%s", manifest)
	}
	// baseline do índice parseável com o nome do workspace
	idx, err := indexstate.Parse([]byte(readFile(t, filepath.Join(root, "golmet.index.json"))))
	if err != nil || idx.Workspace != "MeuApp" {
		t.Errorf("baseline inválido: %v (workspace=%q)", err, idx.Workspace)
	}
	// workspace scaffoldado é CONFORME por construção (zero erros)
	res, rerr := scan.Run(root, scan.Options{Tolerant: true})
	if rerr != nil {
		t.Fatalf("scan do scaffold: %v", rerr)
	}
	if len(res.Errors) != 0 {
		for _, e := range res.Errors {
			t.Errorf("scaffold gerou erro hard: %+v", e)
		}
	}
}

func TestInitIdempotentNeverOverwrites(t *testing.T) {
	tmp := t.TempDir()
	if _, err := Run(tmp, "MeuApp", Options{}); err != nil {
		t.Fatalf("primeira Run: %v", err)
	}
	root := filepath.Join(tmp, "MeuApp")

	// usuário edita o charter — init NUNCA sobrescreve
	charter := filepath.Join(root, "PROJECT_CHARTER.md")
	os.WriteFile(charter, []byte("---\nstatus: confirmed\n---\n# Charter editado\n"), 0o644)
	// e o validate reescreveu o índice — também preservado
	os.WriteFile(filepath.Join(root, "golmet.index.json"), []byte("{\"format\":1,\"generated_at\":\"x\",\"workspace\":\"MeuApp\",\"entities\":{},\"counts\":{}}\n"), 0o644)

	rep, err := Run(tmp, "MeuApp", Options{})
	if err != nil {
		t.Fatalf("segunda Run: %v", err)
	}
	if rep.Skipped == 0 || rep.Created != 0 {
		t.Errorf("segunda Run deveria pular tudo: criados=%d pulados=%d", rep.Created, rep.Skipped)
	}
	if got := readFile(t, charter); !strings.Contains(got, "editado") {
		t.Errorf("charter do usuário foi sobrescrito:\n%s", got)
	}
	idx := readFile(t, filepath.Join(root, "golmet.index.json"))
	if !strings.Contains(idx, "\"x\"") {
		t.Errorf("índice existente foi sobrescrito:\n%s", idx)
	}
}

func TestInitOrgContainer(t *testing.T) {
	tmp := t.TempDir()
	rep, err := Run(tmp, "MinhaOrg", Options{Org: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	root := filepath.Join(tmp, "MinhaOrg")
	if rep.Root != root {
		t.Errorf("rep.Root = %s", rep.Root)
	}
	// §3 — container: true + products: []
	cfg := readFile(t, filepath.Join(root, "golmet.yaml"))
	if !strings.Contains(cfg, "container: true") || !strings.Contains(cfg, "products: []") {
		t.Errorf("golmet.yaml do container inválido:\n%s", cfg)
	}
	for _, rel := range []string{"CONSTITUTION.md", "INDEX.md"} {
		if !fileExists(filepath.Join(root, rel)) {
			t.Errorf("stub ausente no container: %s", rel)
		}
	}
	// container NÃO é workspace — sem entidades
	if fileExists(filepath.Join(root, "REQUIREMENTS.md")) {
		t.Errorf("container não deveria ter REQUIREMENTS.md")
	}
}

func TestInitIntoContainer(t *testing.T) {
	tmp := t.TempDir()
	org := filepath.Join(tmp, "MinhaOrg")
	if _, err := Run(tmp, "MinhaOrg", Options{Org: true}); err != nil {
		t.Fatalf("Run org: %v", err)
	}

	rep, err := Run(org, "MeuApp", Options{Into: org})
	if err != nil {
		t.Fatalf("Run into: %v", err)
	}
	if !rep.ContainerUpdated {
		t.Errorf("container deveria ser atualizado")
	}
	// produto criado DENTRO do container
	if !fileExists(filepath.Join(org, "MeuApp", "golmet.yaml")) {
		t.Errorf("produto não criado dentro do container")
	}
	// products do container registra o produto
	cfg := readFile(t, filepath.Join(org, "golmet.yaml"))
	if !strings.Contains(cfg, "products: [MeuApp]") {
		t.Errorf("products não atualizado:\n%s", cfg)
	}

	// re-run idempotente — sem duplicar products
	if _, err := Run(org, "MeuApp", Options{Into: org}); err != nil {
		t.Fatalf("Run into 2: %v", err)
	}
	cfg = readFile(t, filepath.Join(org, "golmet.yaml"))
	if !strings.Contains(cfg, "products: [MeuApp]") || strings.Contains(cfg, "MeuApp, MeuApp") {
		t.Errorf("products duplicado no re-run:\n%s", cfg)
	}
}

func TestInitIntoNonContainerFails(t *testing.T) {
	tmp := t.TempDir()
	notContainer := filepath.Join(tmp, "ws")
	os.MkdirAll(notContainer, 0o755)
	os.WriteFile(filepath.Join(notContainer, "golmet.yaml"), []byte("format: 1\n"), 0o644)

	if _, err := Run(notContainer, "MeuApp", Options{Into: notContainer}); err == nil {
		t.Errorf("--into em não-container deveria falhar")
	}
	if fileExists(filepath.Join(notContainer, "MeuApp")) {
		t.Errorf("produto não deveria ser criado em não-container")
	}
}

func TestInitOrgAndIntoExclusive(t *testing.T) {
	tmp := t.TempDir()
	if _, err := Run(tmp, "X", Options{Org: true, Into: tmp}); err == nil {
		t.Errorf("--org e --into deveriam ser exclusivos")
	}
}

func TestInitIntoRequiresName(t *testing.T) {
	tmp := t.TempDir()
	if _, err := Run(tmp, "", Options{Into: tmp}); err == nil {
		t.Errorf("--into sem nome de produto deveria falhar")
	}
}
