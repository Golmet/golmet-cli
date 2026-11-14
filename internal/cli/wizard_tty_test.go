package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Golmet/golmet-cli/internal/core/install"
)

// listPrefix: largura fixa de 2 cols em ativo e inativo (a seta substitui
// o espaço, nunca soma).
func TestListPrefixFixedWidth(t *testing.T) {
	a, i := listPrefix(true), listPrefix(false)
	if len([]rune(a)) != len([]rune(i)) {
		t.Errorf("prefixos com larguras diferentes: ativo %q (%d) vs inativo %q (%d)", a, len([]rune(a)), i, len([]rune(i)))
	}
}

// renderListLine: prefixo fixo + checkbox na mesma coluna + gap mínimo de
// 1 char entre nome e atributos — inclusive para o MAIOR nome da lista
// (regressão do azure-devops, onde o pad chegava a 0 e o atributo colava).
func TestRenderListLineAlignment(t *testing.T) {
	nameWidth := len("azure-devops") // 12 — o maior nome do catálogo real

	active := renderListLine(true, "[x]", "azure-devops", "env ADO_PAT_B64 · param ORG", nameWidth)
	inactive := renderListLine(false, "[ ]", "vscode-copilot", "", nameWidth)

	// prefixo de MESMA largura em bytes (marcador ASCII — largura garantida)
	if active[:2] != "> " {
		t.Errorf("prefixo ativo = %q, want %q", active[:2], "> ")
	}
	if inactive[:2] != "  " {
		t.Errorf("prefixo inativo = %q, want 2 espaços", inactive[:2])
	}
	// checkbox na MESMA coluna em ativo e inativo
	idxA := strings.Index(active, "[x]")
	idxI := strings.Index(inactive, "[ ]")
	if idxA != idxI {
		t.Errorf("coluna do checkbox difere: ativo %d inativo %d", idxA, idxI)
	}
	// gap mínimo: atributo começa depois de ao menos 1 espaço após o nome MAIOR
	if !strings.Contains(active, "azure-devops env ADO_PAT_B64") {
		t.Errorf("sem o gap mínimo nome→atributo: %q", active)
	}
}

// frame MCPs do wizard: TODAS as linhas de item têm o checkbox na MESMA
// coluna (prefixo uniforme — a seta nunca soma ao prefixo).
func TestMCPFrameUniformPrefix(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	m := install.NewWizardModel(home, filepath.Join(home, "ws"))
	m.EngineConfirm()
	m.Advance()

	var buf strings.Builder
	c := newTTYColors(false) // sem cores: bytes crus comparáveis
	drawMCPStep(&rawWriter{alt: &buf, c: c}, m, c, 100)

	// extrai o índice da coluna "[" de cada linha de item
	var cols []int
	for _, line := range strings.Split(buf.String(), "\n") {
		if idx := strings.Index(line, "["); idx >= 0 && idx <= 6 {
			cols = append(cols, idx)
		}
	}
	if len(cols) != 12 {
		t.Fatalf("esperava 12 linhas de item, achei %d (%q)", len(cols), buf.String())
	}
	for i, col := range cols[1:] {
		if col != cols[0] {
			t.Errorf("coluna do checkbox difere: linha 0 = %d, linha %d = %d", cols[0], i+1, col)
		}
	}
}

// terminalWidth: fallback 100 e valor injetado.
func TestTerminalWidthValues(t *testing.T) {
	old := termWidthFn
	termWidthFn = func() int { return 100 }
	t.Cleanup(func() { termWidthFn = old })
	if got := terminalWidth(); got != 100 {
		t.Errorf("terminalWidth = %d, want 100 (fallback)", got)
	}
	termWidthFn = func() int { return 200 }
	if got := terminalWidth(); got != 200 {
		t.Errorf("terminalWidth = %d, want 200", got)
	}
}

func TestRenderListLineColoredAlignment(t *testing.T) {
	// regressão do "tudo verde": cores por PARTE, coluna visível preservada
	c := ttyColors{bold: "\x1b[1m", green: "\x1b[32m", gray: "\x1b[90m", yellow: "\x1b[33m", reset: "\x1b[0m"}
	nameWidth := len("azure-devops")

	active := renderListLineColored(true, c, c.green+"[x]"+c.reset, "azure-devops", c.gray+"env ADO_PAT_B64 · param ORG"+c.reset, nameWidth)
	inactive := renderListLineColored(false, c, c.gray+"[ ]"+c.reset, "vscode-copilot", "", nameWidth)

	// ANSI não ocupa coluna: a coluna VISÍVEL do checkbox é idêntica
	colA := strings.Index(stripANSI(active), "[x]")
	colI := strings.Index(stripANSI(inactive), "[ ]")
	if colA != colI {
		t.Errorf("coluna visível do checkbox difere: ativo %d inativo %d", colA, colI)
	}
	// prefixo: verde no ativo, cinza no inativo — seta só no ativo
	if !strings.Contains(active, c.green+"> "+c.reset) {
		t.Errorf("ativo deveria ter prefixo verde")
	}
	if strings.Count(stripANSI(inactive), ">") != 0 {
		t.Errorf("inativo não deveria ter seta")
	}
	// nome do ativo é bold
	if !strings.Contains(active, "\x1b[1mazure-devops\x1b[0m") {
		t.Errorf("nome do ativo deveria ser bold")
	}
	// checkbox marcado continua verde (cor própria, não herdada da linha)
	if !strings.Contains(active, c.green+"[x]"+c.reset) {
		t.Errorf("checkbox marcado deveria ser verde")
	}
}
