// docs_sync_test.go — o guardião do sincronismo da REFERÊNCIA pública
// (M30): docs/reference/commands.md e docs/reference/tools.md são manuais
// sincronizados com a registry do binário (a decisão KISS da onda — sem
// comando gerador). Este teste quebra se um command nascer/sumir sem
// atualizar a referência, ou se uma tool gm_* mudar o número/mesmo.
// O lint L6 vigia os `gm-*` CITADOS nos docs; este teste vigia a COBERTURA.
package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const commandsRef = "../../docs/reference/commands.md"
const toolsRef = "../../docs/reference/tools.md"

func readDocRef(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("doc de referência ausente: %v", err)
	}
	return string(data)
}

// Todo command do binário (exceto `help` — o help do cobra é genérico)
// está citado em docs/reference/commands.md com a sua seção.
func TestDocsReferenceCoversAllCommands(t *testing.T) {
	doc := readDocRef(t, commandsRef)
	root := newRootCmd()
	for _, cmd := range root.Commands() {
		name := cmd.Name()
		if name == "help" {
			continue
		}
		if !strings.Contains(doc, "golmet "+name) {
			t.Errorf("command %q ausente de docs/reference/commands.md — a referência pública ficou para trás (M30: sincronismo manual vigiado)", name)
		}
	}
}

// As 13 tools gm_* do MCP estão citadas em docs/reference/tools.md.
// TestDocsReferenceCoversAllLintChecks trava a referência das camadas do
// lint aos checks implementados: docs/reference/lint.pt-br.md cita todos os
// IDs (L1, L3-L13 + L7-size) — camada nova sem doc = falha (a doc anda com
// o código, não atrás).
func TestDocsReferenceCoversAllLintChecks(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "docs", "reference", "lint.pt-br.md"))
	if err != nil {
		t.Fatalf("docs/reference/lint.pt-br.md ilegível: %v", err)
	}
	content := string(data)
	for _, id := range []string{"L1", "L3", "L4", "L5", "L6", "L7", "L7-size", "L8", "L9", "L10", "L11", "L12", "L13"} {
		if !strings.Contains(content, id) {
			t.Errorf("docs/reference/lint.pt-br.md não cita o check %s — a doc de referência anda com o código", id)
		}
	}
}

func TestDocsReferenceCoversAllMCPTools(t *testing.T) {
	doc := readDocRef(t, toolsRef)
	for _, tool := range []string{
		"gm_validate", "gm_doctor", "gm_lint", "gm_graph", "gm_status", "gm_next",
		"gm_index", "gm_update", "gm_pack_update", "gm_pack_publish",
		"gm_pack_list",
		"gm_eval", "gm_eval_reports", "gm_jev_classify", "gm_upgrade",
	} {
		if !strings.Contains(doc, tool) {
			t.Errorf("tool %q ausente de docs/reference/tools.md", tool)
		}
	}
}

// ---- M33 — a árvore bilingue da doc ----

// Cada doc público tem o par .pt-br.md e AMBOS carregam o badge de
// alternância de idioma com os links cruzados (a convenção declarada no
// FORMAT §2: o base é EN; a tradução PT usa o sufixo .pt-br.md).
func TestDocsBilingualTree(t *testing.T) {
	dirs := []string{"concepts", "reference", "getting-started"}
	for _, dir := range dirs {
		entries, err := os.ReadDir(filepath.Join("..", "..", "docs", dir))
		if err != nil {
			t.Fatalf("docs/%s ilegível: %v", dir, err)
		}
		bases := map[string]bool{}
		for _, e := range entries {
			name := e.Name()
			if !strings.HasSuffix(name, ".md") {
				continue
			}
			data, rerr := os.ReadFile(filepath.Join("..", "..", "docs", dir, name))
			if rerr != nil {
				t.Fatalf("docs/%s/%s ilegível: %v", dir, name, rerr)
			}
			content := string(data)
			badge := "<p align=\"center\">"
			if !strings.Contains(content, badge) {
				t.Errorf("docs/%s/%s sem o badge de idioma no topo", dir, name)
			}
			if strings.HasSuffix(name, ".pt-br.md") {
				base := strings.TrimSuffix(name, ".pt-br.md")
				bases[base] = true
				// o badge PT linka o README raiz e o README.pt-br
				if !strings.Contains(content, "../../README.md") ||
					!strings.Contains(content, "../../README.pt-br.md") {
					t.Errorf("docs/%s/%s: o badge PT deveria linkar os 2 READMEs raiz", dir, name)
				}
			} else {
				// o badge EN linka o par PT
				if !strings.Contains(content, "href=\""+strings.TrimSuffix(name, ".md")+".pt-br.md\"") {
					t.Errorf("docs/%s/%s: o badge EN deveria linkar o par PT", dir, name)
				}
			}
		}
		for base := range bases {
			if _, err := os.Stat(filepath.Join("..", "..", "docs", dir, base+".md")); err != nil {
				t.Errorf("docs/%s/%s.pt-br.md sem o par EN base (%s.md)", dir, base, base)
			}
		}
	}
	// a raiz docs/ e o README raiz também são pares bilingues
	for _, pair := range []struct{ base, dir string }{
		{"recommended", "docs"},
		{"README", "."},
	} {
		for _, f := range []string{pair.base + ".md", pair.base + ".pt-br.md"} {
			p := filepath.Join("..", "..", pair.dir, f)
			if _, err := os.Stat(p); err != nil {
				t.Errorf("par bilingue ausente: %s", p)
				continue
			}
			data, rerr := os.ReadFile(p)
			if rerr != nil {
				t.Fatal(rerr)
			}
			if !strings.Contains(string(data), "Português (Brasil)") && pair.base != "README" {
				t.Errorf("%s sem o badge de idioma", f)
			}
		}
	}
}
