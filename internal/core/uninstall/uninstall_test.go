// Testes do uninstall: sandbox com config fake populado (manifest + personal
// + state pack + plugins + opencode.jsonc com as entradas do Golmet) —
// seleção default exclui pack; exclusão parcial; pack selecionado com aviso
// de repo remoto; engine registry cirúrgico; dirs vazios; dry-run/plan.
package uninstall

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sandbox monta uma máquina fake: home/.config/opencode populado.
func sandbox(t *testing.T) (home, configRoot string) {
	t.Helper()
	home = t.TempDir()
	cfg := filepath.Join(home, ".config", "opencode")
	for _, d := range []string{
		filepath.Join(cfg, ".golmet"),
		filepath.Join(cfg, "agents", "gm-dev"),
		filepath.Join(cfg, "skills", "gm-foo"),
		filepath.Join(cfg, "profile"),
		filepath.Join(cfg, "packs", "org-pack", "rules"),
		filepath.Join(cfg, "plugins"),
	} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	w := func(rel, content string) {
		if err := os.WriteFile(filepath.Join(cfg, filepath.FromSlash(rel)), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// geridos layer core
	w("agents/gm-dev/gm-dev.md", "agent\n")
	w("skills/gm-foo/SKILL.md", "skill\n")
	w("profile/context7.md", "rule\n")
	// pessoais-por-design
	w("AGENTS.md", "agents real\n")
	w("profile/client-profile.md", "cliente\n")
	w("profile/operator-profile.md", "operador\n")
	w("cli.json", "{}\n")
	w("service.json", "{}\n")
	// não-gerido
	w("plugins/x.dll", "bin\n")
	// estado do kit
	w(".golmet/manifest.json", `{"version":"1","files":[
		{"file":"agents/gm-dev/gm-dev.md","sha256":"x","layer":"core"},
		{"file":"skills/gm-foo/SKILL.md","sha256":"x","layer":"core"},
		{"file":"profile/context7.md","sha256":"x","layer":"core"}]}`)
	w(".golmet/personal.json", `{"version":1,"paths":["AGENTS.md","profile/client-profile.md","profile/operator-profile.md","opencode.jsonc","cli.json","service.json"]}`)
	// pack state
	w("packs/org-pack/pack.yaml", "id: org-pack\nversion: 1\nname: Org\nkind: state\n")
	w("packs/org-pack/rules/org.md", "norma\n")
	// engine jsonc (com comentários + resto da config)
	w("opencode.jsonc", strings.Join([]string{
		"// config do engine",
		"{",
		`  "model": "claude-x",`,
		`  "mcp": {`,
		`    // MCP do Golmet`,
		`    "golmet": { "type": "local", "command": ["golmet", "mcp"] },`,
		`    "postgres": { "type": "remote", "url": "https://x" }`,
		`  },`,
		`  "agent": {`,
		`    "gm-dev-backend": { "permission": { "edit": { "dir/**": "allow" } } },`,
		`    "gm-dev-frontend": { "permission": { "edit": { "dir/**": "allow" } } },`,
		`  },`,
		"}",
	}, "\n"))
	return home, cfg
}

func discover(t *testing.T, home string) (*Plan, string) {
	t.Helper()
	plan, err := Discover(Options{EngineID: "opencode", Home: home})
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	return plan, filepath.Join(home, ".config", "opencode")
}

func layerByID(t *testing.T, plan *Plan, id string) Layer {
	t.Helper()
	for _, l := range plan.Layers {
		if l.ID == id {
			return l
		}
	}
	t.Fatalf("camada %s ausente do plano: %+v", id, plan.Layers)
	return Layer{}
}

func exists(cfg, rel string) bool {
	_, err := os.Stat(filepath.Join(cfg, filepath.FromSlash(rel)))
	return err == nil
}

// seleção default: packs FORA; tudo o resto removido; opencode.jsonc só
// perde as entradas do Golmet (resto intacto, JSON válido)
func TestUninstallDefaultsExcludePack(t *testing.T) {
	home, cfg := sandbox(t)
	plan, _ := discover(t, home)
	selected := SelectedDefaults()
	if containsStr(selected, LayerPacks) {
		t.Fatalf("pack deveria vir DESELECIONADO por padrão: %v", selected)
	}
	rep, err := Run(plan, selected)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// pack intacto (working copy local)
	if !exists(cfg, "packs/org-pack/pack.yaml") || !exists(cfg, "packs/org-pack/rules/org.md") {
		t.Errorf("working copy do pack deveria permanecer (default deselecionado)")
	}
	// ferramental removido
	if exists(cfg, "agents/gm-dev/gm-dev.md") || exists(cfg, "skills/gm-foo/SKILL.md") || exists(cfg, "profile/context7.md") {
		t.Errorf("ferramental deveria ser removido")
	}
	// pessoal removido
	if exists(cfg, "AGENTS.md") || exists(cfg, "profile/client-profile.md") || exists(cfg, "cli.json") {
		t.Errorf("dados pessoais deveriam ser removidos")
	}
	// não-gerido removido
	if exists(cfg, "plugins/x.dll") {
		t.Errorf("plugins não-gerido deveria ser removido")
	}
	// estado removido
	if exists(cfg, ".golmet/manifest.json") {
		t.Errorf(".golmet deveria ser removido")
	}
	// engine: entradas golmet saem; resto intacto e JSON válido
	data, _ := os.ReadFile(filepath.Join(cfg, "opencode.jsonc"))
	stripped := stripJSONCComments(data)
	var cfgJSON map[string]any
	if err := json.Unmarshal(stripped, &cfgJSON); err != nil {
		t.Fatalf("opencode.jsonc resultante inválido: %v — %s", err, data)
	}
	if strings.Contains(string(data), "\"golmet\"") || strings.Contains(string(data), "gm-dev-backend") || strings.Contains(string(data), "gm-dev-frontend") {
		t.Errorf("entradas do Golmet deveriam sair do jsonc: %s", data)
	}
	if !strings.Contains(string(data), "\"postgres\"") || !strings.Contains(string(data), "\"model\"") {
		t.Errorf("resto da config deveria permanecer: %s", data)
	}
	if !strings.Contains(string(data), "// config do engine") {
		t.Errorf("comentários do operador deveriam permanecer: %s", data)
	}
	if !rep.PackReminder {
		t.Errorf("PackReminder deveria estar ativo")
	}
	// dirs vazios removidos; com conteúdo preservado
	if dirExists(filepath.Join(cfg, "agents")) || dirExists(filepath.Join(cfg, "profile")) {
		t.Errorf("dirs vazios criados pelo kit deveriam ser removidos")
	}
	if !dirExists(filepath.Join(cfg, "packs")) {
		t.Errorf("packs/ tem working copy remanescente — deveria permanecer")
	}
}

// exclusão parcial: só ferramental — pack, pessoal e resto intactos
func TestUninstallOnlyFerramental(t *testing.T) {
	home, cfg := sandbox(t)
	plan, _ := discover(t, home)
	if _, err := Run(plan, []string{LayerFerramental}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if exists(cfg, "agents/gm-dev/gm-dev.md") || exists(cfg, "skills/gm-foo/SKILL.md") {
		t.Errorf("ferramental deveria ser removido")
	}
	for _, rel := range []string{
		"AGENTS.md", "profile/client-profile.md", "cli.json", "service.json",
		"plugins/x.dll", ".golmet/manifest.json", "packs/org-pack/pack.yaml",
		"opencode.jsonc",
	} {
		if !exists(cfg, rel) {
			t.Errorf("exclusão parcial removeu %s indevidamente", rel)
		}
	}
}

// pack SELECIONADO: working copy apagado + aviso do repo remoto no output
func TestUninstallPackSelected(t *testing.T) {
	home, cfg := sandbox(t)
	plan, _ := discover(t, home)
	rep, err := Run(plan, append(SelectedDefaults(), LayerPacks))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if exists(cfg, "packs/org-pack") {
		t.Errorf("working copy do pack deveria ser apagado quando selecionado")
	}
	if !rep.PackReminder {
		t.Errorf("PackReminder deveria ativar quando o pack é removido")
	}
	human := rep.Human()
	if !strings.Contains(human, "repositório remoto") || !strings.Contains(human, "NÃO é tocado") {
		t.Errorf("aviso do repo remoto ausente do output:\n%s", human)
	}
}

// engine sem entradas do Golmet → no-op (sem erro)
func TestUninstallEngineNoEntries(t *testing.T) {
	home, cfg := sandbox(t)
	os.WriteFile(filepath.Join(cfg, "opencode.jsonc"), []byte("{\n  \"model\": \"x\"\n}"), 0o644)
	plan, _ := discover(t, home)
	l := layerByID(t, plan, LayerEngine)
	if l.Count != 0 {
		t.Fatalf("camada engine deveria ter 0 alvos: %+v", l)
	}
	rep, err := Run(plan, SelectedDefaults())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, l := range rep.Layers {
		if l.ID == LayerEngine && strings.Contains(l.Note, "no-op") == false && len(l.Removed) != 0 {
			t.Errorf("engine no-op inesperado: %+v", l)
		}
	}
	if !exists(cfg, "opencode.jsonc") {
		t.Errorf("opencode.jsonc nunca é apagado")
	}
}

// dry-run: o plano lista os alvos e NÃO apaga nada
func TestUninstallDryRunPlanOnly(t *testing.T) {
	home, cfg := sandbox(t)
	plan, _ := discover(t, home)
	human := plan.Human()
	if !strings.Contains(human, "dry-run") || !strings.Contains(human, "AGENTS.md") || !strings.Contains(human, "packs/org-pack/") {
		t.Fatalf("plan.Human incompleto:\n%s", human)
	}
	// nada foi apagado pelo Discover/Human
	for _, rel := range []string{"AGENTS.md", ".golmet/manifest.json", "packs/org-pack/pack.yaml", "plugins/x.dll"} {
		if !exists(cfg, rel) {
			t.Errorf("dry-run apagou %s", rel)
		}
	}
}

// removeJSONCKey — unitários do util jsonc-aware
func TestRemoveJSONCKey(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		key     string
		wantOut string
		wantOK  bool
	}{
		{
			name:    "bloco no meio com vírgula trailing",
			in:      "{\n  \"a\": {\"x\": 1},\n  \"golmet\": {\"type\": \"local\"},\n  \"b\": 2\n}",
			key:     "golmet",
			wantOut: "{\n  \"a\": {\"x\": 1},\n  \"b\": 2\n}",
			wantOK:  true,
		},
		{
			name:    "bloco no fim com vírgula leading",
			in:      "{\n  \"a\": 1,\n  \"golmet\": {\"x\": 1}\n}",
			key:     "golmet",
			wantOut: "{\n  \"a\": 1\n}",
			wantOK:  true,
		},
		{
			name: "com comentários jsonc dentro do bloco",
			in:   "{\n  // MCP do kit\n  \"golmet\": { // inline\n    \"cmd\": 1 // comentário de linha\n  },\n  \"b\": 2\n}",
			key:  "golmet",
			// comentário órfão antes do bloco remanesce — jsonc válido
			wantOut: "{\n  // MCP do kit\n  \n  \"b\": 2\n}",
			wantOK:  true,
		},
		{
			name:    "idempotente — chave ausente",
			in:      "{\n  \"a\": 1\n}",
			key:     "golmet",
			wantOut: "{\n  \"a\": 1\n}",
			wantOK:  false,
		},
		{
			name:    "chave dentro de string não casa",
			in:      "{\n  \"model\": \"use golmet sempre\",\n  \"a\": {\"golmet_extra\": 1}\n}",
			key:     "golmet",
			wantOut: "{\n  \"model\": \"use golmet sempre\",\n  \"a\": {\"golmet_extra\": 1}\n}",
			wantOK:  false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, ok := removeJSONCKey([]byte(tc.in), tc.key)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v — out: %s", ok, tc.wantOK, out)
			}
			if ok && strings.Join(strings.Fields(string(out)), "") != strings.Join(strings.Fields(tc.wantOut), "") {
				t.Errorf("out = %q, want %q", out, tc.wantOut)
			}
			if ok && !json.Valid(stripJSONCToJSON(out)) {
				t.Errorf("resultado inválido: %s", out)
			}
		})
	}
}

// pós-uninstall completo: o lint no sandbox não flaga manifest stale (o
// inventário saiu junto com os arquivos)
func TestUninstallLintCleanAfter(t *testing.T) {
	home, cfg := sandbox(t)
	plan, _ := discover(t, home)
	if _, err := Run(plan, SelectedDefaults()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	// o uninstall apaga .golmet inteiro — sem manifest, o lint não tem o que
	// flagar de stale; plugins removidos; nada órfão de Golmet resta
	if _, err := os.Stat(filepath.Join(cfg, ".golmet")); !os.IsNotExist(err) {
		t.Fatalf(".golmet deveria ter sido removido com o estado do kit")
	}
}
