package mcpserve

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ---- gm_pack_publish (tool 14 — M32) ----

// makeStatePackFixture — o sandbox git real: um origin bare + o working
// copy do state pack clonado dele (com pack.yaml kind: state + um WIP de
// maturidade). Devolve o caminho do working copy e do origin.
func makeStatePackFixture(t *testing.T, id string) (workCopy, origin string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git indisponível — os testes de publish são skipped")
	}
	base := t.TempDir()
	origin = filepath.Join(base, "origin.git")
	runGit := func(dir string, args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	runGit(base, "init", "--bare", "-b", "main", origin)

	workCopy = filepath.Join(base, id)
	runGit(base, "clone", origin, workCopy)
	if err := os.WriteFile(filepath.Join(workCopy, "pack.yaml"),
		[]byte("id: "+id+"\nversion: 1.0.0\nname: Test pack\nkind: state\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(workCopy, "accumulation", "nuances"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workCopy, "accumulation", "nuances", "meuapp.md"),
		[]byte("nuance: o scheduler tem drift de 2s\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(workCopy, "add", "-A")
	runGit(workCopy, "-c", "user.email=kit@golmet.dev", "-c", "user.name=golmet", "commit", "-m", "pack inicial")
	// WIP de maturidade — o que a retrospective teria gravado
	if err := os.WriteFile(filepath.Join(workCopy, "accumulation", "nuances", "rn001.md"),
		[]byte("nuance: a janela de 15 minutos veio do dono do produto\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// o origin do working copy (o clone já fixa, mas explícito para clareza)
	runGit(workCopy, "remote", "set-url", "origin", origin)
	return workCopy, origin
}

// Sem confirm: erro com o padrão das mutações E o preview da subida
// proposta (working copy, branch, origin, commit, push alvo, o WIP listado).
func TestGmPackPublishPreviewWithoutConfirm(t *testing.T) {
	workCopy, _ := makeStatePackFixture(t, "techminds")
	cs := session(t, context.Background())
	res, out := callTool(t, cs, "gm_pack_publish", map[string]any{
		"pack": workCopy,
	})
	if !res.IsError {
		t.Fatalf("sem confirm deveria falhar: %s", out)
	}
	for _, want := range []string{
		"gm_pack_publish exige confirm=true; nada foi publicado",
		"subida proposta",
		"working copy: " + workCopy,
		"branch:       main",
		"commit:       publish: maturidade via gm_pack_publish",
		"push:         origin main",
		"WIP (1 arquivo(s))",
		"rn001.md",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("preview sem %q:\n%s", want, out)
		}
	}
}

// Confirm=true: o publish executa o MESMO core do CLI (commit do WIP + push
// origin) — o retorno completo e o origin bare recebendo o commit.
func TestGmPackPublishAppliesWithConfirm(t *testing.T) {
	workCopy, origin := makeStatePackFixture(t, "techminds")
	cs := session(t, context.Background())
	res, out := callTool(t, cs, "gm_pack_publish", map[string]any{
		"pack": workCopy, "confirm": true,
	})
	if res.IsError {
		t.Fatalf("confirm deveria aplicar: %s", out)
	}
	for _, want := range []string{
		`"tool":"gm_pack_publish"`,
		`"pack":"techminds"`,
		`"branch":"main"`,
		`"committed":true`,
		`"pushed":true`,
		`"message":"maturidade via gm_pack_publish"`,
		"rn001.md",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("retorno sem %q:\n%s", want, out)
		}
	}
	// o origin bare recebeu o commit (o push real aconteceu no sandbox)
	cmd := exec.Command("git", "-C", origin, "rev-parse", "main")
	if b, err := cmd.CombinedOutput(); err != nil || strings.TrimSpace(string(b)) == "" {
		t.Errorf("o origin não recebeu o commit: %v\n%s", err, b)
	}
}

// Sem WIP: o confirm publica o estado commitado (idempotente — committed
// ausente, pushed true).
func TestGmPackPublishCleanIsIdempotent(t *testing.T) {
	workCopy, _ := makeStatePackFixture(t, "techminds")
	cs := session(t, context.Background())
	if _, out := callTool(t, cs, "gm_pack_publish", map[string]any{
		"pack": workCopy, "confirm": true,
	}); out == "" {
		t.Fatal("sanity: primeiro publish vazio")
	}
	res, out := callTool(t, cs, "gm_pack_publish", map[string]any{
		"pack": workCopy, "confirm": true,
	})
	if res.IsError {
		t.Fatalf("publish idempotente deveria passar: %s", out)
	}
	if !strings.Contains(out, `"pushed":true`) || strings.Contains(out, `"committed":true`) {
		t.Errorf("segundo publish = sem WIP (committed ausente, pushed true):\n%s", out)
	}
}

// N state packs sem o param pack: erro listando os id's.
func TestGmPackPublishMultiplePacksListsIds(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, ".config", "opencode", "packs")
	os.MkdirAll(filepath.Join(root, "pack-a"), 0o755)
	os.MkdirAll(filepath.Join(root, "pack-b"), 0o755)
	for _, id := range []string{"pack-a", "pack-b"} {
		dir := filepath.Join(root, id)
		os.WriteFile(filepath.Join(dir, "pack.yaml"),
			[]byte("id: "+id+"\nversion: 1.0.0\nname: "+id+"\nkind: state\n"), 0o644)
		os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	}
	cs := session(t, context.Background())
	res, out := callTool(t, cs, "gm_pack_publish", map[string]any{"home": home})
	if !res.IsError {
		t.Fatalf("2 packs sem o param deveria listar os id's: %s", out)
	}
	if !strings.Contains(out, "2 state packs instalados") ||
		!strings.Contains(out, "pack-a, pack-b") {
		t.Errorf("erro sem a lista de id's:\n%s", out)
	}
}

// Pack inexistente: erro claro com os id's instalados.
func TestGmPackPublishUnknownPack(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, ".config", "opencode", "packs")
	os.MkdirAll(filepath.Join(root, "techminds", ".git"), 0o755)
	os.WriteFile(filepath.Join(root, "techminds", "pack.yaml"),
		[]byte("id: techminds\nversion: 1.0.0\nname: Techminds\nkind: state\n"), 0o644)
	cs := session(t, context.Background())
	res, out := callTool(t, cs, "gm_pack_publish", map[string]any{
		"home": home, "pack": "inexistente",
	})
	if !res.IsError || !strings.Contains(out, `state pack "inexistente" não encontrado`) ||
		!strings.Contains(out, "techminds") {
		t.Fatalf("erro de pack inexistente inesperado: %s", out)
	}
}
