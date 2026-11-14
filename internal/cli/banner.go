// banner.go — identidade visual do `golmet install` (layout Copilot CLI):
// wordmark bloco à esquerda (âmbar truecolor 240,179,35) + golem TRUECOLOR
// HALF-BLOCK à direita (44 cols ≈ 22 rows — o visual aprovado no primeiro
// screenshot: golem sólido pixelado com quadrado âmbar, chroma-key no
// branco). O alinhamento vive em RUNES em todo o pipeline (█ e ANSI são
// multibyte — a contagem em bytes desalinhava cada linha por um offset
// diferente e "fragmentava" o golem; bug já corrigido e não re-quebrado).
package cli

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png" // decoder do asset embutido
	"strings"

	_ "embed"

	"github.com/Golmet/golmet-cli/internal/core/indexstate"
)

//go:embed assets/logo.png
var logoPNG []byte

// dimensões do golem em células de terminal (célula = 2px verticais):
// full 44 cols × 22 rows (a proporção do primeiro render aprovado) e
// compacta 24 cols × 12 rows — a compacta entra em viewports 84-103 cols
// (a silhueta com o quadrado âmbar lê; abaixo de 84 → wordmark only).
const (
	golemCols        = 44
	golemRows        = 22
	golemCompactCols = 24
	golemCompactRows = 12
)

// bannerWordmarkLetters — letras-bloco 5 rows (leitura G-O-L-M-E-T coluna
// a coluna), largura por letra (7/7/6/8/6/5), gap de 1 col na montagem.
var bannerWordmarkLetters = [][]string{
	{ // G
		"███████",
		"██     ",
		"██  ███",
		"██   ██",
		"███████",
	},
	{ // O
		"███████",
		"██   ██",
		"██   ██",
		"██   ██",
		"███████",
	},
	{ // L — haste nas 4 linhas de cima, barra cheia SÓ na última
		"██    ",
		"██    ",
		"██    ",
		"██    ",
		"██████",
	},
	{ // M
		"██    ██",
		"███  ███",
		"██ ██ ██",
		"██    ██",
		"██    ██",
	},
	{ // E
		"██████",
		"██    ",
		"█████ ",
		"██    ",
		"██████",
	},
	{ // T — 5 cols, haste nas cols 1-2 em TODAS as linhas (pad à largura do topo)
		"█████",
		" ██  ",
		" ██  ",
		" ██  ",
		" ██  ",
	},
}

// wordmarkLines monta as 5 linhas do wordmark (mesma largura por linha).
func wordmarkPlainLines() []string {
	var out []string
	for row := 0; row < 5; row++ {
		var b strings.Builder
		for i, letter := range bannerWordmarkLetters {
			if i > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(letter[row])
		}
		out = append(out, b.String())
	}
	return out
}

// wordmarkComposite — REMOVIDO: as 3 iterações de efeito sticker/sombra
// (backing sólido, contorno fino, composite grid) foram reprovadas pelo
// operador — o wordmark aprovado é o âmbar LIMPO de wordmarkLines.

// wordmarkLines — linhas do wordmark APROVADAS pelo operador: letras
// sólidas █ âmbar com gaps de 1 col LIMPOS entre letras (sem sombra, sem
// contorno — as 3 iterações de efeito sticker foram reprovadas). Sem cor
// (NO_COLOR/pipe) → blocos planos.
func wordmarkLines(color bool) []string {
	c := newTTYColors(color)
	amber := bannerAmber(color)
	var out []string
	for _, l := range wordmarkPlainLines() {
		out = append(out, amber+l+c.reset)
	}
	return out
}

// bannerAmber — cor sólida do wordmark: âmbar truecolor (240,179,35, a cor
// do quadrado da logo) quando colorido; vazio sem cor (NO_COLOR/pipe).
func bannerAmber(color bool) string {
	if !color {
		return ""
	}
	return "\x1b[38;2;240;179;35m"
}

// bannerTagline — linha de tagline (indent 2 é aplicado pelo composer;
// alinhada com o corpo do wordmark; a versão é linha própria abaixo).
func bannerTagline() string {
	return "the stone guardian of your product's documented truth"
}

// isBackground — chroma-key: quase-branco (fundo do asset) vira
// transparente, o golem flutua no fundo do terminal.
func isBackground(r, g, b uint8) bool {
	return r >= 235 && g >= 235 && b >= 235
}

// cropBackground devolve a imagem recortada na bounding box do CONTEÚDO.
func cropBackground(img image.Image) image.Image {
	b := img.Bounds()
	minX, minY, maxX, maxY := b.Max.X, b.Max.Y, b.Min.X, b.Min.Y
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			if !isBackground(uint8(r>>8), uint8(g>>8), uint8(bl>>8)) {
				if x < minX {
					minX = x
				}
				if y < minY {
					minY = y
				}
				if x > maxX {
					maxX = x
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}
	if minX > maxX || minY > maxY {
		return img // imagem toda fundo — degenerada
	}
	return img.(interface {
		SubImage(r image.Rectangle) image.Image
	}).SubImage(image.Rect(minX, minY, maxX+1, maxY+1))
}

// samplePixel pega o pixel nearest da fração (fx, fy) e aplica o chroma-key.
func samplePixel(img image.Image, bw, bh int, fx, fy float64) (uint8, uint8, uint8, bool) {
	x := int(fx * float64(bw))
	y := int(fy * float64(bh))
	if x >= bw {
		x = bw - 1
	}
	if y >= bh {
		y = bh - 1
	}
	ib := img.Bounds()
	r, g, b, _ := img.At(ib.Min.X+x, ib.Min.Y+y).RGBA()
	r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)
	return r8, g8, b8, isBackground(r8, g8, b8)
}

// renderGolem — half-block truecolor: cada célula do terminal cobre 2
// pixels verticais via ▀/▄ com ESC[38;2/48;2:
//
//	ambos conteúdo:   ESC[38;2;top mESC[48;2;bottom m▀ESC[0m
//	só topo conteúdo: ESC[38;2;top m▀ESC[0m   (metade de cima desenha)
//	só baixo:         ESC[38;2;bottom m▄ESC[0m (metade de baixo desenha)
//	nenhum:           espaço (fundo do terminal — nunca retângulo branco)
//
// Nearest-neighbor (3 cores planas — quantização desnecessária).
func renderGolem(img image.Image, cols, rows int) []string {
	img = cropBackground(img)
	b := img.Bounds()
	bw, bh := b.Dx(), b.Dy()
	if bw == 0 || bh == 0 || cols <= 0 || rows <= 0 {
		return nil
	}
	var out []string
	for row := 0; row < rows; row++ {
		var sb strings.Builder
		for col := 0; col < cols; col++ {
			fx := (float64(col) + 0.5) / float64(cols)
			tr, tg, tb, tbg := samplePixel(img, bw, bh, fx, (float64(row*2)+0.5)/float64(rows*2))
			br, bg2, bb, bbg := samplePixel(img, bw, bh, fx, (float64(row*2+1)+0.5)/float64(rows*2))
			switch {
			case !tbg && !bbg: // ambos conteúdo
				fmt.Fprintf(&sb, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀\x1b[0m", tr, tg, tb, br, bg2, bb)
			case !tbg: // só topo: ▀ desenha a metade de cima
				fmt.Fprintf(&sb, "\x1b[38;2;%d;%d;%dm▀\x1b[0m", tr, tg, tb)
			case !bbg: // só baixo: ▄ desenha a metade de baixo
				fmt.Fprintf(&sb, "\x1b[38;2;%d;%d;%dm▄\x1b[0m", br, bg2, bb)
			default:
				sb.WriteByte(' ')
			}
		}
		out = append(out, strings.TrimRight(sb.String(), " "))
	}
	return out
}

// golemFromAsset decodifica o embed (nil = asset quebrado — banner segue
// sem golem).
func golemFromAsset() image.Image {
	img, _, err := image.Decode(bytes.NewReader(logoPNG))
	if err != nil {
		return nil
	}
	return img
}

// composeBanner monta o banner completo (layout Copilot CLI): coluna
// esquerda (wordmark âmbar + linha em branco + tagline cinza + versão
// cinza, indent 2) e coluna direita (golem half-block truecolor 44×22),
// merge row a row com a esquerda centrada verticalmente contra o golem,
// gap mínimo de 4 cols. Golem SÓ com cor (NO_COLOR/pipe omite — o
// wordmark já degrada a plano) e com largura real disponível
// (leftWidth+4+golemCols+1 = 104 ≤ termWidth — QUALIDADE > PRESENÇA:
// abaixo disso wordmark+tagline only, nunca um golem cortado). Larguras em
// RUNES (█/ANSI multibyte). Respiro final separa do header do passo.
func composeBanner(color bool, termWidth int) string {
	c := newTTYColors(color)

	var left []string
	for _, l := range wordmarkLines(color) { // sombra branca só no colorido
		left = append(left, l) // as linhas já vêm colorizadas (âmbar/branco)
	}
	left = append(left, "")
	left = append(left, c.gray+"  "+bannerTagline()+c.reset)
	left = append(left, c.gray+"  v"+indexstate.KitVersion+c.reset)

	leftWidth := 0
	for _, l := range left {
		if w := len([]rune(stripANSI(l))); w > leftWidth { // RUNES — █ é 3 bytes
			leftWidth = w
		}
	}

	var right []string
	if color && termWidth >= leftWidth+4+golemCompactCols+1 {
		if img := golemFromAsset(); img != nil {
			// escada por largura: full (44×22) · compacta (24×12) · omit —
			// QUALIDADE > PRESENÇA: nunca um golem cortado
			switch {
			case termWidth >= leftWidth+4+golemCols+1:
				right = renderGolem(img, golemCols, golemRows)
			default:
				right = renderGolem(img, golemCompactCols, golemCompactRows)
			}
		}
	}

	total := len(left)
	if len(right) > total {
		total = len(right)
	}
	leftOff := 0
	if len(right) > len(left) {
		leftOff = (len(right) - len(left)) / 2
	}
	var lines []string
	for i := 0; i < total; i++ {
		l := ""
		if i >= leftOff && i-leftOff < len(left) {
			l = left[i-leftOff]
		}
		r := ""
		if i < len(right) {
			r = right[i]
		}
		if r == "" {
			lines = append(lines, strings.TrimRight(l, " "))
			continue
		}
		pad := leftWidth - len([]rune(stripANSI(l)))
		if pad < 0 {
			pad = 0
		}
		lines = append(lines, strings.TrimRight(l+strings.Repeat(" ", pad)+strings.Repeat(" ", 4)+r, " "))
	}
	return strings.Join(lines, "\n") + "\n\n"
}
