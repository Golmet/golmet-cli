package scan

import (
	"github.com/Golmet/golmet-cli/internal/core"
)

type Severity = core.Severity
type GolmetError = core.GolmetError

const (
	severityError   = core.SeverityError
	severityWarning = core.SeverityWarning
)

func summary(xs []core.GolmetError) []string {
	var out []string
	for _, e := range xs {
		out = append(out, e.Rule+"@"+e.File+":"+itoa(e.Line))
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
