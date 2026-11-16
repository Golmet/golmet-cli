// Package evidence implementa a gramática de evidência do FORMAT v1 §5:
// forma canônica [caminho:linha @hash], candidatos malformados (H7) e
// evidência legada sem hash (S4).
package evidence

import (
	"regexp"
	"strconv"
)

// Kind classifica o que foi reconhecido.
type Kind string

const (
	KindCanonical Kind = "canonical" // [caminho:linha @hash] — §5
	KindMalformed Kind = "malformed" // parecian evidência mas violam a regex
	KindLegacy    Kind = "legacy"    // [caminho:linha] sem hash — S4
)

// Evidence é uma evidência §5. Hash vazio = legada.
type Evidence struct {
	Path string
	Line int
	Hash string
}

// Finding liga a evidência ao arquivo/linha onde ocorre.
type Finding struct {
	File string
	Line int
	Kind Kind
	Ev   Evidence
}

var (
	strictRe  = regexp.MustCompile(`\[([^\[\]:\n]+):(\d{1,6})\s+@([0-9a-f]{7,40})\]`)
	lenientRe = regexp.MustCompile(`\[([^\[\]:\n]+):(\d{1,6})\s+@([0-9A-Za-z]{1,40})\]`)
	legacyRe  = regexp.MustCompile(`\[([^\[\]:\n]+):(\d{1,6})\]`)
)

// Scan extrai evidências das linhas (1-based). Blocos de código devem vir
// pré-blankados (fsutil.BlankFenced) para não casar exemplos. A passada é
// em cascata: canônica → malformada → legada, com blanking entre passadas.
func Scan(file string, lines []string) []Finding {
	var out []Finding
	work := make([]string, len(lines))
	copy(work, lines)
	work = scanOne(work, strictRe, KindCanonical, file, &out)
	work = scanOne(work, lenientRe, KindMalformed, file, &out)
	scanOne(work, legacyRe, KindLegacy, file, &out)
	return out
}

func scanOne(lines []string, re *regexp.Regexp, kind Kind, file string, out *[]Finding) []string {
	for i, line := range lines {
		locs := re.FindAllStringSubmatchIndex(line, -1)
		if len(locs) == 0 {
			continue
		}
		b := []byte(line)
		for _, loc := range locs {
			m := re.FindStringSubmatch(line[loc[0]:loc[1]])
			evLine, _ := strconv.Atoi(m[2])
			hash := ""
			if kind != KindLegacy {
				hash = m[3]
			}
			*out = append(*out, Finding{
				File: file,
				Line: i + 1,
				Kind: kind,
				Ev:   Evidence{Path: m[1], Line: evLine, Hash: hash},
			})
			for j := loc[0]; j < loc[1]; j++ {
				b[j] = ' '
			}
		}
		lines[i] = string(b)
	}
	return lines
}
