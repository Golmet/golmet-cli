// Package container — contrato da estrutura canônica do CONTAINER de
// organização (FORMATO §2): fonte única consumida pelo doctor (checklist
// de hierarquia) e pelo lint (L7/L1 do container). Read-only — nunca muta;
// quem escreve é o `golmet init` (scaffold).
package container

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Golmet/golmet-cli/internal/core/frontmatter"
)

// CanonicalDirs — pastas canônicas da raiz do container (meetings são
// checkpoints transversais; migrations é a camada de migração/histórico
// da org). docs/ NÃO é canônico (M18): docs de produto vão no produto,
// docs do kit vão no repo do kit, históricos em migrations/.
var CanonicalDirs = []string{"meetings", "migrations"}

// CanonicalFiles — arquivos canônicos da raiz (INDEX.md é a navegação do
// container — IMP-012; CONSTITUTION.md são os princípios da org).
var CanonicalFiles = []string{"CONSTITUTION.md", "INDEX.md"}

// metaNames — arquivos meta/ferramentais da raiz que NÃO são artefato
// org-level (silenciosos). Declarativa e versionada aqui — ampliar com
// julgamento, nunca silenciosamente.
var metaNames = map[string]bool{
	".gitignore": true,
	"README.md":  true,
	"LICENSE":    true,
}

// Config lê e parseia o golmet.yaml do dir. Segundo retorno = o yaml
// declara container: true. Erro = ausente/ilegível (chamador decide).
func Config(dir string) (map[string]frontmatter.Value, bool, error) {
	data, err := os.ReadFile(filepath.Join(dir, "golmet.yaml"))
	if err != nil {
		return nil, false, err
	}
	fields, perr := frontmatter.ParseBlock(string(data))
	if perr != nil {
		return nil, false, perr
	}
	return fields, fields["container"].Str == "true", nil
}

// Products devolve a lista products: declarada no golmet.yaml do container.
func Products(fields map[string]frontmatter.Value) []string {
	return fields["products"].Arr
}

// Structure é o retrato determinístico da hierarquia real do container
// versus o declarado — consumido pelo doctor (checklist) e pelo lint (L7).
type Structure struct {
	MissingProducts  []string // declarados no products: que NÃO existem no disco (stale)
	UndeclaredDirs   []string // dirs de produto na raiz FORA do products: (workspace órfão ou pasta perdida)
	LooseMDs         []string // *.md org-level soltos na raiz (lugar canônico: docs/)
	MissingCanonical []string // arquivos canônicos ausentes (CONSTITUTION.md / INDEX.md)
	DeclaredPresent  []string // declarados e presentes (para a mensagem OK)
}

// Inspect varre a raiz do container e devolve o retrato. products é a
// lista declarada (o chamador já parseou o golmet.yaml). Dir oculto
// (prefixo ".") é meta de ferramenta — sem opinião. Arquivo não-md
// fora do canônico é ferramental — sem opinião (mesma política do
// L3/config: só docs org-level apinam).
func Inspect(root string, products []string) *Structure {
	st := &Structure{}
	declared := map[string]bool{}
	for _, p := range products {
		p = filepath.ToSlash(filepath.Clean(p))
		declared[p] = true
		if isDir(filepath.Join(root, filepath.FromSlash(p))) {
			st.DeclaredPresent = append(st.DeclaredPresent, p)
		} else {
			st.MissingProducts = append(st.MissingProducts, p)
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return st
	}
	canonicalDir := map[string]bool{}
	for _, d := range CanonicalDirs {
		canonicalDir[d] = true
	}
	canonicalFile := map[string]bool{}
	for _, f := range CanonicalFiles {
		canonicalFile[f] = true
	}
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() {
			if strings.HasPrefix(n, ".") || canonicalDir[n] || declared[n] {
				continue
			}
			st.UndeclaredDirs = append(st.UndeclaredDirs, n)
			continue
		}
		if metaNames[n] || canonicalFile[n] {
			continue
		}
		if strings.HasSuffix(strings.ToLower(n), ".md") {
			st.LooseMDs = append(st.LooseMDs, n)
		}
	}
	sort.Strings(st.MissingProducts)
	sort.Strings(st.UndeclaredDirs)
	sort.Strings(st.LooseMDs)
	sort.Strings(st.MissingCanonical)
	sort.Strings(st.DeclaredPresent)
	for _, f := range CanonicalFiles {
		if !fileExists(filepath.Join(root, f)) {
			st.MissingCanonical = append(st.MissingCanonical, f)
		}
	}
	sort.Strings(st.MissingCanonical)
	return st
}

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}
