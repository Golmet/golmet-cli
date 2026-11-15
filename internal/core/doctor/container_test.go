// Testes do checklist de container no doctor (FORMATO §2): hierarquia
// products ↔ disco, base canônica e artefatos org-level soltos; os checks
// de workspace-produto (índice/scan/manifest/capacidades) não rodam na
// raiz do container.
package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func containerFixture(t *testing.T) string {
	t.Helper()
	ws := t.TempDir()
	os.WriteFile(filepath.Join(ws, "golmet.yaml"), []byte("format: 1\ncontainer: true\nproducts: [ProdutoA]\n"), 0o644)
	os.WriteFile(filepath.Join(ws, "CONSTITUTION.md"), []byte("# Constituição\n"), 0o644)
	os.WriteFile(filepath.Join(ws, "INDEX.md"), []byte("# Índice\n"), 0o644)
	os.MkdirAll(filepath.Join(ws, "docs"), 0o755)
	os.MkdirAll(filepath.Join(ws, "meetings"), 0o755)
	os.MkdirAll(filepath.Join(ws, "migrations"), 0o755)
	os.MkdirAll(filepath.Join(ws, "ProdutoA"), 0o755)
	return ws
}

func checksOf(rep *Report, item string) []Check {
	var out []Check
	for _, c := range rep.Checks {
		if c.Item == item {
			out = append(out, c)
		}
	}
	return out
}

func TestDoctorContainerConformant(t *testing.T) {
	ws := containerFixture(t)
	rep, err := Run(ws, Options{NoNetwork: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	prod := checksOf(rep, "container-products")
	if len(prod) != 1 || prod[0].Status != OK {
		t.Fatalf("container-products = %+v, want 1 OK (1 produto declarado e presente)", prod)
	}
	for _, c := range rep.Checks {
		if c.Status == Fail {
			t.Errorf("container conforme não deveria ter fail: %+v", c)
		}
		// checks de workspace-produto não rodam na raiz do container
		switch c.Item {
		case "indice", "worklist-s2", "manifest", "capacidades":
			t.Errorf("check de workspace-produto na raiz do container: %+v", c)
		}
	}
}

func TestDoctorContainerHierarchyFindings(t *testing.T) {
	ws := containerFixture(t)
	// declarado ausente no disco
	os.WriteFile(filepath.Join(ws, "golmet.yaml"), []byte("format: 1\ncontainer: true\nproducts: [ProdutoA, ProdutoFantasma]\n"), 0o644)
	// produto não declarado + artefato org-level solto
	os.MkdirAll(filepath.Join(ws, "ProdutoSolto"), 0o755)
	os.WriteFile(filepath.Join(ws, "AUDIT_2026-01-01.md"), []byte("relatório\n"), 0o644)

	rep, err := Run(ws, Options{NoNetwork: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	prod := checksOf(rep, "container-products")
	var hasFail, hasWarn bool
	for _, c := range prod {
		if c.Status == Fail && strings.Contains(c.Detail, "ProdutoFantasma") {
			hasFail = true
		}
		if c.Status == Warn && strings.Contains(c.Detail, "ProdutoSolto") {
			hasWarn = true
		}
	}
	if !hasFail || !hasWarn {
		t.Fatalf("container-products = %+v, want fail (ProdutoFantasma) + warn (ProdutoSolto)", prod)
	}
	raiz := checksOf(rep, "container-raiz")
	if len(raiz) != 1 || raiz[0].Status != Warn || !strings.Contains(raiz[0].Fix, "docs/") {
		t.Fatalf("container-raiz = %+v, want 1 warn apontando docs/", raiz)
	}
}

func TestDoctorProductWorkspaceUnaffected(t *testing.T) {
	// golmet.yaml SEM container: true → fluxo de workspace-produto intacto
	ws := t.TempDir()
	os.WriteFile(filepath.Join(ws, "golmet.yaml"), []byte("format: 1\n"), 0o644)
	os.WriteFile(filepath.Join(ws, "PROJECT_CHARTER.md"), []byte("---\nstatus: draft\n---\n# Charter\n"), 0o644)
	rep, err := Run(ws, Options{NoNetwork: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, c := range rep.Checks {
		if strings.HasPrefix(c.Item, "container-") {
			t.Errorf("check de container em workspace-produto: %+v", c)
		}
	}
	if checksOf(rep, "indice") == nil {
		t.Errorf("workspace-produto deveria rodar o check de índice")
	}
}
