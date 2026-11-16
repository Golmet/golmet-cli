// Package frontmatter implementa o parser do subset YAML normativo do
// FORMAT v1 §6.1: chaves escalares, valores escalares, arrays inline de
// escalares e mapas de no máximo 1 nível. Sem multiline (|, >), sem
// anchors/aliases (&, *) e sem tags (!!) — rejeitados com erro estruturado.
// Comentários # são tolerados em qualquer linha e ignorados.
package frontmatter

import (
	"fmt"
	"regexp"
	"strings"
)

// Value é um valor do subset: escalar, array inline ou mapa de 1 nível.
type Value struct {
	Str   string
	Arr   []string
	Map   map[string]string
	IsArr bool
	IsMap bool
}

// String devolve a forma escalar (arrays viram join por vírgula — uso
// incidental de log; regras devem ler Arr/Map diretamente).
func (v Value) String() string {
	if v.IsArr {
		return strings.Join(v.Arr, ",")
	}
	if v.IsMap {
		return ""
	}
	return v.Str
}

// Doc é o resultado do parse de um front matter delimitado por ---.
type Doc struct {
	Fields        map[string]Value
	KeyLines      map[string]int // linha 1-based de cada chave de 1º nível
	Body          string         // conteúdo após o fechamento
	BodyStartLine int            // linha 1-based do início do body
}

var keyRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)

// Extract lê front matter delimitado por --- na 1ª linha e na linha de
// fechamento. Sem delimitadores → (nil, false, nil). Violação do subset →
// erro descritivo com linha.
func Extract(content string) (*Doc, bool, error) {
	content = strings.TrimPrefix(strings.ReplaceAll(content, "\r\n", "\n"), "\uFEFF")
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil, false, nil
	}
	closeIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			closeIdx = i
			break
		}
	}
	if closeIdx < 0 {
		return nil, true, fmt.Errorf("delimitador de fechamento '---' ausente")
	}
	fields, keyLines, err := parseFields(lines[1:closeIdx], 2)
	if err != nil {
		return nil, true, err
	}
	return &Doc{
		Fields:        fields,
		KeyLines:      keyLines,
		Body:          strings.Join(lines[closeIdx+1:], "\n"),
		BodyStartLine: closeIdx + 2,
	}, true, nil
}

// ParseBlock parseia um bloco YAML solto (ex.: golmet.yaml) com as mesmas
// regras do subset §6.1.
func ParseBlock(content string) (map[string]Value, error) {
	fields, _, err := parseFields(strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n"), 1)
	return fields, err
}

func parseFields(lines []string, baseLine int) (map[string]Value, map[string]int, error) {
	fields := map[string]Value{}
	keyLines := map[string]int{}
	pendingMap := ""
	prevIndentedEmpty := false
	for i, raw := range lines {
		lineNo := baseLine + i
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indented := raw[0] == ' ' || raw[0] == '\t'
		content := strings.TrimSpace(stripComment(trimmed))
		if content == "" {
			continue
		}
		kv := strings.SplitN(content, ":", 2)
		if len(kv) != 2 {
			return nil, nil, fmt.Errorf("linha %d: esperado 'chave: valor' — encontrei %q", lineNo, content)
		}
		key := strings.TrimSpace(kv[0])
		if !keyRe.MatchString(key) {
			return nil, nil, fmt.Errorf("linha %d: chave inválida %q — use escalares simples (§6.1)", lineNo, key)
		}
		valRaw := strings.TrimSpace(kv[1])
		if indented {
			// linha de mapa de 1 nível: exige chave-pi pendente
			if pendingMap == "" || prevIndentedEmpty {
				return nil, nil, fmt.Errorf("linha %d: aninhamento além de 1 nível não é suportado (§6.1)", lineNo)
			}
			mv := fields[pendingMap]
			if !mv.IsMap {
				if mv.Str != "" || mv.IsArr {
					return nil, nil, fmt.Errorf("linha %d: conflito de tipos na chave %q (§6.1)", lineNo, pendingMap)
				}
				mv = Value{IsMap: true, Map: map[string]string{}}
			}
			v, err := parseValue(valRaw, lineNo)
			if err != nil {
				return nil, nil, err
			}
			if v.IsArr {
				return nil, nil, fmt.Errorf("linha %d: valor de mapa só aceita escalar (§6.1)", lineNo)
			}
			if mv.Map == nil {
				mv.Map = map[string]string{}
				mv.IsMap = true
			}
			mv.Map[key] = v.Str
			fields[pendingMap] = mv
			prevIndentedEmpty = valRaw == ""
			continue
		}
		pendingMap = ""
		prevIndentedEmpty = false
		if valRaw == "" {
			if _, exists := fields[key]; exists {
				return nil, nil, fmt.Errorf("linha %d: chave %q duplicada", lineNo, key)
			}
			fields[key] = Value{}
			keyLines[key] = lineNo
			pendingMap = key
			continue
		}
		v, err := parseValue(valRaw, lineNo)
		if err != nil {
			return nil, nil, err
		}
		if _, exists := fields[key]; exists {
			return nil, nil, fmt.Errorf("linha %d: chave %q duplicada", lineNo, key)
		}
		fields[key] = v
		keyLines[key] = lineNo
	}
	return fields, keyLines, nil
}

// stripComment corta comentários # fora de aspas.
func stripComment(s string) string {
	var inQuote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case inQuote != 0:
			if c == inQuote {
				inQuote = 0
			}
		case c == '"' || c == '\'':
			inQuote = c
		case c == '#':
			return s[:i]
		}
	}
	return s
}

func parseValue(s string, lineNo int) (Value, error) {
	if s == "" {
		return Value{}, nil
	}
	switch s[0] {
	case '[':
		if !strings.HasSuffix(s, "]") {
			return Value{}, fmt.Errorf("linha %d: array inline não fechado em ']' (§6.1)", lineNo)
		}
		inner := strings.TrimSpace(s[1 : len(s)-1])
		v := Value{IsArr: true}
		if inner != "" {
			for _, part := range strings.Split(inner, ",") {
				item := strings.TrimSpace(part)
				if item == "" {
					return Value{}, fmt.Errorf("linha %d: item vazio em array inline (§6.1)", lineNo)
				}
				sc, err := scalar(item, lineNo)
				if err != nil {
					return Value{}, err
				}
				v.Arr = append(v.Arr, sc)
			}
		}
		return v, nil
	case '|', '>':
		return Value{}, fmt.Errorf("linha %d: bloco multiline (%q) não é suportado — use string de uma linha (§6.1)", lineNo, string(s[0]))
	case '&':
		return Value{}, fmt.Errorf("linha %d: anchors (&) não são suportados (§6.1)", lineNo)
	case '*':
		return Value{}, fmt.Errorf("linha %d: aliases (*) não são suportados (§6.1)", lineNo)
	case '!':
		return Value{}, fmt.Errorf("linha %d: tags (!!) não são suportadas (§6.1)", lineNo)
	}
	sc, err := scalar(s, lineNo)
	if err != nil {
		return Value{}, err
	}
	return Value{Str: sc}, nil
}

func scalar(s string, lineNo int) (string, error) {
	if s[0] == '!' || s[0] == '&' || s[0] == '*' {
		return "", fmt.Errorf("linha %d: constructo %q não é suportado (§6.1)", lineNo, string(s[0]))
	}
	if len(s) >= 2 && ((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'')) {
		return s[1 : len(s)-1], nil
	}
	if s[0] == '"' || s[0] == '\'' {
		return "", fmt.Errorf("linha %d: aspas não fechadas (§6.1)", lineNo)
	}
	return s, nil
}
