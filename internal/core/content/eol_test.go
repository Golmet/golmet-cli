package content

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	golmet "github.com/Golmet/golmet-cli"
)

// Regra EOL-tolerante: diferença só de CRLF/CR vs LF não é customização —
// a cura é rematerializar o arquivo com o canônico LF do kit, contando em
// Written, sem flag de conflito e sem perder conteúdo.

func crlf(b []byte) []byte { return []byte(strings.ReplaceAll(string(b), "\n", "\r\n")) }

func mustMaterialize(t *testing.T, root string) *MaterializeReport {
	t.Helper()
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
	return rep
}

func TestSameContentNormalized(t *testing.T) {
	lf := []byte("---\nname: gm-x\n---\n# título\n")
	if !SameContentNormalized(lf, crlf(lf)) {
		t.Errorf("CRLF vs LF deveriam ser iguais normalizados")
	}
	if !SameContentNormalized(lf, []byte("---\rname: gm-x\r---\r# título\r")) {
		t.Errorf("CR solitário vs LF deveriam ser iguais normalizados")
	}
	if SameContentNormalized(lf, []byte("---\nname: gm-x\n---\n# OUTRO\n")) {
		t.Errorf("conteúdo realmente diferente não deveria passar")
	}
	if !SameContentNormalized(lf, lf) {
		t.Errorf("bytes idênticos deveriam ser iguais")
	}
	// BOM UTF-8 inicial é resquício de editor — igual ao canônico sem BOM
	bom := append([]byte{0xEF, 0xBB, 0xBF}, lf...)
	if !SameContentNormalized(bom, lf) {
		t.Errorf("BOM vs sem BOM deveriam ser iguais normalizados")
	}
	if !SameContentNormalized(append([]byte{0xEF, 0xBB, 0xBF}, crlf(lf)...), lf) {
		t.Errorf("BOM + CRLF vs LF deveriam ser iguais normalizados")
	}
	if SameContentNormalized(lf, append([]byte{0xEF, 0xBB, 0xBF}, []byte("---\nname: gm-y\n---\n")...)) {
		t.Errorf("BOM não cura conteúdo realmente diferente")
	}
	// normalizeEOL (curado): disco CRLF vira o canônico LF byte-a-byte
	if got := normalizeEOL(crlf(lf)); string(got) != string(lf) {
		t.Errorf("normalizeEOL deveria devolver o canônico LF: %q", got)
	}
	if got := normalizeEOL([]byte("a\rb\n")); string(got) != "a\nb\n" { // CR solitário → LF
		t.Errorf("CR solitário deveria virar LF: %q", got)
	}
}

// (a) file no disco com o mesmo texto em CRLF + materialize sem force →
// Written (cura), disco vira LF, manifest adota o hash canônico.
func TestMaterializeCuresEOLAsWritten(t *testing.T) {
	root := t.TempDir()
	mustMaterialize(t, root)

	// semeia CRLF (mesmo texto, só EOL) num agent e numa skill
	targets := []string{"agents/gm-dev.md", "skills/gm-reverse/SKILL.md"}
	for _, rel := range targets {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, rel), crlf(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	rep := mustMaterialize(t, root)
	if len(rep.Conflicts) != 0 {
		t.Errorf("EOL-only não deveria gerar conflito: %#v", rep.Conflicts)
	}
	if len(rep.Written) != len(targets) {
		t.Fatalf("cura EOL deveria contar em Written (%d), written=%#v", len(targets), rep.Written)
	}
	for _, rel := range targets {
		found := false
		for _, w := range rep.Written {
			if w == rel {
				found = true
			}
		}
		if !found {
			t.Errorf("%s ausente de Written: %#v", rel, rep.Written)
		}
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "\r") {
			t.Errorf("%s deveria ter sido curado para LF", rel)
		}
	}

	// segunda execução: idempotente (0 escritas, tudo preservado)
	rep2 := mustMaterialize(t, root)
	if len(rep2.Written) != 0 || len(rep2.Conflicts) != 0 {
		t.Errorf("re-run deveria ser idempotente: written=%#v conflicts=%#v", rep2.Written, rep2.Conflicts)
	}
	// manifest adotou o hash canônico (arquivo em disco == hash registrado)
	manifest, err := readContentManifest(ContentManifestPath(root, ""))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range manifest.Files {
		if entry.File == "agents/gm-dev.md" {
			data, _ := os.ReadFile(filepath.Join(root, "agents/gm-dev.md"))
			if entry.SHA256 != sha(data) {
				t.Errorf("manifest não adotou o hash canônico do arquivo curado")
			}
		}
	}
}

// conteúdo REALMENTE diferente continua preservado/conflito (regressão).
func TestMaterializeRealDiffStillConflict(t *testing.T) {
	root := t.TempDir()
	mustMaterialize(t, root)
	target := filepath.Join(root, "agents", "gm-dev.md")
	data, _ := os.ReadFile(target)
	os.WriteFile(target, []byte(strings.ReplaceAll(string(data), "gm-dev", "gm-dev-custom")), 0o644)

	rep := mustMaterialize(t, root)
	if len(rep.Conflicts) != 1 || rep.Conflicts[0] != "agents/gm-dev.md" {
		t.Fatalf("customização real deveria conflitar: %#v", rep.Conflicts)
	}
	got, _ := os.ReadFile(target)
	if !strings.Contains(string(got), "gm-dev-custom") {
		t.Errorf("customização real foi perdida")
	}
}

// (b) overlay de pack sobre arquivo com o mesmo texto em CRLF → Written
// sem conflito; disco vira LF.
func TestApplyOverlayCuresEOL(t *testing.T) {
	root := t.TempDir()
	packData := []byte("---\nname: gm-reverse\n---\npack-owned\n")
	// arquivo pré-existente com o MESMO texto em CRLF (sem entrada de manifest)
	os.MkdirAll(filepath.Join(root, "skills", "gm-reverse"), 0o755)
	os.WriteFile(filepath.Join(root, "skills", "gm-reverse", "SKILL.md"), crlf(packData), 0o644)

	rep, err := ApplyOverlay(root, []OverlayFile{{Rel: "skills/gm-reverse/SKILL.md", Data: packData}}, OverlayOptions{
		Layer: "pack:community", Source: "local@sha",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Conflicts) != 0 {
		t.Errorf("EOL-only não deveria conflitar no overlay: %#v", rep.Conflicts)
	}
	if len(rep.Written) != 1 {
		t.Fatalf("overlay deveria curar como Written: %#v", rep)
	}
	got, _ := os.ReadFile(filepath.Join(root, "skills", "gm-reverse", "SKILL.md"))
	if strings.Contains(string(got), "\r") {
		t.Errorf("overlay deveria ter curado o EOL para LF")
	}
}
