package evidence

import (
	"strings"
	"testing"
)

func TestScanMixed(t *testing.T) {
	lines := []string{
		"ok `[src/A.cs:88 @a1b2c3d4f9]` e legado `[x/y.py:9]` e mal `[z:1 @ABC123]`",
		"hash curto `[b.ts:2 @abc123]` conta malformado",
		"sem evidência aqui",
	}
	f := Scan("doc.md", lines)
	var canonical, legacy, malformed int
	for _, x := range f {
		switch x.Kind {
		case KindCanonical:
			canonical++
			if x.Ev.Path != "src/A.cs" || x.Ev.Line != 88 || x.Ev.Hash != "a1b2c3d4f9" {
				t.Errorf("canônica errada: %+v", x.Ev)
			}
		case KindLegacy:
			legacy++
			if x.Ev.Hash != "" {
				t.Errorf("legada com hash: %+v", x.Ev)
			}
		case KindMalformed:
			malformed++
		}
	}
	if canonical != 1 || legacy != 1 || malformed != 2 {
		t.Errorf("counts canonical=%d legacy=%d malformed=%d", canonical, legacy, malformed)
	}
	// linha do finding aponta o arquivo que contém a evidência
	for _, x := range f {
		if x.Line != 1 && x.Line != 2 {
			t.Errorf("Line = %d", x.Line)
		}
		if x.File != "doc.md" {
			t.Errorf("File = %q", x.File)
		}
	}
}

func TestScanBlankingAvoidsDoubleMatch(t *testing.T) {
	// a evidência canônica não pode ser re-detectada como legada
	lines := []string{"`[p:1 @a1b2c3d]`"}
	f := Scan("f.md", lines)
	if len(f) != 1 || f[0].Kind != KindCanonical {
		t.Fatalf("esperava 1 canônica, veio %+v", f)
	}
}

func TestFenceBlanking(t *testing.T) {
	lines := []string{
		"antes",
		"```md",
		"[x:1 @a1b2c3d]",
		"```",
		"depois `[y:2 @b2c3d4e]`",
	}
	f := Scan("g.md", BlankForTest(lines))
	if len(f) != 1 || f[0].Ev.Hash != "b2c3d4e" {
		t.Errorf("fence não filtrado: %+v", f)
	}
	if len(BlankForTest(lines)) != len(lines) {
		t.Errorf("BlankForTest mudou a contagem de linhas")
	}
}

// BlankForTest replica fsutil.BlankFenced sem import circular de teste.
func BlankForTest(lines []string) []string {
	out := make([]string, len(lines))
	in := false
	var marker string
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if !in && (strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~")) {
			in, marker = true, t[:3]
			out[i] = strings.Repeat(" ", len(line))
			continue
		}
		if in {
			if strings.HasPrefix(t, marker) {
				in = false
			}
			out[i] = strings.Repeat(" ", len(line))
			continue
		}
		out[i] = line
	}
	return out
}
