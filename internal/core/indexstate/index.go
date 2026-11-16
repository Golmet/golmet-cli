// Package indexstate define os structs de golmet.index.json (FORMAT §8).
// Bloco 1: apenas leitura/parse para comparação prosa↔índice (H6/H8).
// Persistência (escrita do índice) fica para o bloco 2.
package indexstate

import (
	"encoding/json"
	"fmt"
)

// EvidenceRef é a referência de evidência gravada no índice (§8).
type EvidenceRef struct {
	Path   string `json:"path"`
	Line   int    `json:"line"`
	Hash   string `json:"hash"`
	HeadOK bool   `json:"head_ok"`
}

// EntityState é o estado de uma entidade no índice. RN/PEND/CR/ADR/CAND/
// Spec usam Estado (+SecaoHash para ADR); Feature usa Status/Anchor/Tasks.
type EntityState struct {
	Estado    string            `json:"estado,omitempty"`
	Status    string            `json:"status,omitempty"`
	Anchor    string            `json:"anchor,omitempty"`
	SecaoHash string            `json:"secao_hash,omitempty"`
	Tasks     map[string]string `json:"tasks,omitempty"`
	Evidencia []EvidenceRef     `json:"evidencia,omitempty"`
}

// Index é a estrutura raiz do golmet.index.json (§8).
type Index struct {
	Format      int                               `json:"format"`
	GeneratedAt string                            `json:"generated_at"`
	Workspace   string                            `json:"workspace"`
	Entities    map[string]map[string]EntityState `json:"entities"`
	Counts      map[string]int                    `json:"counts"`
}

// Parse decodifica e valida o índice (format: 1).
func Parse(data []byte) (*Index, error) {
	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("json inválido: %w", err)
	}
	if idx.Format != 1 {
		return nil, fmt.Errorf("format %d não suportado (esperado 1)", idx.Format)
	}
	return &idx, nil
}

// Estado devolve o estado/status registrado no índice para (ns, id).
func (i *Index) Estado(ns, id string) string {
	if i == nil {
		return ""
	}
	m, ok := i.Entities[ns]
	if !ok {
		return ""
	}
	e, ok := m[id]
	if !ok {
		return ""
	}
	if e.Estado != "" {
		return e.Estado
	}
	return e.Status
}

// Task devolve o estado de uma task no índice (via feature/anchor).
func (i *Index) Task(anchor, taskID string) string {
	if i == nil {
		return ""
	}
	m, ok := i.Entities["feature"]
	if !ok {
		return ""
	}
	e, ok := m[anchor]
	if !ok {
		return ""
	}
	return e.Tasks[taskID]
}
