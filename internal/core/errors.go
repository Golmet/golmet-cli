// Package core define os tipos compartilhados do kit — em especial o
// erro estruturado consumível por agentes (FORMATO v1 §11).
package core

// Severity classifica um finding: hard (bloqueia) ou soft (aviso).
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// GolmetError é o erro estruturado do FORMATO v1 §11. Os campos file,
// entity, id, rule, message e fix são o contrato §11; line e severity
// são extensões v0 (aditivas) para saída human e degradação --tolerant.
type GolmetError struct {
	File     string   `json:"file"`
	Line     int      `json:"line,omitempty"`
	Entity   string   `json:"entity"`
	ID       string   `json:"id,omitempty"`
	Rule     string   `json:"rule"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	Fix      string   `json:"fix"`
}

// Error implementa error para uso incidental em logs.
func (e GolmetError) Error() string {
	loc := e.File
	if e.Line > 0 {
		loc += ":" + itoa(e.Line)
	}
	id := ""
	if e.ID != "" {
		id = " " + e.ID
	}
	return "[" + loc + "] " + e.Entity + id + " [" + e.Rule + "] " + e.Message
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
