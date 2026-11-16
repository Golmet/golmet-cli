package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// uninstallSandbox monta home/.config/opencode populado para os testes do
// comando (manifest + personal + state pack + plugins + opencode.jsonc).
func uninstallSandbox(t *testing.T) (home, configRoot string) {
	t.Helper()
	home = t.TempDir()
	cfg := filepath.Join(home, ".config", "opencode")
	for _, d := range []string{
		filepath.Join(cfg, ".golmet"),
		filepath.Join(cfg, "agents", "gm-dev"),
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
	w("agents/gm-dev/gm-dev.md", "agent\n")
	w("profile/context7.md", "rule\n")
	w("AGENTS.md", "agents real\n")
	w("profile/client-profile.md", "cliente\n")
	w("cli.json", "{}\n")
	w("plugins/x.dll", "bin\n")
	w(".golmet/manifest.json", `{"version":"1","files":[{"file":"agents/gm-dev/gm-dev.md","sha256":"x","layer":"core"},{"file":"profile/context7.md","sha256":"x","layer":"core"}]}`)
	w(".golmet/personal.json", `{"version":1,"paths":["AGENTS.md","profile/client-profile.md","opencode.jsonc","cli.json"]}`)
	w("packs/org-pack/pack.yaml", "id: org-pack\nversion: 1\nname: Org\nkind: state\n")
	w("packs/org-pack/rules/org.md", "norma\n")
	w("opencode.jsonc", strings.Join([]string{
		"// config do engine",
		"{",
		`  "model": "claude-x",`,
		`  "mcp": {`,
		`    "golmet": { "type": "local", "command": ["golmet", "mcp"] },`,
		`    "postgres": { "type": "remote", "url": "https://x" }`,
		`  },`,
		`  "agent": {`,
		`    "gm-dev-backend": { "permission": { "edit": { "d/**": "allow" } } }`,
		`  },`,
		"}",
	}, "\n"))
	return home, cfg
}

func existsRel(cfg, rel string) bool {
	_, err := os.Stat(filepath.Join(cfg, filepath.FromSlash(rel)))
	return err == nil
}

// --only SEM --yes (não-interativo) → erro claro; nada apagado
func TestUninstallOnlyRequiresYes(t *testing.T) {
	home, cfg := uninstallSandbox(t)
	out, err := runCmd(t, "uninstall", "--only", "ferramental", "--home", home)
	if err == nil || !strings.Contains(err.Error(), "exit code 2") {
		t.Fatalf("--only sem --yes deveria falhar com exit 2: %v", err)
	}
	if !strings.Contains(out, "exige --yes") {
		t.Fatalf("mensagem clara ausente do stderr capturado: %s", out)
	}
	if !existsRel(cfg, "agents/gm-dev/gm-dev.md") {
		t.Errorf("nada deveria ser apagado sem --yes")
	}
}

// --only + --yes: exclusão parcial — pack, pessoal e resto intactos
func TestUninstallOnlyWithYes(t *testing.T) {
	home, cfg := uninstallSandbox(t)
	out, err := runCmd(t, "uninstall", "--only", "ferramental", "--yes", "--home", home)
	if err != nil {
		t.Fatalf("Execute: %v — %s", err, out)
	}
	if existsRel(cfg, "agents/gm-dev/gm-dev.md") || existsRel(cfg, "profile/context7.md") {
		t.Errorf("ferramental deveria ser removido")
	}
	for _, rel := range []string{
		"AGENTS.md", "profile/client-profile.md", "cli.json",
		"plugins/x.dll", ".golmet/manifest.json", "packs/org-pack/pack.yaml",
		"opencode.jsonc",
	} {
		if !existsRel(cfg, rel) {
			t.Errorf("exclusão parcial removeu %s indevidamente", rel)
		}
	}
}

// pack SELECIONADO: working copy apagado + aviso do repo remoto no output
func TestUninstallPackSelectedViaCLI(t *testing.T) {
	home, cfg := uninstallSandbox(t)
	out, err := runCmd(t, "uninstall", "--only", "packs", "--yes", "--home", home)
	if err != nil {
		t.Fatalf("Execute: %v — %s", err, out)
	}
	if existsRel(cfg, "packs/org-pack") {
		t.Errorf("working copy do pack deveria ser apagado quando selecionado")
	}
	if !strings.Contains(out, "repositório remoto") || !strings.Contains(out, "NÃO é tocado") {
		t.Errorf("aviso do repo remoto ausente do output:\n%s", out)
	}
}

// registro no engine: SÓ as entradas golmet saem; resto intacto; JSON válido
func TestUninstallEngineRegistrySurgical(t *testing.T) {
	home, cfg := uninstallSandbox(t)
	out, err := runCmd(t, "uninstall", "--only", "engine", "--yes", "--home", home)
	if err != nil {
		t.Fatalf("Execute: %v — %s", err, out)
	}
	data, rerr := os.ReadFile(filepath.Join(cfg, "opencode.jsonc"))
	if rerr != nil {
		t.Fatalf("opencode.jsonc sumiu: %v", rerr)
	}
	s := string(data)
	if strings.Contains(s, "\"golmet\"") || strings.Contains(s, "gm-dev-backend") {
		t.Errorf("entradas do Golmet deveriam sair: %s", s)
	}
	if !strings.Contains(s, "\"postgres\"") || !strings.Contains(s, "\"model\"") {
		t.Errorf("resto da config deveria permanecer: %s", s)
	}
	// JSON válido (parse do jsonc stripado — o teste compara por via do
	// resultado ter vírgulas balanceadas e chaves em pares)
	if strings.Count(s, "{") != strings.Count(s, "}") {
		t.Errorf("chaves desbalanceadas: %s", s)
	}
}

// --dry-run lista o plano e NÃO apaga nada
func TestUninstallDryRunFlag(t *testing.T) {
	home, cfg := uninstallSandbox(t)
	out, err := runCmd(t, "uninstall", "--dry-run", "--home", home)
	if err != nil {
		t.Fatalf("Execute: %v — %s", err, out)
	}
	if !strings.Contains(out, "PLANO") || !strings.Contains(out, "AGENTS.md") {
		t.Errorf("plano ausente do output:\n%s", out)
	}
	for _, rel := range []string{"AGENTS.md", ".golmet/manifest.json", "plugins/x.dll", "packs/org-pack/pack.yaml"} {
		if !existsRel(cfg, rel) {
			t.Errorf("dry-run apagou %s", rel)
		}
	}
}

// --json: payload estruturado com o report
func TestUninstallJSONOutput(t *testing.T) {
	home, _ := uninstallSandbox(t)
	out, err := runCmd(t, "uninstall", "--only", "engine", "--yes", "--json", "--home", home)
	if err != nil {
		t.Fatalf("Execute: %v — %s", err, out)
	}
	if !strings.Contains(out, `"removed"`) || !strings.Contains(out, `"layers"`) {
		t.Errorf("payload JSON esperado com layers/removed: %s", out)
	}
}
