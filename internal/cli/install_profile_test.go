package cli

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// pickProfile (modo numerado): PRIMEIRO passo — copy de 2 linhas por
// opção (o que é + limitação) e ENTER vazio caindo no default essencial.
func TestPickProfilePlainNumbersAndDefaultsEssencial(t *testing.T) {
	var buf strings.Builder
	w := &installWizard{out: &buf, in: bufio.NewReader(strings.NewReader("\n"))}
	profile, err := w.pickProfile()
	if err != nil {
		t.Fatalf("pickProfile: %v", err)
	}
	if profile != "essencial" {
		t.Errorf("ENTER vazio deveria escolher o default essencial, veio %q", profile)
	}
	out := buf.String()
	for _, want := range []string{
		"Escolha o perfil:",
		"  1. essencial",
		"  2. completo",
		"limitação:",
		"número [1-2] (default: 1)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("saída do wizard sem %q:\n%s", want, out)
		}
	}
}

// "2" escolhe completo; resposta inválida repete o prompt sem quebrar.
func TestPickProfileChoosesCompleto(t *testing.T) {
	var buf strings.Builder
	w := &installWizard{out: &buf, in: bufio.NewReader(strings.NewReader("x\n2\n"))}
	profile, err := w.pickProfile()
	if err != nil {
		t.Fatalf("pickProfile: %v", err)
	}
	if profile != "completo" {
		t.Errorf("opção 2 deveria escolher completo, veio %q", profile)
	}
	if !strings.Contains(buf.String(), "resposta inválida") {
		t.Errorf("entrada inválida deveria repetir o prompt:\n%s", buf.String())
	}
}

// copy de contrato: o que é + limitação principal de cada perfil
// (engine-agóstica — a nota por engine aparece no passo engine).
func TestPickProfileShowsContractCopy(t *testing.T) {
	var buf strings.Builder
	w := &installWizard{out: &buf, in: bufio.NewReader(strings.NewReader("\n"))}
	if _, err := w.pickProfile(); err != nil {
		t.Fatalf("pickProfile: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"16 skills gm-* na zona .golmet/",
		"versionadas com o repo",
		"sem agents nativos",
		"125 componentes",
		"config global",
		"compatível apenas com opencode e claude-code",
		"use o perfil essencial",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("copy do perfil sem %q:\n%s", want, out)
		}
	}
}

// help do --engine lista os 7 engines do registry (copilot-cli incluído).
func TestInstallEngineHelpListsSevenEngines(t *testing.T) {
	out, err := runCmd(t, "install", "--help")
	if err != nil {
		t.Fatalf("install --help: %v\n%s", err, out)
	}
	for _, id := range []string{"opencode", "claude-code", "vscode-copilot", "cursor", "gemini-cli", "codex", "copilot-cli"} {
		if !strings.Contains(out, id) {
			t.Errorf("engine %s ausente do help do --engine:\n%s", id, out)
		}
	}
}

// regra de produto: com perfil completo, o passo engine do wizard numerado
// lista SÓ os compatíveis (skills nativas — derive via SkillsDir); com
// essencial, todos os 7.
func TestEngineChoicesFilteredForCompleto(t *testing.T) {
	virtualHome(t)
	home := mustHome()
	all := engineChoices("essencial", home)
	if len(all) != 7 {
		t.Fatalf("essencial deveria listar os 7 engines, veio %v", all)
	}
	filtered := engineChoices("completo", home)
	if len(filtered) != 2 || filtered[0] != "opencode" || filtered[1] != "claude-code" {
		t.Fatalf("completo deveria listar só [opencode claude-code], veio %v", filtered)
	}
}

// update --full em engine sem skills nativas = erro fail-closed (exit 2)
// citando os compatíveis — sem degradação silenciosa para a zona.
func TestUpdateFullWeakEngineErrors(t *testing.T) {
	virtualHome(t)
	out, err := runCmd(t, "update", "--full", "--engine", "cursor")
	if code, ok := ExitCodeOf(err); !ok || code != 2 {
		t.Fatalf("update --full --engine cursor deveria sair 2, veio %d (ok=%v)", code, ok)
	}
	if !strings.Contains(out, "perfil completo exige engine com skills nativas") ||
		!strings.Contains(out, "opencode") || !strings.Contains(out, "essencial") {
		t.Errorf("mensagem de erro incompleta:\n%s", out)
	}
	// nada materializado (o erro vem antes da materialização)
	if _, serr := os.Stat(filepath.Join(mustHome(), ".config", "opencode")); !os.IsNotExist(serr) {
		t.Errorf("update inválido materializou conteúdo: %v", serr)
	}
}

// --profile full é alias interno de completo: relatório cita o perfil
// canônico e a camada de 125 componentes materializa. claude-code não tem
// CLI no PATH do teste → patch JSON direto na home virtual.
func TestInstallProfileFullAliasMaterializesCompleto(t *testing.T) {
	virtualHome(t)
	ws := inTempDir(t)
	out, err := runCmd(t, "install", "--engine", "claude-code", "--profile", "full",
		"--mcp", "golmet", "--workspace", ws, "--yes", "--json")
	// exit 0 (tudo ok) ou exit 1 (parcial — só transportes de MCP
	// dependentes de ambiente); exit 2 = falha de uso/FULL, aí sim falha.
	if err != nil {
		if code, ok := ExitCodeOf(err); !ok || code != 1 {
			t.Fatalf("install --profile full: %v\n%s", err, out)
		}
	}
	rep := decodeInstallReport(t, out)
	if rep["profile"] != "completo" {
		t.Errorf("alias --profile full deveria reportar o perfil completo, veio %v", rep["profile"])
	}
	if n := jsonCount(rep["full_written"]); n == 0 {
		t.Errorf("alias --profile full não materializou os 125 componentes:\n%s", out)
	}
}

// equivalência: --full produz o mesmo relatório que --profile completo
// (perfil canônico completo + componentes materializados).
func TestInstallFullFlagSameShapeAsProfileCompleto(t *testing.T) {
	virtualHome(t)
	ws := inTempDir(t)
	out, err := runCmd(t, "install", "--engine", "claude-code", "--full",
		"--mcp", "golmet", "--workspace", ws, "--yes", "--json")
	if err != nil {
		if code, ok := ExitCodeOf(err); !ok || code != 1 {
			t.Fatalf("install --full: %v\n%s", err, out)
		}
	}
	rep := decodeInstallReport(t, out)
	if rep["profile"] != "completo" {
		t.Errorf("--full deveria reportar o perfil completo, veio %v", rep["profile"])
	}
	if jsonCount(rep["full_written"]) == 0 {
		t.Errorf("--full não materializou os 125 componentes:\n%s", out)
	}
}

// wizard numerado (--plain): perfil vem ANTES do engine; a escolha
// completo (2) flui pelo mesmo canal do --full — relatório cita o perfil
// canônico e os componentes materializam.
func TestInstallPlainWizardNumbersProfileFirstAndCompletoFlows(t *testing.T) {
	virtualHome(t)
	ws := inTempDir(t)

	// stdin canalisédo: o wizard numerado lê as respostas daqui
	old := os.Stdin
	r, wp, perr := os.Pipe()
	if perr != nil {
		t.Fatalf("os.Pipe: %v", perr)
	}
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = old
		r.Close()
	})
	answers := "2\n2\n1\ns\n" // perfil completo · engine claude-code · golmet · confirma
	if _, err := wp.WriteString(answers); err != nil {
		t.Fatalf("stdin do wizard: %v", err)
	}
	wp.Close()

	out, err := runCmd(t, "install", "--plain", "--json", "--workspace", ws)
	if err != nil {
		if code, ok := ExitCodeOf(err); !ok || code != 1 {
			t.Fatalf("install --plain: %v\n%s", err, out)
		}
	}
	// o passo perfil aparece ANTES do passo engine
	idxPerfil := strings.Index(out, "Escolha o perfil:")
	idxEngine := strings.Index(out, "Escolha o engine:")
	if idxPerfil < 0 || idxEngine < 0 || idxPerfil > idxEngine {
		t.Errorf("perfil deveria vir antes do engine (perfil=%d engine=%d):\n%s", idxPerfil, idxEngine, out)
	}
	for _, want := range []string{
		"  1. essencial",
		"  2. completo",
		"número [1-2] (default: 1)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("wizard numerado sem %q:\n%s", want, out)
		}
	}
	// resumo do confirm mostra a escolha; o relatório cita o canônico
	if !strings.Contains(out, "perfil:   completo") {
		t.Errorf("resumo deveria mostrar o perfil escolhido (completo):\n%s", out)
	}
	rep := decodeInstallReport(t, out)
	if rep["engine"] != "claude-code" {
		t.Errorf("engine escolhido no wizard não chegou ao Run: %v", rep["engine"])
	}
	if rep["profile"] != "completo" {
		t.Errorf("escolha completo do wizard deveria reportar o perfil completo, veio %v", rep["profile"])
	}
	if jsonCount(rep["full_written"]) == 0 {
		t.Errorf("escolha completo do wizard não materializou os 125 componentes:\n%s", out)
	}
}

// decodeInstallReport extrai o JSON do relatório da saída mista (no modo
// --plain os prompts do wizard compartilham o buffer antes do relatório).
func decodeInstallReport(t *testing.T, out string) map[string]any {
	t.Helper()
	start := strings.Index(out, "{")
	if start < 0 {
		t.Fatalf("saída sem relatório JSON:\n%s", out)
	}
	var rep map[string]any
	if err := json.Unmarshal([]byte(out[start:]), &rep); err != nil {
		t.Fatalf("json inválido: %v\n%s", err, out)
	}
	return rep
}

// jsonCount converte um campo lista do relatório em contagem (0 = ausente).
func jsonCount(v any) int {
	list, ok := v.([]any)
	if !ok {
		return 0
	}
	return len(list)
}
