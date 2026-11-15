package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Golmet/golmet-cli/internal/core/orgs"
)

// ---- opção 2: seleção de organizações do catálogo global ----

// makeOrgContainer cria um dir container válido (golmet.yaml).
func makeOrgContainer(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "golmet.yaml"), []byte("format: 1\ncontainer: true\nproducts: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Orgs do catálogo com path VIVO aparecem na lista (com detail = path);
// path MORTO é pulado com o aviso discreto.
func TestPickOrgListsRegisteredAliveSkipsDead(t *testing.T) {
	home := t.TempDir()
	viva := filepath.Join(home, "OrgViva")
	morta := filepath.Join(home, "OrgMorta")
	makeOrgContainer(t, viva)
	c := orgs.Catalog{Organizations: []orgs.Entry{
		{Name: "OrgViva", Path: viva},
		{Name: "OrgMorta", Path: morta},
	}}
	if err := orgs.Write(home, &c); err != nil {
		t.Fatal(err)
	}
	oldHome := userHomeFn
	userHomeFn = func() (string, error) { return home, nil }
	t.Cleanup(func() { userHomeFn = oldHome })

	cwd := t.TempDir()
	w, out := newTestWizard(cwd, "1\nNovoApp\ns\n") // escolhe a 1ª (a única viva)
	org, cancelled, err := w.pickOrg("", false)
	if err != nil {
		t.Fatalf("pickOrg: %v\n%s", err, out)
	}
	if cancelled {
		t.Fatalf("pickOrg cancelado:\n%s", out)
	}
	if org != viva {
		t.Errorf("org = %s, want %s", org, viva)
	}
	s := out.String()
	if !strings.Contains(s, "OrgViva — "+viva) {
		t.Errorf("org viva não listada com o path:\n%s", s)
	}
	if strings.Contains(s, "OrgMorta —") {
		t.Errorf("org morta não deveria ser listada:\n%s", s)
	}
	if !strings.Contains(s, "1 organização(ões) do catálogo com caminho ausente") {
		t.Errorf("aviso discreto de path morto ausente:\n%s", s)
	}
	// a opção de fallback manual está por último
	if !strings.Contains(s, "Outra — caminho manual") {
		t.Errorf("'Outra — caminho manual' ausente:\n%s", s)
	}
}

// 0 organizações registradas → direto no caminho manual (o fluxo anterior),
// com o aviso de catálogo vazio.
func TestPickOrgEmptyCatalogManualPath(t *testing.T) {
	home := t.TempDir()
	oldHome := userHomeFn
	userHomeFn = func() (string, error) { return home, nil }
	t.Cleanup(func() { userHomeFn = oldHome })

	cwd := t.TempDir()
	w, out := newTestWizard(cwd, "manual\nNovoApp\ns\n")
	org, cancelled, err := w.pickOrg("", false)
	if err != nil || cancelled {
		t.Fatalf("pickOrg = %s/%v/%v\n%s", org, cancelled, err, out)
	}
	if org != "manual" {
		t.Errorf("org = %s, want caminho manual", org)
	}
	s := out.String()
	if !strings.Contains(s, "nenhuma Organização no catálogo — crie com 'golmet init --org'") {
		t.Errorf("aviso de catálogo vazio ausente:\n%s", s)
	}
	if strings.Contains(s, "Outra — caminho manual") {
		t.Errorf("com catálogo vazio não há lista de seleção:\n%s", s)
	}
}

// "Outra — caminho manual" (última opção) cai no fluxo manual.
func TestPickOrgOtherManualFallback(t *testing.T) {
	home := t.TempDir()
	viva := filepath.Join(home, "OrgViva")
	makeOrgContainer(t, viva)
	c := orgs.Catalog{Organizations: []orgs.Entry{{Name: "OrgViva", Path: viva}}}
	if err := orgs.Write(home, &c); err != nil {
		t.Fatal(err)
	}
	oldHome := userHomeFn
	userHomeFn = func() (string, error) { return home, nil }
	t.Cleanup(func() { userHomeFn = oldHome })

	cwd := t.TempDir()
	// 2 itens: 1 = OrgViva, 2 = Outra → escolhe 2, digita o caminho manual
	w, _ := newTestWizard(cwd, "2\nmanual\nNovoApp\ns\n")
	org, cancelled, err := w.pickOrg("", false)
	if err != nil || cancelled {
		t.Fatalf("pickOrg = %s/%v/%v", org, cancelled, err)
	}
	if org != "manual" {
		t.Errorf("org = %s, want caminho manual", org)
	}
}

// ---- init --org registra no catálogo global (criação e adoção) ----

// criação nova: container + entrada no catálogo.
func TestInitOrgRegistersCatalog(t *testing.T) {
	home := t.TempDir()
	oldHome := userHomeFn
	userHomeFn = func() (string, error) { return home, nil }
	t.Cleanup(func() { userHomeFn = oldHome })
	base := inTempDir(t)

	if _, err := runCmd(t, "init", "NovaOrg", "--org"); err != nil {
		t.Fatalf("init --org: %v", err)
	}
	c, err := orgs.Load(home)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(c.Organizations) != 1 {
		t.Fatalf("entries = %d, want 1", len(c.Organizations))
	}
	e := c.Organizations[0]
	if e.Name != "NovaOrg" || !strings.HasSuffix(e.Path, filepath.Join(base, "NovaOrg")) {
		t.Errorf("entry inesperada: %+v", e)
	}
}

// ADOÇÃO: golmet.yaml container: true JÁ EXISTE e NADA é sobrescrito —
// o init registra e reporta a adoção.
func TestInitOrgAdoptionRegisters(t *testing.T) {
	home := t.TempDir()
	oldHome := userHomeFn
	userHomeFn = func() (string, error) { return home, nil }
	t.Cleanup(func() { userHomeFn = oldHome })
	base := inTempDir(t)

	existing := filepath.Join(base, "Adotada")
	makeOrgContainer(t, existing)
	sentinel := filepath.Join(existing, "INDEX.md")
	if err := os.WriteFile(sentinel, []byte("conteúdo do operador\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, "init", "Adotada", "--org")
	if err != nil {
		t.Fatalf("init --org adoção: %v\n%s", err, out)
	}
	// idempotência do scaffold intacta: nada sobrescrito
	if !strings.Contains(out, "pulou (já existe): golmet.yaml") {
		t.Errorf("golmet.yaml existente deveria ser pulado:\n%s", out)
	}
	if got, _ := os.ReadFile(sentinel); !strings.Contains(string(got), "conteúdo do operador") {
		t.Errorf("arquivo existente foi alterado")
	}
	if !strings.Contains(out, "registrada no catálogo global (adoção") {
		t.Errorf("mensagem de adoção ausente:\n%s", out)
	}
	c, _ := orgs.Load(home)
	if len(c.Organizations) != 1 || c.Organizations[0].Name != "Adotada" {
		t.Errorf("catálogo após adoção: %+v", c.Organizations)
	}
	// re-executar: idempotente (uma entrada só)
	if _, err := runCmd(t, "init", "Adotada", "--org"); err != nil {
		t.Fatalf("re-init --org: %v", err)
	}
	c, _ = orgs.Load(home)
	if len(c.Organizations) != 1 {
		t.Errorf("re-adoção duplicou o catálogo: %+v", c.Organizations)
	}
}

// Nome duplicado com path DIFERENTE: erro claro ANTES de criar (o kit
// nunca cria organização que não possa registrar).
func TestInitOrgDuplicateNameFailClosed(t *testing.T) {
	home := t.TempDir()
	oldHome := userHomeFn
	userHomeFn = func() (string, error) { return home, nil }
	t.Cleanup(func() { userHomeFn = oldHome })

	if err := orgs.Register(home, "Colisao", filepath.Join(home, "original")); err != nil {
		t.Fatal(err)
	}
	// mesmo NOME em path diferente (cwd novo) → bloqueado ANTES de criar
	base := inTempDir(t)
	out, err := runCmd(t, "init", "Colisao", "--org")
	if err == nil {
		t.Fatalf("nome duplicado com path diferente deveria falhar:\n%s", out)
	}
	if !strings.Contains(out, "já existe uma Organização") {
		t.Errorf("erro sem a mensagem clara:\n%s", out)
	}
	if _, serr := os.Stat(filepath.Join(base, "Colisao")); serr == nil {
		t.Errorf("nada deveria ter sido criado (pré-checagem)")
	}
}

// O wizard opção 1 também pré-checa (nome duplicado = erro antes do
// confirm — nada criado).
func TestWizardOrgDuplicateNameBeforeConfirm(t *testing.T) {
	home := t.TempDir()
	oldHome := userHomeFn
	userHomeFn = func() (string, error) { return home, nil }
	t.Cleanup(func() { userHomeFn = oldHome })
	base := t.TempDir()
	if err := orgs.Register(home, "Tomada", filepath.Join(home, "original")); err != nil {
		t.Fatal(err)
	}

	w, out := newTestWizard(base, "1\nTomada\ns\n")
	opt, _, _, err := runInitWizard(w, false, false)
	if err == nil {
		t.Fatalf("nome duplicado no wizard deveria falhar antes da confirmação\n%s", out)
	}
	if opt != nil {
		t.Errorf("nada deveria seguir para o scaffold: %+v", opt)
	}
	if _, serr := os.Stat(filepath.Join(base, "Tomada")); serr == nil {
		t.Errorf("nada criado")
	}
}
