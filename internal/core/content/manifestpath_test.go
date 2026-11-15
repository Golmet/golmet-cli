package content

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	golmet "github.com/Golmet/golmet-cli"
)

// Override de ManifestPath: o config root da zona do workspace usa nome
// distinto (manifest-content.json) — o default .golmet/manifest.json NÃO é
// criado nem lido, preservando o manifest BASE do update.Install.
func TestMaterializeManifestPathOverride(t *testing.T) {
	root := t.TempDir()
	core, err := fs.Sub(golmet.CoreDataFS(), "coredata")
	if err != nil {
		t.Fatal(err)
	}
	data, err := fs.ReadFile(core, "registry.yaml")
	if err != nil {
		t.Fatal(err)
	}
	reg := ParseRegistry(string(data))
	override := filepath.Join(root, "manifest-content.json")
	rep, err := Materialize(core, reg, MaterializeOptions{
		ConfigRoot: root, Engine: "opencode", KitVersion: "test",
		ManifestPath: override,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Total != 129 || len(rep.Written) != 129 {
		t.Fatalf("materialize override: total=%d written=%d", rep.Total, len(rep.Written))
	}
	if _, err := os.Stat(override); err != nil {
		t.Fatalf("manifest de conteúdo não escrito no override: %v", err)
	}
	// default NÃO criado (é território do manifest BASE na zona do workspace)
	if _, err := os.Stat(filepath.Join(root, ".golmet", "manifest.json")); !os.IsNotExist(err) {
		t.Errorf("override criou o manifest default (.golmet/manifest.json): %v", err)
	}

	// re-run lê o estado prévio DO OVERRIDE (129 preservadas = prev visto)
	rep2, err := Materialize(core, reg, MaterializeOptions{
		ConfigRoot: root, Engine: "opencode", KitVersion: "test",
		ManifestPath: override,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep2.Conflicts) != 0 {
		t.Errorf("re-run com override deveria ser sem conflitos: %#v", rep2.Conflicts)
	}
	if len(rep2.Written) != 0 {
		t.Errorf("re-run deveria preservar tudo (prev lido do override): written=%d", len(rep2.Written))
	}

	// e o caminho default continua funcionando sem override
	root2 := t.TempDir()
	if _, err := Materialize(core, reg, MaterializeOptions{
		ConfigRoot: root2, Engine: "opencode", KitVersion: "test",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root2, ".golmet", "manifest.json")); err != nil {
		t.Errorf("default path quebrou: %v", err)
	}
}

// default com override vazio = caminho canônico .golmet/manifest.json.
func TestContentManifestPathDefaultAndOverride(t *testing.T) {
	root := `C:\tmp\engine-config`
	if got, want := ContentManifestPath(root, ""), filepath.Join(root, ".golmet", "manifest.json"); got != want {
		t.Errorf("default = %q, want %q", got, want)
	}
	if got, want := ContentManifestPath(root, `C:\tmp\ws\.golmet\manifest-content.json`), `C:\tmp\ws\.golmet\manifest-content.json`; got != want {
		t.Errorf("override = %q, want %q", got, want)
	}
}
