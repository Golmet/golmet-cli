package rules

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// skills.go — leitura das skills vendored em .golmet/skills/ (S3/S5 e o
// agregador de capacidades do doctor). O front matter de skill usa YAML
// completo (§10), NÃO o subset §6.1 — então name/requires/degrades são
// extraídos por regex, sem rejeitar o arquivo.

type skillInfo struct {
	name        string
	file        string // caminho relativo com '/'
	requires    []string
	requiresErr bool
	degrades    map[string]string // capacidade ausente → estratégia declarada (§10)
}

var (
	skillNameRe     = regexp.MustCompile(`(?m)^name:\s*(\S+)`)
	skillRequiresRe = regexp.MustCompile(`(?m)^requires:\s*\[([^\]]*)\]`)
	degradePairRe   = regexp.MustCompile(`^([A-Za-z][\w.-]*):\s*(\S.*)$`)
)

// parseDegrades extrai o bloco `degrades:` do front matter (capacidade →
// estratégia, ex.: `ask: chat-stop`). Varredura linha a linha: o bloco
// abre na chave `degrades:` e fecha na primeira linha não-indentada
// não-vazia (nunca alcança o corpo do markdown).
func parseDegrades(content string) map[string]string {
	out := map[string]string{}
	inBlock := false
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue // vazio: antes do bloco ignora; dentro tolera
		}
		indented := strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")
		if !inBlock {
			if trimmed == "degrades:" {
				inBlock = true
			}
			continue
		}
		if !indented {
			break // fim do bloco (próxima chave de 1º nível ou `---`)
		}
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if m := degradePairRe.FindStringSubmatch(trimmed); m != nil {
			out[strings.ToLower(m[1])] = strings.TrimSpace(m[2])
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// readSkills lê .golmet/skills/*/SKILL.md.
func readSkills(root string) []skillInfo {
	base := filepath.Join(root, ".golmet", "skills")
	dirs, err := os.ReadDir(base)
	if err != nil {
		return nil
	}
	var out []skillInfo
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		sk := skillInfo{
			name: d.Name(),
			file: filepath.ToSlash(filepath.Join(".golmet", "skills", d.Name(), "SKILL.md")),
		}
		data, err := os.ReadFile(filepath.Join(base, d.Name(), "SKILL.md"))
		if err != nil {
			sk.requiresErr = true
			out = append(out, sk)
			continue
		}
		content := string(data)
		if m := skillNameRe.FindStringSubmatch(content); m != nil {
			sk.name = m[1]
		}
		if m := skillRequiresRe.FindStringSubmatch(content); m != nil {
			for _, p := range strings.Split(m[1], ",") {
				if item := strings.TrimSpace(p); item != "" {
					sk.requires = append(sk.requires, item)
				}
			}
		} else {
			sk.requiresErr = true
		}
		sk.degrades = parseDegrades(content)
		out = append(out, sk)
	}
	return out
}

// detectEngine heurística (bloco 3): qualquer engine prove no mínimo
// BASE (file+shell); .mcp.json ou .opencode/ acrescentam mcp; o kit
// (este binário) prove os checkpoints `kit` (§10). Perfis FULL são F3.
func detectEngine(root string) map[string]bool {
	caps := map[string]bool{"file": true, "shell": true, "kit": true}
	if _, err := os.Stat(filepath.Join(root, ".mcp.json")); err == nil {
		caps["mcp"] = true
	}
	if st, err := os.Stat(filepath.Join(root, ".opencode")); err == nil && st.IsDir() {
		caps["mcp"] = true
	}
	return caps
}

// ReadSkills exporta a leitura das skills vendored (.golmet/skills/) para
// o doctor — mesma fonte que S3/S5 consomem (inclui Degrades §10).
func ReadSkills(root string) []SkillInfo {
	sks := readSkills(root)
	out := make([]SkillInfo, 0, len(sks))
	for _, sk := range sks {
		out = append(out, SkillInfo{
			Name: sk.name, File: sk.file,
			Requires: sk.requires, RequiresErr: sk.requiresErr,
			Degrades: sk.degrades,
		})
	}
	return out
}

// DetectEngine exporta a heurística de capacidades do engine detectado
// (S3 — file+shell sempre; .mcp.json/.opencode/ → mcp).
func DetectEngine(root string) map[string]bool { return detectEngine(root) }

// ValidCapability reporta se a capacidade está no vocabulário (§10).
func ValidCapability(cap string) bool { return capabilityVocab[cap] }

// SkillInfo é a visão pública de uma skill vendored (doctor).
type SkillInfo struct {
	Name        string
	File        string
	Requires    []string
	RequiresErr bool
	Degrades    map[string]string // capacidade ausente → estratégia declarada (§10)
}

// RequiresKit reporta se a skill declara a capacidade kit.<cap> (§10).
func (s SkillInfo) RequiresKit(cap string) bool {
	for _, r := range s.Requires {
		if r == "kit."+cap {
			return true
		}
	}
	return false
}

// DegradeFor devolve a estratégia de degradação declarada para a
// capacidade ("" se a skill não declara fallback para ela).
func (s SkillInfo) DegradeFor(capability string) string {
	return s.Degrades[capability]
}
