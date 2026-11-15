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

// CanonicalDirs — pastas canônicas de CONTEÚDO na raiz do container.
// VAZIA desde M19: o container não tem pastas de conteúdo canônicas na
// raiz — o trabalho do kit (checkpoints em meetings/, histórico em
// migrations/) vive na zona `.golmet/` (gitignored); dirs conhecidos que
// ainda aparecem na raiz (docs/, meetings/, migrations/) caem em
// UndeclaredDirs e ganham mensagem própria de migração no doctor/lint.
var CanonicalDirs = []string{}

// WorkZoneDir — a zona de trabalho do kit na raiz do container (M19):
// mesma pasta do kit em qualquer contexto, semântica CONTEXTUAL (FORMATO
// §2.3) — no workspace-produto é VERSIONADA (DR-008, kit do repositório);
// no CONFIG do engine é estado do kit (manifest/personal.json); no
// CONTAINER da organização é a zona de TRABALHO não-versionada
// (gitignored): checkpoints (meetings/) e histórico de processo
// (migrations/) são trabalho do kit, não conteúdo da organização.
var WorkZoneDir = ".golmet"

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
	UndeclaredDirs   []string // dirs na raiz FORA do products: (workspace órfão, pasta perdida ou dir legado pós-M19)
	LooseMDs         []string // *.md org-level soltos na raiz (destino: zona de trabalho .golmet/)
	MissingCanonical []string // arquivos canônicos ausentes (CONSTITUTION.md / INDEX.md)
	DeclaredPresent  []string // declarados e presentes (para a mensagem OK)
	HasWorkZone      bool     // zona de trabalho .golmet/ presente na raiz (M19)
}

// Inspect varre a raiz do container e devolve o retrato. products é a
// lista declarada (o chamador já parseou o golmet.yaml). Dir oculto
// (prefixo ".") é meta de ferramenta — sem opinião, EXCETO a zona de
// trabalho do kit (`.golmet/`, M19), detectada em HasWorkZone. Arquivo
// não-md fora do canônico é ferramental — sem opinião (mesma política do
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
			if n == WorkZoneDir {
				st.HasWorkZone = true // zona de trabalho do kit (gitignored na org)
				continue
			}
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
