// render.go — renderização do registry para o formato de cada engine.
// Port byte-fiel do protótipo render-agents.mjs (piloto F3): o golden test
// compara a saída daqui com a saída dele (arquivos ausentes são reportados
// por item — o lote não aborta, igual ao `console.error` do protótipo).
package content

import (
	"errors"
	"io/fs"
	"strings"
)

// Render projeta o registry para o engine (opencode | claude-code):
// agents → agents/<id>.md; skills → skills/<id>/SKILL.md. Itens sem corpo
// são pulados e reportados no retorno (padrão do piloto). Ao final, anexa
// as rules canônicas da fonte (rules/*.md — camada core de comportamento,
// ex.: rules/context7.md) byte-fiéis, normalizados para o canônico LF.
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
	renderRules(fsys, &out, &missing)
	renderTemplates(fsys, &out, &missing)
	return out, missing
}

// renderRules anexa as rules canônicas da fonte (coredata/rules — *.md e
// *.yaml, ex.: context7.md, stack-templates.yaml, templates de perfil) à
// saída do render com Rel **profile/<nome>** — o destino no CONFIG é o
// perfil de operação do agente (`profile/`); a FONTE continua coredata/rules
// (o conteúdo é rules canônicas; só o destino muda — M13). Byte-fiéis após a
// normalização de resquícios de editor (BOM/EOL → canônico LF, a mesma do
// materialize). Diretório ausente = nada a anexar (fontes externas antigas
// sem rules/ seguem renderizando); arquivo ilegível vira item "sem corpo"
// (mesma reportagem dos bodies do registry).
func renderRules(fsys fs.FS, out *[]Rendered, missing *[]ErrMissingBody) {
	entries, err := fs.ReadDir(fsys, "rules")
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || (!strings.HasSuffix(e.Name(), ".md") && !strings.HasSuffix(e.Name(), ".yaml")) {
			continue
		}
		// FONTE = rules/<nome> (coredata); DESTINO no config = profile/<nome>
		src := "rules/" + e.Name()
		rel := "profile/" + e.Name()
		data, rerr := fs.ReadFile(fsys, src)
		if rerr != nil {
			*missing = append(*missing, ErrMissingBody{ID: rel, Path: src})
			continue
		}
		doc := rightTrimWS(string(Normalize(data)))
		*out = append(*out, Rendered{Rel: rel, Content: doc + "\n"})
	}
}

// renderTemplates anexa assets-raiz não-item do registry (hoje: o
// AGENTS.template.md — template versionado e neutro do config do engine,
// IMP-105) à saída do render, byte-fiéis após a normalização de resquícios
// de editor (BOM/EOL → canônico LF). Diretório/arquivo ausente = skip
// (cores externos antigos seguem renderizando); leitura com outro erro
// vira item "sem corpo" (mesma reportagem dos bodies).
func renderTemplates(fsys fs.FS, out *[]Rendered, missing *[]ErrMissingBody) {
	data, err := fs.ReadFile(fsys, "templates/AGENTS.template.md")
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			*missing = append(*missing, ErrMissingBody{ID: "AGENTS.template.md", Path: "templates/AGENTS.template.md"})
		}
		return
	}
	doc := rightTrimWS(string(Normalize(data)))
	*out = append(*out, Rendered{Rel: "AGENTS.template.md", Content: doc + "\n"})
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
