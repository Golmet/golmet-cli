// Package content implementa a camada L1 do modelo de conteúdo (DR-021):
// parse do registry canônico (subset YAML do FORMATO) e render por engine.
// O registry guarda a FONTE (itens + ponteiros para corpos); o kit renderiza
// para o formato de cada engine — um renderizador, N engines (D2: o kit é o
// escritor único dos artefatos de conteúdo no config).
//
// Port fiel do protótipo render-agents.mjs (piloto F3) — o golden test
// compara byte a byte com a saída dele. Quirks do subset são preservados
// de propósito (o parser é normativo pelo comportamento, não por spec).
package content

import (
	"fmt"
	"regexp"
	"strings"
)

// Item é um componente do registry (primary, subagent ou skill). Campos
// desconhecidos ficam guardados — o parser não descarta (subset evolutivo).
type Item struct {
	scalars map[string]string
	lists   map[string][]string
	bools   map[string]bool
	order   []string // chaves na ordem de aparição (diagnóstico)
}

func (it *Item) Str(k string) string {
	if it == nil || it.scalars == nil {
		return ""
	}
	return it.scalars[k]
}

func (it *Item) List(k string) []string {
	if it == nil || it.lists == nil {
		return nil
	}
	return it.lists[k]
}

func (it *Item) Bool(k string) bool {
	if it == nil || it.bools == nil {
		return false
	}
	return it.bools[k]
}

// Registry é o registry.yaml parseado.
type Registry struct {
	Version string
	Engines map[string]string
	Items   []*Item
}

var (
	reVersion = regexp.MustCompile(`^version:\s*(.+)$`)
	reEngine  = regexp.MustCompile(`^([a-z-]+):\s*(\S.*)$`)
	reItemID  = regexp.MustCompile(`^-\s+id:\s*(\S+)$`)
	reSubKey  = regexp.MustCompile(`^([A-Za-z_-]+):\s*(.*)$`)
)

// ParseRegistry port do parseRegistry do protótipo mjs (subset fixo do
// piloto). Quirks preservados: comentário só em linha inteira; bloco
// escalar `>-`/`>`/`|` com continuação por indentação; lista flow `[a, b]`;
// booleanos; subchaves em indent ≥ 4.
func ParseRegistry(text string) *Registry {
	out := &Registry{Engines: map[string]string{}}
	var cur *Item
	blockKey := ""
	blockIndent := 0
	appendBlock := func(s string) {
		if cur.scalars == nil {
			cur.scalars = map[string]string{}
		}
		cur.scalars[blockKey] += " " + s
	}
	for _, rawLine := range strings.Split(text, "\n") {
		rawLine = strings.TrimSuffix(rawLine, "\r")
		if strings.HasPrefix(rawLine, "#") || strings.TrimSpace(rawLine) == "" {
			continue
		}
		line := strings.ReplaceAll(rawLine, "\t", "  ")
		indent := len(line) - len(strings.TrimLeft(line, " "))
		trimmed := strings.TrimSpace(line)
		if blockKey != "" && indent > blockIndent {
			appendBlock(trimmed)
			continue
		}
		blockKey = ""
		if m := reVersion.FindStringSubmatch(trimmed); m != nil && indent == 0 {
			out.Version = strings.TrimSpace(m[1])
			continue
		}
		if m := reEngine.FindStringSubmatch(trimmed); m != nil && indent == 2 &&
			(m[1] == "opencode" || m[1] == "claude-code") {
			out.Engines[m[1]] = strings.TrimSpace(m[2])
			continue
		}
		if m := reItemID.FindStringSubmatch(trimmed); m != nil {
			cur = &Item{scalars: map[string]string{"id": m[1]}, lists: map[string][]string{}, bools: map[string]bool{}}
			cur.scalars["id"] = m[1]
			out.Items = append(out.Items, cur)
			continue
		}
		if cur == nil {
			continue
		}
		if m := reSubKey.FindStringSubmatch(trimmed); m != nil && indent >= 4 {
			key, val := m[1], strings.TrimSpace(m[2])
			switch {
			case val == ">-" || val == ">" || val == "|":
				cur.scalars[key] = ""
				blockKey = key
				blockIndent = indent
			case strings.HasPrefix(val, "[") && strings.HasSuffix(val, "]"):
				inner := val[1 : len(val)-1]
				for _, s := range strings.Split(inner, ",") {
					s = strings.TrimSpace(s)
					if s != "" {
						cur.lists[key] = append(cur.lists[key], s)
					}
				}
			case val == "true" || val == "false":
				cur.bools[key] = val == "true"
			default:
				cur.scalars[key] = val
			}
			cur.order = append(cur.order, key)
		}
	}
	return out
}

// ErrMissingBody reporta item sem corpo legível (o piloto pula com stderr).
type ErrMissingBody struct {
	ID   string
	Path string
}

func (e ErrMissingBody) Error() string {
	return fmt.Sprintf("corpo ausente %s — item %s pulado", e.Path, e.ID)
}

// Rendered é um arquivo renderizado (caminho relativo ao config do engine).
type Rendered struct {
	Rel     string // agents/<id>.md | skills/<id>/SKILL.md
	Content string
}

// kindOf resolve o kind com o default do emitter (kind vazio = skill).
func kindOf(it *Item) string {
	k := it.Str("kind")
	if k == "" {
		return "skill"
	}
	return k
}

// jsSpaceSet é a classe \s do JAVASCRIPT (ECMA-262) — inclui U+FEFF (BOM),
// U+00A0 e espaços unicode. O protótipo mjs depende desse comportamento
// (o \s* consome o BOM entre blocos de headers legados) — o port precisa
// da mesma semântica para o golden byte-a-byte.
const jsSpaceSet = "\t\n\v\f\r \u00a0\u1680\u2000\u2001\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u2028\u2029\u202f\u205f\u3000\ufeff"

func isJSSpace(r rune) bool { return strings.ContainsRune(jsSpaceSet, r) }

// leadBlock é o regex de strip de bloco `---` no início do corpo (sources
// twin têm até 2 headers legados). Não-greedy; trailing com a classe \s do
// JS (consome BOM entre blocos — quirk normativo do protótipo).
var leadBlock = regexp.MustCompile(
	`(?s)\A---\r?\n.*?\r?\n---\r?\n[\t\n\v\f\r \x{00A0}\x{1680}\x{2000}-\x{200A}\x{2028}\x{2029}\x{202F}\x{205F}\x{3000}\x{FEFF}]*`)

// stripLeadBlocks port do loop iterativo: repete enquanto o corpo começar
// com bloco; para se malformado (não engole conteúdo).
func stripLeadBlocks(body string) string {
	s := body
	for {
		if !leadBlock.MatchString(s) {
			return s
		}
		next := leadBlock.ReplaceAllString(s, "")
		if next == s {
			return s
		}
		s = next
	}
}

// rightTrimWS port de .replace(/\s+$/,”) com a classe \s do JS.
func rightTrimWS(s string) string {
	return strings.TrimRightFunc(s, isJSSpace)
}

// kind/permission maps — ordem de inserção do JS preservada (Go map não
// tem ordem; slices de pares garantem o byte-a-byte do golden).
type kv struct{ k, v string }

var opencodePerms = map[string][]kv{
	"primary": {
		{"read", "allow"}, {"edit", "allow"}, {"glob", "allow"}, {"grep", "allow"},
		{"list", "allow"}, {"bash", "allow"}, {"write", "allow"}, {"task", "allow"},
		{"todowrite", "allow"}, {"question", "allow"}, {"skill", "allow"},
		{"webfetch", "allow"}, {"websearch", "allow"}, {"lsp", "allow"},
		{"external_directory", "allow"},
	},
	"subagent": {
		{"read", "allow"}, {"glob", "allow"}, {"grep", "allow"}, {"skill", "allow"},
		{"edit", "ask"}, {"write", "ask"}, {"bash", "ask"}, {"question", "allow"},
		{"webfetch", "deny"},
	},
}

var claudeTools = map[string]string{
	"primary":  "Read, Write, Edit, Glob, Grep, Bash, Task, AskUserQuestion, Skill, WebSearch, WebFetch",
	"subagent": "Read, Glob, Grep, Skill, Bash, WebSearch",
}

func opencodeFrontMatter(it *Item) string {
	mode := it.Str("mode")
	perms, ok := opencodePerms[mode]
	if !ok {
		perms = opencodePerms["subagent"]
	}
	var permLines []string
	for _, p := range perms {
		permLines = append(permLines, "  "+p.k+": "+p.v)
	}
	return fmt.Sprintf("---\ndescription: >-\n  %s\nmode: %s\ntemperature: %s\npermission:\n%s\n---\n",
		it.Str("description"), mode, it.Str("temperature"), strings.Join(permLines, "\n"))
}

func claudeFrontMatter(it *Item) string {
	mode := it.Str("mode")
	tools, ok := claudeTools[mode]
	if !ok {
		tools = claudeTools["subagent"]
	}
	return fmt.Sprintf("---\nname: %s\ndescription: %s\ntools: %s\n---\n",
		it.Str("id"), it.Str("description"), tools)
}

func skillFrontMatter(it *Item) string {
	var b strings.Builder
	b.WriteString("---\nname: " + it.Str("id") + "\ndescription: >-\n  " + it.Str("description") + "\n")
	if req := it.List("requires"); len(req) > 0 {
		b.WriteString("requires: [" + strings.Join(req, ", ") + "]\n")
	}
	if deg := it.List("degrades"); len(deg) > 0 {
		b.WriteString("degrades: [" + strings.Join(deg, ", ") + "]\n")
	}
	if cp := it.Str("checkpoints"); cp != "" {
		b.WriteString("kit:\n  checkpoints: " + cp + "\n")
	}
	b.WriteString("---\n")
	return b.String()
}
