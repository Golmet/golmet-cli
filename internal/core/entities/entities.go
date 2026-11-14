// Package entities parseia as entidades do workspace (FORMATO v1 §7):
// Charter/Feature via front matter §6.1; RN/PEND/CR/ADR/CAND/Task via
// seções h2/h3 com diretivas "> estado:"; Spec via diretiva do arquivo.
package entities

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/Golmet/golmet-cli/internal/adapters/fsutil"
	"github.com/Golmet/golmet-cli/internal/core/frontmatter"
)

var (
	headingIDRe   = regexp.MustCompile(`^(#{1,6})\s+([A-Z]{2,6}-\d{1,4}(?:-\d{1,3})?)(?:\s*[.—–:-]\s*(.*\S))?\s*$`)
	headingAnyRe  = regexp.MustCompile(`^(#{1,6})\s+`)
	directiveRe   = regexp.MustCompile(`^\s{0,3}>\s*estado:\s*(.+?)\s*$`)
	reasonRe      = regexp.MustCompile(`reason:\s*([A-Za-z]{2,6}-\d{1,4})`)
	featureNameRe = regexp.MustCompile(`^RN-\d{1,4}\s*[—–-]\s*.+\s*[—–-]\s*Feature\.md$`)
	taskIDRe      = regexp.MustCompile(`^RN-\d{1,4}-\d{1,3}$`)
	analysisDirRe = regexp.MustCompile(`^(RN-\d{1,4})`)
	reposHeadRe   = regexp.MustCompile(`(?i)^#{1,6}\s+.*\brepos?\b`)
)

// Directive é a diretiva de estado "> estado:" (§6).
type Directive struct {
	Line   int    // linha 1-based no arquivo
	State  string // ex.: active
	Dest   string // destino após → (RN-142 / CR-013 / ADR-004)
	Reason string // reason: CR-XXX (obrigatória em Spec stale — H9)
}

// Section é uma seção com ID (RN/PEND/CR/ADR/CAND/Task).
type Section struct {
	File     string
	Line     int // linha do heading, 1-based
	Level    int
	ID       string
	Title    string
	Lines    []string // corpo plain (fences blankados)
	RawLines []string // corpo original (hash de seção — H8)
	State    *Directive
}

// RepoRef é uma linha da tabela de repos do charter §1.
type RepoRef struct {
	Raw    string // caminho como escrito no charter
	Branch string
	Head   string
	Abs    string // resolvido relativo à raiz do workspace
	Line   int
}

// Charter é a entidade PROJECT_CHARTER.md (§7.1).
type Charter struct {
	File    string
	HasFM   bool
	FM      map[string]frontmatter.Value
	FMError error
	Repos   []RepoRef
}

// Feature é a entidade delivery/<grupo>/RN-NNN — <título> — Feature.md (§7.5).
type Feature struct {
	File     string
	HasFM    bool
	FM       map[string]frontmatter.Value
	FMError  error
	KeyLines map[string]int
	Tasks    []Section
}

// Spec é um arquivo de analysis/ (§7.7) — estado por diretiva do arquivo.
type Spec struct {
	File string
	ID   string // RN herdado da pasta analysis/<RN-NNN — <título>>
	Dir  *Directive
}

// IsTaskID reporta se o ID tem forma de Task (RN-NNN-M).
func IsTaskID(id string) bool { return taskIDRe.MatchString(id) }

// IsFeatureFile reporta se o caminho relativo é uma Feature (§7.5,
// incluindo a variante legada dentro de pasta própria — tolerada).
func IsFeatureFile(rel string) bool {
	return strings.HasPrefix(rel, "delivery/") && featureNameRe.MatchString(filepath.Base(rel))
}

// ParseDirective interpreta uma linha "> estado: ..." (§6). Retorna nil
// quando a linha não é diretiva.
func ParseDirective(line string, lineNo int) *Directive {
	m := directiveRe.FindStringSubmatch(line)
	if m == nil {
		return nil
	}
	d := &Directive{Line: lineNo}
	main := m[1]
	if loc := reasonRe.FindStringSubmatchIndex(main); loc != nil {
		d.Reason = main[loc[2]:loc[3]]
		main = strings.TrimRight(main[:loc[0]], " \t·;,")
	}
	sep := strings.Index(main, "→")
	if sep < 0 {
		sep = strings.Index(main, "->")
	}
	if sep >= 0 {
		d.State = strings.TrimSpace(main[:sep])
		rest := strings.TrimSpace(main[sep:])
		rest = strings.TrimPrefix(rest, "→")
		rest = strings.TrimPrefix(rest, "->")
		d.Dest = strings.TrimSpace(rest)
	} else {
		d.State = strings.TrimSpace(main)
	}
	return d
}

// ParseSections extrai seções com ID cujo (id, nível) passe em want.
// Boundaries consideram TODOS os headings (com ou sem ID) e ignoram
// headings dentro de fences (plain já vem blankado).
func ParseSections(file string, plain, raw []string, want func(id string, level int) bool) []Section {
	type heading struct {
		line, level int
		id, title   string
	}
	var bounds []heading
	for i, line := range plain {
		m := headingAnyRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		h := heading{line: i + 1, level: len(m[1])}
		if mm := headingIDRe.FindStringSubmatch(line); mm != nil {
			h.id = mm[2]
			h.title = strings.TrimSpace(mm[3])
		}
		bounds = append(bounds, h)
	}
	var out []Section
	for bi, h := range bounds {
		if h.id == "" || !want(h.id, h.level) {
			continue
		}
		end := len(plain)
		for _, nb := range bounds[bi+1:] {
			if nb.level <= h.level {
				end = nb.line - 1
				break
			}
		}
		sec := Section{
			File:     file,
			Line:     h.line,
			Level:    h.level,
			ID:       h.id,
			Title:    h.title,
			Lines:    plain[h.line:end],
			RawLines: raw[h.line:end],
		}
		for j, l := range sec.Lines {
			if d := ParseDirective(l, h.line+j); d != nil {
				sec.State = d // última diretiva da seção = estado corrente
			}
		}
		out = append(out, sec)
	}
	return out
}

// ParseCharter parseia o charter: front matter §6.1 + tabela de repos §1.
func ParseCharter(doc *fsutil.FileDoc, root string) *Charter {
	c := &Charter{File: doc.Rel}
	d, has, err := frontmatter.Extract(strings.Join(doc.Lines, "\n"))
	c.HasFM = has
	if err != nil {
		c.FMError = err
	} else if has {
		c.FM = d.Fields
	}
	c.Repos = parseRepos(doc.Plain, root)
	return c
}

// ParseFeature parseia uma Feature: front matter + seções de Task (§7.6).
func ParseFeature(doc *fsutil.FileDoc) *Feature {
	f := &Feature{File: doc.Rel}
	d, has, err := frontmatter.Extract(strings.Join(doc.Lines, "\n"))
	f.HasFM = has
	if err != nil {
		f.FMError = err
	} else if has {
		f.FM = d.Fields
		f.KeyLines = d.KeyLines
	}
	f.Tasks = ParseSections(doc.Rel, doc.Plain, doc.Lines, func(id string, level int) bool {
		return level == 3 && taskIDRe.MatchString(id)
	})
	return f
}

// FMString devolve o valor escalar de uma chave do front matter ("" se ausente).
func (f *Feature) FMString(key string) string {
	if f.FM == nil {
		return ""
	}
	return f.FM[key].String()
}

// ParseSpec parseia um arquivo de analysis/: RN herdado da pasta e a
// primeira diretiva "> estado:" como estado do arquivo (§7.7).
func ParseSpec(doc *fsutil.FileDoc) *Spec {
	s := &Spec{File: doc.Rel}
	// o Abs pode chegar com separador \\ de Windows em qualquer GOOS —
	// Dir/Base são dependentes de plataforma, então normaliza antes
	norm := strings.ReplaceAll(doc.Abs, "\\", "/")
	dir := filepath.Base(filepath.Dir(norm))
	if m := analysisDirRe.FindStringSubmatch(dir); m != nil {
		s.ID = m[1]
	}
	for i, l := range doc.Plain {
		if d := ParseDirective(l, i+1); d != nil {
			s.Dir = d
			break
		}
	}
	return s
}

// parseRepos lê a tabela | Repo | Branch | HEAD | do charter §1.
func parseRepos(plain []string, root string) []RepoRef {
	start := -1
	for i, l := range plain {
		if reposHeadRe.MatchString(l) {
			start = i
			break
		}
	}
	if start < 0 {
		return nil
	}
	var out []RepoRef
	seenSep := false
	for i := start + 1; i < len(plain); i++ {
		l := plain[i]
		if headingAnyRe.MatchString(l) {
			break
		}
		t := strings.TrimSpace(l)
		if !strings.HasPrefix(t, "|") {
			continue
		}
		cells := strings.Split(t, "|")
		if len(cells) < 5 {
			continue
		}
		get := func(k int) string { return strings.TrimSpace(cells[k]) }
		if isSeparator(get(1)) && isSeparator(get(2)) {
			seenSep = true
			continue
		}
		if !seenSep || get(1) == "" || get(2) == "" {
			continue
		}
		ref := RepoRef{Raw: get(1), Branch: get(2), Head: get(3), Line: i + 1}
		if filepath.IsAbs(ref.Raw) {
			ref.Abs = filepath.Clean(filepath.FromSlash(ref.Raw))
		} else {
			ref.Abs = filepath.Clean(filepath.Join(root, filepath.FromSlash(ref.Raw)))
		}
		out = append(out, ref)
	}
	return out
}

func isSeparator(s string) bool {
	s = strings.Trim(s, ": ")
	if s == "" {
		return false
	}
	for _, r := range s {
		if r != '-' {
			return false
		}
	}
	return true
}

// Ita converte int para string (helpers de formatação).
func Ita(n int) string { return strconv.Itoa(n) }

// defaults — estado default quando a diretiva está ausente (FORMATO §6:
// "Diretiva ausente = estado default da entidade"; §7 não nomeia os
// defaults — adota-se o primeiro estado do ciclo de vida de cada entidade).
var defaults = map[string]string{
	"RN": "draft", "PEND": "open", "CR": "pending", "ADR": "proposed",
	"CAND": "pending", "Task": "todo", "Spec": "current",
}

// Default devolve o estado default da entidade (§6/§7).
func Default(kind string) string { return defaults[kind] }

// StateOf devolve o estado corrente da seção (diretiva "> estado:" ou o
// default informado). Fonte única usada pelas regras e pelos comandos
// status/índice — mesma resolução em todo o kit.
func StateOf(s Section, def string) string {
	if s.State != nil {
		return strings.ToLower(s.State.State)
	}
	return def
}

// FeatureKey é a chave da Feature no golmet.index.json (§8): o anchor
// quando presente (H5 exige em ready/done), senão o caminho do arquivo —
// mesma chave que H6 consulta.
func FeatureKey(f *Feature) string {
	if fid := strings.TrimSpace(f.FMString("anchor")); fid != "" {
		return fid
	}
	return f.File
}
