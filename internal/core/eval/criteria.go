// Package eval implementa o checker determinístico do `golmet eval`
// (DR-017, forma mínima): avalia criteria.yaml de uma fixture contra o
// workspace onde o engine produziu os artefatos. O kit NÃO semeia nem
// executa a skill — só pontua (semântica do runner de referência,
// porta fiel: substring exata para contains, varredura *.md recursiva
// em paths de diretório, Get-NumList com 1º capture group, etc.).
//
// O parser de criteria usa YAML COMPLETO (gopkg.in/yaml.v3 — o schema
// tem mapas aninhados e description multiline) — dialeto distinto do
// subset §6.1 usado nos documentos do workspace.
package eval

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Severity dos checks.
const (
	SeverityError   = "error"
	SeverityWarning = "warning"
)

// Status possíveis de um check.
const (
	StatusPass    = "PASS"
	StatusFail    = "FAIL" // severity error quebrado
	StatusWarn    = "WARN" // severity warning quebrado
	StatusSkipped = "SKIPPED"
)

// Check é um item de criteria.yaml. Campos específicos por type:
// contains/not_contains→Pattern · section_exists→Section · no_todos→Patterns ·
// id_sequence/cross_ref→Pattern(+AllowGaps/Sources) · line_count_range→Min/Max ·
// section_order→Sections · table_has_columns→Columns · reference_match→Expected ·
// qualquer→Expect("fail", IMP-100) e Severity (default error).
type Check struct {
	Type      string   `yaml:"type"`
	Path      string   `yaml:"path"`
	Severity  string   `yaml:"severity"`
	Pattern   string   `yaml:"pattern"`
	Patterns  []string `yaml:"patterns"`
	Section   string   `yaml:"section"`
	Sections  []string `yaml:"sections"`
	Sources   []string `yaml:"sources"`
	Min       *int     `yaml:"min"`
	Max       *int     `yaml:"max"`
	AllowGaps *bool    `yaml:"allow_gaps"`
	Columns   []string `yaml:"columns"`
	Expected  string   `yaml:"expected"`
	Expect    string   `yaml:"expect"`
}

// criteriaDoc é o envelope do criteria.yaml.
type criteriaDoc struct {
	Name           string   `yaml:"name"`
	Description    string   `yaml:"description"`
	InputsRequired []string `yaml:"inputs_required"`
	Checks         []Check  `yaml:"checks"`
}

// Criteria é o criteria.yaml parseado e validado.
type Criteria struct {
	Name           string
	Description    string
	InputsRequired []string
	Checks         []Check
}

// ParseCriteria decodifica e valida o criteria.yaml — campo obrigatório
// ausente = erro estruturado (§11-style, CLI sai 2). Algumas fixtures
// legadas trazem um header pré-FORMATO ("mode: manual") seguido de "—"
// — YAML multi-documento: o critério real é o documento que contém checks.
func ParseCriteria(data []byte) (*Criteria, error) {
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	var lastErr error
	for {
		var doc criteriaDoc
		err := decoder.Decode(&doc)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			lastErr = err
			break
		}
		if len(doc.Checks) == 0 {
			continue // documento sem checks (header legado) — próximo
		}
		return buildCriteria(&doc)
	}
	if lastErr != nil {
		return nil, fmt.Errorf("criteria.yaml ilegível: %w", lastErr)
	}
	return nil, fmt.Errorf("criteria.yaml sem checks — nada a pontuar")
}

// buildCriteria valida os checks do documento escolhido.
func buildCriteria(doc *criteriaDoc) (*Criteria, error) {
	c := &Criteria{
		Name:           doc.Name,
		Description:    strings.TrimSpace(doc.Description),
		InputsRequired: doc.InputsRequired,
		Checks:         doc.Checks,
	}
	for i := range c.Checks {
		chk := &c.Checks[i]
		if strings.TrimSpace(chk.Type) == "" {
			return nil, fmt.Errorf("check #%d sem type", i+1)
		}
		if chk.Severity == "" {
			chk.Severity = SeverityError
		}
		if err := validateParams(chk); err != nil {
			return nil, fmt.Errorf("check #%d (%s): %w", i+1, chk.Type, err)
		}
	}
	return c, nil
}

// validateParams confere os campos obrigatórios por tipo (mesmo contrato
// do runner de referência — campo faltante é erro de uso, não falha silenciosa).
func validateParams(chk *Check) error {
	switch chk.Type {
	case "contains", "not_contains", "id_sequence", "cross_ref":
		if chk.Pattern == "" {
			return fmt.Errorf("pattern é obrigatório para %s", chk.Type)
		}
	case "section_exists":
		if chk.Section == "" {
			return fmt.Errorf("section é obrigatória para section_exists")
		}
	case "no_todos":
		if len(chk.Patterns) == 0 {
			return fmt.Errorf("patterns é obrigatório para no_todos")
		}
	case "line_count_range":
		if chk.Min == nil || chk.Max == nil {
			return fmt.Errorf("min e max são obrigatórios para line_count_range")
		}
	case "section_order":
		if len(chk.Sections) == 0 {
			return fmt.Errorf("sections é obrigatório para section_order")
		}
	case "table_has_columns":
		if len(chk.Columns) == 0 {
			return fmt.Errorf("columns é obrigatório para table_has_columns")
		}
	case "file_exists", "not_file_exists", "dir_has_files", "frontmatter_valid", "mermaid_valid":
		if chk.Path == "" {
			return fmt.Errorf("path é obrigatório para %s", chk.Type)
		}
	case "reference_match":
		if chk.Expected == "" {
			return fmt.Errorf("expected é obrigatório para reference_match (baseline relativo à fixture)")
		}
	default:
		// tipo desconhecido NÃO é erro de parse — vira SKIPPED no engine
		// (mesma semântica do runner de referência)
	}
	return nil
}
