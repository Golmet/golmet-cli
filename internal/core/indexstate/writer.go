// writer.go — persistência do golmet.index.json (FORMAT §8, D2).
// O kit é o ÚNICO escritor do índice: uma validate sem erros hard
// reescreve (última validação boa = estado do índice); com erro hard
// NÃO escreve (história preservada para H6/H8); --fix força a
// recomposição ignorando divergências (recuperação pós-merge).
package indexstate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"time"
)

// KitVersion é a versão do subset BASE gravada no .golmet/manifest.json
// pelo update (bloco 3) e usada como versão do servidor MCP.
const KitVersion = "1.0.0"

// SectionHash é a normalização canônica de hash de seção (§8): SHA-256
// hex das linhas ORIGINAIS da seção (RawLines) unidas por "\n" — a MESMA
// normalização que H8 usa para imutabilidade de ADR aceito (§7.8).
func SectionHash(rawLines []string) string {
	sum := sha256.Sum256([]byte(strings.Join(rawLines, "\n")))
	return hex.EncodeToString(sum[:])
}

// SectionSnap é o retrato de uma entidade seccional (rn/pend/cr/adr/cand)
// pronto para o Build. Evidencia é opcional (status não resolve hashes).
type SectionSnap struct {
	NS        string // namespace no índice: rn | pend | cr | adr | cand
	ID        string
	Estado    string // já resolvido (diretiva ou default)
	RawLines  []string
	Evidencia []EvidenceRef
}

// SpecSnap é o retrato de um Spec (§7.7) — chave = RN herdado da pasta.
type SpecSnap struct {
	ID     string
	Estado string
}

// FeatureSnap é o retrato de uma Feature (§7.5) + Tasks (§7.6).
type FeatureSnap struct {
	Key    string // anchor ou caminho do arquivo (entities.FeatureKey)
	Status string
	Anchor string
	Tasks  map[string]string
}

// Snapshot é o retrato neutro da prosa consumido por Build — sem depender
// do motor de regras (scan monta; status monta sem evidência/resolver).
type Snapshot struct {
	CharterFound  bool
	CharterStatus string
	Sections      []SectionSnap
	Specs         []SpecSnap
	Features      []FeatureSnap
}

// Build compõe o índice (§8) a partir do retrato. Todas as namespaces
// existem (mesmo vazias) — shape estávio para diffs determinísticos.
func Build(workspace string, s Snapshot) *Index {
	idx := &Index{
		Format:      1,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Workspace:   workspace,
		Entities: map[string]map[string]EntityState{
			"rn": {}, "pend": {}, "cr": {}, "adr": {}, "cand": {},
			"feature": {}, "spec": {}, "charter": {},
		},
		Counts: map[string]int{},
	}
	for _, sec := range s.Sections {
		m := idx.Entities[sec.NS]
		if m == nil {
			continue // namespace desconhecida — ignora (defensivo)
		}
		m[sec.ID] = EntityState{
			Estado:    sec.Estado,
			SecaoHash: SectionHash(sec.RawLines),
			Evidencia: sec.Evidencia,
		}
	}
	for _, sp := range s.Specs {
		if sp.ID == "" {
			continue
		}
		idx.Entities["spec"][sp.ID] = EntityState{Estado: sp.Estado}
	}
	for _, f := range s.Features {
		idx.Entities["feature"][f.Key] = EntityState{
			Status: f.Status,
			Anchor: f.Anchor,
			Tasks:  f.Tasks,
		}
	}
	if s.CharterFound {
		idx.Entities["charter"]["PROJECT_CHARTER.md"] = EntityState{Status: s.CharterStatus}
	}
	idx.Recount()
	return idx
}

// Recount recalcula counts (§8) a partir de Entities: total por namespace,
// por estado (`<ns>_<estado>`), tasks agregadas da feature e os aliases
// nominais do FORMAT (pend_open, cr_pending, features_done).
func (i *Index) Recount() {
	c := map[string]int{}
	for ns, m := range i.Entities {
		if ns == "charter" {
			continue // charter não entra em totals (§8 exemplifica entidades seccionais)
		}
		c[ns] = len(m)
		for _, e := range m {
			st := e.Estado
			if st == "" {
				st = e.Status
			}
			if st != "" {
				c[ns+"_"+st]++
			}
			if ns == "feature" {
				for _, ts := range e.Tasks {
					c["task"]++
					c["task_"+ts]++
				}
			}
		}
	}
	c["features_done"] = c["feature_done"] // alias literal do §8
	i.Counts = c
}

// SecaoHashOf devolve o secao_hash registrado no índice para (ns, id).
// H8 compara contra ESTE campo (não contra o estado).
func (i *Index) SecaoHashOf(ns, id string) string {
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
	return e.SecaoHash
}

// WriteIfChanged grava o índice quando o estado semântico diverge do
// arquivo existente (format/workspace/entities/counts — generated_at não
// conta como divergência: evita churn de commit a cada validate).
// Retorna true quando o arquivo foi (re)escrito. Índice ilegível no
// disco = reescreve (o kit é o único escritor e ele está são).
func WriteIfChanged(path string, idx *Index) (bool, error) {
	if data, err := os.ReadFile(path); err == nil {
		if old, perr := Parse(data); perr == nil && sameState(old, idx) {
			return false, nil
		}
	}
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return false, err
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return false, err
	}
	return true, nil
}

// sameState compara dois índices ignorando generated_at (cache semântico)
// e a diferença nil-vs-vazio introduzida pelo round-trip JSON (omitempty
// grava feature sem tasks como ausente; o parse lê nil; o rebuild produz
// mapa vazio — semanticamente iguais).
func sameState(a, b *Index) bool {
	return a.Format == b.Format &&
		a.Workspace == b.Workspace &&
		reflect.DeepEqual(normalizedEntities(a), normalizedEntities(b)) &&
		reflect.DeepEqual(a.Counts, b.Counts)
}

// normalizedEntities devolve uma cópia das entidades com nil-vs-vazio
// normalizado (Tasks/Evidencia nil → vazio) para comparação estável.
func normalizedEntities(i *Index) map[string]map[string]EntityState {
	out := make(map[string]map[string]EntityState, len(i.Entities))
	for ns, m := range i.Entities {
		nm := make(map[string]EntityState, len(m))
		for id, e := range m {
			ne := e
			if ne.Tasks == nil {
				ne.Tasks = map[string]string{}
			}
			if ne.Evidencia == nil {
				ne.Evidencia = []EvidenceRef{}
			}
			nm[id] = ne
		}
		out[ns] = nm
	}
	return out
}

// Marshal renderiza o índice no formato commitado (indentado + \n final) —
// usado pelo init para o baseline.
func (i *Index) Marshal() ([]byte, error) {
	data, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
