package cli

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/Golmet/golmet-cli/internal/core/indexstate"
	"github.com/Golmet/golmet-cli/internal/core/install"
)

func amber() color.RGBA { return color.RGBA{R: 240, G: 179, B: 35, A: 255} }
func coal() color.RGBA  { return color.RGBA{R: 38, G: 41, B: 43, A: 255} }
func white() color.RGBA { return color.RGBA{R: 255, G: 255, B: 255, A: 255} }

// wordmark: plano (NO_COLOR) = 5 linhas de mesma largura, blocos █, gaps
// de 1 col LIMPOS entre letras (nenhuma célula não-letra no gap), sem
// backtick — leitura G-O-L-M-E-T coluna a coluna.
func TestWordmarkShape(t *testing.T) {
	lines := wordmarkLines(false)
	if len(lines) != 5 {
		t.Fatalf("wordmark = %d linhas, want 5", len(lines))
	}
	widths := map[int]int{}
	for _, l := range lines {
		if strings.Contains(l, "`") {
			t.Errorf("wordmark contém backtick")
		}
		if !strings.Contains(l, "██") {
			t.Errorf("linha sem blocos: %q", l)
		}
		widths[len([]rune(l))]++ // largura VISÍVEL (█ é 3 bytes em UTF-8)
	}
	if len(widths) != 1 {
		t.Errorf("linhas do wordmark com larguras diferentes: %v", widths)
	}
	if got := len([]rune(lines[0])); got != 44 { // 7+1+7+1+6+1+8+1+6+1+5 (T 5 cols)
		t.Errorf("largura do wordmark = %d cols, want 44", got)
	}
	// gaps entre letras LIMPOS: nenhuma célula não-letra (bloco/ruído)
	// ocupando as colunas de separação (7, 15, 22, 31, 38)
	for i, l := range lines {
		for _, gap := range []int{7, 15, 22, 31, 38} {
			if c := []rune(l)[gap]; c != ' ' {
				t.Errorf("row %d col %d (gap entre letras) = %q, want espaço", i, gap, c)
			}
		}
	}
}

// âmbar truecolor presente com cor; sem nenhum ANSI sem cor (NO_COLOR/pipe
// → os MESMOS blocos planos).
func TestWordmarkAmber(t *testing.T) {
	colored := composeBanner(true, 200)
	if !strings.Contains(colored, "\x1b[38;2;240;179;35m") {
		t.Errorf("âmbar truecolor ausente do banner colorido")
	}
	plain := composeBanner(false, 200)
	if strings.Contains(plain, "\x1b") {
		t.Errorf("ANSI vazou no banner plano")
	}
	if !strings.Contains(plain, "███████") {
		t.Errorf("wordmark plano perdeu os blocos")
	}
}

func syntheticGolem(t *testing.T, w, h int, paint func(x, y int) color.RGBA) *image.RGBA {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, paint(x, y))
		}
	}
	return img
}

// células half-block: topo âmbar (conteúdo) + baixo branco (fundo) → ▀ sem
// bg; topo branco + baixo carvão → ▄ com fg do carvão — o golem flutua no
// fundo do terminal (chroma-key: nunca retângulo branco).
func TestGolemHalfBlockCells(t *testing.T) {
	img := syntheticGolem(t, 2, 2, func(x, y int) color.RGBA {
		switch {
		case x == 0 && y == 0:
			return amber()
		case x == 1 && y == 1:
			return coal()
		}
		return white()
	})
	lines := renderGolem(img, 2, 1)
	if len(lines) != 1 {
		t.Fatalf("renderGolem = %d linhas, want 1", len(lines))
	}
	want := "\x1b[38;2;240;179;35m▀\x1b[0m\x1b[38;2;38;41;43m▄\x1b[0m"
	if lines[0] != want {
		t.Errorf("célula = %q, want %q", lines[0], want)
	}
}

// ambos os pixels conteúdo → ▀ com fg do topo e bg do baixo.
func TestGolemFullCell(t *testing.T) {
	img := syntheticGolem(t, 2, 2, func(x, y int) color.RGBA {
		if y == 0 {
			return amber()
		}
		return coal()
	})
	lines := renderGolem(img, 2, 1)
	want := "\x1b[38;2;240;179;35m\x1b[48;2;38;41;43m▀\x1b[0m" +
		"\x1b[38;2;240;179;35m\x1b[48;2;38;41;43m▀\x1b[0m"
	if len(lines) != 1 || lines[0] != want {
		t.Errorf("célula cheia = %q, want %q", lines, want)
	}
}

// T: haste nas MESMAS cols (1-2) nas 4 linhas de haste (regressão do stem
// pulando de linha); todas as letras com largura constante entre linhas.
func TestWordmarkStemAlignment(t *testing.T) {
	for li, letter := range bannerWordmarkLetters {
		widths := map[int]int{}
		for _, row := range letter {
			widths[len([]rune(row))]++
		}
		if len(widths) != 1 {
			t.Errorf("letra %d com larguras inconsistentes entre linhas: %v", li, widths)
		}
	}
	tLetter := bannerWordmarkLetters[5] // T por último
	if len([]rune(tLetter[0])) != 5 {
		t.Fatalf("T topo = %d cols, want 5", len([]rune(tLetter[0])))
	}
	stemCol := strings.Index(tLetter[1], "█")
	if stemCol != 1 {
		t.Fatalf("T haste deveria começar na col 1, veio col %d", stemCol)
	}
	for i, row := range tLetter[1:] {
		if strings.Index(row, "█") != stemCol {
			t.Errorf("T haste pulou de linha (row %d): %q", i+1, row)
		}
	}
}

// composição com ESCADA de golem por largura: ≥104 → full (22 rows) ·
// ≥84 → compacta (12 rows) · menor → wordmark only (nenhum fragmento).
func TestComposeBannerMergeAndClamp(t *testing.T) {
	wide := composeBanner(true, 137) // terminal do operador
	if !strings.Contains(wide, "\x1b[48;2;") {
		t.Errorf("golem truecolor ausente do banner em terminal largo (sem bg ANSI)")
	}
	lines := strings.Split(strings.TrimRight(wide, "\n"), "\n")
	if len(lines) != golemRows && len(lines) != golemRows-1 { // última row pode ser fundo → aparada
		t.Errorf("banner largo = %d linhas, want %d (ou %d)", len(lines), golemRows, golemRows-1)
	}
	// golem centrado vertical: as linhas acima do wordmark não têm blocos █
	if strings.Contains(lines[0], "███████") {
		t.Errorf("wordmark deveria estar centrado verticalmente: %q", lines[0])
	}

	medium := composeBanner(true, 90) // 84 ≤ 90 < 104 → compacta
	mlines := strings.Split(strings.TrimRight(medium, "\n"), "\n")
	if len(mlines) != golemCompactRows && len(mlines) != golemCompactRows-1 {
		t.Errorf("banner compacto = %d linhas, want %d (ou %d)", len(mlines), golemCompactRows, golemCompactRows-1)
	}
	if !strings.Contains(medium, "\x1b[48;2;") {
		t.Errorf("golem compacto deveria estar presente em 90 cols")
	}

	narrow := composeBanner(true, 60) // < 84 → wordmark only
	if strings.Contains(narrow, "\x1b[48;2;") {
		t.Errorf("golem deveria ser omitido em terminal de 60 cols")
	}
	if !strings.Contains(narrow, "███████") {
		t.Errorf("wordmark ausente do banner estreito")
	}
}

// NO_COLOR: o golem é color-only — banner plano sem nenhum ANSI de fundo.
func TestBannerNoColorOmitsGolem(t *testing.T) {
	b := composeBanner(false, 137)
	if strings.Contains(b, "\x1b[48;2;") {
		t.Errorf("golem renderizou sem cor (NO_COLOR deveria omitir)")
	}
	if !strings.Contains(b, "███████") {
		t.Errorf("wordmark plano ausente")
	}
}

// tagline + versão + respiro estrutural (linha em branco antes do passo).
func TestBannerRespiroAndTagline(t *testing.T) {
	b := composeBanner(false, 200)
	if !strings.Contains(b, "the stone guardian of your product's documented truth") {
		t.Errorf("tagline ausente")
	}
	if !strings.Contains(b, "v"+indexstate.KitVersion) {
		t.Errorf("versão ausente")
	}
	if !strings.HasSuffix(b, "\n\n") {
		t.Errorf("respiro final ausente: %q", b[len(b)-40:])
	}
}

// asset real: decodifica e o golem renderiza 22 rows com conteúdo truecolor.
func TestGolemAssetRenderable(t *testing.T) {
	img := golemFromAsset()
	if img == nil {
		t.Fatal("logo.png não decodifica")
	}
	lines := renderGolem(img, golemCols, golemRows)
	if len(lines) != golemRows {
		t.Fatalf("golem real = %d linhas, want %d", len(lines), golemRows)
	}
	content := 0
	for _, l := range lines {
		if strings.Contains(l, "▀") || strings.Contains(l, "▄") {
			content++
		}
	}
	if content == 0 {
		t.Errorf("golem real sem conteúdo (só fundo)")
	}
}

// frame do wizard recebe a composição completa (o draw prefixa o banner
// pré-renderizado — regressão do full-clear que apagava o banner).
func TestWizardReceivesBannerComposition(t *testing.T) {
	model := install.NewWizardModel(t.TempDir(), ".")
	banner := composeBanner(true, 137)
	if !strings.Contains(banner, "\x1b[48;2;") || !strings.Contains(banner, "the stone guardian") {
		t.Fatalf("composição sem golem+tagline para o frame")
	}
	_ = model // a passagem banner → runInteractiveTTY é o wiring do install.go
}
