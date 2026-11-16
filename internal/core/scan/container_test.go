package scan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Golmet/golmet-cli/internal/core"
)

// ---- contexto container (FORMAT §2.3 — M21) ----

func makeContainer(t *testing.T, products string) string {
	t.Helper()
	root := t.TempDir()
	cfg := "format: 1\ncontainer: true\nproducts: " + products + "\n"
	if err := os.WriteFile(filepath.Join(root, "golmet.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func makeProduct(t *testing.T, root, name string, adopted bool) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if adopted {
		if err := os.WriteFile(filepath.Join(dir, "golmet.yaml"), []byte("format: 1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// Declaração íntegra: 1 adotado + 1 não-adotado → exit 0 (nenhum error),
// relatório com os contadores e mensagem de adoção.
func TestValidateContainerDeclarationIntact(t *testing.T) {
	root := makeContainer(t, "[alfa, beta]")
	makeProduct(t, root, "alfa", true)  // golmet.yaml próprio = adotado
	makeProduct(t, root, "beta", false) // só dir = não-adotado (estado legítimo)

	res, err := Run(root, Options{WriteIndex: true})
	if err != nil {
		t.Fatalf("container íntegro não deveria falhar: %v", err)
	}
	if res.Context != ContextContainer {
		t.Errorf("contexto = %q, want %q", res.Context, ContextContainer)
	}
	if res.Container == nil {
		t.Fatalf("retrato do container ausente do Result")
	}
	c := res.Container
	if c.Total != 2 || len(c.Adopted) != 1 || len(c.NotAdopted) != 1 {
		t.Errorf("contadores = total %d, adotados %v, não-adotados %v — want 2/1/1", c.Total, c.Adopted, c.NotAdopted)
	}
	if c.Adopted[0] != "alfa" || c.NotAdopted[0] != "beta" {
		t.Errorf("adoção invertida: %v / %v", c.Adopted, c.NotAdopted)
	}
	if len(res.Errors) != 0 || len(res.Warnings) != 0 {
		t.Errorf("declaração íntegra não deveria ter findings: %v / %v", res.Errors, res.Warnings)
	}
	// container NÃO escreve golmet.index.json (não é workspace-produto)
	if res.IndexWritten || res.Index != nil {
		t.Errorf("container não deveria compor índice de estados")
	}
	if _, serr := os.Stat(filepath.Join(root, "golmet.index.json")); serr == nil {
		t.Errorf("golmet.index.json não deveria existir na raiz do container")
	}
	s := c.Summary()
	for _, want := range []string{"2 produto(s)", "1 adotado(s)", "1 não-adotado(s)", "golmet init <produto> --into <container>"} {
		if !strings.Contains(s, want) {
			t.Errorf("summary sem %q: %s", want, s)
		}
	}
}

// Produto declarado sem dir = finding C1 (declaração stale) — exit 1 no CLI.
func TestValidateContainerStaleProduct(t *testing.T) {
	root := makeContainer(t, "[alfa, fantasma]")
	makeProduct(t, root, "alfa", false)

	res, err := Run(root, Options{})
	if err != nil {
		t.Fatalf("erro de uso (exit 2) não é falha de declaração: %v", err)
	}
	if len(res.Errors) != 1 {
		t.Fatalf("want 1 finding, got %d: %v", len(res.Errors), res.Errors)
	}
	e := res.Errors[0]
	if e.Rule != "C1" || e.File != "golmet.yaml" || e.Severity != core.SeverityError {
		t.Errorf("finding inesperado: %+v", e)
	}
	if !strings.Contains(e.Message, "fantasma") {
		t.Errorf("finding não cita o produto stale: %s", e.Message)
	}
	if e.Fix == "" {
		t.Errorf("finding sem remediação")
	}
}

// Dir dado sem golmet.yaml: NEM workspace NEM container — erro claro com
// os dois caminhos de init (exit 2 no CLI).
func TestValidateContainerNoneContext(t *testing.T) {
	res, err := Run(t.TempDir(), Options{})
	if err == nil {
		t.Fatalf("dir sem golmet.yaml deveria falhar (exit 2); res = %+v", res)
	}
	for _, want := range []string{"não é workspace nem container", "golmet init", "--into <container>"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("mensagem de erro sem %q: %v", want, err)
		}
	}
}

// Workspace-produto intacto: golmet.yaml SEM container → gramática
// integral, contexto workspace-produto e sem retrato de container.
func TestValidateWorkspaceContextUnchanged(t *testing.T) {
	ws := t.TempDir()
	if err := os.WriteFile(filepath.Join(ws, "golmet.yaml"), []byte("format: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Run(ws, Options{})
	if err != nil {
		t.Fatalf("workspace mínimo deveria validar: %v", err)
	}
	if res.Context != ContextWorkspace {
		t.Errorf("contexto = %q, want %q", res.Context, ContextWorkspace)
	}
	if res.Container != nil {
		t.Errorf("workspace-produto não deveria carregar retrato de container")
	}
}
