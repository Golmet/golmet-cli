// Testes do contrato do container de organização (FORMATO §2): hierarquia
// products ↔ disco, base canônica e artefatos org-level soltos — fonte
// única consumida pelo doctor e pelo lint.
package container

import (
	"os"
	"path/filepath"
	"testing"
)

func mkContainer(t *testing.T) string {
	t.Helper()
	ws := t.TempDir()
	os.WriteFile(filepath.Join(ws, "golmet.yaml"), []byte("format: 1\ncontainer: true\nproducts: []\n"), 0o644)
	os.WriteFile(filepath.Join(ws, "CONSTITUTION.md"), []byte("# Constituição\n"), 0o644)
	os.WriteFile(filepath.Join(ws, "INDEX.md"), []byte("# Índice\n"), 0o644)
	// zona de trabalho do kit (M19): meetings/ + migrations/ DENTRO de .golmet/
	os.MkdirAll(filepath.Join(ws, ".golmet", "meetings"), 0o755)
	os.MkdirAll(filepath.Join(ws, ".golmet", "migrations"), 0o755)
	return ws
}

func TestInspectDeclaredAndPresent(t *testing.T) {
	ws := mkContainer(t)
	os.MkdirAll(filepath.Join(ws, "ProdutoA"), 0o755)
	os.WriteFile(filepath.Join(ws, "golmet.yaml"), []byte("format: 1\ncontainer: true\nproducts: [ProdutoA]\n"), 0o644)
	st := Inspect(ws, []string{"ProdutoA"})
	if len(st.MissingProducts) != 0 || len(st.UndeclaredDirs) != 0 || len(st.LooseMDs) != 0 || len(st.MissingCanonical) != 0 {
		t.Fatalf("container conforme deveria zerar o retrato: %+v", st)
	}
	if len(st.DeclaredPresent) != 1 || st.DeclaredPresent[0] != "ProdutoA" {
		t.Errorf("DeclaredPresent = %v, want [ProdutoA]", st.DeclaredPresent)
	}
}

func TestInspectMissingDeclaredProduct(t *testing.T) {
	ws := mkContainer(t)
	st := Inspect(ws, []string{"ProdutoFantasma"})
	if len(st.MissingProducts) != 1 || st.MissingProducts[0] != "ProdutoFantasma" {
		t.Fatalf("MissingProducts = %v, want [ProdutoFantasma]", st.MissingProducts)
	}
}

func TestInspectUndeclaredProductDir(t *testing.T) {
	ws := mkContainer(t)
	// dir de produto (com ou sem golmet.yaml interno) fora do products:
	os.MkdirAll(filepath.Join(ws, "ProdutoSolto"), 0o755)
	os.MkdirAll(filepath.Join(ws, ".oculto"), 0o755) // meta — sem opinião
	st := Inspect(ws, nil)
	if len(st.UndeclaredDirs) != 1 || st.UndeclaredDirs[0] != "ProdutoSolto" {
		t.Fatalf("UndeclaredDirs = %v, want [ProdutoSolto] (oculto isento)", st.UndeclaredDirs)
	}
}

func TestInspectDocsIsNotCanonical(t *testing.T) {
	// M18: docs/ deixou de ser canônico — cai em UndeclaredDirs para o
	// doctor/lint emitirem a mensagem própria de extinção
	ws := mkContainer(t)
	os.MkdirAll(filepath.Join(ws, "docs"), 0o755)
	st := Inspect(ws, nil)
	if len(st.UndeclaredDirs) != 1 || st.UndeclaredDirs[0] != "docs" {
		t.Fatalf("UndeclaredDirs = %v, want [docs] (não-canônico desde M18)", st.UndeclaredDirs)
	}
}

func TestInspectWorkZoneAndLegacyDirs(t *testing.T) {
	// M19: a zona .golmet/ é detectada (HasWorkZone); meetings/migrations
	// na RAIZ são legados → UndeclaredDirs (mensagens próprias downstream);
	// dentro da zona continuam legítimos (fixture já cria .golmet/{meetings,migrations})
	ws := mkContainer(t)
	os.MkdirAll(filepath.Join(ws, "meetings"), 0o755)
	os.MkdirAll(filepath.Join(ws, "migrations"), 0o755)
	st := Inspect(ws, nil)
	if !st.HasWorkZone {
		t.Fatalf("HasWorkZone = false, want true (.golmet/ no fixture)")
	}
	if len(st.UndeclaredDirs) != 2 || st.UndeclaredDirs[0] != "meetings" || st.UndeclaredDirs[1] != "migrations" {
		t.Fatalf("UndeclaredDirs = %v, want [meetings migrations] (legados na raiz, M19)", st.UndeclaredDirs)
	}
}

func TestInspectLooseOrgArtifactsAndCanonical(t *testing.T) {
	ws := mkContainer(t)
	os.WriteFile(filepath.Join(ws, "AUDIT_2026-01-01.md"), []byte("relatório solto\n"), 0o644)
	os.WriteFile(filepath.Join(ws, ".gitignore"), []byte("ignorado\n"), 0o644)
	os.WriteFile(filepath.Join(ws, "validacoes.yml"), []byte("ferramental\n"), 0o644) // não-md — sem opinião
	st := Inspect(ws, nil)
	if len(st.LooseMDs) != 1 || st.LooseMDs[0] != "AUDIT_2026-01-01.md" {
		t.Fatalf("LooseMDs = %v, want [AUDIT_2026-01-01.md] (.gitignore meta; não-md isento)", st.LooseMDs)
	}
	// base canônica ausente
	os.Remove(filepath.Join(ws, "INDEX.md"))
	st2 := Inspect(ws, nil)
	if len(st2.MissingCanonical) != 1 || st2.MissingCanonical[0] != "INDEX.md" {
		t.Fatalf("MissingCanonical = %v, want [INDEX.md]", st2.MissingCanonical)
	}
}

func TestConfigDetectsContainer(t *testing.T) {
	ws := mkContainer(t)
	_, isCont, err := Config(ws)
	if err != nil || !isCont {
		t.Fatalf("Config = container=%v err=%v, want container=true sem erro", isCont, err)
	}
	// workspace-produto (sem container:) NÃO é container
	os.WriteFile(filepath.Join(ws, "golmet.yaml"), []byte("format: 1\n"), 0o644)
	_, isCont, err = Config(ws)
	if err != nil || isCont {
		t.Fatalf("workspace-produto detectado como container: container=%v err=%v", isCont, err)
	}
}
