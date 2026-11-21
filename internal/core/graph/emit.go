// emit.go — emissão dos 3 artefatos (graph.json / knowledge-graph.html /
// knowledge-graph.md) — integral: strings montados em memória, escrita só no
// fim. JSON com EscapeHTML off + Attrs ordenados por inserção = paridade
// byte-a-byte com o extrator de referência (exceto gerado_em). HTML via
// template go:embed (port verbatim do .mjs validado — 6 bugs de interação
// corrigidos já no template).
package graph

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed template.html
var htmlTemplate string

// ---- Attrs: marshal/unmarshal com ordem de inserção ----

func marshalValue(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	b := buf.Bytes()
	return bytes.TrimRight(b, "\n"), nil
}

func (a *Attrs) MarshalJSON() ([]byte, error) {
	if a == nil {
		return []byte("{}"), nil
	}
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, p := range a.pairs {
		if p.v == omit {
			continue
		}
		if i > 0 && buf.Len() > 1 {
			buf.WriteByte(',')
		}
		kb, err := marshalValue(p.k)
		if err != nil {
			return nil, err
		}
		buf.Write(kb)
		buf.WriteByte(':')
		vb, err := marshalValue(p.v)
		if err != nil {
			return nil, err
		}
		buf.Write(vb)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

func (a *Attrs) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return fmt.Errorf("attrs: esperado objeto")
	}
	a.pairs = nil
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return err
		}
		key, _ := keyTok.(string)
		var v any
		if err := dec.Decode(&v); err != nil {
			return err
		}
		// normaliza json.Number para float64 apenas quando integral vira int64
		if n, ok := v.(json.Number); ok {
			if i, perr := n.Int64(); perr == nil {
				v = i
			} else {
				v, _ = n.Float64()
			}
		}
		a.pairs = append(a.pairs, attrPair{k: key, v: v})
	}
	_, err = dec.Token() // '}'
	return err
}

// marshalGraphCompact/Indent — JSON sem escape HTML (paridade com
// JSON.stringify do JS).
func marshalJSONIndent(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

func marshalJSONCompact(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// ---- from-json ----

func runFromJSON(opt Options) (*Graph, error) {
	data, err := os.ReadFile(opt.FromJSON)
	if err != nil {
		return nil, fmt.Errorf("ERRO: graph.json ilegível em %s: %w", opt.FromJSON, err)
	}
	var frozen Graph
	if err := json.Unmarshal(data, &frozen); err != nil {
		return nil, fmt.Errorf("ERRO: %s não é um graph.json válido ({meta, nodes, edges}): %w", opt.FromJSON, err)
	}
	avisos := frozen.Meta.Avisos
	if len(avisos) == 0 {
		avisos = []string{"regenerado de " + opt.FromJSON + " — sem re-parseio"}
	}
	escopo := frozen.Meta.Escopo
	if escopo == "" {
		escopo = "herdado do grafo congelado"
	}
	return &Graph{
		Meta: Meta{
			Modo:     "regeneração from-json (grafo congelado)",
			Escopo:   escopo,
			Fontes:   frozen.Meta.Fontes,
			GeradoEm: nowMinuteUTC(),
			Avisos:   avisos,
			Podado:   normalizePodado(frozen.Meta.Podado),
		},
		Nodes: frozen.Nodes,
		Edges: frozen.Edges,
	}, nil
}

// normalizePodado — podado vindo de JSON (map) vira *PodadoInfo (ordem de
// chaves preservada na re-emissão); bool/false atravessa.
func normalizePodado(v any) any {
	switch t := v.(type) {
	case map[string]any:
		pi := &PodadoInfo{}
		if x, ok := t["antes"].(float64); ok {
			pi.Antes = int(x)
		}
		if x, ok := t["removidos"].(float64); ok {
			pi.Removidos = int(x)
		}
		if x, ok := t["criterio"].(string); ok {
			pi.Criterio = x
		}
		return pi
	case bool:
		return t
	case nil:
		return false
	}
	return false
}

// ---- emissão ----

func renderMD(g *Graph, maxNodes int) string {
	grau := map[string]int{}
	for _, e := range g.Edges {
		grau[e.From]++
		grau[e.To]++
	}
	top := make([]*Node, len(g.Nodes))
	copy(top, g.Nodes)
	sort.SliceStable(top, func(i, j int) bool {
		a, b := top[i], top[j]
		if grau[a.ID] != grau[b.ID] {
			return grau[a.ID] > grau[b.ID]
		}
		return a.ID < b.ID
	})
	if len(top) > 30 {
		top = top[:30]
	}
	ids := map[string]bool{}
	for _, n := range top {
		ids[n.ID] = true
	}
	var mEdges []*Edge
	for _, e := range g.Edges {
		if ids[e.From] && ids[e.To] {
			mEdges = append(mEdges, e)
			if len(mEdges) >= 60 {
				break
			}
		}
	}
	sanitize := func(s string) string {
		var b strings.Builder
		for _, r := range s {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
				b.WriteRune(r)
			} else {
				b.WriteByte('_')
			}
		}
		return b.String()
	}
	var stale []string
	porTipo := map[string]int{}
	var tipoOrder []string
	for _, n := range g.Nodes {
		if porTipo[n.Type] == 0 {
			tipoOrder = append(tipoOrder, n.Type)
		}
		porTipo[n.Type]++
		if n.Attrs != nil {
			for _, p := range n.Attrs.pairs {
				if p.k == "stale" {
					if bv, ok := p.v.(bool); ok && bv {
						stale = append(stale, n.Label)
					}
				}
			}
		}
	}
	staleTxt := "nenhuma"
	if len(stale) > 0 {
		staleTxt = strings.Join(stale, ", ")
	}
	podadoTxt := ""
	if pi, ok := g.Meta.Podado.(*PodadoInfo); ok && pi != nil {
		podadoTxt = " · **PODADO:** " + itoa(pi.Removidos) + " nós (teto " + itoa(maxNodes) + ")"
	}
	var b strings.Builder
	b.WriteString("# Grafo do Conhecimento (T1-T5)\n\n")
	b.WriteString("> Gerado por golmet graph (motor determinístico do kit — ADR-019).\n")
	b.WriteString("> Gerado em " + g.Meta.GeradoEm + " · modo: " + g.Meta.Modo + " · escopo: " + g.Meta.Escopo + "\n\n")
	b.WriteString("## Contagem por tipo\n\n")
	b.WriteString("| Tipo | Nós |\n|---|---|\n")
	for i, t := range tipoOrder {
		b.WriteString("| " + t + " | " + itoa(porTipo[t]) + " |")
		if i < len(tipoOrder)-1 {
			b.WriteString("\n")
		}
	}
	b.WriteString("\n\n")
	b.WriteString("**Arestas:** " + itoa(len(g.Edges)) + podadoTxt + "\n\n")
	b.WriteString("**Bases T1 stale:** " + staleTxt + "\n\n")
	b.WriteString("## Mermaid (top " + itoa(len(top)) + " nós por grau)\n\n")
	b.WriteString("```mermaid\ngraph LR\n")
	for i, n := range top {
		b.WriteString("  " + sanitize(n.ID) + "[\"" + strings.ReplaceAll(n.Label, "\"", "'") + "\"]")
		if i < len(top)-1 {
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	for i, e := range mEdges {
		b.WriteString("  " + sanitize(e.From) + " -->|" + e.Type + "| " + sanitize(e.To))
		if i < len(mEdges)-1 {
			b.WriteString("\n")
		}
	}
	b.WriteString("\n```\n\n")
	b.WriteString("## Avisos (" + itoa(len(g.Meta.Avisos)) + ")\n\n")
	if len(g.Meta.Avisos) > 0 {
		for i, a := range g.Meta.Avisos {
			b.WriteString("- " + a)
			if i < len(g.Meta.Avisos)-1 {
				b.WriteString("\n")
			}
		}
	} else {
		b.WriteString("- nenhum")
	}
	b.WriteString("\n")
	return b.String()
}

func renderHTML(g *Graph) (string, error) {
	dados := struct {
		GeradoEm string  `json:"gerado_em"`
		Nodes    []*Node `json:"nodes"`
		Edges    []*Edge `json:"edges"`
	}{GeradoEm: g.Meta.GeradoEm, Nodes: g.Nodes, Edges: g.Edges}
	compact, err := marshalJSONCompact(dados)
	if err != nil {
		return "", err
	}
	jsonEsc := strings.ReplaceAll(string(compact), "<", `\u003c`)
	return strings.ReplaceAll(htmlTemplate, "__GRAPH_JSON__", jsonEsc), nil
}

func emitAll(g *Graph, opt Options) *Summary {
	gJSON, err := marshalJSONIndent(g)
	if err != nil {
		gJSON = []byte("{}")
	}
	md := renderMD(g, opt.MaxNodes)
	html, herr := renderHTML(g)
	if herr != nil {
		html = "<html><body>erro de render</body></html>"
	}
	out := opt.Out
	if out == "" {
		base := opt.WorkspaceHint
		if base == "" {
			base, _ = os.Getwd()
		}
		out = filepath.Join(base, "knowledge-graph")
	}
	outAbs := out
	if !filepath.IsAbs(outAbs) {
		outAbs, _ = filepath.Abs(outAbs)
	}
	_ = os.MkdirAll(outAbs, 0o755)
	pJSON := filepath.Join(outAbs, "graph.json")
	pHTML := filepath.Join(outAbs, "knowledge-graph.html")
	pMD := filepath.Join(outAbs, "knowledge-graph.md")
	// paridade: graph.json SEM newline final (JSON.stringify do JS não adiciona)
	_ = os.WriteFile(pJSON, gJSON, 0o644)
	_ = os.WriteFile(pHTML, []byte(html), 0o644)
	_ = os.WriteFile(pMD, []byte(md), 0o644)

	// resumo
	porTipo := map[string]int{}
	var stale []string
	for _, n := range g.Nodes {
		porTipo[n.Type]++
		if n.Attrs != nil {
			for _, p := range n.Attrs.pairs {
				if p.k == "stale" {
					if bv, ok := p.v.(bool); ok && bv {
						stale = append(stale, n.Label)
					}
				}
			}
		}
	}
	podadoOut := any(false)
	if pi, ok := g.Meta.Podado.(*PodadoInfo); ok && pi != nil {
		podadoOut = pi
	}
	avisos := g.Meta.Avisos
	if len(avisos) == 0 {
		avisos = []string{}
	}
	return &Summary{
		OK:        true,
		Modo:      g.Meta.Modo,
		Escopo:    g.Meta.Escopo,
		Nodes:     len(g.Nodes),
		Edges:     len(g.Edges),
		PorTipo:   porTipo,
		StaleT1:   stale,
		Avisos:    avisos,
		Podado:    podadoOut,
		Artefatos: Artefatos{GraphJSON: pJSON, HTML: pHTML, MD: pMD},
		OutDir:    outAbs,
	}
}

// ConsoleLines — as linhas de resumo no padrão do extrator (CLI imprime).
func (s *Summary) ConsoleLines() []string {
	lines := []string{
		"OK: " + itoa(s.Nodes) + " nós · " + itoa(s.Edges) + " arestas · " + s.OutDir,
		"Por tipo: " + summaryPorTipo(s.PorTipo),
		"T1 stale: " + orDefault(strings.Join(s.StaleT1, ", "), "nenhuma"),
	}
	if pi, ok := s.Podado.(*PodadoInfo); ok && pi != nil {
		lines = append(lines, "PODADO: "+mustJSON(pi))
	}
	lines = append(lines, "Avisos: "+itoa(len(s.Avisos)))
	for i, a := range s.Avisos {
		if i >= 15 {
			break
		}
		lines = append(lines, "  - "+a)
	}
	return lines
}

func summaryPorTipo(m map[string]int) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString("\"" + k + "\":" + itoa(m[k]))
	}
	b.WriteByte('}')
	return b.String()
}

func mustJSON(v any) string {
	b, err := marshalJSONCompact(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}
