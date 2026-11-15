package cli

import (
	"errors"
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
// com o aviso de catálogo vazio, exemplo GENÉRICO e validação com
// re-pergunta (caminho ruim → mensagem → o próximo input segue).
func TestPickOrgEmptyCatalogManualPath(t *testing.T) {
	home := t.TempDir()
	oldHome := userHomeFn
	userHomeFn = func() (string, error) { return home, nil }
	t.Cleanup(func() { userHomeFn = oldHome })

	// a org válida é APONTADA no manual (fora do catálogo)
	valid := filepath.Join(t.TempDir(), "OrgApontada")
	makeOrgContainer(t, valid)

	cwd := t.TempDir()
	// 1ª tentativa: caminho inexistente → re-pergunta; 2ª: válido
	w, out := newTestWizard(cwd, "C:\\Projetos\\Sumiu\n"+valid+"\nNovoApp\ns\n")
	org, cancelled, err := w.pickOrg("", false)
	if err != nil || cancelled {
		t.Fatalf("pickOrg = %s/%v/%v\n%s", org, cancelled, err, out)
	}
	if org != valid {
		t.Errorf("org = %s, want %s", org, valid)
	}
	s := out.String()
	if !strings.Contains(s, "nenhuma Organização no catálogo — crie com 'golmet init --org'") {
		t.Errorf("aviso de catálogo vazio ausente:\n%s", s)
	}
	if strings.Contains(s, "Outra — caminho manual") {
		t.Errorf("com catálogo vazio não há lista de seleção:\n%s", s)
	}
	if !strings.Contains(s, "a pasta não existe — verifique o caminho") {
		t.Errorf("validação de caminho inexistente ausente:\n%s", s)
	}
	// exemplo GENÉRICO (nenhuma registrada no catálogo)
	if !strings.Contains(s, "(ex.: C:\\Projetos\\MinhaOrganizacao") {
		t.Errorf("exemplo genérico esperado sem catálogo:\n%s", s)
	}
}

// "Outra — caminho manual" (última opção) cai no fluxo manual com
// validação — o caminho precisa ser pasta de Organização real.
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

	// org MANUAL (fora do catálogo) — fixture própria, não a registrada
	manual := filepath.Join(home, "OrgManual")
	makeOrgContainer(t, manual)

	cwd := t.TempDir()
	// 2 itens: 1 = OrgViva, 2 = Outra → escolhe 2, digita o caminho manual
	w, out := newTestWizard(cwd, "2\n"+manual+"\nNovoApp\ns\n")
	org, cancelled, err := w.pickOrg("", false)
	if err != nil || cancelled {
		t.Fatalf("pickOrg = %s/%v/%v\n%s", org, cancelled, err, out)
	}
	if org != manual {
		t.Errorf("org = %s, want %s (absoluto resolvido)", org, manual)
	}
	// o prompt manual novo: instrução + exemplo PLACEHOLDER genérico
	// (o path do catálogo NUNCA entra no prompt — privacidade M24; ele é
	// funcional apenas na lista de seleção acima)
	s := out.String()
	if !strings.Contains(s, "Caminho completo da pasta que contém o golmet.yaml da Organização") {
		t.Errorf("instrução do caminho manual ausente:\n%s", s)
	}
	if !strings.Contains(s, "(ex.: C:\\Projetos\\MinhaOrganizacao") {
		t.Errorf("placeholder genérico esperado no prompt:\n%s", s)
	}
	// a lista de seleção (acima) segue com o path funcional da registrada
	if !strings.Contains(s, "OrgViva — "+viva) {
		t.Errorf("path funcional da org na lista de seleção ausente:\n%s", s)
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

// ---- input manual do caminho: normalização, validação com re-pergunta ----

// Pasta inexistente → mensagem + RE-PERGUNTA (o próximo input segue).
func TestAskOrgPathNonexistentReasks(t *testing.T) {
	cwd := t.TempDir()
	valid := filepath.Join(t.TempDir(), "OrgOK")
	makeOrgContainer(t, valid)
	w, out := newTestWizard(cwd, cwd+"\\Sumiu\n"+valid+"\n")
	org, cancelled, err := w.askOrgPathManual("")
	if err != nil || cancelled {
		t.Fatalf("askOrgPathManual = %s/%v/%v\n%s", org, cancelled, err, out)
	}
	if org != valid {
		t.Errorf("org = %s, want %s", org, valid)
	}
	if !strings.Contains(out.String(), "a pasta não existe — verifique o caminho") {
		t.Errorf("mensagem de pasta inexistente ausente:\n%s", out)
	}
	if strings.Count(out.String(), "Caminho da Organização:") != 2 {
		t.Errorf("o input deveria ser re-perguntado (2 prompts):\n%s", out)
	}
}

// Pasta sem golmet.yaml → mensagem de "não é Organização declarada" +
// re-pergunta.
func TestAskOrgPathWithoutGolmetYamlReasks(t *testing.T) {
	cwd := t.TempDir()
	sem := t.TempDir() // pasta existe, sem golmet.yaml
	valid := filepath.Join(t.TempDir(), "OrgOK")
	makeOrgContainer(t, valid)
	w, out := newTestWizard(cwd, sem+"\n"+valid+"\n")
	org, cancelled, err := w.askOrgPathManual("")
	if err != nil || cancelled {
		t.Fatalf("askOrgPathManual = %s/%v/%v\n%s", org, cancelled, err, out)
	}
	if !strings.Contains(out.String(), "a pasta não é uma Organização declarada — rode golmet init --org nela, ou informe outra") {
		t.Errorf("mensagem de pasta sem golmet.yaml ausente:\n%s", out)
	}
}

// golmet.yaml sem container: true (workspace-produto) → mensagem própria +
// re-pergunta.
func TestAskOrgPathWorkspaceProductReasks(t *testing.T) {
	cwd := t.TempDir()
	produto := filepath.Join(t.TempDir(), "MeuApp")
	os.MkdirAll(produto, 0o755)
	os.WriteFile(filepath.Join(produto, "golmet.yaml"), []byte("format: 1\n"), 0o644)
	valid := filepath.Join(t.TempDir(), "OrgOK")
	makeOrgContainer(t, valid)
	w, out := newTestWizard(cwd, produto+"\n"+valid+"\n")
	org, cancelled, err := w.askOrgPathManual("")
	if err != nil || cancelled {
		t.Fatalf("askOrgPathManual = %s/%v/%v\n%s", org, cancelled, err, out)
	}
	if !strings.Contains(out.String(), "essa pasta é um workspace-produto, não uma Organização") {
		t.Errorf("mensagem de workspace-produto ausente:\n%s", out)
	}
}

// Válido → segue direto (1 prompt), caminho absoluto limpo.
func TestAskOrgPathValidSinglePrompt(t *testing.T) {
	cwd := t.TempDir()
	valid := filepath.Join(cwd, "OrgAqui")
	makeOrgContainer(t, valid)
	w, out := newTestWizard(filepath.Join(cwd, "sub", "sub"), "\""+valid+"//\"\n") // aspas + trailing slashes
	org, cancelled, err := w.askOrgPathManual("")
	if err != nil || cancelled {
		t.Fatalf("askOrgPathManual = %s/%v/%v\n%s", org, cancelled, err, out)
	}
	if org != valid {
		t.Errorf("org = %s, want %s (aspas/trailing removidos, clean)", org, valid)
	}
	if strings.Count(out.String(), "Caminho da Organização:") != 1 {
		t.Errorf("caminho válido não deveria re-perguntar:\n%s", out)
	}
}

// Relativo ao cwd resolvido para absoluto; ENTER aceita a sugestão
// (também validada).
func TestAskOrgPathRelativeAndSuggestion(t *testing.T) {
	base := t.TempDir()
	cwd := filepath.Join(base, "sub")
	os.MkdirAll(cwd, 0o755)
	valid := filepath.Join(base, "OrgRel")
	makeOrgContainer(t, valid)

	// relativo: "../OrgRel"
	w, _ := newTestWizard(cwd, "../OrgRel\n")
	org, cancelled, err := w.askOrgPathManual("")
	if err != nil || cancelled || org != valid {
		t.Fatalf("relativo = %s/%v/%v", org, cancelled, err)
	}
	// sugestão (pai container) aceita no ENTER e validada — aqui a
	// sugestão é inválida → mensagem e o próximo input resolve
	w2, out := newTestWizard(cwd, "\n"+valid+"\n")
	org2, cancelled2, err2 := w2.askOrgPathManual(filepath.Join(base, "NaoExiste"))
	if err2 != nil || cancelled2 || org2 != valid {
		t.Fatalf("sugestão = %s/%v/%v\n%s", org2, cancelled2, err2, out)
	}
	if !strings.Contains(out.String(), "a pasta não existe — verifique o caminho") {
		t.Errorf("sugestão inválida deveria ser validada:\n%s", out)
	}
}

// `esc` cancela o wizard inteiro (nada criado).
func TestAskOrgPathEscCancels(t *testing.T) {
	w, out := newTestWizard(t.TempDir(), "esc\n")
	org, cancelled, err := w.askOrgPathManual("")
	if err != nil && !errors.Is(err, errWizardCancelled) {
		t.Fatalf("esc = %v\n%s", err, out)
	}
	if !cancelled || org != "" {
		t.Errorf("esc deveria cancelar: org %s, cancelled %v", org, cancelled)
	}
}

// GUARDIÃO DO FIX (M24 — privacidade): o exemplo do prompt do caminho
// manual é SEMPRE o placeholder genérico — NUNCA um path do catálogo. O
// fixture usa um path pessoal plausível (username + pasta pessoal); se
// alguém reintroduzir o exemplo do catálogo no prompt, este teste quebra.
// O catálogo alimenta APENAS a lista de seleção (onde o path é funcional).
func TestAskOrgPathPromptNeverLeaksCatalogPaths(t *testing.T) {
	home := isolatedHome(t)
	// path pessoal plausível no catálogo — fragmentos que NUNCA podem
	// aparecer no prompt
	personal := filepath.Join(home, "Users", "alguem", "Pessoal", "MinhaOrg")
	makeOrgContainer(t, personal)
	if err := orgs.Register(home, "OrgPessoal", personal); err != nil {
		t.Fatal(err)
	}
	valid := filepath.Join(t.TempDir(), "OrgOK")
	makeOrgContainer(t, valid)
	w, out := newTestWizard(t.TempDir(), valid+"\n")
	if _, cancelled, err := w.askOrgPathManual(""); err != nil || cancelled {
		t.Fatalf("askOrgPathManual: %v\n%s", err, out)
	}
	s := out.String()
	for _, fragment := range []string{personal, "alguem", "Pessoal", "OrgPessoal", home} {
		if strings.Contains(s, fragment) {
			t.Errorf("PRIVACIDADE: o prompt do caminho manual contém fragmento do catálogo %q:\n%s", fragment, s)
		}
	}
	if !strings.Contains(s, "(ex.: C:\\Projetos\\MinhaOrganizacao") {
		t.Errorf("exemplo deveria ser o placeholder genérico:\n%s", s)
	}
} // O wizard opção 1 também pré-checa (nome duplicado = erro antes do
// confirm — nada criado).
func TestWizardOrgDuplicateNameBeforeConfirm(t *testing.T) {
	home := isolatedHome(t)
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
