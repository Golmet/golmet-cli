// Checks do CONTAINER de organização (FORMATO §2) — consumidos pelo Run
// quando o golmet.yaml declara container: true. Fonte da estrutura
// canônica: internal/core/container (compartilhada com o doctor). O
// workspace-produto (golmet.yaml sem container) segue pelos checks
// existentes como hoje.
package lint

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Golmet/golmet-cli/internal/core/container"
)

// lintL7Container — estrutura canônica do CONTAINER: products declarados
// existem no disco (stale = error), dirs de produto não declarados =
// warning (workspace órfão ou pasta perdida), base canônica
// (CONSTITUTION.md/INDEX.md) presente, artefato org-level solto na raiz =
// warning com remediação (docs/ é o hub).
func lintL7Container(rep *Report, ws string, st *container.Structure) {
	structErr := func(file, desc, fix string) {
		rep.Findings = append(rep.Findings, Finding{Check: "L7", Severity: "error", File: file, Description: desc, Fix: fix})
	}
	structWarn := func(file, desc, fix string) {
		rep.Findings = append(rep.Findings, Finding{Check: "L7", Severity: "warning", File: file, Description: desc, Fix: fix})
	}
	for _, p := range st.MissingProducts {
		structErr(p, "produto declarado no products: do container não existe no disco",
			"remova do products: do golmet.yaml (o produto não existe mais) ou restaure o diretório")
	}
	for _, d := range st.UndeclaredDirs {
		structWarn(d, "dir de produto na raiz fora do products: declarado",
			"declare no golmet.yaml do container ('golmet init <produto> --into <container>') ou mova para docs/ (hub) se não for produto")
	}
	for _, f := range st.MissingCanonical {
		structWarn(f, "arquivo canônico do container ausente (INDEX.md é a navegação — IMP-012)",
			"crie o arquivo (o stub vem do 'golmet init --org' — idempotente, nada existente é sobrescrito)")
	}
	for _, f := range st.LooseMDs {
		structWarn(f, "artefato org-level na raiz do container",
			"mova para docs/ (hub de docs da org) ou declare allowlist")
	}
}

// lintL1Container — referências mortas na CAMADA DE NAVEGAÇÃO do
// container: canônicos da raiz (CONSTITUTION.md, INDEX.md, README.md) +
// docs/ + meetings/ (recursivo). Os *.md org-level SOLTOS na raiz ficam
// FORA do L1 — são registros históricos e o L7 já aponta a movimentação
// para docs/ (varrê-los produz ruído de refs que descrevem o passado).
// Candidatos resolvem contra [dir do arquivo, raiz do container, CONFIG
// ROOT do engine] — o config entra como raiz quando conhecido
// (--engine/--home): docs do container citam o runtime do agente (ex.:
// `profile/client-profile.md` existe em ~/.config/opencode/profile/) e
// não são refs mortas. A isenção de store é POR CANDIDATO: refs
// ancoradas no container (docs/, meetings/, migrations/, canônicos) são
// arquivos próprios e verificáveis — as demais stores de workspace de
// produto permanecem isentas (mesma política do kit: citadas por design).
func lintL1Container(rep *Report, ws, configRoot string) {
	var rels []string
	for _, f := range []string{"CONSTITUTION.md", "INDEX.md", "README.md"} {
		if fileExists(filepath.Join(ws, f)) {
			rels = append(rels, f)
		}
	}
	rels = append(rels, mdFilesRecursive(ws, "docs")...)
	rels = append(rels, mdFilesRecursive(ws, "meetings")...)
	for _, rel := range rels {
		fpath := filepath.Join(ws, filepath.FromSlash(rel))
		roots := []string{filepath.Dir(fpath), ws}
		if configRoot != "" {
			roots = append(roots, configRoot)
		}
		checkMDRefs(rep, rel, fpath, roots, nil, containerAnchoredStoreExempt)
	}
}

// containerAnchoredStoreExempt — no container, uma ref de store é ISENTA
// quando NÃO é ancorada no container: prefixos canônicos (docs/,
// meetings/, migrations/ e os canônicos de raiz) apontam para arquivos
// PRÓPRIOS do container → verificados; as demais stores (delivery/,
// specs/, quality/…) são stores de workspace de produto citadas por
// design → isentas (mesma política do kit).
func containerAnchoredStoreExempt(cand string) bool {
	seg := cand
	if i := strings.Index(seg, "/"); i >= 0 {
		seg = seg[:i]
	}
	switch seg {
	case "docs", "meetings", "migrations", "CONSTITUTION.md", "INDEX.md":
		return false // ancorada no container — verificável
	}
	return true
}

// mdFilesRecursive — *.md sob root/sub em QUALQUER profundidade (salta
// dirs ocultos); rel ao root, slash.
func mdFilesRecursive(root, sub string) []string {
	dir := filepath.Join(root, filepath.FromSlash(sub))
	var out []string
	filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		if rel, rerr := filepath.Rel(root, p); rerr == nil {
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	return out
}
