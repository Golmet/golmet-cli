// container.go — contexto CONTAINER do validate (FORMAT §2.3): um
// container de organização NÃO é workspace-produto — aplicar a gramática
// integral (H1-H10/S1-S5) sobre ele produzia centenas de findings falsos
// sobre os produtos ainda não adotados ("RN-022 não existe no workspace").
// Aqui o validate aplica o contrato da DECLARAÇÃO da org: produto
// declarado sem dir = finding C1 (declaração stale); produto com dir e
// golmet.yaml próprio = ADOTADO (validate de produto disponível via
// `golmet validate <produto>`); com dir sem golmet.yaml = NÃO-ADOTADO —
// estado legítimo, sem finding. Exit 0 ⇔ declaração íntegra.
package scan

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/Golmet/golmet-cli/internal/core"
	"github.com/Golmet/golmet-cli/internal/core/frontmatter"
)

// ContainerReport é o retrato do validate em contexto container: a
// declaração (products:) ↔ disco e o estado de adoção de cada produto.
type ContainerReport struct {
	Name       string   `json:"name"`
	Total      int      `json:"products"`
	Adopted    []string `json:"adopted"`
	NotAdopted []string `json:"not_adopted"`
}

// Summary — a linha de estado da declaração (relatório humano e insumo do
// MCP `gm_validate`).
func (c *ContainerReport) Summary() string {
	return fmt.Sprintf("container %s: %d produto(s) — %d adotado(s) · %d não-adotado(s) — o validate de produto fica disponível ao adotar com `golmet init <produto> --into <container>`",
		c.Name, c.Total, len(c.Adopted), len(c.NotAdopted))
}

// validateContainer valida a declaração da org contra o disco. Nome do
// produto aceita subpath (mesma semântica do container.Inspect — §2.3).
func validateContainer(root string, fields map[string]frontmatter.Value) (*ContainerReport, []core.GolmetError) {
	rep := &ContainerReport{Name: filepath.Base(root)}
	var errs []core.GolmetError
	for _, p := range fields["products"].Arr {
		p = filepath.ToSlash(filepath.Clean(p))
		rep.Total++
		if !isContainerDir(filepath.Join(root, filepath.FromSlash(p))) {
			errs = append(errs, core.GolmetError{
				Rule: "C1", Entity: "Container", File: "golmet.yaml", Severity: core.SeverityError,
				Message: fmt.Sprintf("produto declarado no products: sem dir no disco: %s", p),
				Fix:     "remova do products: do golmet.yaml (o produto não existe mais) ou crie o dir com 'golmet init <produto> --into <container>'",
			})
			continue
		}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(p), "golmet.yaml")); err == nil {
			rep.Adopted = append(rep.Adopted, p)
		} else {
			rep.NotAdopted = append(rep.NotAdopted, p)
		}
	}
	sort.Strings(rep.Adopted)
	sort.Strings(rep.NotAdopted)
	return rep, errs
}

func isContainerDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}
