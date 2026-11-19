// Testes do lint: um check por fixture de sandbox — refs mortas com
// falsos positivos tratados, órfãos com/sem allowlist, manifest stale,
// nomes fora do padrão, fantasmas por contexto e estrutura canônica.
package lint

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// kitFixture monta um sandbox kit mínimo: registry + sources/rules/docs.
func kitFixture(t *testing.T) string {
	t.Helper()
	ws := t.TempDir()
	os.MkdirAll(filepath.Join(ws, "coredata", "sources"), 0o755)
	os.WriteFile(filepath.Join(ws, "coredata", "registry.yaml"), []byte(
		"version: pilot-1\nitems:\n  - id: gm-a\n    kind: skill\n    body: sources/gm-a.md\n"), 0o644)
	return ws
}

func findingsOf(rep *Report, check string) []Finding {
	var out []Finding
	for _, f := range rep.Findings {
		if f.Check == check {
			out = append(out, f)
		}
	}
	return out
}

func runLint(t *testing.T, ws, home string) *Report {
	t.Helper()
	rep, err := Run(Options{Workspace: ws, EngineID: "opencode", Home: home})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return rep
}

// ---- L1 ----

func TestL1DeadRefsAndFalsePositives(t *testing.T) {
	ws := kitFixture(t)
	src := filepath.Join(ws, "coredata", "sources", "gm-a.md")
	os.WriteFile(src, []byte(strings.Join([]string{
		"- [ok](README.md) e `FORMAT.md` — existem na raiz do kit",
		"- `docs/guia.md` — existe em docs/",
		"- [morto](sub/nada-disso.md) — não existe → L1",
		"- `profile/context7.md` — resolve via coredata/rules (mapping canônico)",
		"- `agents/gm-a.md` — resolve via registry (gm-a)",
		"- <caminho-placeholder> e `pkg/*/x.go` e [url](https://x.com/a.md) — não verificáveis",
		"- `packs/<id>/x.md` — placeholder, não verificável",
		"- `precos/nao-existe.md` — fora das raízes → L1",
		"- `angular.json` — nome solto de repo de produto: não verificado (heurística)",
		"",
		"```exemplo",
		"- `exemplo/nao-verificar.md` — bloco marcado como exemplo",
		"```",
		"",
		"```json",
		"- `json/nao-existe.md` — bloco normal é verificado → L1",
		"```",
	}, "\n")), 0o644)
	os.MkdirAll(filepath.Join(ws, "docs"), 0o755)
	os.WriteFile(filepath.Join(ws, "docs", "guia.md"), []byte("guia\n"), 0o644)
	os.WriteFile(filepath.Join(ws, "README.md"), []byte("readme\n"), 0o644)
	os.WriteFile(filepath.Join(ws, "FORMAT.md"), []byte("formato\n"), 0o644)
	os.MkdirAll(filepath.Join(ws, "coredata", "rules"), 0o755)
	os.WriteFile(filepath.Join(ws, "coredata", "rules", "context7.md"), []byte("rule\n"), 0o644)

	rep := runLint(t, ws, t.TempDir()) // home vazio de config → sem runtime view
	l1 := findingsOf(rep, "L1")
	var dead []string
	for _, f := range l1 {
		dead = append(dead, strings.SplitN(f.Description, "`", 3)[1])
	}
	if len(l1) != 3 {
		t.Fatalf("L1 = %v (%d findings), want 3 (sub/nada-disso, precos, json/nao-existe)", dead, len(l1))
	}
	for _, want := range []string{"sub/nada-disso.md", "precos/nao-existe.md", "json/nao-existe.md"} {
		if !containsStr(dead, want) {
			t.Errorf("L1 sem a ref morta %s: %v", want, dead)
		}
	}
}

func TestL1RuntimeViewConfig(t *testing.T) {
	ws := t.TempDir() // workspace NÃO-kit → só runtime view
	home := t.TempDir()
	cfg := filepath.Join(home, ".config", "opencode")
	os.MkdirAll(filepath.Join(cfg, "profile"), 0o755)
	os.WriteFile(filepath.Join(cfg, "AGENTS.md"), []byte("- `profile/viva.md` — existe\n- `profile/morta.md` — não existe → L1\n"), 0o644)
	os.WriteFile(filepath.Join(cfg, "profile", "viva.md"), []byte("ok\n"), 0o644)

	// state pack com md citando arquivo do próprio pack — L1 de pack é
	// SKIP v0.1 (docs de negócio citam paths de produto por design)
	pack := filepath.Join(cfg, "packs", "org-pack")
	os.MkdirAll(pack, 0o755)
	os.WriteFile(filepath.Join(pack, "pack.yaml"), []byte("id: org-pack\nversion: 1\nname: Org\nkind: state\n"), 0o644)
	os.WriteFile(filepath.Join(pack, "guia.md"), []byte("- `<repo>/deploy.yml` — exemplo de produto, não verificado\n"), 0o644)
	os.MkdirAll(filepath.Join(pack, "rules"), 0o755)
	os.WriteFile(filepath.Join(pack, "rules", "org.md"), []byte("norma\n"), 0o644)

	rep := runLint(t, ws, home)
	l1 := findingsOf(rep, "L1")
	if len(l1) != 1 || l1[0].File != "AGENTS.md" {
		t.Fatalf("L1 = %+v, want 1 (AGENTS.md profile/morta — pack fora do scan v0.1)", l1)
	}
}

// ---- L3 ----

func TestL3OrphansWithAllowlistAndCited(t *testing.T) {
	ws := t.TempDir()
	home := t.TempDir()
	cfg := filepath.Join(home, ".config", "opencode")
	os.MkdirAll(filepath.Join(cfg, ".golmet"), 0o755)
	os.MkdirAll(filepath.Join(cfg, "plugins"), 0o755)
	// gerido pelo manifest
	os.WriteFile(filepath.Join(cfg, "skills", "gm-a", "SKILL.md"), []byte("---\nname: gm-a\n---\n"), 0o644)
	os.MkdirAll(filepath.Join(cfg, "skills", "gm-a"), 0o755)
	os.WriteFile(filepath.Join(cfg, "skills", "gm-a", "SKILL.md"), []byte("skill\n"), 0o644)
	os.WriteFile(filepath.Join(cfg, ".golmet", "manifest.json"), []byte(
		`{"version":"1","updated_at":"2026-01-01T00:00:00Z","layers":{"core":{"source":"embedded"}},"files":[{"file":"skills/gm-a/SKILL.md","sha256":"x","layer":"core"}]}`), 0o644)
	// pessoal-por-design
	os.WriteFile(filepath.Join(cfg, "profile", "client-profile.md"), []byte("dado\n"), 0o644)
	os.WriteFile(filepath.Join(cfg, ".golmet", "personal.json"), []byte(`{"version":1,"paths":["profile/client-profile.md"]}`), 0o644)
	// allowlist
	os.WriteFile(filepath.Join(cfg, "plugins", "ext.dll"), []byte("bin\n"), 0o644)
	// settings raiz allowlist
	os.WriteFile(filepath.Join(cfg, "service.json"), []byte("{}\n"), 0o644)
	// citado pelo AGENTS.md
	os.WriteFile(filepath.Join(cfg, "guia.md"), []byte("citado\n"), 0o644)
	os.WriteFile(filepath.Join(cfg, "AGENTS.md"), []byte("- `guia.md` — leia\n"), 0o644)
	// órfão de verdade
	os.WriteFile(filepath.Join(cfg, "health.json"), []byte("{}\n"), 0o644)
	os.WriteFile(filepath.Join(cfg, "solto.md"), []byte("solto\n"), 0o644)

	rep := runLint(t, ws, home)
	l3 := findingsOf(rep, "L3")
	var files []string
	for _, f := range l3 {
		files = append(files, f.File)
	}
	if len(l3) != 2 || !containsStr(files, "solto.md") || !containsStr(files, "health.json") {
		t.Fatalf("L3 = %v, want exatamente solto.md + health.json", files)
	}
	for _, f := range l3 {
		if f.Severity != "warning" {
			t.Errorf("órfão deveria ser warning: %+v", f)
		}
	}
}

// ---- L4 ----

func TestL4ManifestStale(t *testing.T) {
	ws := t.TempDir()
	home := t.TempDir()
	cfg := filepath.Join(home, ".config", "opencode")
	os.MkdirAll(filepath.Join(cfg, ".golmet", "skills", "gm-a"), 0o755)
	os.WriteFile(filepath.Join(cfg, ".golmet", "skills", "gm-a", "SKILL.md"), []byte("ok\n"), 0o644)
	os.WriteFile(filepath.Join(cfg, ".golmet", "manifest.json"), []byte(
		`{"version":"1","updated_at":"2026-01-01T00:00:00Z","layers":{"core":{"source":"embedded"}},"files":[{"file":".golmet/skills/gm-a/SKILL.md","sha256":"x","layer":"core"},{"file":"rules/packs/org.md","sha256":"y","layer":"core"}]}`), 0o644)

	rep := runLint(t, ws, home)
	l4 := findingsOf(rep, "L4")
	if len(l4) != 1 || l4[0].Severity != "error" || l4[0].File != "rules/packs/org.md" {
		t.Fatalf("L4 = %+v, want 1 error rules/packs/org.md (stale)", l4)
	}
}

// ---- L5 ----

func TestL5NamesPTAndNonKebab(t *testing.T) {
	ws := t.TempDir()
	home := t.TempDir()
	cfg := filepath.Join(home, ".config", "opencode")
	os.MkdirAll(filepath.Join(cfg, "Assets"), 0o755) // dir 1º nível fora do kebab
	pack := filepath.Join(cfg, "packs", "org-pack")
	os.MkdirAll(filepath.Join(pack, "rules"), 0o755)
	os.MkdirAll(filepath.Join(pack, "accumulation", "patterns", "processo", "Padrão X"), 0o755)
	os.MkdirAll(filepath.Join(pack, "accumulation", "nuances"), 0o755)
	os.WriteFile(filepath.Join(pack, "pack.yaml"), []byte("id: org-pack\nversion: 1\nname: Org\nkind: state\n"), 0o644)
	os.WriteFile(filepath.Join(pack, "rules", "Norma Deploy.md"), []byte("norma\n"), 0o644)
	os.WriteFile(filepath.Join(pack, "accumulation", "patterns", "processo", "Padrão X", "PATTERN.md"), []byte("p\n"), 0o644)
	os.WriteFile(filepath.Join(pack, "accumulation", "nuances", "Reengenharia-nuances.md"), []byte("n\n"), 0o644)

	rep := runLint(t, ws, home)
	l5 := findingsOf(rep, "L5")
	var files []string
	for _, f := range l5 {
		files = append(files, f.File)
	}
	// 4 findings: Assets, Norma Deploy.md (kebab), Padrão X (slug kebab),
	// Reengenharia (tema kebab) — a taxonomia PT (processo) cedeu ao L10 (M20)
	if len(l5) != 4 {
		t.Fatalf("L5 = %v (%d), want 4 (Assets, Norma Deploy, Padrão X, Reengenharia — tipo processo agora é L10)", files, len(l5))
	}
	for _, want := range []string{
		"Assets",
		"packs/org-pack/rules/Norma Deploy.md",
		"packs/org-pack/accumulation/patterns/processo/Padrão X",
		"packs/org-pack/accumulation/nuances/Reengenharia-nuances.md",
	} {
		if !containsStr(files, want) {
			t.Errorf("L5 sem %s: %v", want, files)
		}
	}
	// sem duplicação: o tipo PT é do L10 (teste completo em TestL10PackState)
}

// ---- L6 ----

func TestL6GhostsByContext(t *testing.T) {
	ws := kitFixture(t)
	os.WriteFile(filepath.Join(ws, "coredata", "sources", "gm-a.md"), []byte("usa `gm-fantasma` e `gm-a` ok\n"), 0o644)
	os.MkdirAll(filepath.Join(ws, "docs"), 0o755)
	os.WriteFile(filepath.Join(ws, "docs", "historia.md"), []byte("no passado o `gm-antigo` existia\n"), 0o644)

	rep := runLint(t, ws, t.TempDir())
	l6 := findingsOf(rep, "L6")
	if len(l6) != 2 {
		t.Fatalf("L6 = %+v, want 2 (gm-fantasma error em source, gm-antigo warning em docs)", l6)
	}
	for _, f := range l6 {
		if f.File == "coredata/sources/gm-a.md" && f.Severity != "error" {
			t.Errorf("fantasma em source deveria ser error: %+v", f)
		}
		if f.File == "docs/historia.md" && f.Severity != "warning" {
			t.Errorf("fantasma em docs deveria ser warning: %+v", f)
		}
	}

	// allowlist declarativa silencia (com motivo) — save/restore
	ghostAllowlist["gm-antigo"] = "renomeada na onda M6 (histórico)"
	defer delete(ghostAllowlist, "gm-antigo")
	rep2 := runLint(t, ws, t.TempDir())
	if len(findingsOf(rep2, "L6")) != 1 {
		t.Fatalf("L6 pós-allowlist = %+v, want só gm-fantasma", findingsOf(rep2, "L6"))
	}
	// allowlist em uso é SUPERFICIADO (L6-allowlist) — exceção nunca é
	// silêncio invisível
	al := findingsOf(rep2, "L6-allowlist")
	if len(al) != 1 || !strings.Contains(al[0].Description, "gm-antigo") {
		t.Fatalf("L6-allowlist = %+v, want 1 citando gm-antigo", al)
	}
}

func TestL6ScopeRegistryGoldensSkills(t *testing.T) {
	ws := kitFixture(t)
	// fantasma citado na descrição DENTRO do registry
	os.WriteFile(filepath.Join(ws, "coredata", "registry.yaml"), []byte(
		"version: pilot-1\nitems:\n  - id: gm-a\n    kind: skill\n    description: usa o `gm-fantasma-reg`\n    body: sources/gm-a.md\n"), 0o644)
	// fantasma em golden de render (árvore aninhada)
	os.MkdirAll(filepath.Join(ws, "internal", "core", "content", "testdata", "golden"), 0o755)
	os.WriteFile(filepath.Join(ws, "internal", "core", "content", "testdata", "golden", "x.md"), []byte("render citando `gm-fantasma-gold`\n"), 0o644)
	// fantasma em skill BASE (subdir — o walker do L6 é recursivo)
	os.MkdirAll(filepath.Join(ws, "skills", "gm-b"), 0o755)
	os.WriteFile(filepath.Join(ws, "skills", "gm-b", "SKILL.md"), []byte("delega para `gm-fantasma-skill`\n"), 0o644)

	rep := runLint(t, ws, t.TempDir())
	l6 := findingsOf(rep, "L6")
	if len(l6) != 3 {
		t.Fatalf("L6 = %+v, want 3 (registry, golden, skill)", l6)
	}
	files := map[string]bool{}
	for _, f := range l6 {
		if f.Severity != "error" {
			t.Errorf("escopo gerido deveria ser error: %+v", f)
		}
		files[f.File] = true
	}
	for _, want := range []string{
		"coredata/registry.yaml",
		filepath.ToSlash(filepath.Join("internal", "core", "content", "testdata", "golden", "x.md")),
		filepath.ToSlash(filepath.Join("skills", "gm-b", "SKILL.md")),
	} {
		if !files[want] {
			t.Errorf("L6 deveria cobrir %s: %+v", want, l6)
		}
	}
}

func TestL6AllowlistSurfacesInUse(t *testing.T) {
	ws := kitFixture(t)
	os.WriteFile(filepath.Join(ws, "coredata", "sources", "gm-a.md"), []byte("menciona `gm-legado` duas vezes: `gm-legado`\n"), 0o644)
	ghostAllowlist["gm-legado"] = "renomeada na onda M6 (motivo)"
	defer delete(ghostAllowlist, "gm-legado")

	rep := runLint(t, ws, t.TempDir())
	if got := findingsOf(rep, "L6"); len(got) != 0 {
		t.Fatalf("L6 deveria silenciar gm-legado: %+v", got)
	}
	al := findingsOf(rep, "L6-allowlist")
	if len(al) != 1 {
		t.Fatalf("L6-allowlist = %+v, want 1 (agregado por nome)", al)
	}
	if !strings.Contains(al[0].Description, "gm-legado") || !strings.Contains(al[0].Description, "2 ocorrência") {
		t.Errorf("descrição deveria citar nome e contagem: %+v", al[0])
	}
}

// ---- L7 ----

func TestL7CanonicalStructure(t *testing.T) {
	// pack state com dir não canônico
	home := t.TempDir()
	cfg := filepath.Join(home, ".config", "opencode")
	pack := filepath.Join(cfg, "packs", "org-pack")
	os.MkdirAll(filepath.Join(pack, "uploads"), 0o755)
	os.MkdirAll(filepath.Join(pack, "accumulation", "downloads"), 0o755)
	os.WriteFile(filepath.Join(pack, "pack.yaml"), []byte("id: org-pack\nversion: 1\nname: Org\nkind: state\n"), 0o644)

	rep := runLint(t, t.TempDir(), home)
	l7 := findingsOf(rep, "L7")
	if len(l7) != 2 || l7[0].Severity != "error" {
		t.Fatalf("L7 = %+v, want 2 error (uploads, downloads)", l7)
	}

	// kit: coredata com dir extra + sources com subdir + skills != 16
	ws := kitFixture(t)
	os.MkdirAll(filepath.Join(ws, "coredata", "extra"), 0o755)
	os.MkdirAll(filepath.Join(ws, "coredata", "sources", "aninhado"), 0o755)
	os.MkdirAll(filepath.Join(ws, "skills"), 0o755)
	for i := 0; i < 15; i++ {
		d := filepath.Join(ws, "skills", "gm-"+string(rune('a'+i)))
		os.MkdirAll(d, 0o755)
		os.WriteFile(filepath.Join(d, "SKILL.md"), []byte("---\nname: x\n---\n"), 0o644)
	}
	rep2 := runLint(t, ws, t.TempDir())
	l7kit := findingsOf(rep2, "L7")
	if len(l7kit) != 3 {
		t.Fatalf("L7 kit = %+v, want 3 (coredata/extra, sources/aninhado, skills != 16)", l7kit)
	}
}

// ---- integração: sandbox limpo = zero findings; workspace inválido = erro ----

func TestLintCleanSandboxAndInvalidWorkspace(t *testing.T) {
	ws := kitFixture(t)
	os.WriteFile(filepath.Join(ws, "coredata", "sources", "gm-a.md"), []byte("- `gm-a.md` no registry e `FORMAT.md` na raiz\n"), 0o644)
	os.WriteFile(filepath.Join(ws, "FORMAT.md"), []byte("formato\n"), 0o644)
	rep := runLint(t, ws, t.TempDir())
	if rep.Errors != 0 || rep.Warnings != 0 {
		t.Fatalf("sandbox limpo deveria zerar findings: %+v", rep.Findings)
	}
	if _, err := Run(Options{Workspace: filepath.Join(t.TempDir(), "nada"), Home: t.TempDir()}); err == nil {
		t.Errorf("workspace inexistente deveria erro")
	}
}

// ---- L8 — conformidade estrutural definitivo ↔ template ----

func TestL8StructuralConformance(t *testing.T) {
	home := t.TempDir()
	cfg := filepath.Join(home, ".config", "opencode")

	// definitivo CONFORMANTE: 5 blocos abertos+fechados + seções mínimas +
	// seção EXTRA (evolução legítima — nunca finding)
	agents := strings.Join([]string{
		// token ÚNICO repetido (convenção da casa para azure-devops) —
		// ≥2 delimitadores = aberto e fechado
		"<!-- azure-devops -->", "tracker", "<!-- azure-devops -->",
		"<!-- org-pack -->", "org", "<!-- /org-pack -->",
		"### Tools MCP do kit (servidor `golmet`)", "13 tools",
		"<!-- projeto-padrao -->", "dir", "<!-- /projeto-padrao -->",
		"<!-- gm-pack-accumulation -->", "pack", "<!-- /gm-pack-accumulation -->",
		"## Índice de Referências — lazy loading",
		"<!-- knowledge-routing -->", "T1-T5", "<!-- /knowledge-routing -->",
		"## Seção Extra que o definitivo ganhou (evolução — nunca finding)",
	}, "\n")
	os.MkdirAll(filepath.Join(cfg, "profile"), 0o755)
	os.WriteFile(filepath.Join(cfg, "AGENTS.md"), []byte(agents), 0o644)
	os.WriteFile(filepath.Join(cfg, "profile", "client-profile.md"), []byte(
		strings.Join([]string{
			"## Tracker (estrutura migrada da org — não é mínimo)",
			"## Identidade e infraestrutura (IMP-105)", "| org_github | x |", "| ghcr_registry | y |", "| dns_domain | z |",
			"## Packs (IMP-111)", "| packs | a |",
			"## Seção extra do cliente (evolução — ok)",
		}, "\n")), 0o644)
	os.WriteFile(filepath.Join(cfg, "profile", "operator-profile.md"), []byte(
		strings.Join([]string{
			"## Identidade", "| Nome | x |",
			"## Identidade git e contas", "| git user.name | a |", "| git user.email | b |", "| username | c |",
		}, "\n")), 0o644)

	rep := runLint(t, t.TempDir(), home)
	if l8 := findingsOf(rep, "L8"); len(l8) != 0 {
		t.Fatalf("definitivos conformantes deveriam zerar L8: %+v", l8)
	}

	// definitivo que PERDE estrutura: bloco não fechado + seção/campos ausentes
	os.WriteFile(filepath.Join(cfg, "AGENTS.md"), []byte(
		strings.Join([]string{
			"<!-- azure-devops -->", "sem fechamento",
			"### Tools MCP do kit",
		}, "\n")), 0o644)
	os.WriteFile(filepath.Join(cfg, "profile", "client-profile.md"), []byte("sem estrutura\n"), 0o644)
	rep2 := runLint(t, t.TempDir(), home)
	l8 := findingsOf(rep2, "L8")
	if len(l8) == 0 {
		t.Fatalf("definitivo sem estrutura mínima deveria flagar L8")
	}
	agentsFlags, clientFlags := 0, 0
	for _, f := range l8 {
		if f.Severity != "warning" {
			t.Errorf("L8 é warning: %+v", f)
		}
		switch f.File {
		case "AGENTS.md":
			agentsFlags++
		case "profile/client-profile.md":
			clientFlags++
		default:
			t.Errorf("L8 inesperado em %s", f.File)
		}
	}
	if agentsFlags < 2 || clientFlags < 1 { // bloco não fechado + seção Índice; client sem estrutura mínima
		t.Fatalf("flags por arquivo insuficientes: %+v", l8)
	}

	// operator-profile CONFORMANTE não flaga (isolando o ruído do par acima)
	for _, f := range findingsOf(rep2, "L8") {
		if f.File == "profile/operator-profile.md" {
			t.Errorf("operator-profile conformante não deveria flagar: %+v", f)
		}
	}

	// definitivo AUSENTE (máquina nova antes do setup) → skip silencioso
	home2 := t.TempDir()
	rep4 := runLint(t, t.TempDir(), home2)
	if len(findingsOf(rep4, "L8")) != 0 {
		t.Fatalf("sem definitivos o L8 é silencioso: %+v", findingsOf(rep4, "L8"))
	}
}

// template depositado no config → L7 warning específico (M12)

func TestL7TemplateDepositFlag(t *testing.T) {
	home := t.TempDir()
	cfg := filepath.Join(home, ".config", "opencode")
	os.MkdirAll(cfg, 0o755)
	os.WriteFile(filepath.Join(cfg, "AGENTS.template.md"), []byte("template\n"), 0o644)
	rep := runLint(t, t.TempDir(), home)
	found := false
	for _, f := range findingsOf(rep, "L7") {
		if f.File == "AGENTS.template.md" {
			found = true
		}
	}
	if !found {
		t.Fatalf("L7 deveria flagar template depositado no config: %+v", findingsOf(rep, "L7"))
	}
}

// ---- L7 — agregados de spec do workspace: unitário | quebrado (§7.7.1) ----

func TestL7SpecAggregatesUnitaryAndBroken(t *testing.T) {
	ws := t.TempDir()
	os.MkdirAll(filepath.Join(ws, "specs"), 0o755)
	// unitário PEQUENO → sem warning
	os.WriteFile(filepath.Join(ws, "specs", "DATA_MODEL.md"), []byte(strings.Repeat("coluna\n", 10)), 0o644)
	rep := runLint(t, ws, t.TempDir())
	if l7 := findingsOf(rep, "L7"); len(l7) != 0 {
		t.Fatalf("unitário pequeno não deveria flagar: %+v", l7)
	}

	// unitário acima do limiar → warning apontando o split
	os.WriteFile(filepath.Join(ws, "specs", "DATA_MODEL.md"), []byte(strings.Repeat("coluna\n", DataModelSplitLines+1)), 0o644)
	rep2 := runLint(t, ws, t.TempDir())
	l7 := findingsOf(rep2, "L7-size")
	if len(l7) != 1 || l7[0].Severity != "warning" || l7[0].File != "specs/DATA_MODEL.md" {
		t.Fatalf("unitário grande = %+v, want 1 warning em specs/DATA_MODEL.md", l7)
	}
	if !strings.Contains(l7[0].Fix, "gm-datamodel Split") {
		t.Errorf("warning deveria remeter ao gm-datamodel Split: %+v", l7[0])
	}

	// split simulado (procedimento gm-datamodel): INDEX + escopos +
	// original removido → workspace limpo
	os.Remove(filepath.Join(ws, "specs", "DATA_MODEL.md"))
	dm := filepath.Join(ws, "specs", "data_model")
	os.MkdirAll(dm, 0o755)
	os.WriteFile(filepath.Join(dm, "INDEX.md"), []byte("# Modelo de Dados — índice\n- [agendamento](agendamento.md)\n- [financeiro](financeiro.md)\n"), 0o644)
	os.WriteFile(filepath.Join(dm, "agendamento.md"), []byte("# Agendamento\nTABELA_AGENDAMENTO — RN-001\n"), 0o644)
	os.WriteFile(filepath.Join(dm, "financeiro.md"), []byte("# Financeiro\nTABELA_PAGAMENTO — RN-002\n"), 0o644)
	rep3 := runLint(t, ws, t.TempDir())
	if l7 := findingsOf(rep3, "L7"); len(l7) != 0 {
		t.Fatalf("quebrado COM INDEX deveria ser limpo: %+v", l7)
	}
	if l7s := findingsOf(rep3, "L7-size"); len(l7s) != 0 {
		t.Fatalf("quebrado COM INDEX não deveria ter aviso de tamanho: %+v", l7s)
	}

	// merge no quebrado: entidade nova no escopo certo + escopo novo com
	// entrada no INDEX continuam limpos
	os.WriteFile(filepath.Join(dm, "clientes.md"), []byte("# Clientes\nTABELA_CLIENTE — RN-003\n"), 0o644)
	os.WriteFile(filepath.Join(dm, "INDEX.md"), []byte("# Modelo de Dados — índice\n- [agendamento](agendamento.md)\n- [financeiro](financeiro.md)\n- [clientes](clientes.md)\n"), 0o644)
	rep4 := runLint(t, ws, t.TempDir())
	if l7 := findingsOf(rep4, "L7"); len(l7) != 0 {
		t.Fatalf("escopo novo + INDEX atualizado deveria ser limpo: %+v", l7)
	}
	if l7s := findingsOf(rep4, "L7-size"); len(l7s) != 0 {
		t.Fatalf("escopo novo + INDEX atualizado não deveria ter aviso de tamanho: %+v", l7s)
	}

	// quebrado SEM INDEX → error com remediação do split
	os.Remove(filepath.Join(dm, "INDEX.md"))
	rep5 := runLint(t, ws, t.TempDir())
	l75 := findingsOf(rep5, "L7")
	if len(l75) != 1 || l75[0].Severity != "error" || l75[0].File != "specs/data_model" {
		t.Fatalf("quebrado sem INDEX = %+v, want 1 error em specs/data_model", l75)
	}
	if !strings.Contains(l75[0].Fix, "crie o INDEX") || !strings.Contains(l75[0].Fix, "gm-datamodel Split") {
		t.Errorf("remediação deveria citar o INDEX e o Split: %+v", l75[0])
	}
}

// ---- L7-size — aviso precoce dos 4 candidatos sem gramática declarada ----

func TestL7SizeWarningsEarlyCandidates(t *testing.T) {
	// limiares PINADOS (mudança de número quebra aqui de propósito)
	wantLines := map[string]int{
		"DECISIONS.md":            800,
		"specs/ERROR_CATALOG.md":  600,
		"support/KNOWN_ERRORS.md": 400,
		"GLOSSARY.md":             300,
	}
	for _, a := range aggregateSizeWarns {
		if a.Lines != wantLines[a.Paths[0]] {
			t.Errorf("limiar de %s = %d, want %d", a.Paths[0], a.Lines, wantLines[a.Paths[0]])
		}
	}

	cases := []struct {
		path  string
		lines int
	}{
		{"DECISIONS.md", 800},
		{"specs/ERROR_CATALOG.md", 600},
		{"support/KNOWN_ERRORS.md", 400},
		{"GLOSSARY.md", 300},
	}
	for _, tc := range cases {
		ws := t.TempDir()
		os.MkdirAll(filepath.Dir(filepath.Join(ws, tc.path)), 0o755)
		// abaixo do limiar → silêncio
		os.WriteFile(filepath.Join(ws, filepath.FromSlash(tc.path)), []byte(strings.Repeat("linha\n", tc.lines)), 0o644)
		rep := runLint(t, ws, t.TempDir())
		if n := len(findingsOf(rep, "L7-size")) + len(findingsOf(rep, "L7")); n != 0 {
			t.Fatalf("%s com %d linhas (no limiar) deveria silenciar: %+v", tc.path, tc.lines, rep.Findings)
		}
		// acima do limiar → warning L7-size com remediação de PLANEJAMENTO
		os.WriteFile(filepath.Join(ws, filepath.FromSlash(tc.path)), []byte(strings.Repeat("linha\n", tc.lines+1)), 0o644)
		rep2 := runLint(t, ws, t.TempDir())
		l7s := findingsOf(rep2, "L7-size")
		if len(l7s) != 1 {
			t.Fatalf("%s com %d linhas = %d avisos, want 1: %+v", tc.path, tc.lines+1, len(l7s), rep2.Findings)
		}
		f := l7s[0]
		if f.Severity != "warning" || f.File != tc.path {
			t.Errorf("aviso %s = %+v, want warning em %s", tc.path, f, tc.path)
		}
		if !strings.Contains(f.Fix, "gm-doc") || !strings.Contains(f.Fix, "ainda não está declarada no FORMAT") {
			t.Errorf("remediação deveria apontar planejamento com gm-doc (sem gramática declarada): %+v", f)
		}
	}

	// fallback de path: root legado do ERROR_CATALOG também dispara (1º
	// existente vale, sem duplicar findings)
	ws := t.TempDir()
	os.WriteFile(filepath.Join(ws, "ERROR_CATALOG.md"), []byte(strings.Repeat("linha\n", 601)), 0o644)
	rep := runLint(t, ws, t.TempDir())
	l7s := findingsOf(rep, "L7-size")
	if len(l7s) != 1 || l7s[0].File != "ERROR_CATALOG.md" {
		t.Fatalf("ERROR_CATALOG legado na raiz = %+v, want 1 aviso apontando a raiz", l7s)
	}
}

// ---- container de organização: L7 (hierarquia) + L1 (refs do container) ----

func TestL7L1Container(t *testing.T) {
	ws := t.TempDir()
	os.WriteFile(filepath.Join(ws, "golmet.yaml"), []byte("format: 1\ncontainer: true\nproducts: [ProdutoA]\n"), 0o644)
	// ref ancorada no legado docs/ (verificável): ok + morta → L1
	os.WriteFile(filepath.Join(ws, "CONSTITUTION.md"), []byte("- `docs/guia.md` ok e `docs/morta.md` morta → L1\n"), 0o644)
	os.MkdirAll(filepath.Join(ws, ".golmet", "meetings"), 0o755)
	os.MkdirAll(filepath.Join(ws, ".golmet", "migrations"), 0o755)
	// ref morta DENTRO da zona de trabalho → L1 NÃO varre .golmet/ (skip M19)
	os.WriteFile(filepath.Join(ws, ".golmet", "meetings", "checkpoint.md"), []byte("- `fantasma/inexistente.md` dentro da zona → sem opinião\n"), 0o644)
	os.MkdirAll(filepath.Join(ws, "ProdutoA"), 0o755)
	os.WriteFile(filepath.Join(ws, "AUDIT.md"), []byte("solto\n"), 0o644)

	rep := runLint(t, ws, t.TempDir())
	if !rep.Container {
		t.Errorf("Report.Container deveria ser true para golmet.yaml de container")
	}
	var missingProd, undeclared, loose, deadRef, zoneRef int
	for _, f := range rep.Findings {
		switch {
		case f.Check == "L7" && f.Severity == "error" && f.File == "ProdutoFantasma":
			missingProd++
		case f.Check == "L7" && f.Severity == "warning" && f.File == "ProdutoSolto":
			undeclared++
		case f.Check == "L7" && f.Severity == "warning" && f.File == "AUDIT.md":
			loose++
		case f.Check == "L1" && strings.Contains(f.Description, "docs/morta.md"):
			deadRef++
		case f.Check == "L1" && f.File == ".golmet/meetings/checkpoint.md":
			zoneRef++
		}
	}
	if missingProd != 0 || undeclared != 0 || loose != 1 || deadRef != 1 || zoneRef != 0 {
		t.Fatalf("findings inesperados: missingProd=%d undeclared=%d loose=%d deadRef=%d zoneRef=%d — %+v", missingProd, undeclared, loose, deadRef, zoneRef, rep.Findings)
	}

	// declarado ausente + produto não declarado + legados meetings/ e
	// migrations/ na raiz (M19 → mensagens de migração para a zona)
	os.Remove(filepath.Join(ws, "golmet.yaml"))
	os.WriteFile(filepath.Join(ws, "golmet.yaml"), []byte("format: 1\ncontainer: true\nproducts: [ProdutoA, ProdutoFantasma]\n"), 0o644)
	os.MkdirAll(filepath.Join(ws, "ProdutoSolto"), 0o755)
	os.MkdirAll(filepath.Join(ws, "meetings"), 0o755)
	os.MkdirAll(filepath.Join(ws, "migrations"), 0o755)
	rep2 := runLint(t, ws, t.TempDir())
	l7 := findingsOf(rep2, "L7")
	var errs, prodWarns, legacyWarns int
	for _, f := range l7 {
		if f.Severity == "error" && f.File == "ProdutoFantasma" {
			errs++
		}
		if f.Severity == "warning" && f.File == "ProdutoSolto" {
			prodWarns++
		}
		if f.Severity == "warning" && (f.File == "meetings" || f.File == "migrations") && strings.Contains(f.Fix, ".golmet/") {
			legacyWarns++
		}
	}
	if errs != 1 || prodWarns != 1 || legacyWarns != 2 {
		t.Fatalf("L7 = %+v, want 1 error (ProdutoFantasma) + 1 warning (ProdutoSolto) + 2 warnings de migração (meetings/migrations → .golmet/)", l7)
	}

	// workspace-produto (golmet.yaml sem container) NÃO ganha checks de container
	wsProd := t.TempDir()
	os.WriteFile(filepath.Join(wsProd, "golmet.yaml"), []byte("format: 1\n"), 0o644)
	rep3 := runLint(t, wsProd, t.TempDir())
	if rep3.Container {
		t.Errorf("workspace-produto detectado como container")
	}
	if len(findingsOf(rep3, "L7")) != 0 || len(findingsOf(rep3, "L1")) != 0 {
		t.Fatalf("workspace-produto vazio deveria zerar L7/L1: %+v", rep3.Findings)
	}
}

// L1 do container resolve o CONFIG ROOT do engine quando conhecido:
// refs ao runtime do agente (profile/client-profile.md) existem no config,
// não no container — não são refs mortas.

func TestL1ContainerResolvesEngineConfigRoot(t *testing.T) {
	ws := t.TempDir()
	os.WriteFile(filepath.Join(ws, "golmet.yaml"), []byte("format: 1\ncontainer: true\nproducts: []\n"), 0o644)
	// docs/ legado continua varrido pelo L1 (transição M18) — é onde a ref
	// ao runtime do agente aparece
	os.MkdirAll(filepath.Join(ws, "docs"), 0o755)
	os.WriteFile(filepath.Join(ws, "docs", "nota.md"), []byte(
		"- `profile/client-profile.md` — existe no config do engine\n"+
			"- `profile/fantasma.md` — não existe nem no config nem no container → L1\n"), 0o644)

	// config root COM o arquivo referido
	home := t.TempDir()
	cfg := filepath.Join(home, ".config", "opencode")
	os.MkdirAll(filepath.Join(cfg, "profile"), 0o755)
	os.WriteFile(filepath.Join(cfg, "profile", "client-profile.md"), []byte("perfil\n"), 0o644)

	rep := runLint(t, ws, home)
	l1 := findingsOf(rep, "L1")
	if len(l1) != 1 || !strings.Contains(l1[0].Description, "profile/fantasma.md") {
		t.Fatalf("L1 = %+v, want só profile/fantasma.md (client-profile resolve no config)", l1)
	}

	// sem config root na máquina (home vazia) → a ref de config vira finding
	// (o contexto não pode verificar — caminho continua declarado como morto)
	rep2 := runLint(t, ws, t.TempDir())
	l12 := findingsOf(rep2, "L1")
	if len(l12) != 2 {
		t.Fatalf("sem config root conhecido as 2 refs deveriam flagar: %+v", l12)
	}
}

// ---- L9 — schema do registry de stack-templates (M14) ----

// l9Valid é um registry CONFORMANTE: boilerplate_org placeholder (aceito —
// IMP-105), template completo (.NET) e template placeholder (boilerplate
// null, sem description — aceito).
const l9Valid = `# Stack Templates Registry
boilerplate_org: <do client-profile — campo boilerplate_org (IMP-105)>

templates:
  .NET:
    boilerplate: <boilerplate_org>/Boilerplate.NET-API
    detect: ["*.csproj", "*.sln"]
    description: |
      Clean Architecture .NET + CQRS/MediatR + PostgreSQL + Docker.
    post_setup: |
      # 1. Clona boilerplate
      # 2. Roda rename-project.ps1
  Angular:
    boilerplate: null      # não existe ainda (a criar na org)
    detect: ["angular.json"]
    description: "Angular MFE — a criar boilerplate"
`

func l9Write(t *testing.T, content string) (ws, home string) {
	t.Helper()
	home = t.TempDir()
	cfg := filepath.Join(home, ".config", "opencode", "profile")
	os.MkdirAll(cfg, 0o755)
	os.WriteFile(filepath.Join(cfg, "stack-templates.yaml"), []byte(content), 0o644)
	return t.TempDir(), home
}

func TestL9StackTemplatesSchema(t *testing.T) {
	// (a)-(e) conformante → zero L9; by_check registra L9 apenas com findings
	ws, home := l9Write(t, l9Valid)
	rep := runLint(t, ws, home)
	if l9 := findingsOf(rep, "L9"); len(l9) != 0 {
		t.Fatalf("registry conforme deveria zerar L9: %+v", l9)
	}
	if rep.ByCheck["L9"] != 0 {
		t.Errorf("by_check L9 = %d, want 0", rep.ByCheck["L9"])
	}

	// arquivo AUSENTE (máquina nova antes do install) → skip silencioso
	rep0 := runLint(t, t.TempDir(), t.TempDir())
	if l9 := findingsOf(rep0, "L9"); len(l9) != 0 {
		t.Fatalf("sem stack-templates.yaml o L9 é silencioso: %+v", l9)
	}

	tt := map[string]struct {
		content string
		want    string // substring esperada na description do finding
	}{
		"yaml quebrado":       {"boilerplate_org: [unclosed\n  templates: {", "não é YAML parseável"},
		"arquivo vazio":       {"\n", "vazio"},
		"sem boilerplate_org": {"templates:\n  .NET:\n    boilerplate: null\n    detect: [\"x\"]\n", "boilerplate_org"},
		"boilerplate_org vazio": {"boilerplate_org: \"\"\ntemplates:\n  .NET:\n    boilerplate: null\n    detect: [\"x\"]\n",
			"boilerplate_org"},
		"sem templates": {"boilerplate_org: Acme\n", "templates"},
		"template sem detect": {"boilerplate_org: Acme\ntemplates:\n  .NET:\n    boilerplate: null\n",
			"detect"},
		"detect vazio": {"boilerplate_org: Acme\ntemplates:\n  .NET:\n    boilerplate: null\n    detect: []\n",
			"detect"},
		"sem chave boilerplate": {"boilerplate_org: Acme\ntemplates:\n  .NET:\n    detect: [\"x\"]\n    description: d\n",
			"chave `boilerplate`"},
		"sem description": {"boilerplate_org: Acme\ntemplates:\n  .NET:\n    boilerplate: Acme/Boilerplate.NET-API\n    detect: [\"x\"]\n",
			"description"},
	}
	for name, tc := range tt {
		ws, home := l9Write(t, tc.content)
		rep := runLint(t, ws, home)
		l9 := findingsOf(rep, "L9")
		if len(l9) != 1 {
			t.Errorf("%s: want 1 finding L9, got %d: %+v", name, len(l9), l9)
			continue
		}
		f := l9[0]
		if f.Severity != "error" {
			t.Errorf("%s: L9 é error, got %s", name, f.Severity)
		}
		if f.File != "profile/stack-templates.yaml" {
			t.Errorf("%s: file = %s, want profile/stack-templates.yaml", name, f.File)
		}
		if !strings.Contains(f.Description, tc.want) {
			t.Errorf("%s: description %q não contém %q", name, f.Description, tc.want)
		}
		if f.Fix == "" {
			t.Errorf("%s: finding sem remediação", name)
		}
	}

	// múltiplas violações num mesmo registry → um finding por template
	multi := "boilerplate_org: Acme\ntemplates:\n" +
		"  A:\n    detect: [\"x\"]\n" + // sem chave boilerplate
		"  B:\n    boilerplate: Acme/B\n    detect: []\n    description: d\n" + // detect vazio
		"  C:\n    boilerplate: Acme/C\n    detect: [\"y\"]\n" // sem description
	ws, home = l9Write(t, multi)
	repM := runLint(t, ws, home)
	if l9 := findingsOf(repM, "L9"); len(l9) != 3 {
		t.Fatalf("registry com 3 violações: want 3 findings L9, got %d: %+v", len(l9), l9)
	}
}

// ---- L9 — registries de pack (M15: registry da org IN-PLACE no pack) ----

// l9WritePack grava o registry no working copy do pack no config
// (`<home>/.config/opencode/packs/techminds/rules/stack-templates.yaml`).
func l9WritePack(t *testing.T, content string) (ws, home string) {
	t.Helper()
	home = t.TempDir()
	pack := filepath.Join(home, ".config", "opencode", "packs", "techminds", "rules")
	os.MkdirAll(pack, 0o755)
	os.WriteFile(filepath.Join(pack, "stack-templates.yaml"), []byte(content), 0o644)
	return t.TempDir(), home
}

func TestL9PackRegistrySchema(t *testing.T) {
	// registry de pack CONFORME → zero L9 (mesmo schema do kit)
	ws, home := l9WritePack(t, l9Valid)
	rep := runLint(t, ws, home)
	if l9 := findingsOf(rep, "L9"); len(l9) != 0 {
		t.Fatalf("registry de pack conforme deveria zerar L9: %+v", l9)
	}

	// pack SEM registry (e sem profile) → skip silencioso (estado legítimo)
	rep0 := runLint(t, t.TempDir(), t.TempDir())
	if l9 := findingsOf(rep0, "L9"); len(l9) != 0 {
		t.Fatalf("sem registry nenhum o L9 é silencioso: %+v", l9)
	}

	// violação no registry do pack → finding L9 apontando o arquivo DO pack,
	// severity error, com remediação apontando o pack publish
	broken := "boilerplate_org: Acme\ntemplates:\n  Rust:\n    boilerplate: null\n"
	ws, home = l9WritePack(t, broken)
	rep = runLint(t, ws, home)
	l9 := findingsOf(rep, "L9")
	if len(l9) != 1 {
		t.Fatalf("pack registry violado: want 1 finding L9, got %d: %+v", len(l9), l9)
	}
	f := l9[0]
	if f.File != "packs/techminds/rules/stack-templates.yaml" {
		t.Errorf("file = %s, want packs/techminds/rules/stack-templates.yaml", f.File)
	}
	if f.Severity != "error" {
		t.Errorf("L9 de pack é error, got %s", f.Severity)
	}
	if !strings.Contains(f.Description, "detect") {
		t.Errorf("description %q não menciona detect", f.Description)
	}
	if f.Fix == "" {
		t.Errorf("finding sem remediação")
	}

	// violação de RAIZ no registry do pack: yaml quebrado → remediação aponta
	// o working copy do pack; arquivo vazio → remediação com golmet pack publish
	ws, home = l9WritePack(t, "boilerplate_org: [unclosed\n")
	rep = runLint(t, ws, home)
	l9 = findingsOf(rep, "L9")
	if len(l9) != 1 {
		t.Fatalf("pack yaml quebrado: want 1 finding L9, got %d: %+v", len(l9), l9)
	}
	if !strings.Contains(l9[0].Fix, "packs/techminds/") {
		t.Errorf("fix %q deveria apontar o working copy do pack", l9[0].Fix)
	}
	ws, home = l9WritePack(t, "\n")
	rep = runLint(t, ws, home)
	l9 = findingsOf(rep, "L9")
	if len(l9) != 1 {
		t.Fatalf("pack registry vazio: want 1 finding L9, got %d: %+v", len(l9), l9)
	}
	if !strings.Contains(l9[0].Fix, "packs/techminds/") || !strings.Contains(l9[0].Fix, "pack publish") {
		t.Errorf("fix %q deveria apontar o working copy do pack + pack publish", l9[0].Fix)
	}

	// registry do kit E do pack quebrados ao mesmo tempo → 2 findings
	// independentes (o pack NÃO mascara o kit nem vice-versa)
	home2 := t.TempDir()
	prof := filepath.Join(home2, ".config", "opencode", "profile")
	os.MkdirAll(prof, 0o755)
	os.WriteFile(filepath.Join(prof, "stack-templates.yaml"), []byte("boilerplate_org: [unclosed\n"), 0o644)
	os.MkdirAll(filepath.Join(home2, ".config", "opencode", "packs", "techminds", "rules"), 0o755)
	os.WriteFile(filepath.Join(home2, ".config", "opencode", "packs", "techminds", "rules", "stack-templates.yaml"), []byte(broken), 0o644)
	rep2 := runLint(t, t.TempDir(), home2)
	l9both := findingsOf(rep2, "L9")
	if len(l9both) != 2 {
		t.Fatalf("kit+pack quebrados: want 2 findings L9, got %d: %+v", len(l9both), l9both)
	}
	files := []string{l9both[0].File, l9both[1].File}
	sort.Strings(files)
	if files[0] != "packs/techminds/rules/stack-templates.yaml" || files[1] != "profile/stack-templates.yaml" {
		t.Errorf("files = %v, want [packs/techminds/..., profile/...]", files)
	}
}

// ---- L11 - diagramas Mermaid ----

// TestL11Mermaid cobre as heurísticas do check L11 via lintMermaidFile
// direto (função package-private): válidos passam sem finding; os erros
// comuns (tipo inválido, delimitadores desbalanceados, aspas ímpar, seta
// órfã, subgraph/end, fence aberto) flagam; construções exóticas válidas
// (sequenceDiagram ->>, C4*, comentários) NÃO flagam (anti-falso-positivo).
func TestL11Mermaid(t *testing.T) {
	findings := func(content string) []Finding {
		t.Helper()
		rep := &Report{Findings: []Finding{}, ByCheck: map[string]int{}}
		lintMermaidFile(rep, "doc.md", content)
		return rep.Findings
	}
	l11 := func(fs []Finding) []Finding {
		var out []Finding
		for _, f := range fs {
			if f.Check == "L11" {
				out = append(out, f)
			}
		}
		return out
	}

	// válido: flowchart com labels fechados, aspas pares, subgraph/end
	if got := l11(findings("```mermaid\nflowchart TD\n  %% comentario\n  A[Entrada] -->|texto| B(Fim)\n  subgraph s1 [grupo]\n    B --> C{Decisão}\n  end\n```\n")); len(got) != 0 {
		t.Fatalf("flowchart válido flagado: %+v", got)
	}
	// tolerado: sequenceDiagram com ->> (pontas de sequence NÃO validadas) e C4
	if got := l11(findings("```mermaid\nsequenceDiagram\n  Alice->>Bob: oi\n```\n")); len(got) != 0 {
		t.Fatalf("sequenceDiagram válido flagado: %+v", got)
	}
	// tolerado: sequenceDiagram com blocos alt...end (o opener par com o end)
	if got := l11(findings("```mermaid\nsequenceDiagram\n  A->>B: ok\n  alt contradição\n    A-->>B: 400\n  end\n```\n")); len(got) != 0 {
		t.Fatalf("sequenceDiagram alt/end válido flagado: %+v", got)
	}
	// tolerado: erDiagram com attribute-blocks multilinha E cardinalidade
	// }o--|| (falso-positivo real pego na 1ª execução sobre o gm-datamodel)
	if got := l11(findings("```mermaid\nerDiagram\n  TABELA {\n    string id PK\n  }\n  TABELA }o--|| TABELA2 : \"fk\"\n```\n")); len(got) != 0 {
		t.Fatalf("erDiagram válido flagado: %+v", got)
	}
	if got := l11(findings("```mermaid\nC4Context\n  Person(user, \"Usuário\")\n```\n")); len(got) != 0 {
		t.Fatalf("C4 válido flagado: %+v", got)
	}
	// tipo inválido na 1ª linha
	if got := l11(findings("```mermaid\nfoobar TD\n  A --> B\n```\n")); len(got) != 1 {
		t.Fatalf("tipo inválido: want 1 finding, got %d: %+v", len(got), got)
	}
	// delimitadores desbalanceados (colchete aberto)
	if got := l11(findings("```mermaid\nflowchart TD\n  A[Entrada --> B\n```\n")); len(got) != 1 {
		t.Fatalf("colchete desbalanceado: want 1 finding, got %d: %+v", len(got), got)
	}
	// aspas ímpar
	if got := l11(findings("```mermaid\nflowchart TD\n  A[\"texto --> B\n```\n")); len(got) != 1 {
		t.Fatalf("aspas ímpar: want 1 finding, got %d: %+v", len(got), got)
	}
	// seta sem destino
	if got := l11(findings("```mermaid\nflowchart TD\n  A -->\n```\n")); len(got) != 1 {
		t.Fatalf("seta sem destino: want 1 finding, got %d: %+v", len(got), got)
	}
	// subgraph sem end
	if got := l11(findings("```mermaid\nflowchart TD\n  subgraph s1\n    A --> B\n```\n")); len(got) != 1 {
		t.Fatalf("subgraph sem end: want 1 finding, got %d: %+v", len(got), got)
	}
	// fence aberto sem fechamento
	if got := l11(findings("```mermaid\nflowchart TD\n  A --> B\n")); len(got) != 1 {
		t.Fatalf("fence aberto: want 1 finding, got %d: %+v", len(got), got)
	}
	// bloco vazio
	if got := l11(findings("```mermaid\n```\n")); len(got) != 1 {
		t.Fatalf("bloco vazio: want 1 finding, got %d: %+v", len(got), got)
	}
	// Line aponta a linha absoluta do problema
	rep := &Report{Findings: []Finding{}, ByCheck: map[string]int{}}
	lintMermaidFile(rep, "doc.md", "texto antes\n\n```mermaid\nflowchart TD\n  A -->\n```\n")
	if len(rep.Findings) != 1 || rep.Findings[0].Line != 5 {
		t.Fatalf("Line = %+v, want linha 5 (a seta órfã)", rep.Findings)
	}
}

// ---- L12 - encoding ----

// TestL12Encoding cobre as heurísticas do check L12 via l12CheckFile
// direto: UTF-8 limpo passa; UTF-16/BOM UTF-16, bytes não-UTF-8, U+FFFD e
// dupla codificação (Ã©) flagam error; BOM UTF-8 flaga warning.
func TestL12Encoding(t *testing.T) {
	findings := func(data []byte) []Finding {
		t.Helper()
		rep := &Report{Findings: []Finding{}, ByCheck: map[string]int{}}
		l12CheckFile(rep, "doc.md", data)
		return rep.Findings
	}
	clean := []byte("# Título em UTF-8\n\n- linha com acento: ação, referência, ∑\n")
	if got := findings(clean); len(got) != 0 {
		t.Fatalf("UTF-8 limpo flagado: %+v", got)
	}
	// UTF-16 BOM
	u16 := append([]byte{0xFF, 0xFE}, []byte("x\x00")...)
	if got := findings(u16); len(got) != 1 || got[0].Severity != "error" {
		t.Fatalf("UTF-16: want 1 error, got %+v", got)
	}
	// BOM UTF-8 (warning) — corpo válido limpo
	bom := append([]byte{0xEF, 0xBB, 0xBF}, []byte("# ok\n")...)
	if got := findings(bom); len(got) != 1 || got[0].Severity != "warning" {
		t.Fatalf("BOM UTF-8: want 1 warning, got %+v", got)
	}
	// bytes não-UTF-8 (Latin-1 cru: "ação" em CP1252)
	raw1252 := []byte{0x61, 0xE7, 0xE3, 0x6F} // "açao" em latin-1
	if got := findings(raw1252); len(got) != 1 || got[0].Severity != "error" {
		t.Fatalf("latin-1 cru: want 1 error, got %+v", got)
	}
	// replacement character
	if got := findings([]byte("# t\uFFFDtulo\n")); len(got) != 1 || got[0].Severity != "error" {
		t.Fatalf("U+FFFD: want 1 error, got %+v", got)
	}
	// dupla codificação ("ação" → UTF-8 lido como latin-1 → "Ã§Ã£o")
	double := []byte("# Ã§Ã£o\n")
	if got := findings(double); len(got) != 1 || got[0].Severity != "error" {
		t.Fatalf("dupla codificação: want 1 error, got %+v", got)
	}
}

// ---- L13 - âncoras de idioma ----

// TestL13Anchors cobre o check L13 via lintL13Anchors direto (sandboxes
// temporários, não-kit): doc PT linkando o par EN que existe = flag; link
// já PT = ok; link sem par PT no disco = ok (o EN é o único destino);
// externo e âncora pura = ok. O doc EN (base) NÃO é alvo do L13 (o L13
// valida só docs PT — um .md base linkando o par PT é legítimo: badge,
// cross-links); link morto = skip (o L1 cobre).
func TestL13Anchors(t *testing.T) {
	// sandbox 1 — doc PT com os casos de link
	ws1 := t.TempDir()
	write1 := func(rel, content string) {
		t.Helper()
		p := filepath.Join(ws1, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write1("guide.pt-br.md", "[base](guide.md)\n[par PT](guide.pt-br.md)\n[sem par](only-en.md)\n[externo](https://exemplo.io)\n[âncora pura](#seção)\n[vídeo](video.mp4)\n")
	write1("guide.md", "# Guide EN\n")
	write1("only-en.md", "# Only EN\n")

	rep := &Report{Findings: []Finding{}, ByCheck: map[string]int{}}
	lintL13Anchors(rep, ws1, false, "")
	if len(rep.Findings) != 1 {
		t.Fatalf("want 1 finding (o link [base] deve apontar o par PT), got %d: %+v", len(rep.Findings), rep.Findings)
	}
	f := rep.Findings[0]
	if f.Check != "L13" || f.Severity != "warning" || !strings.Contains(f.Description, "guide.md") {
		t.Fatalf("finding = %+v", f)
	}
	if !strings.Contains(f.Fix, "guide.pt-br.md") {
		t.Fatalf("fix sem o par PT: %+v", f)
	}

	// sandbox 2 — doc EN (base) linkando o par PT: o .md base não é alvo
	// do L13 (o L13 valida só *.pt-br.md) — zero findings
	ws2 := t.TempDir()
	if err := os.MkdirAll(filepath.Join(ws2, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws2, "docs", "guide.md"), []byte("# Guide EN\n[par PT](guide.pt-br.md)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rep2 := &Report{Findings: []Finding{}, ByCheck: map[string]int{}}
	lintL13Anchors(rep2, ws2, false, "")
	if len(rep2.Findings) != 0 {
		t.Fatalf("doc EN base não é alvo do L13 — findings: %+v", rep2.Findings)
	}
}

// ---- L13 - âncoras de idioma ----
