package cli

import (
	"encoding/json"
	"io"
)

// stubs.go — nenhum comando do kit é mais stub: os 11 (validate, init,
// adopt, status, doctor, update, tools, mcp, index, next, eval) são
// funcionais desde o bloco 5. Este arquivo mantém apenas o encoder JSON
// compartilhado pelos comandos com --json.

// newJSONEncoder padroniza a serialização --json (indentado, UTF-8).
func newJSONEncoder(w io.Writer) *json.Encoder {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc
}
