// Golden test do render: a saída Go deve ser BYTE-A-BYTE igual ao golden
// neutralizado (o piloto render-agents.mjs/F3 foi regenerado já neutro —
// convenção BASE MeuApp/AppIrmao; nenhum nome real de projeto no repo).
package content

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	golmet "github.com/Golmet/golmet-cli"
)

func coreFS(t *testing.T) fs.FS {
	t.Helper()
	sub, err := fs.Sub(golmet.CoreDataFS(), "coredata")
	if err != nil {
		t.Fatalf("fs.Sub coredata: %v", err)
	}
	return sub
}

func TestRenderGoldenOpencodeByteEquality(t *testing.T) {
	regData, err := fs.ReadFile(coreFS(t), "registry.yaml")
	if err != nil {
		t.Fatalf("registry.yaml ilegível: %v", err)
	}
	reg := ParseRegistry(string(regData))
	if len(reg.Items) != 124 {
		t.Fatalf("registry: esperados 124 itens, got %d", len(reg.Items))
	}
	rendered, missing := Render(coreFS(t), reg, "opencode")
	if len(missing) != 0 {
		t.Fatalf("itens sem corpo: %v", missing)
	}
	if len(rendered) != 129 {
		t.Fatalf("renderizados: esperados 129 (124 itens + rules/templates), got %d", len(rendered))
	}

	goldenRoot := filepath.Join("testdata", "golden-opencode")

	// Regeneração de goldens: GOLMET_UPDATE_GOLDEN=1 sobrescreve os goldens
	// com o render corrente (use após editar coredata/sources e rever o diff).
	if os.Getenv("GOLMET_UPDATE_GOLDEN") == "1" {
		for _, r := range rendered {
			dst := filepath.Join(goldenRoot, filepath.FromSlash(r.Rel))
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				t.Fatalf("mkdir %s: %v", filepath.Dir(dst), err)
			}
			if err := os.WriteFile(dst, []byte(r.Content), 0o644); err != nil {
				t.Fatalf("gravar golden %s: %v", dst, err)
			}
		}
		t.Logf("goldens regenerados: %d arquivos", len(rendered))
		return
	}

	var goldenFiles []string
	err = filepath.WalkDir(goldenRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && (strings.HasSuffix(p, ".md") || strings.HasSuffix(p, ".yaml")) {
			rel, _ := filepath.Rel(goldenRoot, p)
			goldenFiles = append(goldenFiles, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("golden walk: %v", err)
	}
	sort.Strings(goldenFiles)

	got := FileSet(rendered)
	if strings.Join(got, "|") != strings.Join(goldenFiles, "|") {
		t.Fatalf("file set diverge:\n got:  %s\n want: %s", strings.Join(got, "|"), strings.Join(goldenFiles, "|"))
	}

	byRel := map[string]string{}
	for _, r := range rendered {
		byRel[r.Rel] = r.Content
	}
	for _, rel := range goldenFiles {
		want, err := os.ReadFile(filepath.Join(goldenRoot, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("golden ilegível %s: %v", rel, err)
		}
		gotBytes := byRel[rel]
		if gotBytes != string(want) {
			t.Errorf("BYTE DIVERGE: %s", rel)
		}
	}
}

func TestRenderClaudeCodeShape(t *testing.T) {
	regData, _ := fs.ReadFile(coreFS(t), "registry.yaml")
	reg := ParseRegistry(string(regData))
	rendered, missing := Render(coreFS(t), reg, "claude-code")
	if len(missing) != 0 || len(rendered) != 129 {
		t.Fatalf("claude-code: 129 esperados (124 itens + rules/templates), got %d (missing %v)", len(rendered), missing)
	}
	// fm = bloco entre o primeiro --- e o segundo; corpo pode mencionar "mode:" legitimately
	for _, r := range rendered {
		if !strings.HasPrefix(r.Rel, "agents/") {
			continue
		}
		end := strings.Index(r.Content[4:], "\n---")
		fm := r.Content
		if end >= 0 {
			fm = r.Content[:4+end]
		}
		if strings.Contains(fm, "mode:") {
			t.Errorf("agente claude-code não deve ter mode no fm: %s", r.Rel)
		}
	}
	// primary gm-req deve ter tools com Task
	for _, r := range rendered {
		if r.Rel == "agents/gm-req.md" && !strings.Contains(r.Content, "Task, AskUserQuestion") {
			t.Errorf("gm-req claude sem tools de primary")
		}
	}
}

func TestNeutralidadeConteudoEmbarcado(t *testing.T) {
	// gate: o conteúdo FULL embarcado não pode carregar identidade de org ou
	// de projetos de clientes. Os tokens proibidos são referenciados por
	// HASH (SHA-256 do token lowercase) — o repo nunca publica a lista em
	// texto plano. Atualize a lista e os hashes juntos, nunca em texto.
	forbidden := map[string]struct{}{
		"1a26b7dee6fc52c891e769ddab4fc79f4b6cce1b07a90fe95e53dcec05567f09": {},
		"3d9293c729b2ebd4dc37f1cb69b2cdb1ba8aefc200846cb3d65bb39a2ce2fa14": {},
		"b86c24fb6243540aa4b0c30411c3c3f521dcbaf597a61c4360382beae152acdb": {},
		"0b02488a99bd12c9974ea4470674b61476b5efe29bb9339fdad47a9e711676b8": {},
		"456f45fddc2e725c2948d03317e2262c2ec8e86745af28dc7b2221987b858b50": {},
		"670d47ccb9a6cdef180e87d03ef6a3b783c771ecac19fb76c3ba5551ab369da8": {},
		"fa07f2b1a17a72c9bd041936448df7e94b0536b63c7134f028b8a4d77a180677": {},
		"28fd515d1d0c50c6742fa30d86bb25f28b584abd0220c18f465f758160386aa2": {},
		"99bde068af2d49ed7fc8b8fa79abe13a6059e0db320bb73459fd96624bb4b33f": {},
		"5cca655ac072f27f6dc6a48c54a499e5f4ccd8d3eee65708bc557ceb3d6202f2": {},
		"6dee04b73e8d3336654bca37a2d58c001f0bfc572f4bf0d46c38fc08002d4da7": {},
	}
	bad := regexp.MustCompile(`[A-Za-z][A-Za-z0-9]*`)
	err := fs.WalkDir(golmet.CoreDataFS(), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || (!strings.HasSuffix(p, ".md") && !strings.HasSuffix(p, ".yaml")) {
			return err
		}
		data, err := fs.ReadFile(golmet.CoreDataFS(), p)
		if err != nil {
			return err
		}
		for _, tok := range bad.FindAllString(string(data), -1) {
			sum := sha256.Sum256([]byte(strings.ToLower(tok)))
			if _, hit := forbidden[hex.EncodeToString(sum[:])]; hit {
				t.Errorf("NEUTRALIDADE: %s contém token proibido (hash %s…)", p, hex.EncodeToString(sum[:])[:12])
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
