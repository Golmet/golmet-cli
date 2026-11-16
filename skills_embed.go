// Package golmet expõe os assets do kit embutidos no binário — o subset
// BASE de skills (FORMAT §2: zona .golmet/ é vendored pelo `golmet
// update`). Package na raiz do módulo porque go:embed não sobe
// diretórios: a pasta skills/ vive na raiz.
package golmet

import (
	"embed"
	"io/fs"
)

//go:embed all:skills
var skillsFS embed.FS

// SkillsFS devolve o FS com skills/gm-*/SKILL.md embutidas.
func SkillsFS() fs.FS { return skillsFS }
