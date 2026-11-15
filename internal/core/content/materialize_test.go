package content

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	golmet "github.com/Golmet/golmet-cli"
)

func TestMaterializeWritesFullAndManifest(t *testing.T) {
	root := t.TempDir()
	core, err := fs.Sub(golmet.CoreDataFS(), "coredata")
	if err != nil {
		t.Fatal(err)
	}
	data, err := fs.ReadFile(core, "registry.yaml")
	if err != nil {
		t.Fatal(err)
	}
	rep, err := Materialize(core, ParseRegistry(string(data)), MaterializeOptions{
		ConfigRoot: root, Engine: "opencode", KitVersion: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Total != 129 || len(rep.Written) != 129 || len(rep.Conflicts) != 0 {
		t.Fatalf("materialize: total=%d written=%d conflicts=%d", rep.Total, len(rep.Written), len(rep.Conflicts))
	}
	if _, err := os.Stat(filepath.Join(root, "agents", "gm-dev.md")); err != nil {
		t.Fatalf("agent não materializado: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "skills", "gm-reverse", "SKILL.md")); err != nil {
		t.Fatalf("skill não materializada: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".golmet", "manifest.json")); err != nil {
		t.Fatalf("manifest não materializado: %v", err)
	}
}

func TestMaterializeDryRunDoesNotWriteFilesOrManifest(t *testing.T) {
	root := t.TempDir()
	core, err := fs.Sub(golmet.CoreDataFS(), "coredata")
	if err != nil {
		t.Fatal(err)
	}
	data, err := fs.ReadFile(core, "registry.yaml")
	if err != nil {
		t.Fatal(err)
	}
	rep, err := Materialize(core, ParseRegistry(string(data)), MaterializeOptions{
		ConfigRoot: root, Engine: "opencode", KitVersion: "test", DryRun: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Written) != rep.Total || len(rep.Preserved) != 0 || len(rep.Conflicts) != 0 {
		t.Fatalf("plano dry-run inesperado: total=%d written=%d preserved=%d conflicts=%d", rep.Total, len(rep.Written), len(rep.Preserved), len(rep.Conflicts))
	}
	if _, err := os.Stat(filepath.Join(root, ".golmet")); !os.IsNotExist(err) {
		t.Fatalf("dry-run criou manifest/zona: %v", err)
	}
}

func TestMaterializePreservesPackOverlapAndManifestOwnership(t *testing.T) {
	root := t.TempDir()
	core, _ := fs.Sub(golmet.CoreDataFS(), "coredata")
	data, _ := fs.ReadFile(core, "registry.yaml")
	reg := ParseRegistry(string(data))
	if _, err := Materialize(core, reg, MaterializeOptions{ConfigRoot: root, Engine: "opencode", KitVersion: "test"}); err != nil {
		t.Fatal(err)
	}
	overlap := filepath.Join(root, "skills", "gm-reverse", "SKILL.md")
	packData := []byte("---\nname: gm-reverse\n---\npack-owned\n")
	if _, err := ApplyOverlay(root, []OverlayFile{{Rel: "skills/gm-reverse/SKILL.md", Data: packData}}, OverlayOptions{
		Layer: "pack:community", Source: "local@sha",
	}); err != nil {
		t.Fatal(err)
	}
	rep, err := Materialize(core, reg, MaterializeOptions{ConfigRoot: root, Engine: "opencode", KitVersion: "next", Force: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Conflicts) != 1 || rep.Conflicts[0] != "skills/gm-reverse/SKILL.md" {
		t.Fatalf("overlap deveria ser conflito: %#v", rep.Conflicts)
	}
	got, _ := os.ReadFile(overlap)
	if string(got) != string(packData) {
		t.Fatal("update FULL sobrescreveu arquivo gerido pelo pack")
	}
	manifest, err := readContentManifest(ContentManifestPath(root, ""))
	if err != nil {
		t.Fatal(err)
	}
	foundPackEntry := false
	for _, entry := range manifest.Files {
		if entry.File == "skills/gm-reverse/SKILL.md" {
			foundPackEntry = true
			if entry.Layer != "pack:community" {
				t.Fatalf("manifest perdeu a proveniência do pack: %#v", entry)
			}
		}
	}
	if !foundPackEntry {
		t.Fatal("manifest não preservou a entrada sobreposta do pack")
	}
}

func TestMaterializeRejectsUnsafeRenderedPath(t *testing.T) {
	reg := &Registry{Items: []*Item{{
		scalars: map[string]string{"id": "../escape", "body": "body.md", "description": "x"},
		lists:   map[string][]string{}, bools: map[string]bool{},
	}}}
	_, err := Materialize(fstest.MapFS{"body.md": &fstest.MapFile{Data: []byte("body")}}, reg, MaterializeOptions{
		ConfigRoot: t.TempDir(), Engine: "opencode", KitVersion: "test",
	})
	if err == nil || !strings.Contains(err.Error(), "fora do escopo") {
		t.Fatalf("path traversal deveria falhar fechado: %v", err)
	}
}

func TestMaterializeRejectsSymlinkedDestination(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "skills", "gm-x")); err != nil {
		t.Skipf("symlink indisponível neste ambiente: %v", err)
	}
	reg := &Registry{Items: []*Item{{
		scalars: map[string]string{"id": "gm-x", "body": "body.md", "description": "x"},
		lists:   map[string][]string{}, bools: map[string]bool{},
	}}}
	_, err := Materialize(fstest.MapFS{"body.md": &fstest.MapFile{Data: []byte("body")}}, reg, MaterializeOptions{
		ConfigRoot: root, Engine: "opencode", KitVersion: "test",
	})
	if err == nil || !strings.Contains(err.Error(), "escapa") {
		t.Fatalf("destino symlinkado deveria falhar fechado: %v", err)
	}
}

func TestMaterializePreservesLocalLayerWithoutForce(t *testing.T) {
	root := t.TempDir()
	core, _ := fs.Sub(golmet.CoreDataFS(), "coredata")
	data, _ := fs.ReadFile(core, "registry.yaml")
	reg := ParseRegistry(string(data))
	if _, err := Materialize(core, reg, MaterializeOptions{ConfigRoot: root, Engine: "opencode", KitVersion: "test"}); err != nil {
		t.Fatal(err)
	}
	// registra um arquivo como camada local (customização do operador)
	if _, err := ApplyOverlay(root, []OverlayFile{{Rel: "skills/gm-reverse/SKILL.md", Data: []byte("---\nname: gm-reverse\n---\n# local tweak\n")}}, OverlayOptions{
		Layer: "local", Source: "operator",
	}); err != nil {
		t.Fatal(err)
	}
	rep, err := Materialize(core, reg, MaterializeOptions{ConfigRoot: root, Engine: "opencode", KitVersion: "next"})
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "skills", "gm-reverse", "SKILL.md")
	got, _ := os.ReadFile(target)
	if !strings.Contains(string(got), "local tweak") {
		t.Fatal("arquivo da camada local foi sobrescrito sem --force")
	}
	if len(rep.Conflicts) == 0 {
		t.Fatalf("camada local deveria ser reportada como conflito: %#v", rep)
	}
	// --force é a exceção explícita
	if _, err := Materialize(core, reg, MaterializeOptions{ConfigRoot: root, Engine: "opencode", KitVersion: "next", Force: true}); err != nil {
		t.Fatal(err)
	}
	got, _ = os.ReadFile(target)
	if strings.Contains(string(got), "local tweak") {
		t.Fatal("--force não substituiu camada local")
	}
}

func TestApplyOverlayPreservesLocalLayerWithoutForce(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "skills", "shared", "SKILL.md")
	if _, err := ApplyOverlay(root, []OverlayFile{{Rel: "skills/shared/SKILL.md", Data: []byte("operator")}}, OverlayOptions{
		Layer: "local", Source: "operator",
	}); err != nil {
		t.Fatal(err)
	}
	rep, err := ApplyOverlay(root, []OverlayFile{{Rel: "skills/shared/SKILL.md", Data: []byte("pack")}}, OverlayOptions{
		Layer: "pack:community", Source: "local@sha",
	})
	if err == nil && len(rep.Conflicts) == 0 {
		t.Fatalf("pack deveria conflitar com camada local sem force: %#v", rep)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "operator" {
		t.Fatal("pack sobrescreveu camada local sem --force")
	}
	if _, err := ApplyOverlay(root, []OverlayFile{{Rel: "skills/shared/SKILL.md", Data: []byte("pack")}}, OverlayOptions{
		Layer: "pack:community", Source: "local@sha", Force: true,
	}); err != nil {
		t.Fatal(err)
	}
	got, _ = os.ReadFile(path)
	if string(got) != "pack" {
		t.Fatal("--force não aplicou o pack sobre camada local")
	}
}

func TestMaterializePreservesLocalOverrideAndReportsConflict(t *testing.T) {
	root := t.TempDir()
	core, _ := fs.Sub(golmet.CoreDataFS(), "coredata")
	data, _ := fs.ReadFile(core, "registry.yaml")
	reg := ParseRegistry(string(data))
	if _, err := Materialize(core, reg, MaterializeOptions{ConfigRoot: root, Engine: "opencode", KitVersion: "test"}); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "skills", "gm-reverse", "SKILL.md")
	local := "---\nname: gm-reverse\n---\n# custom local override\n"
	if err := os.WriteFile(target, []byte(local), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := Materialize(core, reg, MaterializeOptions{ConfigRoot: root, Engine: "opencode", KitVersion: "next"})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Conflicts) != 1 || rep.Conflicts[0] != "skills/gm-reverse/SKILL.md" {
		t.Fatalf("conflito inesperado: %#v", rep.Conflicts)
	}
	got, _ := os.ReadFile(target)
	if !strings.Contains(string(got), "custom local override") {
		t.Fatal("override local foi sobrescrito")
	}
}

func TestMaterializeForceReplacesLocalOverride(t *testing.T) {
	root := t.TempDir()
	core, _ := fs.Sub(golmet.CoreDataFS(), "coredata")
	data, _ := fs.ReadFile(core, "registry.yaml")
	reg := ParseRegistry(string(data))
	if _, err := Materialize(core, reg, MaterializeOptions{ConfigRoot: root, Engine: "opencode", KitVersion: "test"}); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "skills", "gm-reverse", "SKILL.md")
	if err := os.WriteFile(target, []byte("local"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Materialize(core, reg, MaterializeOptions{ConfigRoot: root, Engine: "opencode", KitVersion: "next", Force: true}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(target)
	if string(got) == "local" {
		t.Fatal("--force não substituiu override")
	}
}

func TestCoreUpdatePreservesPackLayerInManifest(t *testing.T) {
	root := t.TempDir()
	core, _ := fs.Sub(golmet.CoreDataFS(), "coredata")
	data, _ := fs.ReadFile(core, "registry.yaml")
	reg := ParseRegistry(string(data))
	if _, err := Materialize(core, reg, MaterializeOptions{ConfigRoot: root, Engine: "opencode", KitVersion: "test"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyOverlay(root, []OverlayFile{{Rel: "skills/community/SKILL.md", Data: []byte("pack")}}, OverlayOptions{Layer: "pack:community", Source: "local@sha"}); err != nil {
		t.Fatal(err)
	}
	if _, err := Materialize(core, reg, MaterializeOptions{ConfigRoot: root, Engine: "opencode", KitVersion: "next"}); err != nil {
		t.Fatal(err)
	}
	manifest, err := readContentManifest(ContentManifestPath(root, ""))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := manifest.Layers["pack:community"]; !ok {
		t.Fatalf("update do core apagou layer do pack: %#v", manifest.Layers)
	}
	if _, err := os.Stat(filepath.Join(root, "skills", "community", "SKILL.md")); err != nil {
		t.Fatalf("arquivo do pack desapareceu: %v", err)
	}
}

func TestOverlayDryRunDoesNotWriteFilesOrManifest(t *testing.T) {
	root := t.TempDir()
	rep, err := ApplyOverlay(root, []OverlayFile{{Rel: "skills/community/SKILL.md", Data: []byte("pack")}}, OverlayOptions{
		Layer: "pack:community", Source: "local@sha", DryRun: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Written) != 1 || rep.Written[0] != "skills/community/SKILL.md" {
		t.Fatalf("plano overlay inesperado: %#v", rep)
	}
	if _, err := os.Stat(filepath.Join(root, "skills")); !os.IsNotExist(err) {
		t.Fatalf("dry-run overlay criou arquivo/zona: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".golmet", "manifest.json")); !os.IsNotExist(err) {
		t.Fatalf("dry-run overlay criou manifest: %v", err)
	}
}

func TestOverlayNoOverrideAllowsOwnPackUpdateButNotOtherLayerForce(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "skills", "shared", "SKILL.md")
	if _, err := ApplyOverlay(root, []OverlayFile{{Rel: "skills/shared/SKILL.md", Data: []byte("v1")}}, OverlayOptions{
		Layer: "pack:community", Source: "local@1", NoOverride: true,
	}); err != nil {
		t.Fatal(err)
	}
	rep, err := ApplyOverlay(root, []OverlayFile{{Rel: "skills/shared/SKILL.md", Data: []byte("v2")}}, OverlayOptions{
		Layer: "pack:community", Source: "local@2", NoOverride: true,
	})
	if err != nil || len(rep.Written) != 1 || len(rep.Conflicts) != 0 {
		t.Fatalf("próprio pack deveria atualizar: rep=%#v err=%v", rep, err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "v2" {
		t.Fatalf("atualização do próprio pack não aplicada: %q", got)
	}
	rep, err = ApplyOverlay(root, []OverlayFile{{Rel: "skills/shared/SKILL.md", Data: []byte("core")}}, OverlayOptions{
		Layer: "core", Source: "embedded", NoOverride: true, Force: true,
	})
	if err != nil || len(rep.Conflicts) != 1 {
		t.Fatalf("force não deveria furar no-override: rep=%#v err=%v", rep, err)
	}
	got, _ = os.ReadFile(path)
	if string(got) != "v2" {
		t.Fatalf("force de outra camada sobrescreveu pack: %q", got)
	}
}
