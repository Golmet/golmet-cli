// Package graph implementa o motor determinístico do grafo de conhecimento
// (T1-T5) — porta fiel do extrator da skill gm-knowledge-graph (ADR-019:
// "determinístico mora no CLI; o que é da organização mora no PACK; o
// julgamento mora no RUNTIME"). Fontes read-only; ausente = aviso, EXCETO
// o registry.yaml do config do engine (strict: ausente/inválido = erro).
// Saída integral: os 3 artefatos só são escritos após o grafo montado em
// memória. Única variação entre execuções é meta.gerado_em (precisão de
// MINUTO — idempotência de saída). O seletor --what filtra stores:
// workflow→T5 · pack→T1+T2+T3 · docs→T4 (combinável com '+'); arestas
// cruzadas (indice_lazy T5→T3, fundado_em T5→T1, aplica_em T2→T4) só entram
// quando origem E destino estão selecionados.
package graph

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Options controla uma execução do motor (espelha as flags do comando
// `golmet graph` e os params da tool gm_graph).
type Options struct {
	ConfigRoot   string // default: <homedir>/.config/opencode
	PacksRoot    string // default: ConfigRoot (raiz onde vive packs/)
	Pack         string // default: auto-descoberta (único pack com domains/)
	ProjectsRoot string // vazio → T4 pulado com aviso (quando what inclui docs)
	Out          string // default: <cwd>/knowledge-graph
	Scope        string // delta: restringe docs ao módulo nomeado
	MaxNodes     int    // teto de nós (default 3500; poda RN/PEND do global primeiro)
	FromJSON     string // regeneração de grafo congelado (sem re-parseio)
	What         string // "workflow|pack|docs|all" combinável com '+' (default all)

	// WorkspaceHint — contexto da tool MCP: resolve o default do Out
	// (<hint>/knowledge-graph) quando Out vazio (CLI usa cwd).
	WorkspaceHint string

	// disp — caminhos COMO PASSADOS (relativo permanece relativo:
	// determinismo entre máquinas); vazio = igual ao resolvido.
	DispCfg, DispPacks, DispProjects string
}

// Summary é o relatório compacto da execução (console do CLI / payload da
// tool MCP).
type Summary struct {
	OK        bool           `json:"ok"`
	Modo      string         `json:"modo"`
	Escopo    string         `json:"escopo"`
	What      string         `json:"what"`
	Pack      string         `json:"pack,omitempty"`
	Nodes     int            `json:"nodes"`
	Edges     int            `json:"edges"`
	PorTipo   map[string]int `json:"por_tipo"`
	StaleT1   []string       `json:"stale_t1"`
	Avisos    []string       `json:"avisos"`
	Podado    any            `json:"podado,omitempty"`
	Artefatos Artefatos      `json:"artefatos"`
	OutDir    string         `json:"out_dir"`
}

// Artefatos são os caminhos dos 3 arquivos escritos.
type Artefatos struct {
	GraphJSON string `json:"graph_json"`
	HTML      string `json:"html"`
	MD        string `json:"md"`
}

// Attrs — mapa ordenado por inserção, paridade com o JSON do extrator de
// referência (Object.assign do JS preserva ordem de inserção). Valores:
// string, bool, nil (→ null) e omit (sentinela que OMITE a chave — análogo
// a `undefined`, que JSON.stringify não emite).
type omitType struct{}

var omit = omitType{}

type attrPair struct {
	k string
	v any
}

type Attrs struct {
	pairs []attrPair
}

func newAttrs() *Attrs { return &Attrs{} }

func (a *Attrs) set(k string, v any) *Attrs {
	for i := range a.pairs {
		if a.pairs[i].k == k {
			a.pairs[i].v = v
			return a
		}
	}
	a.pairs = append(a.pairs, attrPair{k, v})
	return a
}

func (a *Attrs) merge(other *Attrs) {
	if other == nil {
		return
	}
	for _, p := range other.pairs {
		a.set(p.k, p.v)
	}
}

// selected — o que o seletor --what habilita.
type selected struct{ t1, t2, t3, t4, t5 bool }

func (s selected) store(name string) bool {
	switch name {
	case "T1":
		return s.t1
	case "T2":
		return s.t2
	case "T3":
		return s.t3
	case "T4":
		return s.t4
	case "T5":
		return s.t5
	}
	return false
}

func parseWhat(what string) (selected, string, error) {
	w := strings.TrimSpace(what)
	if w == "" {
		w = "all"
	}
	tokens := strings.Split(w, "+")
	var sel selected
	seen := map[string]bool{}
	for _, t := range tokens {
		t = strings.TrimSpace(strings.ToLower(t))
		switch t {
		case "all":
			return selected{true, true, true, true, true}, "all", nil
		case "workflow":
			sel.t5 = true
		case "pack":
			sel.t1, sel.t2, sel.t3 = true, true, true
		case "docs":
			sel.t4 = true
		default:
			return selected{}, "", fmt.Errorf("valor inválido em --what: %q (use workflow|pack|docs|all, combináveis com +)", t)
		}
		seen[t] = true
	}
	labels := make([]string, 0, len(seen))
	for _, l := range []string{"workflow", "pack", "docs"} {
		if seen[l] {
			labels = append(labels, l)
		}
	}
	return sel, strings.Join(labels, "+"), nil
}

// StrictError marca a falha de registry (o único aborto do motor — exit 1).
type StrictError struct{ Msg string }

func (e *StrictError) Error() string { return e.Msg }

// ---- tipos do grafo (ordem dos campos = ordem das chaves no JSON — paridade) ----

type Node struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Label string `json:"label"`
	Store string `json:"store"`
	Attrs *Attrs `json:"attrs"`
}

type Edge struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Type  string `json:"type"`
	Attrs *Attrs `json:"attrs"`
}

type PodadoInfo struct {
	Antes     int    `json:"antes"`
	Removidos int    `json:"removidos"`
	Criterio  string `json:"criterio"`
}

type Meta struct {
	Modo     string   `json:"modo"`
	Escopo   string   `json:"escopo"`
	Fontes   []string `json:"fontes"`
	GeradoEm string   `json:"gerado_em"`
	Avisos   []string `json:"avisos"`
	Podado   any      `json:"podado"`
}

type Graph struct {
	Meta  Meta    `json:"meta"`
	Nodes []*Node `json:"nodes"`
	Edges []*Edge `json:"edges"`
}

// ---- engine ----

type engine struct {
	opt       Options
	nodes     []*Node
	edges     []*Edge
	byID      map[string]*Node
	edgeKeys  map[string]bool
	avisos    []string
	avisoSet  map[string]bool
	regSkills map[string]string // id → domain (skills inline do registry)
	pack      string
	t1Domains []string
	t2Slugs   []string
	t4Modulos []string
	podado    any
}

func newEngine(opt Options) *engine {
	return &engine{
		opt:       opt,
		byID:      map[string]*Node{},
		edgeKeys:  map[string]bool{},
		avisoSet:  map[string]bool{},
		regSkills: map[string]string{},
	}
}

func (g *engine) node(id, typ, label, store string, attrs *Attrs) *Node {
	if n, ok := g.byID[id]; ok {
		if attrs != nil {
			n.Attrs.merge(attrs)
		}
		return n
	}
	if attrs == nil {
		attrs = newAttrs()
	}
	n := &Node{ID: id, Type: typ, Label: label, Store: store, Attrs: attrs}
	g.byID[id] = n
	g.nodes = append(g.nodes, n)
	return n
}

func (g *engine) edge(from, to, typ string, attrs *Attrs) {
	k := from + "|" + to + "|" + typ
	if g.edgeKeys[k] {
		return
	}
	g.edgeKeys[k] = true
	if attrs == nil {
		attrs = newAttrs()
	}
	g.edges = append(g.edges, &Edge{From: from, To: to, Type: typ, Attrs: attrs})
}

func (g *engine) aviso(m string) {
	if g.avisoSet[m] {
		return
	}
	g.avisoSet[m] = true
	g.avisos = append(g.avisos, m)
}

// disp — caminho exibido (como passado; relativo permanece relativo).
func (g *engine) cfgDisp() string {
	if g.opt.DispCfg != "" {
		return g.opt.DispCfg
	}
	return g.opt.ConfigRoot
}

func (g *engine) packsDisp() string {
	if g.opt.DispPacks != "" {
		return g.opt.DispPacks
	}
	return g.cfgDisp()
}

func (g *engine) projectsDisp() string {
	if g.opt.DispProjects != "" {
		return g.opt.DispProjects
	}
	return g.opt.ProjectsRoot
}

// Run executa o motor e escreve os 3 artefatos (integral ou nada).
func Run(opt Options) (*Summary, error) {
	if opt.MaxNodes <= 0 {
		opt.MaxNodes = 3500
	}
	sel, whatLabel, err := parseWhat(opt.What)
	if err != nil {
		return nil, err
	}

	// defaults + disp (como passado)
	defCfg := filepath.Join(homeDir(), ".config", "opencode")
	if opt.ConfigRoot == "" {
		opt.ConfigRoot = defCfg
	}
	if opt.DispCfg == "" {
		opt.DispCfg = opt.ConfigRoot
	}

	// modo from-json: regeneração sem re-parseio (correção de template
	// reaplicada a artefatos já gerados — grafo congelado)
	if opt.FromJSON != "" {
		g, serr := runFromJSON(opt)
		if serr != nil {
			return nil, serr
		}
		sum := emitAll(g, opt)
		sum.What = "grafo congelado"
		return sum, nil
	}

	g := newEngine(opt)

	// ---- T5: registry.yaml SEMPRE parseado (strict — espinha dorsal) ----
	if err := g.parseT5(); err != nil {
		return nil, err
	}

	// ---- pack auto-discovery + T1/T2/T3 (só quando pack selecionado) ----
	if sel.t1 || sel.t2 || sel.t3 {
		g.discoverPack()
		if g.pack != "" {
			if sel.t1 {
				g.parseT1()
			}
			if sel.t3 {
				g.parseT3()
			}
			if sel.t2 {
				g.parseT2()
			}
		}
	}

	// ---- fundado_em (cross T5→T1; o filtro final exige ambos os stores) ----
	if sel.t1 && sel.t5 {
		g.scanFund()
	}

	// ---- T4 ----
	if sel.t4 {
		g.parseT4All()
	}

	g.prune()
	g.sanitize()
	g.applySelector(sel)
	sortAll(g.nodes, g.edges)
	meta := g.buildMeta(sel, whatLabel)

	graph := &Graph{Meta: meta, Nodes: g.nodes, Edges: g.edges}
	sum := emitAll(graph, opt)
	sum.What = whatLabel
	sum.Pack = g.pack
	return sum, nil
}

func (g *engine) buildMeta(sel selected, whatLabel string) Meta {
	fontes := []string{
		g.cfgDisp() + "/registry.yaml",
		g.cfgDisp() + "/AGENTS.md",
	}
	if g.pack != "" {
		fontes = append(fontes,
			g.packsDisp()+"/packs/"+g.pack+"/domains/INDEX.md",
			g.packsDisp()+"/packs/"+g.pack+"/rules/",
			g.packsDisp()+"/packs/"+g.pack+"/accumulation/patterns/INDEX.md")
	}
	fontes = append(fontes, g.cfgDisp()+"/skills/*/SKILL.md", g.cfgDisp()+"/agents/*.md")
	if sel.t4 && g.opt.ProjectsRoot != "" && dirExists(g.opt.ProjectsRoot) {
		fontes = append(fontes, g.projectsDisp()+"/{REQUIREMENTS.md, DECISIONS.md, quality/TRACEABILITY.md, delivery/**, specs/}")
	}
	if g.opt.Scope != "" && len(g.t4Modulos) == 0 {
		g.aviso("delta sem módulo correspondente — grafo sem T4")
	}
	var escopo string
	if sel.t4 && g.opt.ProjectsRoot != "" && dirExists(g.opt.ProjectsRoot) && len(g.t4Modulos) > 0 {
		if g.opt.Scope != "" {
			escopo = g.opt.Scope
		} else {
			escopo = strings.Join(g.t4Modulos, ", ")
		}
	} else {
		escopo = "estrutura do próprio Golmet (T1-T5)"
		if g.pack != "" {
			escopo += " · pack " + g.pack
		}
	}
	if whatLabel != "" && whatLabel != "all" {
		escopo += " · what: " + whatLabel
	}
	modo := "global"
	if g.opt.Scope != "" {
		modo = "delta (scope: " + g.opt.Scope + ")"
	}
	podado := any(false)
	if g.podado != nil {
		podado = g.podado
	}
	return Meta{
		Modo:     modo,
		Escopo:   escopo,
		Fontes:   fontes,
		GeradoEm: nowMinuteUTC(),
		Avisos:   g.avisos,
		Podado:   podado,
	}
}

func (g *engine) prune() {
	if len(g.nodes) <= g.opt.MaxNodes {
		return
	}
	antes := len(g.nodes)
	grau := map[string]int{}
	for _, e := range g.edges {
		grau[e.From]++
		grau[e.To]++
	}
	drop := map[string]bool{}
	rnPend := []string{}
	for _, n := range g.nodes {
		if n.Type == "t4_rn" || n.Type == "t4_pend" {
			rnPend = append(rnPend, n.ID)
		}
	}
	sort.Strings(rnPend)
	for _, id := range rnPend {
		if len(g.nodes)-len(drop) <= g.opt.MaxNodes {
			break
		}
		drop[id] = true
	}
	if len(g.nodes)-len(drop) > g.opt.MaxNodes {
		type gd struct {
			id string
			g  int
		}
		resto := []gd{}
		for _, n := range g.nodes {
			if drop[n.ID] || n.Type == "t5_config" {
				continue
			}
			resto = append(resto, gd{n.ID, grau[n.ID]})
		}
		sort.Slice(resto, func(i, j int) bool {
			if resto[i].g != resto[j].g {
				return resto[i].g < resto[j].g
			}
			return resto[i].id < resto[j].id
		})
		for _, r := range resto {
			if len(g.nodes)-len(drop) <= g.opt.MaxNodes {
				break
			}
			drop[r.id] = true
		}
	}
	for id := range drop {
		delete(g.byID, id)
	}
	kept := g.nodes[:0]
	for _, n := range g.nodes {
		if !drop[n.ID] {
			kept = append(kept, n)
		}
	}
	g.nodes = kept
	g.podado = &PodadoInfo{Antes: antes, Removidos: len(drop), Criterio: "RN/PEND do modo global primeiro, depois menor grau (id asc)"}
}

func (g *engine) sanitize() {
	validas := make([]*Edge, 0, len(g.edges))
	for _, e := range g.edges {
		if g.byID[e.From] != nil && g.byID[e.To] != nil {
			validas = append(validas, e)
		} else {
			g.aviso("aresta descartada (ponta solta): " + e.From + " --" + e.Type + "--> " + e.To)
		}
	}
	g.edges = validas
}

func (g *engine) applySelector(sel selected) {
	kept := g.nodes[:0]
	for _, n := range g.nodes {
		if sel.store(n.Store) {
			kept = append(kept, n)
		}
	}
	g.nodes = kept
	ved := g.edges[:0]
	for _, e := range g.edges {
		fn, tn := g.byID[e.From], g.byID[e.To]
		if fn != nil && tn != nil && sel.store(fn.Store) && sel.store(tn.Store) {
			ved = append(ved, e)
		}
		// aresta cruzada com ponta fora do seletor: descartada silenciosamente
		// (decisão do seletor — não é ponta solta de dado)
	}
	g.edges = ved
}

func sortAll(nodes []*Node, edges []*Edge) {
	sort.SliceStable(nodes, func(i, j int) bool {
		a, b := nodes[i], nodes[j]
		if a.Store != b.Store {
			return a.Store < b.Store
		}
		if a.Type != b.Type {
			return a.Type < b.Type
		}
		return a.ID < b.ID
	})
	sort.SliceStable(edges, func(i, j int) bool {
		a, b := edges[i], edges[j]
		if a.Type != b.Type {
			return a.Type < b.Type
		}
		if a.From != b.From {
			return a.From < b.From
		}
		return a.To < b.To
	})
}

func nowMinuteUTC() string {
	return time.Now().UTC().Format("2006-01-02T15:04")
}

func homeDir() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return h
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func itoa(n int) string { return strconv.Itoa(n) }
