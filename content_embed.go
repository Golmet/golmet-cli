// Conteúdo FULL do workflow (L1) — registry.yaml + sources/ embutidos no
// binário. Padrão espelha skills_embed.go: go:embed não sobe diretórios,
// então coredata/ vive na raiz. `golmet install --full` renderiza este FS
// para o engine alvo e materializa agents/ + skills/ no config (com
// proveniência por arquivo no manifest — modelo de camadas DR-021).
package golmet

import (
	"embed"
	"io/fs"
)

//go:embed all:coredata
var coredataFS embed.FS

// CoreDataFS devolve o FS com coredata/registry.yaml + coredata/sources/.
func CoreDataFS() fs.FS { return coredataFS }
