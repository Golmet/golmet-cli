package rules

import (
	"strings"

	"github.com/Golmet/golmet-cli/internal/core"
)

// S3 — requires de skill não satisfeitas pelo engine detectado
// (FORMATO §10/§9). Capacidades kit.* ficam no S5.
func S3(ctx *Context) []core.GolmetError {
	skills := readSkills(ctx.Root)
	if len(skills) == 0 {
		return nil
	}
	var out []core.GolmetError
	detected := detectEngine(ctx.Root)
	for _, sk := range skills {
		if sk.requiresErr {
			out = append(out, warn("S3", "Skill", sk.name, sk.file, 1,
				"skill sem requires parseável",
				"declara 'requires: [capacidade, ...]' no front matter"))
			continue
		}
		var missing []string
		for _, r := range sk.requires {
			if strings.HasPrefix(r, "kit.") {
				continue
			}
			if !capabilityVocab[r] || !detected[r] {
				missing = append(missing, r)
			}
		}
		if len(missing) > 0 {
			out = append(out, warn("S3", "Skill", sk.name, sk.file, 1,
				"requires não satisfeitas pelo engine detectado: "+strings.Join(missing, ", "),
				"use a escada de degradação da skill ou instale um engine FULL (§10)"))
		}
	}
	return out
}

// capabilityVocab — vocabulário de capacidades (§10). `kit` é o metadado
// declarado pelo subset BASE (ex.: gm-adopt) para skills que usam
// checkpoints do kit — provida pelo próprio binário golmet (detectEngine).
var capabilityVocab = map[string]bool{
	"file": true, "shell": true, "mcp": true, "ask": true, "subagent": true,
	"parallel": true, "workspace-ops": true, "db-introspect": true, "kit": true,
}
