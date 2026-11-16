package uninstall

import "strings"

// stripDanglingCommas remove vírgulas pendentes antes de }/] (fora de strings) -
// a remoção de keys do JSONC pode deixar a vírgula da key anterior pendente
// (JSONC tolera trailing comma; o JSON resultante precisa ser válido).
// Preserva comentários de linha e bloco.
func stripDanglingCommas(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inStr, inLine, inBlock := false, false, false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inStr {
			b.WriteByte(ch)
			if ch == '\\' && i+1 < len(s) {
				i++
				b.WriteByte(s[i])
			} else if ch == '"' {
				inStr = false
			}
			continue
		}
		if inLine {
			b.WriteByte(ch)
			if ch == '\n' {
				inLine = false
			}
			continue
		}
		if inBlock {
			b.WriteByte(ch)
			if ch == '*' && i+1 < len(s) && s[i+1] == '/' {
				i++
				b.WriteByte('/')
				inBlock = false
			}
			continue
		}
		switch {
		case ch == '"':
			inStr = true
			b.WriteByte(ch)
		case ch == '/' && i+1 < len(s) && s[i+1] == '/':
			inLine = true
			b.WriteByte(ch)
		case ch == '/' && i+1 < len(s) && s[i+1] == '*':
			inBlock = true
			b.WriteByte(ch)
		case ch == ',':
			// lookahead: pula ws/comentários; se o próximo significativo for
			// } ou ], a vírgula está pendente - descarta (não escreve)
			j := i + 1
			for j < len(s) {
				c := s[j]
				if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
					j++
					continue
				}
				if c == '/' && j+1 < len(s) && s[j+1] == '/' {
					j += 2
					for j < len(s) && s[j] != '\n' {
						j++
					}
					continue
				}
				if c == '/' && j+1 < len(s) && s[j+1] == '*' {
					j += 2
					for j+1 < len(s) && !(s[j] == '*' && s[j+1] == '/') {
						j++
					}
					continue
				}
				break
			}
			if j < len(s) && (s[j] == '}' || s[j] == ']') {
				continue
			}
			b.WriteByte(ch)
		default:
			b.WriteByte(ch)
		}
	}
	return b.String()
}
