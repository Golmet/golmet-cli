// Package update implementa o `golmet update` (FORMATO §2, DR-008):
// instala o subset BASE de skills embutido no binário em
// <workspace>/.golmet/skills/ e grava o manifest. A zona .golmet/ é
// kit-gerida — o kit é o ÚNICO escritor dela; re-run sobrescreve
// skills/manifest (preservando skills modificadas localmente) e NUNCA
// toca o resto do workspace. `--check` compara embutido vs instalado
// sem escrever nada.
package update

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	golmet "github.com/Golmet/golmet-cli"
	"github.com/Golmet/golmet-cli/internal/core/content"
	"github.com/Golmet/golmet-cli/internal/core/frontmatter"
	"github.com/Golmet/golmet-cli/internal/core/indexstate"
)

// Entry é uma skill registrada no manifest (hash do arquivo EMBUTIDO —
// a referência do kit; conteúdo divergente no disco é reportado como
// modificado pelo --check).
type Entry struct {
	Name   string `json:"name"`
	File   string `json:"file"` // caminho relativo à zona .golmet/, separador '/'
	SHA256 string `json:"sha256"`
	Layer  string `json:"layer,omitempty"` // base (default) | core | local — modelo de camadas (DR-021)
}

// LayerInfo registra a origem de uma camada instalada (modelo de camadas
// DR-021: embedded | <repo>@<sha>).
type LayerInfo struct {
	Source      string `json:"source"`
	KitVersion  string `json:"kit_version,omitempty"`
	InstalledAt string `json:"installed_at,omitempty"`
}

// Manifest é o .golmet/manifest.json (compatível com o shape do bloco 2:
// "version" = versão do kit que instalou; vazio = scaffold pré-update).
// Layers/layer são opcionais — manifests de manifests pré-0.1 parseiam igual.
type Manifest struct {
	Version     string               `json:"version"`
	InstalledAt string               `json:"installed_at,omitempty"`
	Skills      []Entry              `json:"skills"`
	Layers      map[string]LayerInfo `json:"layers,omitempty"`
}

// ParseManifest lê o manifest do workspace (ausente → nil, sem erro).
func ParseManifest(root string) (*Manifest, error) {
	data, err := os.ReadFile(filepath.Join(root, ".golmet", "manifest.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("manifest ilegível: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("manifest ilegível: %w", err)
	}
	return &m, nil
}

// EmbeddedSkill é uma skill do subset BASE embutida no binário.
type EmbeddedSkill struct {
	Name    string // campo name do front matter
	Dir     string // pasta skills/<dir>
	File    string // caminho no FS embutido, separador '/'
	Content []byte
	SHA256  string
}

// SkillError é o erro estruturado de validação de front matter (§10).
type SkillError struct {
	File    string `json:"file"`
	Problem string `json:"problem"`
}

func (e SkillError) Error() string { return "[" + e.File + "] " + e.Problem }

// EmbeddedSkills lê e valida TODAS as skills embutidas. Qualquer front
// matter inválido aborta (erro estruturado) — o kit nunca instala
// skills corrompidas.
func EmbeddedSkills() ([]EmbeddedSkill, []SkillError, error) {
	var out []EmbeddedSkill
	entries, err := fs.ReadDir(golmet.SkillsFS(), "skills")
	if err != nil {
		return nil, nil, fmt.Errorf("subset BASE embutido ilegível: %w", err)
	}
	var skillErrs []SkillError
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "gm-") {
			continue
		}
		rel := "skills/" + e.Name() + "/SKILL.md"
		data, rerr := fs.ReadFile(golmet.SkillsFS(), rel)
		if rerr != nil {
			skillErrs = append(skillErrs, SkillError{File: rel,
				Problem: "SKILL.md ausente na pasta da skill"})
			continue
		}
		name, serr := validateFrontMatter(rel, data)
		if serr != nil {
			skillErrs = append(skillErrs, *serr)
			continue
		}
		sum := sha256.Sum256(data)
		out = append(out, EmbeddedSkill{
			Name:    name,
			Dir:     e.Name(),
			File:    rel,
			Content: data,
			SHA256:  hex.EncodeToString(sum[:]),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	if len(skillErrs) > 0 {
		return nil, skillErrs, nil
	}
	return out, nil, nil
}

// validateFrontMatter valida o front matter §10 com o parser §6.1 real:
// delimitadores ---, name escalar obrigatório (prefixo gm- no BASE) e
// requires como array inline.
func validateFrontMatter(rel string, data []byte) (string, *SkillError) {
	doc, has, err := frontmatter.Extract(string(data))
	if err != nil {
		return "", &SkillError{File: rel, Problem: "front matter ilegível (" + err.Error() + ")"}
	}
	if !has {
		return "", &SkillError{File: rel, Problem: "front matter ausente (esperado --- … --- §10)"}
	}
	nameV, ok := doc.Fields["name"]
	if !ok || nameV.Str == "" {
		return "", &SkillError{File: rel, Problem: "campo name ausente (§10)"}
	}
	if !strings.HasPrefix(nameV.Str, "gm-") {
		return "", &SkillError{File: rel, Problem: "name \"" + nameV.Str + "\" sem prefixo gm- (§10 BASE)"}
	}
	if reqV, ok := doc.Fields["requires"]; ok && !reqV.IsArr {
		return "", &SkillError{File: rel, Problem: "requires deve ser array inline de escalares (§6.1/§10)"}
	}
	return nameV.Str, nil
}

// InstallReport resume a instalação.
type InstallReport struct {
	Workspace  string
	KitVersion string
	Installed  int      // skills escritas (novas ou atualizadas)
	Preserved  []string // modificadas localmente — preservadas (DR-008)
	Total      int      // skills no subset BASE
}

// CheckReport é o resultado do --check (consulta, sem escrita).
type CheckReport struct {
	KitVersion       string   `json:"kit_version"`
	InstalledVersion string   `json:"installed_version,omitempty"`
	InstalledAt      string   `json:"installed_at,omitempty"`
	NotInstalled     bool     `json:"not_installed,omitempty"`
	Faltantes        []string `json:"faltantes,omitempty"` // no subset, ausentes do disco
	Extras           []string `json:"extras,omitempty"`    // em disco, fora do subset
	Modificados      []string `json:"modificados,omitempty"`
	VerConforme      bool     `json:"-"`
}

// Install instala o subset BASE no workspace. Modified-skill preservation:
// skill cujo hash em disco diverge do manifest instalado é PRESERVADA
// (DR-008) e listada no report; as demais são sobrescritas pelo embed.
// O manifest é sempre reescrito (kit é o único escritor da zona).
func Install(root string) (*InstallReport, error) {
	skills, skillErrs, err := EmbeddedSkills()
	if err != nil {
		return nil, err
	}
	if len(skillErrs) > 0 {
		var msgs []string
		for _, se := range skillErrs {
			msgs = append(msgs, se.Error())
		}
		return nil, fmt.Errorf("subset BASE com front matter inválido — não instalado:\n%s", strings.Join(msgs, "\n"))
	}

	prev, perr := ParseManifest(root)
	if perr != nil {
		return nil, perr // manifest corrompido: não adivinhar — exigir correção manual
	}
	prevHash := map[string]string{}
	if prev != nil {
		for _, en := range prev.Skills {
			prevHash[en.Name] = en.SHA256
		}
	}

	zone := filepath.Join(root, ".golmet", "skills")
	rep := &InstallReport{Workspace: root, KitVersion: indexstate.KitVersion, Total: len(skills)}
	for _, sk := range skills {
		target := filepath.Join(root, ".golmet", filepath.FromSlash(sk.File))
		// DR-008: conteúdo modificado localmente (diverge do instalado mesmo
		// normalizando EOL) é preservado — diferença só de EOL não é
		// customização: rematerializa com o canônico LF (cura silenciosa).
		// Byte-idêntico = no-op (idempotente — re-run não conta escrita).
		if data, rerr := os.ReadFile(target); rerr == nil {
			sum := sha256.Sum256(data)
			diskHash := hex.EncodeToString(sum[:])
			switch {
			case diskHash == sk.SHA256:
				continue
			case prevHash[sk.Name] != "" &&
				diskHash != prevHash[sk.Name] &&
				!content.SameContentNormalized(data, sk.Content):
				rep.Preserved = append(rep.Preserved, sk.Name)
				continue
			}
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil, fmt.Errorf("falha ao criar a zona do kit: %w", err)
		}
		if err := os.WriteFile(target, sk.Content, 0o644); err != nil {
			return nil, fmt.Errorf("falha ao escrever %s: %w", sk.File, err)
		}
		rep.Installed++
	}
	// pasta de skills existe mesmo com preservadas (zona kit-gerida)
	if err := os.MkdirAll(zone, 0o755); err != nil {
		return nil, fmt.Errorf("falha ao criar a zona do kit: %w", err)
	}

	manifest := Manifest{
		Version:     indexstate.KitVersion,
		InstalledAt: time.Now().UTC().Format(time.RFC3339),
		Layers: map[string]LayerInfo{
			"base": {Source: "embedded", KitVersion: indexstate.KitVersion},
		},
	}
	for _, sk := range skills {
		manifest.Skills = append(manifest.Skills, Entry{
			Name:   sk.Name,
			File:   sk.File,
			SHA256: sk.SHA256,
			Layer:  "base",
		})
	}
	if err := writeManifest(root, &manifest); err != nil {
		return nil, err
	}
	return rep, nil
}

func writeManifest(root string, m *Manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("falha ao serializar o manifest: %w", err)
	}
	data = append(data, '\n')
	target := filepath.Join(root, ".golmet", "manifest.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("falha ao criar a zona do kit: %w", err)
	}
	if err := os.WriteFile(target, data, 0o644); err != nil {
		return fmt.Errorf("falha ao escrever o manifest: %w", err)
	}
	return nil
}

// WriteManifest grava o manifest no root (exportado — o materialize do
// engine config reusa a zona .golmet/ com o mesmo formato).
func WriteManifest(root string, m *Manifest) error { return writeManifest(root, m) }

// Check compara o subset embutido vs o instalado — consulta pura,
// nunca escreve.
func Check(root string) (*CheckReport, error) {
	skills, skillErrs, err := EmbeddedSkills()
	if err != nil {
		return nil, err
	}
	if len(skillErrs) > 0 {
		var msgs []string
		for _, se := range skillErrs {
			msgs = append(msgs, se.Error())
		}
		return nil, fmt.Errorf("subset BASE embutido inválido:\n%s", strings.Join(msgs, "\n"))
	}

	rep := &CheckReport{KitVersion: indexstate.KitVersion}
	manifest, merr := ParseManifest(root)
	if merr != nil {
		return nil, merr
	}
	if manifest == nil || len(manifest.Skills) == 0 {
		rep.NotInstalled = true
		return rep, nil
	}
	rep.InstalledVersion = manifest.Version
	rep.InstalledAt = manifest.InstalledAt

	inSubset := map[string]string{}     // name -> hash embutido
	embeddedData := map[string][]byte{} // name -> conteúdo embutido (para comparação EOL-tolerante)
	for _, sk := range skills {
		inSubset[sk.Name] = sk.SHA256
		embeddedData[sk.Name] = sk.Content
	}
	inManifest := map[string]bool{}
	for _, en := range manifest.Skills {
		inManifest[en.Name] = true
		if _, ok := inSubset[en.Name]; !ok {
			rep.Extras = append(rep.Extras, en.Name) // no manifest, fora do subset
			continue
		}
		target := filepath.Join(root, ".golmet", filepath.FromSlash(en.File))
		data, rerr := os.ReadFile(target)
		if rerr != nil {
			rep.Faltantes = append(rep.Faltantes, en.Name)
			continue
		}
		// EOL-tolerante: diferença só de CRLF/CR vs LF não é modificação
		if !content.SameContentNormalized(data, embeddedData[en.Name]) {
			rep.Modificados = append(rep.Modificados, en.Name)
		}
	}
	// skills do subset presentes em disco mas ausentes do manifest
	for _, sk := range skills {
		if !inManifest[sk.Name] {
			if _, rerr := os.Stat(filepath.Join(root, ".golmet", filepath.FromSlash(sk.File))); rerr == nil {
				rep.Extras = append(rep.Extras, sk.Name)
			} else {
				rep.Faltantes = append(rep.Faltantes, sk.Name)
			}
		}
	}
	sort.Strings(rep.Faltantes)
	sort.Strings(rep.Extras)
	sort.Strings(rep.Modificados)
	return rep, nil
}
