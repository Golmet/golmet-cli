// Testes do lint: um check por fixture de sandbox — refs mortas com
// falsos positivos tratados, órfãos com/sem allowlist, manifest stale,
// nomes fora do padrão, fantasmas por contexto e estrutura canônica.
package lint

import (
	"os"
	"path/filepath"
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
		"- [ok](README.md) e `FORMATO.md` — existem na raiz do kit",
		"- `docs/guia.md` — existe em docs/",
		"- [morto](sub/nada-disso.md) — não existe → L1",
		"- `rules/context7.md` — resolve via coredata/rules (mapping canônico)",
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
	os.WriteFile(filepath.Join(ws, "FORMATO.md"), []byte("formato\n"), 0o644)
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
	os.MkdirAll(filepath.Join(cfg, "rules"), 0o755)
	os.WriteFile(filepath.Join(cfg, "AGENTS.md"), []byte("- `rules/viva.md` — existe\n- `rules/morta.md` — não existe → L1\n"), 0o644)
	os.WriteFile(filepath.Join(cfg, "rules", "viva.md"), []byte("ok\n"), 0o644)

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
		t.Fatalf("L1 = %+v, want 1 (AGENTS.md rules/morta — pack fora do scan v0.1)", l1)
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
	os.WriteFile(filepath.Join(cfg, "rules", "client-profile.md"), []byte("dado\n"), 0o644)
	os.WriteFile(filepath.Join(cfg, ".golmet", "personal.json"), []byte(`{"version":1,"paths":["rules/client-profile.md"]}`), 0o644)
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
	if len(l5) != 5 {
		t.Fatalf("L5 = %v (%d), want 5 (Assets, Norma Deploy, processo, Padrão X, Reengenharia)", files, len(l5))
	}
	for _, want := range []string{
		"Assets",
		"packs/org-pack/rules/Norma Deploy.md",
		"packs/org-pack/accumulation/patterns/processo",
		"packs/org-pack/accumulation/patterns/processo/Padrão X",
		"packs/org-pack/accumulation/nuances/Reengenharia-nuances.md",
	} {
		if !containsStr(files, want) {
			t.Errorf("L5 sem %s: %v", want, files)
		}
	}
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
	os.WriteFile(filepath.Join(ws, "coredata", "sources", "gm-a.md"), []byte("- `gm-a.md` no registry e `FORMATO.md` na raiz\n"), 0o644)
	os.WriteFile(filepath.Join(ws, "FORMATO.md"), []byte("formato\n"), 0o644)
	rep := runLint(t, ws, t.TempDir())
	if rep.Errors != 0 || rep.Warnings != 0 {
		t.Fatalf("sandbox limpo deveria zerar findings: %+v", rep.Findings)
	}
	if _, err := Run(Options{Workspace: filepath.Join(t.TempDir(), "nada"), Home: t.TempDir()}); err == nil {
		t.Errorf("workspace inexistente deveria erro")
	}
}
