package cli

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---- wizard do init bare (M22 — FORMATO §2) ----

// newTestWizard monta o wizard com entrada simulada e cwd informado.
func newTestWizard(cwd, input string) (*initWizard, *strings.Builder) {
	var b strings.Builder
	w := &initWizard{out: &b, in: bufio.NewReader(strings.NewReader(input)), cwd: cwd}
	return w, &b
}

// Opção 1 — Organização com nome: perguntas, confirmação e parâmetros
// resolvidos para o scaffold (o wizard fala "Organização"; o container
// continua sendo o termo interno).
func TestInitWizardOrgFlow(t *testing.T) {
	cwd := t.TempDir()
	w, out := newTestWizard(cwd, "1\nMinhaOrg\ns\n")
	opt, baseDir, name, err := runInitWizard(w, false)
	if err != nil {
		t.Fatalf("wizard org: %v\n%s", err, out)
	}
	if opt == nil {
		t.Fatalf("wizard deveria produzir opções (cancelado?)\n%s", out)
	}
	if !opt.Org || opt.Into != "" {
		t.Errorf("opções = %+v, want Org", opt)
	}
	if baseDir != cwd || name != "MinhaOrg" {
		t.Errorf("base/name = %s/%s, want %s/MinhaOrg", baseDir, name, cwd)
	}
	if !strings.Contains(out.String(), "Organização") {
		t.Errorf("wizard não fala 'Organização':\n%s", out)
	}
	// o wizard NÃO cria — quem cria é o scaffold depois da confirmação
	if _, serr := os.Stat(filepath.Join(cwd, "golmet.yaml")); serr == nil {
		t.Errorf("o wizard não deveria criar nada por conta própria")
	}
}

// Opção 2 — workspace dentro de Organização: o dir PAI do cwd sendo
// container vira a sugestão (ENTER aceita); o wizard não globa o disco.
func TestInitWizardIntoFlowWithParentContainerSuggestion(t *testing.T) {
	base := t.TempDir()
	if err := os.WriteFile(filepath.Join(base, "golmet.yaml"), []byte("format: 1\ncontainer: true\nproducts: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cwd := filepath.Join(base, "SubDir")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := suggestedOrg(cwd); got != base {
		t.Fatalf("sugestão = %s, want %s", got, base)
	}
	w, out := newTestWizard(cwd, "2\n\nProdutoX\ns\n")
	opt, baseDir, name, err := runInitWizard(w, false)
	if err != nil {
		t.Fatalf("wizard into: %v\n%s", err, out)
	}
	if opt == nil || opt.Into != base {
		t.Errorf("opções = %+v, want Into=%s", opt, base)
	}
	if baseDir != cwd || name != "ProdutoX" {
		t.Errorf("base/name = %s/%s", baseDir, name)
	}
	if !strings.Contains(out.String(), "[sugestão: "+base+"]") {
		t.Errorf("sugestão do pai container não exibida:\n%s", out)
	}
}

// Opção 2 sem pai container: nenhuma sugestão — pergunta o caminho
// (o operador aponta a Organização).
func TestInitWizardIntoFlowWithoutSuggestion(t *testing.T) {
	cwd := t.TempDir()
	if got := suggestedOrg(cwd); got != "" {
		t.Fatalf("pai não-container deveria dar sugestão vazia, got %q", got)
	}
	org := t.TempDir()
	if err := os.WriteFile(filepath.Join(org, "golmet.yaml"), []byte("format: 1\ncontainer: true\nproducts: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	w, _ := newTestWizard(cwd, "2\n"+org+"\nProdutoY\ns\n")
	opt, _, name, err := runInitWizard(w, false)
	if err != nil {
		t.Fatalf("wizard into sem sugestão: %v", err)
	}
	if opt == nil || opt.Into != org || name != "ProdutoY" {
		t.Errorf("opções = %+v, name = %s", opt, name)
	}
}

// Opção 3 — raiz livre (nome vazio = o próprio cwd).
func TestInitWizardFreeRootFlow(t *testing.T) {
	cwd := t.TempDir()
	w, out := newTestWizard(cwd, "3\n\ns\n")
	opt, baseDir, name, err := runInitWizard(w, false)
	if err != nil {
		t.Fatalf("wizard raiz livre: %v\n%s", err, out)
	}
	if opt == nil || opt.Org || opt.Into != "" {
		t.Errorf("opções = %+v, want raiz livre", opt)
	}
	if baseDir != cwd || name != "" {
		t.Errorf("base/name = %s/%s, want cwd/\"\"", baseDir, name)
	}
}

// Confirmação negativa: NADA é criado (o wizard devolve nil).
func TestInitWizardCancelled(t *testing.T) {
	cwd := t.TempDir()
	w, out := newTestWizard(cwd, "1\nMinhaOrg\nn\n")
	opt, _, _, err := runInitWizard(w, false)
	if err != nil || opt != nil {
		t.Fatalf("cancelamento = opt %v, err %v\n%s", opt, err, out)
	}
	if _, serr := os.Stat(filepath.Join(cwd, "golmet.yaml")); serr == nil {
		t.Errorf("cancelado não deveria criar nada")
	}
}

// --yes no wizard: pula a pergunta de confirmação.
func TestInitWizardYesSkipsConfirm(t *testing.T) {
	cwd := t.TempDir()
	w, out := newTestWizard(cwd, "3\nLivre\n") // sem resposta de confirmação
	opt, _, name, err := runInitWizard(w, true)
	if err != nil {
		t.Fatalf("wizard --yes: %v\n%s", err, out)
	}
	if opt == nil || name != "Livre" {
		t.Errorf("opções = %+v, name = %s", opt, name)
	}
	if !strings.Contains(out.String(), "confirmação pulada") {
		t.Errorf("--yes deveria pular a confirmação:\n%s", out)
	}
}

// EOF (stdin fechado — CI/pipe): erro claro com o caminho scriptável.
func TestInitWizardEOF(t *testing.T) {
	w, _ := newTestWizard(t.TempDir(), "")
	_, _, _, err := runInitWizard(w, false)
	if err == nil || !strings.Contains(err.Error(), "--org ou --into <container> e --yes") {
		t.Fatalf("EOF deveria falhar com a remediação scriptável: %v", err)
	}
}

// ---- dispatch do CLI ----

// init bare SEM TTY: fail-closed — exit 2 e NADA criado no cwd (o acidente
// do home não se repete nem em script).
func TestInitBareFailClosedNoTTY(t *testing.T) {
	oldTTY := isTTY
	isTTY = func() bool { return false }
	t.Cleanup(func() { isTTY = oldTTY })
	cwd := inTempDir(t)
	_, err := runCmd(t, "init")
	if err == nil {
		t.Fatalf("init bare sem TTY deveria fail-closed")
	}
	if code, ok := ExitCodeOf(err); !ok || code != 2 {
		t.Errorf("exit = %v, want 2", err)
	}
	if _, serr := os.Stat(filepath.Join(cwd, "golmet.yaml")); serr == nil {
		t.Errorf("fail-closed não deveria criar nada no cwd")
	}
	if _, serr := os.Stat(filepath.Join(cwd, "PROJECT_CHARTER.md")); serr == nil {
		t.Errorf("fail-closed não deveria criar charter")
	}
}

// Flags explícitos (--org | --into | nome): comportamento scriptável
// atual INTACTO mesmo sem TTY (regressão zero nos testes existentes).
func TestInitExplicitFlagsUnchangedNoTTY(t *testing.T) {
	oldTTY := isTTY
	isTTY = func() bool { return false }
	t.Cleanup(func() { isTTY = oldTTY })
	base := inTempDir(t)

	if _, err := runCmd(t, "init", "MinhaOrg", "--org"); err != nil {
		t.Fatalf("init --org: %v", err)
	}
	org := filepath.Join(base, "MinhaOrg")
	if _, err := os.Stat(filepath.Join(org, "golmet.yaml")); err != nil {
		t.Fatalf("container não criado: %v", err)
	}
	if _, err := runCmd(t, "init", "MeuApp", "--into", org); err != nil {
		t.Fatalf("init --into: %v", err)
	}
	if _, err := os.Stat(filepath.Join(org, "MeuApp", "golmet.yaml")); err != nil {
		t.Fatalf("produto não criado: %v", err)
	}
	// nome sem flags: destino explícito (o operador nomeou) — intacto
	if _, err := runCmd(t, "init", "RaizLivre"); err != nil {
		t.Fatalf("init <produto>: %v", err)
	}
	if _, err := os.Stat(filepath.Join(base, "RaizLivre", "golmet.yaml")); err != nil {
		t.Fatalf("produto raiz livre não criado: %v", err)
	}
}

// ---- confirmação do wizard vs guarda do home: o wizard confirma, mas a
// guarda do core é a linha dura (testada no pacote scaffold) ----
