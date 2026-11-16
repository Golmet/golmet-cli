package pack

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Golmet/golmet-cli/internal/core/content"
)

// ---- fixtures git para state packs (herméticos; pulam sem git) ----

// hermeticGitEnv isola o git do ambiente da máquina (autocrlf=false — o
// global do Windows reescreve EOL no checkout/rebase do working copy que o
// KIT clona; identity fixa; defaultBranch=main).
func hermeticGitEnv(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	cfg := "[init]\n\tdefaultBranch = main\n[core]\n\tautocrlf = false\n[user]\n\tname = Test\n\temail = test@example.com\n"
	global := filepath.Join(dir, "gitconfig-global")
	if err := os.WriteFile(global, []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	system := filepath.Join(dir, "gitconfig-system")
	if err := os.WriteFile(system, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	t.Setenv("GIT_CONFIG_SYSTEM", system)
}

type gitStateTest struct {
	t   *testing.T
	dir string
}

func newGitStateTest(t *testing.T, dir string) *gitStateTest {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git indisponível")
	}
	return &gitStateTest{t: t, dir: dir}
}

func (g *gitStateTest) run(args ...string) string {
	g.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		g.t.Fatalf("git %v: %v (%s)", args, err, out)
	}
	return string(out)
}

func (g *gitStateTest) commitAll(msg string) {
	g.t.Helper()
	g.run("-c", "user.name=Test", "-c", "user.email=test@example.com", "add", "-A")
	g.run("-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-m", msg)
}

func (g *gitStateTest) write(rel, content string) {
	g.t.Helper()
	p := filepath.Join(g.dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		g.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		g.t.Fatal(err)
	}
}

func (g *gitStateTest) read(rel string) string {
	g.t.Helper()
	data, err := os.ReadFile(filepath.Join(g.dir, filepath.FromSlash(rel)))
	if err != nil {
		g.t.Fatalf("arquivo esperado ausente: %s: %v", rel, err)
	}
	return string(data)
}

func (g *gitStateTest) exists(rel string) bool {
	_, err := os.Stat(filepath.Join(g.dir, filepath.FromSlash(rel)))
	return err == nil
}

// newStatePackRemote cria um remote bare (acessado via file:/// — bare repo
// é diretório e sem o scheme seria tratado como fonte local) com um pack
// state completo: skills/, agents/, knowledge/, accumulation/ e
// rules/packs/<id>.md. seed é um clone comum para avançar o remote.
func newStatePackRemote(t *testing.T, root string) (remoteURL, seed string) {
	t.Helper()
	hermeticGitEnv(t)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	newGitStateTest(t, root)
	remote := filepath.Join(root, "origin.git")
	seed = filepath.Join(root, "seed")
	g := newGitStateTest(t, root)
	g.run("init", "--bare", "-b", "main", remote)
	g.run("clone", remote, seed)
	s := newGitStateTest(t, seed)
	s.write("pack.yaml", "id: state-pack\nversion: 1.0.0\nname: State pack\nkind: state\nrequires-core: \">=0.1\"\n")
	s.write("skills/gm-state/SKILL.md", "---\ndescription: skill do pack\n---\nconteúdo original\n")
	s.write("agents/gm-state-reviewer.md", "agent do pack\n")
	s.write("knowledge/banco/indices.md", "conhecimento acumulado\n")
	s.write("accumulation/patterns/exemplo.md", "padrão acumulado\n")
	s.write("rules/packs/state-pack.md", "regra do pack para o lazy index\n")
	s.commitAll("seed do pack state")
	s.run("branch", "-M", "main")
	s.run("push", "-u", "origin", "main")
	return "file:///" + filepath.ToSlash(remote), seed
}

// ---- parse do kind ----

func TestParseSpecStateKind(t *testing.T) {
	s, err := ParseSpec([]byte("id: st\nversion: 1.0.0\nname: St\nkind: state\n"))
	if err != nil {
		t.Fatal(err)
	}
	if s.Kind != KindState {
		t.Fatalf("Kind = %q, want state", s.Kind)
	}
	// default: sem kind → overlay (comportamento atual preservado)
	s, err = ParseSpec([]byte("id: ov\nversion: 1.0.0\nname: Ov\nprovides: [skills/x/SKILL.md]\n"))
	if err != nil || s.Kind != KindOverlay {
		t.Fatalf("kind default = %q (%v), want overlay", s.Kind, err)
	}
}

func TestParseSpecStateRejectsProvides(t *testing.T) {
	_, err := ParseSpec([]byte("id: st\nversion: 1.0.0\nname: St\nkind: state\nprovides: [skills/x/SKILL.md]\n"))
	if err == nil || !strings.Contains(err.Error(), "provides") {
		t.Fatalf("state com provides deveria falhar: %v", err)
	}
}

func TestParseSpecUnknownKind(t *testing.T) {
	_, err := ParseSpec([]byte("id: st\nversion: 1.0.0\nname: St\nkind: mystery\n"))
	if err == nil || !strings.Contains(err.Error(), "kind desconhecido") {
		t.Fatalf("kind desconhecido deveria falhar: %v", err)
	}
}

func TestParseSpecToleratesBOM(t *testing.T) {
	// autores de pack no Windows gravam UTF-8 com BOM — o id não pode sumir
	s, err := ParseSpec([]byte("\xEF\xBB\xBFid: bom-pack\nversion: 1.0.0\nname: BOM\nkind: state\n"))
	if err != nil || s.ID != "bom-pack" {
		t.Fatalf("BOM deveria ser tolerado: %+v (%v)", s, err)
	}
}

// ---- (a) add clona working copy + materializa skills/agents (dados ficam in-place) ----

func TestAddStatePackClonesWorkingCopyAndMaterializes(t *testing.T) {
	root := t.TempDir()
	remoteURL, _ := newStatePackRemote(t, root)
	home := t.TempDir()
	configRoot := t.TempDir()

	rep, err := AddWithOptions(home, configRoot, "opencode", remoteURL, false, false)
	if err != nil {
		t.Fatalf("AddWithOptions state: %v", err)
	}
	if rep.Spec.Kind != KindState || rep.StateSync == nil {
		t.Fatalf("relatório não é de state pack: %+v", rep)
	}
	if !rep.Sync.Cloned {
		t.Error("Cloned = false, want true (primeiro add clona o working copy)")
	}
	cfg := newGitStateTest(t, configRoot)
	work := filepath.Join(configRoot, "packs", "state-pack")
	if _, err := os.Lstat(filepath.Join(work, ".git")); err != nil {
		t.Fatalf("working copy não foi criado em packs/<id>: %v", err)
	}
	// skills/agents materializados no config (posse do pack)
	if got := cfg.read("skills/gm-state/SKILL.md"); got == "" {
		t.Error("skill do pack não materializada")
	}
	if got := cfg.read("agents/gm-state-reviewer.md"); got == "" {
		t.Error("agent do pack não materializado")
	}
	// rules/packs/<id>.md NÃO é copiada — lida IN-PLACE do working copy
	if cfg.exists("rules/packs/state-pack.md") {
		t.Error("rules foi copiada para o config — deveria ser lida in-place do working copy")
	}
	// knowledge/ e accumulation/ ficam NO working copy — nunca no config
	if cfg.exists("knowledge/banco/indices.md") {
		t.Error("knowledge foi materializado — deveria viver só no working copy")
	}
	if cfg.exists("accumulation/patterns/exemplo.md") {
		t.Error("accumulation foi materializado — deveria viver só no working copy")
	}
	if got := newGitStateTest(t, work).read("knowledge/banco/indices.md"); got == "" {
		t.Error("knowledge ausente no working copy")
	}
	if got := newGitStateTest(t, work).read("rules/packs/state-pack.md"); got != "regra do pack para o lazy index\n" {
		t.Errorf("rules ausente no working copy: %q", got)
	}
	// nenhum arquivo do pack registrado como gerido (só skills/agents do overlay)
	manifest, err := content.ReadContentManifest(configRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range manifest.Files {
		if strings.HasPrefix(e.File, "rules/packs/") {
			t.Errorf("rules do pack registrada como gerida (%s layer %s) — in-place não é gerido", e.File, e.Layer)
		}
	}
	// segundo add no MESMO working copy: adoção idempotente (não reclone)
	rep2, err := AddWithOptions(home, configRoot, "opencode", remoteURL, false, false)
	if err != nil {
		t.Fatalf("segundo add (adoção): %v", err)
	}
	if rep2.Sync.Cloned {
		t.Error("segundo add reclonou — deveria adotar o working copy")
	}
	if got := cfg.read("skills/gm-state/SKILL.md"); got == "" {
		t.Error("materialização perdida no segundo add")
	}
}

// ---- (g) add em working copy existente ADOTA (migração), com WIP ----

func TestAddStateAdoptsExistingWorkingCopy(t *testing.T) {
	root := t.TempDir()
	remoteURL, _ := newStatePackRemote(t, root)
	home := t.TempDir()
	configRoot := t.TempDir()

	// working copy pré-existente (o cenário de migração do operador), com WIP
	work := filepath.Join(configRoot, "packs", "state-pack")
	g := newGitStateTest(t, configRoot)
	g.run("clone", remoteURL, work)
	w := newGitStateTest(t, work)
	w.write("skills/gm-state/SKILL.md", "skill migrada pelo operador\n")

	rep, err := AddWithOptions(home, configRoot, "opencode", remoteURL, false, false)
	if err != nil {
		t.Fatalf("adoção deveria sincronizar e materializar: %v", err)
	}
	if rep.Sync.Cloned {
		t.Error("adoção reclonou o working copy")
	}
	if !rep.StateSync.AutoCommit {
		t.Error("WIP do working copy adotado não foi commitado (dado se perderia)")
	}
	// o WIP foi preservado e materializado no config
	if got := w.read("skills/gm-state/SKILL.md"); got != "skill migrada pelo operador\n" {
		t.Errorf("dado do operador perdido na adoção: %q", got)
	}
	if got := newGitStateTest(t, configRoot).read("skills/gm-state/SKILL.md"); got != "skill migrada pelo operador\n" {
		t.Errorf("config não reflete o WIP adotado: %q", got)
	}
}

// ---- (b) update com WIP local: auto-commit + rebase + re-materializa ----

func TestUpdateStatePreservesWIPAndIntegratesRemote(t *testing.T) {
	root := t.TempDir()
	remoteURL, seed := newStatePackRemote(t, root)
	home := t.TempDir()
	configRoot := t.TempDir()
	if _, err := AddWithOptions(home, configRoot, "opencode", remoteURL, false, false); err != nil {
		t.Fatalf("add: %v", err)
	}

	// WIP local: o operador edita a skill no WORKING COPY
	work := filepath.Join(configRoot, "packs", "state-pack")
	w := newGitStateTest(t, work)
	w.write("skills/gm-state/SKILL.md", "conteúdo acumulado local\n")

	// o remote avança em OUTRO arquivo (sem conflito)
	s := newGitStateTest(t, seed)
	s.write("accumulation/patterns/remoto.md", "padrão do remoto\n")
	s.commitAll("padrão remoto")
	s.run("push", "origin", "main")

	rep, err := AddWithOptions(home, configRoot, "opencode", remoteURL, false, false)
	if err != nil {
		t.Fatalf("update state: %v", err)
	}
	if !rep.StateSync.AutoCommit || !rep.StateSync.Rebased {
		t.Errorf("AutoCommit = %v, Rebased = %v; want true, true", rep.StateSync.AutoCommit, rep.StateSync.Rebased)
	}
	// dado local PRESERVADO (commitado, não descartado)
	if got := w.read("skills/gm-state/SKILL.md"); got != "conteúdo acumulado local\n" {
		t.Errorf("WIP local perdido no update: %q", got)
	}
	// rebase trouxe o remoto para o working copy
	if got := w.read("accumulation/patterns/remoto.md"); got != "padrão do remoto\n" {
		t.Errorf("rebase não integrou o remote: %q", got)
	}
	// re-materialização reflete o WIP no config
	if got := newGitStateTest(t, configRoot).read("skills/gm-state/SKILL.md"); got != "conteúdo acumulado local\n" {
		t.Errorf("config não re-materializado com o WIP: %q", got)
	}
}

// ---- (c) conflito de rebase no update: falha com remediação, dado intacto ----

func TestUpdateStateConflictFailsPreservingData(t *testing.T) {
	root := t.TempDir()
	remoteURL, seed := newStatePackRemote(t, root)
	home := t.TempDir()
	configRoot := t.TempDir()
	if _, err := AddWithOptions(home, configRoot, "opencode", remoteURL, false, false); err != nil {
		t.Fatalf("add: %v", err)
	}
	work := filepath.Join(configRoot, "packs", "state-pack")
	g := newGitStateTest(t, root)

	// o remote muda a mesma linha da skill
	s := newGitStateTest(t, seed)
	s.write("skills/gm-state/SKILL.md", "versão do remoto\n")
	s.commitAll("mudança remota")
	s.run("push", "origin", "main")

	// WIP local conflitante no working copy
	w := newGitStateTest(t, work)
	w.write("skills/gm-state/SKILL.md", "versão local\n")

	_, err := AddWithOptions(home, configRoot, "opencode", remoteURL, false, false)
	if err == nil || !strings.Contains(err.Error(), "resolva no git") {
		t.Fatalf("conflito deveria falhar com remediação clara: %v", err)
	}
	// dado preservado: o WIP foi commitado antes do rebase (ORIG_HEAD)
	subject := g.run("-C", work, "log", "-1", "--format=%s", "ORIG_HEAD")
	if !strings.Contains(subject, "sync: acumulação local") {
		t.Errorf("WIP não foi commitado antes do rebase conflitante: %q", subject)
	}
	// remediação aplicável e acumulação intacta após o abort
	g.run("-C", work, "rebase", "--abort")
	if got := w.read("skills/gm-state/SKILL.md"); got != "versão local\n" {
		t.Errorf("acumulação local perdida: %q", got)
	}
}

// ---- (d) publish commita WIP e faz push (remote = path local) ----

func TestPublishStateViaCLIContract(t *testing.T) {
	root := t.TempDir()
	remoteURL, _ := newStatePackRemote(t, root)
	home := t.TempDir()
	configRoot := t.TempDir()
	if _, err := AddWithOptions(home, configRoot, "opencode", remoteURL, false, false); err != nil {
		t.Fatalf("add: %v", err)
	}
	work := filepath.Join(configRoot, "packs", "state-pack")
	w := newGitStateTest(t, work)
	w.write("accumulation/patterns/publicado.md", "vai para o remote\n")

	// resolve pelo caminho do working copy (sem engine)
	rep, err := PublishState(work, "publicação do colaborador", nil)
	if err != nil {
		t.Fatalf("PublishState: %v", err)
	}
	if rep.Pack != "state-pack" || !rep.Committed || !rep.Pushed {
		t.Errorf("relatório do publish inesperado: %+v", rep)
	}
	// o remote recebeu
	g := newGitStateTest(t, root)
	fresh := filepath.Join(root, "fresh")
	g.run("clone", remoteURL, fresh)
	if got := newGitStateTest(t, fresh).read("accumulation/patterns/publicado.md"); got != "vai para o remote\n" {
		t.Errorf("push do publish não chegou ao remote: %q", got)
	}
	// resolve pela ORIGEM (varre as raízes de packs dos engines)
	w.write("accumulation/patterns/publicado2.md", "segunda publicação\n")
	packsRoot := filepath.Join(configRoot, "packs")
	rep2, err := PublishState(remoteURL, "segunda", []string{packsRoot})
	if err != nil {
		t.Fatalf("PublishState por origem: %v", err)
	}
	if rep2.Pack != "state-pack" || !rep2.Pushed {
		t.Errorf("publish por origem falhou: %+v", rep2)
	}
	if _, err := PublishState(remoteURL, "x", nil); err == nil || !strings.Contains(err.Error(), "nenhum working copy") {
		t.Fatalf("publish sem working copy deveria falhar claro: %v", err)
	}
}

// ---- (f) requires-core em range é rejeitado com erro claro ----

func TestStatePackRequiresCoreRangeRejected(t *testing.T) {
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "pack.yaml"), []byte("id: rng\nversion: 1.0.0\nname: Rng\nkind: state\nrequires-core: \">=0.1 <0.3\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := AddWithOptions(t.TempDir(), t.TempDir(), "opencode", src, false, false)
	if err == nil || !strings.Contains(err.Error(), "ranges") || !strings.Contains(err.Error(), ">=X.Y") {
		t.Fatalf("range em requires-core deveria falhar citando o formato aceito: %v", err)
	}
}

// ---- dry-run não clona nem sincroniza ----

func TestAddStatePackDryRunDoesNotClone(t *testing.T) {
	root := t.TempDir()
	remoteURL, _ := newStatePackRemote(t, root)
	home := t.TempDir()
	configRoot := t.TempDir()

	rep, err := AddWithOptions(home, configRoot, "opencode", remoteURL, false, true)
	if err != nil {
		t.Fatalf("dry-run state: %v", err)
	}
	if _, err := os.Stat(filepath.Join(configRoot, "packs")); !os.IsNotExist(err) {
		t.Error("dry-run criou o working copy")
	}
	if _, err := os.Stat(filepath.Join(home, ".golmet", "packs")); !os.IsNotExist(err) {
		t.Error("dry-run criou o cache overlay")
	}
	if rep.StateSync == nil || !rep.StateSync.DryRun {
		t.Errorf("relatório de dry-run inesperado: %+v", rep)
	}
	if len(rep.Written) == 0 {
		t.Error("plano de dry-run não calculou escritas")
	}
}

// ---- origem divergente na adoção é erro claro (nunca reclone/sobrescreva) ----

func TestAddStateRejectsDivergentOrigin(t *testing.T) {
	root := t.TempDir()
	remoteURL, seed := newStatePackRemote(t, root)
	otherURL, _ := newStatePackRemote(t, filepath.Join(root, "outro"))
	home := t.TempDir()
	configRoot := t.TempDir()

	// working copy apontando para OUTRA origem
	work := filepath.Join(configRoot, "packs", "state-pack")
	g := newGitStateTest(t, configRoot)
	g.run("clone", remoteURL, work)
	// o pack.yaml semeado aponta para state-pack; other é outro repo idêntico
	_ = seed
	if _, err := AddWithOptions(home, configRoot, "opencode", otherURL, false, false); err == nil || !strings.Contains(err.Error(), "origem divergente") {
		t.Fatalf("origem divergente deveria falhar claro: %v", err)
	}
}

// ---- overlay segue intocado: local dir com provides continua funcionando ----

func TestOverlayPackStillWorks(t *testing.T) {
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "pack.yaml"), []byte("id: ov\nversion: 1.0.0\nname: Ov\nprovides:\n  - skills/gm-ov/SKILL.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(src, "skills", "gm-ov"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "skills", "gm-ov", "SKILL.md"), []byte("overlay\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	configRoot := t.TempDir()
	rep, err := AddWithOptions(t.TempDir(), configRoot, "opencode", src, false, false)
	if err != nil {
		t.Fatalf("overlay regrediu: %v", err)
	}
	if rep.Spec.Kind != KindOverlay || rep.StateSync != nil {
		t.Errorf("overlay deveria seguir sem state_sync: %+v", rep)
	}
	data, err := os.ReadFile(filepath.Join(configRoot, "skills", "gm-ov", "SKILL.md"))
	if err != nil || string(data) != "overlay\n" {
		t.Errorf("overlay não materializado: %v", err)
	}
}
