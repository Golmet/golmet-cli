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
		switch d {
		case "docs":
			structWarn(d, "docs/ não é estrutura canônica do container",
				"docs de produto vão no produto; docs do kit vão no repo do kit; históricos em .golmet/migrations/")
		case "meetings", "migrations":
			structWarn(d, d+"/ na raiz do container — migrou para a zona de trabalho do kit (M19)",
				"mova para .golmet/"+d+"/ (a zona é gitignored — trabalho do kit, não conteúdo da org)")
		default:
			structWarn(d, "dir de produto na raiz fora do products: declarado",
				"declare no golmet.yaml do container ('golmet init <produto> --into <container>') ou mova para .golmet/ (trabalho/histórico) se não for produto")
		}
	}
	for _, f := range st.MissingCanonical {
		structWarn(f, "arquivo canônico do container ausente (INDEX.md é a navegação — IMP-012)",
			"crie o arquivo (o stub vem do 'golmet init --org' — idempotente, nada existente é sobrescrito)")
	}
	for _, f := range st.LooseMDs {
		structWarn(f, "artefato org-level na raiz do container",
			"mova para .golmet/ (zona de trabalho do kit — gitignored na org) ou declare allowlist")
	}
}

// lintL1Container — referências mortas na CAMADA DE NAVEGAÇÃO do
// container: canônicos da raiz (CONSTITUTION.md, INDEX.md, README.md) +
// `docs/` legado (transição M18). A zona de trabalho `.golmet/` (M19)
// NÃO é varrida — skip declarado: trabalho temporário do kit
// (meetings/, migrations/), não camada de navegação; os `*.md`
// org-level SOLTOS na raiz também ficam fora (o L7 já aponta a
// movimentação para `.golmet/`). Candidatos resolvem contra [dir do
// arquivo, raiz do container, CONFIG ROOT do engine] — o config entra
// quando conhecido (--engine/--home): docs do container citam o runtime
// do agente (ex.: `profile/client-profile.md` existe em
// `~/.config/opencode/profile/`) e não são refs mortas. A isenção de
// store é POR CANDIDATO: refs ancoradas no container legado (`docs/`)
// são verificáveis; as demais stores de workspace de produto (incl.
// `meetings/`, `migrations/` — agora dentro da zona) permanecem isentas.
func lintL1Container(rep *Report, ws, configRoot string) {
	var rels []string
	for _, f := range []string{"CONSTITUTION.md", "INDEX.md", "README.md"} {
		if fileExists(filepath.Join(ws, f)) {
			rels = append(rels, f)
		}
	}
	rels = append(rels, mdFilesRecursive(ws, "docs")...)
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
// quando NÃO é ancorada no container: o prefixo `docs/` (legado M18,
// transição) aponta para arquivos próprios → verificado; as demais
// stores de workspace de produto (`meetings/`, `migrations/` — agora
// dentro da zona `.golmet/` — e as demais) são citadas por design →
// isentas (mesma política do kit).
func containerAnchoredStoreExempt(cand string) bool {
	seg := cand
	if i := strings.Index(seg, "/"); i >= 0 {
		seg = seg[:i]
	}
	if seg == "docs" {
		return false // ancorada no container legado — verificável
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
