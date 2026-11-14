// render.go — renderização do registry para o formato de cada engine.
// Port byte-fiel do protótipo render-agents.mjs (piloto F3): o golden test
// compara a saída daqui com a saída dele (arquivos ausentes são reportados
// por item — o lote não aborta, igual ao `console.error` do protótipo).
package content

import (
	"io/fs"
)

// Render projeta o registry para o engine (opencode | claude-code):
// agents → agents/<id>.md; skills → skills/<id>/SKILL.md. Itens sem corpo
// são pulados e reportados no retorno (padrão do piloto).
func Render(fsys fs.FS, reg *Registry, engine string) ([]Rendered, []ErrMissingBody) {
	var out []Rendered
	var missing []ErrMissingBody
	for _, it := range reg.Items {
		bodyPath := it.Str("body")
		data, err := fs.ReadFile(fsys, bodyPath)
		if err != nil {
			missing = append(missing, ErrMissingBody{ID: it.Str("id"), Path: bodyPath})
			continue
		}
		body := string(data)
		kind := kindOf(it)
		isAgent := kind == "primary" || kind == "subagent"
		var fm string
		if isAgent {
			if engine == "opencode" {
				fm = opencodeFrontMatter(it)
			} else {
				fm = claudeFrontMatter(it)
			}
		} else {
			fm = skillFrontMatter(it)
		}
		stripped := stripLeadBlocks(body)
		doc := fm + "\n" + rightTrimWS(stripped) + "\n"
		rel := "agents/" + it.Str("id") + ".md"
		if !isAgent {
			rel = "skills/" + it.Str("id") + "/SKILL.md"
		}
		out = append(out, Rendered{Rel: rel, Content: doc})
	}
	return out, missing
}

// FileSet devolve o conjunto de caminhos relativos renderizados (ordenado
// — convenção do kit para comparações e manifests).
func FileSet(rendered []Rendered) []string {
	set := make([]string, 0, len(rendered))
	for _, r := range rendered {
		set = append(set, r.Rel)
	}
	// ordenação lexicográfica estável (sort.Slice não é estável; strings já
	// são distintas — sort.Strings basta)
	for i := 1; i < len(set); i++ {
		for j := i; j > 0 && set[j] < set[j-1]; j-- {
			set[j], set[j-1] = set[j-1], set[j]
		}
	}
	return set
}
