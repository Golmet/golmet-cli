// parse.go — parsers de lista fechada (porta fiel do extrator .mjs):
// T5 registry em blocos + skills inline · T1 domains INDEX (staleness) ·
// T3 rules + índice lazy do AGENTS.md · T2 patterns INDEX · fundado_em ·
// T4 REQUIREMENTS/DECISIONS/TRACEABILITY/delivery/specs.
// Iterações de diretório são ORDENADAS (determinismo estrito do port).
package graph

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var (
	reSec      = regexp.MustCompile(`^([a-z_]+):\s*$`)
	reBlockDef = regexp.MustCompile(`^ {2}- `)
	reBlockID  = regexp.MustCompile(`^- id:\s*`)
	reField    = regexp.MustCompile(`^ {4}[a-z_]+:`)
	reFieldKV  = regexp.MustCompile(`^([a-z_]+):\s*(.*)$`)
	reInlineID = regexp.MustCompile(`id:\s*([^,}]+)`)
	reInlineDo = regexp.MustCompile(`domain:\s*([^,}]+)`)
	reInlineD1 = regexp.MustCompile(`description:\s*(.*)\s*\}$`)
	reInlineD2 = regexp.MustCompile(`description:\s*(.*)$`)
	reH3       = regexp.MustCompile(`(?m)^### `)
	reVer      = regexp.MustCompile(`\(v(\d+\.\d+\.\d+)\)`)
	reProx1    = regexp.MustCompile(`(?i)pr[oó]xima[_ ]revis[aã][oõ]?\**\s*:?\s*(\d{4}-\d{2}-\d{2})`)
	reProx2    = regexp.MustCompile(`(?i)proxima_revisao\**\s*:?\s*(\d{4}-\d{2}-\d{2})`)
	reRowPat   = regexp.MustCompile(`\| \[([^\]]+)\]\(([^)]+)\) \| ([^|]+) \| ([^|]+) \| ([^|]+) \| ([^|]+) \| ([^|]+) \|`)
	reFund     = regexp.MustCompile(`(?:domains|knowledge)/([a-z0-9-]+)/KNOWLEDGE\.md`)
	reReqSec   = regexp.MustCompile(`^#{2,3}\s((?:RN|PEND)-\d+)\s*[-—:]\s*(.+)\s*$`)
	rePromo1   = regexp.MustCompile("(?i)\\bPEND-(\\d+)\\b[^#\\n]{0,160}?promovid[oa]\\b[^#\\n]{0,160}?\\bRN-(\\d+)\\b")
	rePromo2   = regexp.MustCompile("(?i)\\bRN-(\\d+)\\b[^#\\n]{0,160}?promot[ao] d[oa] \\bPEND-(\\d+)\\b")
	reAdr      = regexp.MustCompile(`^##\s((?:ADR|DEC)-\d+)\s*[-—:]\s*(.+)\s*$`)
	reRef      = regexp.MustCompile(`\b(RN|PEND)-(\d+)\b`)
	reHeadStop = regexp.MustCompile(`^#{1,3}\s`)
	reTrHead   = regexp.MustCompile(`^\|\s*RN\b([^|]*)\|`)
	reTrRow    = regexp.MustCompile(`^\|\s*((?:RN|PEND)-\d+)\s*\|(.+)\|\s*$`)
	reFeature  = regexp.MustCompile(`(?i)^((?:RN|PEND)-\d+)\s*[-—]\s*(.+?)\s*[-—]\s*(Feature|Tasks)\.md$`)
	reFM       = regexp.MustCompile(`(?s)\A---\r?\n(.*?)\r?\n---`)
	reStatus   = regexp.MustCompile(`(?m)^status:\s*(.+)\s*$`)
)

// readText — ausente/ilegível = aviso + ok=false (tolerante: T1-T4).
func (g *engine) readText(abs, disp string) (string, bool) {
	st, err := os.Stat(abs)
	if err != nil || st.IsDir() {
		g.aviso("fonte ausente: " + disp)
		return "", false
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		g.aviso("fonte ilegível: " + disp)
		return "", false
	}
	return string(data), true
}

// readTextStrict — ausente/ilegível = StrictError (registry: espinha dorsal).
func readTextStrict(abs, disp string) (string, error) {
	st, err := os.Stat(abs)
	if err != nil || st.IsDir() {
		return "", &StrictError{Msg: "ERRO: fonte ausente: " + disp}
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return "", &StrictError{Msg: "ERRO: fonte ilegível: " + disp}
	}
	return string(data), nil
}

func splitLines(s string) []string { return strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n") }

func stripBold(s string) string { return strings.ReplaceAll(s, "**", "") }

func trimSpace(s string) string { return strings.TrimSpace(s) }

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func hojeUTC() string { return time.Now().UTC().Format("2006-01-02") }

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}


func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// ---- T5: registry.yaml (strict) ----

type regBlock struct {
	sec string
	id  string
	f   map[string]string
}

func (g *engine) parseT5() error {
	regDisp := g.cfgDisp() + "/registry.yaml"
	regTxt, err := readTextStrict(filepath.Join(g.opt.ConfigRoot, "registry.yaml"), regDisp)
	if err != nil {
		return err
	}

	g.node("t5:config:agents", "t5_config", "AGENTS.md (config)", "T5", newAttrs().set("papel", "índice lazy + convenções"))

	blocks := parseRegistryBlocks(regTxt)

	// skills inline (linhas `  - { id: ..., domain: ..., description: ... }`)
	for _, raw := range splitLines(regTxt) {
		t := trimSpace(raw)
		if !strings.HasPrefix(t, "- {") || !strings.Contains(t, "id:") {
			continue
		}
		if strings.Contains(t, "domain:") {
			id := ""
			if m := reInlineID.FindStringSubmatch(t); m != nil {
				id = trimSpace(m[1])
			}
			domain := ""
			if m := reInlineDo.FindStringSubmatch(t); m != nil {
				domain = trimSpace(m[1])
			}
			desc := ""
			if m := reInlineD1.FindStringSubmatch(t); m != nil {
				desc = trimSpace(m[1])
			} else if m := reInlineD2.FindStringSubmatch(t); m != nil {
				desc = trimSpace(m[1])
			}
			if id != "" {
				g.regSkills[id] = domain
				g.node("t5:skill:"+id, "t5_skill", id, "T5", newAttrs().set("domain", domain).set("description", desc))
			}
		}
	}
	// STRICT: registry sem NENHUM componente reconhecível = inválido → ABORTA
	if len(blocks) == 0 && len(g.regSkills) == 0 {
		return &StrictError{Msg: "ERRO: registry.yaml inválido (nenhum primary/subagent/domain/skill reconhecido): " + regDisp + " — camada T5 é a espinha dorsal do grafo (strict)."}
	}

	for _, b := range blocks {
		switch b.sec {
		case "primary_agents":
			g.node("t5:primary:"+b.id, "t5_primary", b.id, "T5", newAttrs().set("description", b.f["description"]))
		case "subagents":
			g.node("t5:subagent:"+b.id, "t5_subagent", b.id, "T5", newAttrs().set("description", b.f["description"]).set("owner", b.f["owner"]))
		case "domains":
			g.node("t5:dominio:"+b.id, "t5_dominio", b.id, "T5", newAttrs().set("orchestrator", orDefault(b.f["orchestrator"], "null")))
		}
	}
	for _, b := range blocks {
		switch b.sec {
		case "primary_agents":
			for _, d := range flowList(b.f["delegates"]) {
				g.edge("t5:primary:"+b.id, "t5:subagent:"+d, "delega", nil)
			}
			for _, s := range flowList(b.f["invokes_direct"]) {
				g.edge("t5:primary:"+b.id, "t5:skill:"+s, "invoca_direta", nil)
			}
		case "subagents":
			if b.f["owner"] != "" {
				g.edge("t5:subagent:"+b.id, "t5:primary:"+b.f["owner"], "owner", nil)
			}
			for _, a := range flowList(b.f["available_to"]) {
				g.edge("t5:primary:"+a, "t5:subagent:"+b.id, "disponivel_para", nil)
			}
			for _, s := range flowList(b.f["invokes"]) {
				g.edge("t5:subagent:"+b.id, "t5:skill:"+s, "invoca", nil)
			}
		case "domains":
			orch := b.f["orchestrator"]
			if orch != "" && orch != "null" {
				g.edge("t5:subagent:"+orch, "t5:dominio:"+b.id, "orquestra", nil)
			}
			for _, s := range flowList(b.f["skills"]) {
				g.edge("t5:dominio:"+b.id, "t5:skill:"+s, "agrega", nil)
			}
		}
	}
	// skills com domain → agrega implícito (quando o bloco domains não cobriu)
	// ordem determinística: iterar skills inline pela ordem de APARIÇÃO no txt
	// (rebuilt abaixo pela mesma varredura de linhas do registry)
	seenImplicit := map[string]bool{}
	for _, raw := range splitLines(regTxt) {
		t := trimSpace(raw)
		if !strings.HasPrefix(t, "- {") {
			continue
		}
		m := reInlineID.FindStringSubmatch(t)
		if m == nil {
			continue
		}
		id := trimSpace(m[1])
		domain, ok := g.regSkills[id]
		if !ok || domain == "" {
			continue
		}
		from, to := "t5:dominio:"+domain, "t5:skill:"+id
		if g.byID[from] == nil || g.byID[to] == nil {
			continue
		}
		key := from + "->" + to
		if seenImplicit[key] {
			continue
		}
		dup := false
		for _, e := range g.edges {
			if e.Type == "agrega" && e.From == from && e.To == to {
				dup = true
				break
			}
		}
		if !dup {
			seenImplicit[key] = true
			g.edge(from, to, "agrega", newAttrs().set("implicita", "via domain do registro da skill"))
		}
	}
	return nil
}

// parseRegistryBlocks — port do parseBlocks JS (passada única; quirks do
// registry regular preservados de propósito).
func parseRegistryBlocks(txt string) []regBlock {
	var out []regBlock
	sec := ""
	var block *regBlock
	push := func() {
		if block != nil {
			out = append(out, *block)
			block = nil
		}
	}
	for _, raw := range splitLines(txt) {
		line := strings.ReplaceAll(raw, "\t", "  ")
		if m := reSec.FindStringSubmatch(line); m != nil {
			push()
			sec = m[1]
			continue
		}
		if reBlockDef.MatchString(line) {
			push()
			t := trimSpace(line)
			if strings.HasPrefix(t, "- {") {
				continue // skill inline — tratada pelo scan dedicado
			}
			id := reBlockID.ReplaceAllString(t, "")
			block = &regBlock{sec: sec, id: trimSpace(id), f: map[string]string{}}
			continue
		}
		if block != nil && reField.MatchString(line) {
			if m := reFieldKV.FindStringSubmatch(trimSpace(line)); m != nil {
				block.f[m[1]] = trimSpace(m[2])
			}
		}
	}
	push()
	filtrados := out[:0]
	for _, b := range out {
		if b.sec != "" && b.id != "" {
			filtrados = append(filtrados, b)
		}
	}
	return filtrados
}

// flowList — lista flow `[a, b, c]` do subset YAML.
func flowList(s string) []string {
	s = trimSpace(s)
	if !strings.HasPrefix(s, "[") || !strings.Contains(s, "]") {
		return nil
	}
	inner := s[1 : strings.LastIndex(s, "]")]
	parts := strings.Split(inner, ",")
	var out []string
	for _, p := range parts {
		p = trimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// ---- pack auto-discovery (neutro: sem default de org) ----

func (g *engine) discoverPack() {
	if g.opt.Pack != "" {
		g.pack = g.opt.Pack
		return
	}
	packsDir := filepath.Join(g.opt.PacksRootOrCfg(), "packs")
	entries, err := os.ReadDir(packsDir)
	if err != nil {
		g.pack = ""
		return
	}
	var comDomains []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if fileExists(filepath.Join(packsDir, e.Name(), "domains", "INDEX.md")) {
			comDomains = append(comDomains, e.Name())
		}
	}
	sort.Strings(comDomains)
	if len(comDomains) == 0 {
		g.pack = ""
		return
	}
	if len(comDomains) > 1 {
		g.aviso("múltiplos packs com domains/ — usando \"" + comDomains[0] + "\" (use --pack para fixar)")
	}
	g.pack = comDomains[0]
}

func (o Options) PacksRootOrCfg() string {
	if o.PacksRoot != "" {
		return o.PacksRoot
	}
	return o.ConfigRoot
}

// ---- T1: domains INDEX.md ----

func (g *engine) parseT1() {
	if g.pack == "" {
		g.aviso("nenhum pack com domains/ encontrado em " + g.packsDisp() + "/packs — T1 pulado")
		return
	}
	disp := g.packsDisp() + "/packs/" + g.pack + "/domains/INDEX.md"
	txt, _ := g.readText(filepath.Join(g.opt.PacksRootOrCfg(), "packs", g.pack, "domains", "INDEX.md"), disp)
	if txt == "" {
		return
	}
	g.node("t1:indice", "t1_indice", "domains/INDEX.md ("+g.pack+")", "T1", nil)
	chunks := reH3.Split(txt, -1)
	if len(chunks) > 1 {
		chunks = chunks[1:]
	} else {
		chunks = nil
	}
	for _, c := range chunks {
		name := trimSpace(splitLines(c)[0])
		if name == "" || strings.HasPrefix(name, "#") {
			continue
		}
		ver := ""
		if m := reVer.FindStringSubmatch(c); m != nil {
			ver = m[1]
		}
		prox := ""
		if m := reProx1.FindStringSubmatch(c); m != nil {
			prox = m[1]
		} else if m := reProx2.FindStringSubmatch(c); m != nil {
			prox = m[1]
		}
		var stale any
		if prox != "" {
			stale = prox < hojeUTC()
		} else {
			stale = nil
		}
		versao := ""
		if ver != "" {
			versao = "v" + ver
		}
		a := newAttrs().set("versao", nilIfEmpty(versao)).set("proxima_revisao", nilIfEmpty(prox)).set("stale", stale)
		g.node("t1:dominio:"+name, "t1_dominio", name, "T1", a)
		g.edge("t1:indice", "t1:dominio:"+name, "indice", nil)
		g.t1Domains = append(g.t1Domains, name)
	}
}

// ---- T3: rules + índice lazy do AGENTS.md ----

func (g *engine) parseT3() {
	lazyDisp := g.cfgDisp() + "/AGENTS.md"
	lazyTxt, _ := g.readText(filepath.Join(g.opt.ConfigRoot, "AGENTS.md"), lazyDisp)
	if g.pack == "" {
		return
	}
	rulesDir := filepath.Join(g.opt.PacksRootOrCfg(), "packs", g.pack, "rules")
	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		return
	}
	var names []string
	for _, e := range entries {
		n := e.Name()
		if !e.IsDir() && strings.HasSuffix(n, ".md") && !strings.HasSuffix(n, ".template.md") {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	for _, f := range names {
		name := strings.TrimSuffix(f, ".md")
		g.node("t3:rule:"+name, "t3_rule", name, "T3", newAttrs().set("arquivo", "packs/"+g.pack+"/rules/"+f))
		if strings.Contains(lazyTxt, "rules/"+f) {
			g.edge("t5:config:agents", "t3:rule:"+name, "indice_lazy", nil)
		} else {
			g.aviso("rule fora do índice lazy: " + f + " (sinal válido — reportar ao gm-rule-keeper)")
		}
	}
	// registry T3 em yaml (ex.: stack-templates.yaml)
	if fileExists(filepath.Join(rulesDir, "stack-templates.yaml")) {
		g.node("t3:rule:stack-templates.yaml", "t3_rule", "stack-templates.yaml", "T3",
			newAttrs().set("arquivo", "packs/"+g.pack+"/rules/stack-templates.yaml").set("tipo", "registry T3 (yaml)"))
		// IMP-011: registry yaml é elegível a indice_lazy pelo mesmo match do
		// índice lazy (attrs.arquivo contra a referência "rules/<arquivo>",
		// independente de extensão .md) — sem isso o nó fica órfão permanente
		if strings.Contains(lazyTxt, "rules/stack-templates.yaml") {
			g.edge("t5:config:agents", "t3:rule:stack-templates.yaml", "indice_lazy", nil)
		} else {
			g.aviso("registry yaml fora do índice lazy: stack-templates.yaml (sinal válido — reportar ao gm-rule-keeper)")
		}
	}
}

// ---- T2: patterns INDEX.md ----

func (g *engine) parseT2() {
	if g.pack == "" {
		return
	}
	patDisp := g.packsDisp() + "/packs/" + g.pack + "/accumulation/patterns/INDEX.md"
	txt, _ := g.readText(filepath.Join(g.opt.PacksRootOrCfg(), "packs", g.pack, "accumulation", "patterns", "INDEX.md"), patDisp)
	if txt == "" {
		return
	}
	g.node("t2:pack:"+g.pack, "t2_pack", "pack "+g.pack, "T2", nil)
	inTable := false
	for _, line := range splitLines(txt) {
		if strings.HasPrefix(line, "| Padr") {
			inTable = true
			continue
		}
		if !inTable {
			continue
		}
		m := reRowPat.FindStringSubmatch(line)
		if m == nil {
			if !strings.HasPrefix(line, "|") {
				inTable = false
			}
			continue
		}
		slug := trimSpace(m[1])
		tipo := trimSpace(m[3])
		projetos := trimSpace(m[5])
		dataRev := trimSpace(m[6])
		status := trimSpace(m[7])
		g.node("t2:pattern:"+slug, "t2_pattern", slug, "T2",
			newAttrs().set("tipo", tipo).set("projetos", projetos).set("data_ultima_revisao", dataRev).set("status", status))
		g.edge("t2:pattern:"+slug, "t2:pack:"+g.pack, "pertence", nil)
		g.t2Slugs = append(g.t2Slugs, slug)
	}
}

// ---- fundado_em (skills/agents que citam KNOWLEDGE.md de domínio) ----

func (g *engine) scanFund() {
	g.scanFundDir(filepath.Join(g.opt.ConfigRoot, "skills"), g.cfgDisp()+"/skills", "skill")
	g.scanFundDir(filepath.Join(g.opt.ConfigRoot, "agents"), g.cfgDisp()+"/agents", "primary")
}

func (g *engine) scanFundDir(dir, disp, kind string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		g.aviso("dir ausente para fundado_em: " + disp)
		return
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	for _, entry := range names {
		p := filepath.Join(dir, entry)
		file := p
		suffix := ""
		if kind == "skill" {
			file = filepath.Join(p, "SKILL.md")
			suffix = "/SKILL.md"
		}
		st, err := os.Stat(file)
		if err != nil || st.IsDir() {
			continue
		}
		data, rerr := os.ReadFile(file)
		if rerr != nil {
			continue
		}
		found := map[string]bool{}
		var order []string
		for _, m := range reFund.FindAllStringSubmatch(string(data), -1) {
			if !found[m[1]] {
				found[m[1]] = true
				order = append(order, m[1])
			}
		}
		for _, d := range order {
			t1id := "t1:dominio:" + d
			from := "t5:primary:" + strings.TrimSuffix(entry, ".md")
			if kind == "skill" {
				from = "t5:skill:" + entry
			}
			if g.byID[t1id] != nil && g.byID[from] != nil {
				g.edge(from, t1id, "fundado_em", nil)
			} else if g.byID[t1id] == nil {
				g.aviso("fundado_em: domínio T1 inexistente \"" + d + "\" citado em " + disp + "/" + entry + suffix)
			}
		}
	}
}

// ---- T4: módulos de produto ----

func (g *engine) parseT4All() {
	if g.opt.ProjectsRoot == "" {
		g.aviso("sem --projects-root — camada T4 (módulos) pulada (modo \"estrutura do próprio Golmet\")")
		return
	}
	root := g.opt.ProjectsRoot
	if !dirExists(root) {
		g.aviso("--projects-root inexistente: " + g.projectsDisp() + " — T4 pulado")
		return
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		g.aviso("--projects-root inexistente: " + g.projectsDisp() + " — T4 pulado")
		return
	}
	var dirs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if fileExists(filepath.Join(root, name, "REQUIREMENTS.md")) ||
			dirExists(filepath.Join(root, name, "delivery")) ||
			dirExists(filepath.Join(root, name, "specs")) {
			dirs = append(dirs, name)
		}
	}
	sort.Strings(dirs)
	if g.opt.Scope != "" {
		found := false
		for _, d := range dirs {
			if d == g.opt.Scope {
				found = true
				break
			}
		}
		if !found {
			g.aviso("módulo do --scope não encontrado em " + g.projectsDisp() + ": " + g.opt.Scope + " — T4 vazio (delta)")
		}
		var filtered []string
		for _, d := range dirs {
			if d == g.opt.Scope {
				filtered = append(filtered, d)
			}
		}
		dirs = filtered
	}
	for _, mod := range dirs {
		g.parseT4Module(root, mod)
	}
}

func (g *engine) parseT4Module(root, mod string) {
	g.t4Modulos = append(g.t4Modulos, mod)
	modDisp := g.projectsDisp() + "/" + mod
	modDir := filepath.Join(root, mod)
	modID := "t4:modulo:" + mod
	g.node(modID, "t4_modulo", mod, "T4", newAttrs().set("raiz", modDisp))

	// ---- REQUIREMENTS.md: RNs/PENDs + promoção PEND→RN (IMP-094) ----
	reqTxt, _ := g.readText(filepath.Join(modDir, "REQUIREMENTS.md"), modDisp+"/REQUIREMENTS.md")
	if reqTxt != "" {
		for _, line := range splitLines(reqTxt) {
			m := reReqSec.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			idFull, titulo := m[1], stripBold(trimSpace(m[2]))
			if strings.HasPrefix(idFull, "RN") {
				g.node("t4:rn:"+mod+":"+idFull, "t4_rn", idFull+" — "+titulo, "T4", newAttrs().set("modulo", mod))
				g.edge(modID, "t4:rn:"+mod+":"+idFull, "define", nil)
			} else {
				g.node("t4:pend:"+mod+":"+idFull, "t4_pend", idFull+" — "+titulo, "T4", newAttrs().set("modulo", mod))
				g.edge(modID, "t4:pend:"+mod+":"+idFull, "abre_questao", nil)
			}
		}
		for _, line := range splitLines(reqTxt) {
			if m := rePromo1.FindStringSubmatch(line); m != nil {
				g.edge("t4:pend:"+mod+":PEND-"+m[1], "t4:rn:"+mod+":RN-"+m[2], "promove", newAttrs().set("imp", "IMP-094"))
			} else if m := rePromo2.FindStringSubmatch(line); m != nil {
				g.edge("t4:pend:"+mod+":PEND-"+m[2], "t4:rn:"+mod+":RN-"+m[1], "promove", newAttrs().set("imp", "IMP-094"))
			}
		}
	}

	// ---- DECISIONS.md: ADRs/DECs + refs → decide; citação de pattern → aplica_em ----
	decTxt, _ := g.readText(filepath.Join(modDir, "DECISIONS.md"), modDisp+"/DECISIONS.md")
	if decTxt != "" {
		curAdr := ""
		decSlugs := map[string]bool{}
		var decSlugOrder []string
		for _, line := range splitLines(decTxt) {
			if m := reAdr.FindStringSubmatch(line); m != nil {
				curAdr = m[1]
				adrID := "t4:adr:" + mod + ":" + curAdr
				g.node(adrID, "t4_adr", curAdr+" — "+stripBold(trimSpace(m[2])), "T4", newAttrs().set("modulo", mod))
				g.edge(adrID, modID, "artefato", nil)
				continue
			}
			if curAdr == "" {
				continue
			}
			if reHeadStop.MatchString(line) {
				curAdr = ""
				continue
			}
			adrID := "t4:adr:" + mod + ":" + curAdr
			for _, rm := range reRef.FindAllStringSubmatch(line, -1) {
				full := rm[1] + "-" + rm[2]
				target := "t4:pend:" + mod + ":" + full
				if rm[1] == "RN" {
					target = "t4:rn:" + mod + ":" + full
				}
				if g.byID[target] != nil {
					g.edge(adrID, target, "decide", nil)
				}
			}
			for _, slug := range g.t2Slugs {
				if strings.Contains(line, slug) && !decSlugs[slug] {
					decSlugs[slug] = true
					decSlugOrder = append(decSlugOrder, slug)
				}
			}
		}
		for _, slug := range decSlugOrder {
			if g.byID["t2:pattern:"+slug] != nil {
				g.edge("t2:pattern:"+slug, modID, "aplica_em", nil)
			}
		}
	}

	// ---- specs/*.md: nós t4_spec ----
	specsDir := filepath.Join(modDir, "specs")
	if entries, err := os.ReadDir(specsDir); err == nil {
		var files []string
		for _, e := range entries {
			n := e.Name()
			if !e.IsDir() && strings.HasSuffix(strings.ToLower(n), ".md") {
				files = append(files, n)
			}
		}
		sort.Strings(files)
		for _, f := range files {
			name := strings.TrimSuffix(f, ".md")
			specID := "t4:spec:" + mod + ":" + name
			g.node(specID, "t4_spec", name+".md", "T4", newAttrs().set("arquivo", modDisp+"/specs/"+f))
			g.edge(specID, modID, "artefato", nil)
		}
	}

	// ---- quality/TRACEABILITY.md: matriz RN × downstream → cobre ----
	tr1 := filepath.Join(modDir, "quality", "TRACEABILITY.md")
	tr2 := filepath.Join(modDir, "QUALITY", "TRACEABILITY.md")
	trPath := tr1
	if !fileExists(tr1) && fileExists(tr2) {
		trPath = tr2
	}
	trTxt, _ := g.readText(trPath, modDisp+"/quality/TRACEABILITY.md")
	if trTxt != "" {
		var cols []string
		for _, line := range splitLines(trTxt) {
			if cols == nil {
				if m := reTrHead.FindStringSubmatch(line); m != nil && strings.Count(line, "|") >= 3 {
					raw := strings.Split(line, "|")
					if len(raw) >= 3 {
						for _, c := range raw[2 : len(raw)-1] {
							c = strings.ReplaceAll(strings.ReplaceAll(c, "*", ""), "`", "")
							c = trimSpace(c)
							if c != "" {
								cols = append(cols, c)
							}
						}
					}
				}
				continue
			}
			m := reTrRow.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			rnFull := m[1]
			rawCells := strings.Split(m[2], "|")
			cells := make([]string, len(rawCells))
			for i, c := range rawCells {
				cells[i] = trimSpace(c)
			}
			for i, col := range cols {
				cell := ""
				if i < len(cells) {
					cell = strings.ToUpper(cells[i])
				}
				if cell == "" || cell == "—" || cell == "-" {
					continue
				}
				var status string
				if strings.HasPrefix(cell, "OK") {
					status = "OK"
				} else if strings.HasPrefix(cell, "FALHA") {
					status = "FALHA"
				} else {
					continue
				}
				colClean := col
				if strings.HasSuffix(strings.ToLower(col), ".md") {
					colClean = col[:len(col)-3]
				}
				specID := "t4:spec:" + mod + ":" + colClean
				if g.byID[specID] == nil {
					g.node(specID, "t4_spec", colClean+".md", "T4",
						newAttrs().set("arquivo", modDisp+"/specs/"+colClean+".md").set("origem", "TRACEABILITY (spec ausente)"))
				}
				rnNodeID := "t4:pend:" + mod + ":" + rnFull
				rnType := "t4_pend"
				if strings.HasPrefix(rnFull, "RN") {
					rnNodeID = "t4:rn:" + mod + ":" + rnFull
					rnType = "t4_rn"
				}
				if g.byID[rnNodeID] == nil {
					g.node(rnNodeID, rnType, rnFull, "T4", newAttrs().set("modulo", mod).set("origem", "TRACEABILITY"))
				}
				g.edge(rnNodeID, specID, "cobre", newAttrs().set("downstream", colClean).set("status", status))
			}
		}
	}

	// ---- delivery/**: features com âncora (IMP-092/093; fallback Tasks.md) ----
	g.walkDelivery(filepath.Join(modDir, "delivery"), mod, modDisp)
}

func (g *engine) walkDelivery(dir, mod, modDisp string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, e := range entries {
		p := filepath.Join(dir, e.Name())
		if e.IsDir() {
			g.walkDelivery(p, mod, modDisp)
			continue
		}
		m := reFeature.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		anchor, titulo, tipoArq := m[1], m[2], m[3]
		disp := modDisp + "/delivery/" + e.Name()
		legada := strings.HasPrefix(anchor, "PEND")
		var status any
		if data, rerr := os.ReadFile(p); rerr == nil {
			if fm := reFM.FindStringSubmatch(string(data)); fm != nil {
				if sm := reStatus.FindStringSubmatch(fm[1]); sm != nil {
					status = trimSpace(sm[1])
				}
			}
		}
		a := newAttrs().set("modulo", mod).set("arquivo", disp).set("tipo", strings.ToLower(tipoArq)).set("status", status)
		if legada {
			a.set("ancora_legada", true)
		}
		featID := "t4:feature:" + mod + ":" + anchor
		g.node(featID, "t4_feature", anchor+" — "+stripBold(trimSpace(titulo)), "T4", a)
		g.edge(featID, "t4:modulo:"+mod, "artefato", nil)
		if !legada {
			g.edge("t4:rn:"+mod+":"+anchor, featID, "ancora", nil)
		}
		// âncora PEND legada: NENHUMA aresta inventada (promove/ancora —
		// IMP-094; a promoção é papel do gm-requirements e só vira aresta com
		// evidência explícita na REQUIREMENTS, detectada acima).
	}
}
