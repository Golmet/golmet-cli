// Package pack implementa packs de conteúdo L2. Um pack é um repo Git (ou
// diretório local) com pack.yaml e arquivos em agents/ e/ou skills/ (kind
// overlay — cache somente-leitura) ou um working copy Git bidirecional
// (kind state). O kit sincroniza, valida e aplica o overlay sem destruir
// customizações locais.
package pack

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Golmet/golmet-cli/internal/core/content"
	"github.com/Golmet/golmet-cli/internal/core/indexstate"
	"github.com/Golmet/golmet-cli/internal/core/source"
)

type Spec struct {
	ID           string   `json:"id"`
	Version      string   `json:"version"`
	Name         string   `json:"name"`
	Kind         string   `json:"kind"` // overlay (default) | state — working copy bidirecional
	RequiresCore string   `json:"requires_core,omitempty"`
	Provides     []string `json:"provides"`
	NoOverride   bool     `json:"no_override"`
}

// Arquétipos de pack (campo kind do pack.yaml). Overlay é o comportamento
// original: provides restritos a agents//skills/, cache somente-leitura e
// overlay unidirecional. State é o working copy Git bidirecional em
// <config>/packs/<id> — o conteúdo é o próprio clone (exceto pack.yaml).
const (
	KindOverlay = "overlay"
	KindState   = "state"
)

type Report struct {
	Spec      Spec                    `json:"spec"`
	Sync      *source.SyncReport      `json:"sync"`
	StateSync *source.StateSyncReport `json:"state_sync,omitempty"` // presente só em state packs
	Written   []string                `json:"written,omitempty"`
	Preserved []string                `json:"preserved,omitempty"`
	Conflicts []string                `json:"conflicts,omitempty"`
}

func ParseSpec(data []byte) (Spec, error) {
	var s Spec
	// tolerante a BOM de editores Windows (autores de pack geram UTF-8 com
	// BOM — sem strip a 1ª chave viraria \ufeffid e o id sairia vazio)
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	var listKey string
	for _, raw := range strings.Split(strings.ReplaceAll(string(data), "\r", ""), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "- ") && listKey == "provides" {
			s.Provides = append(s.Provides, strings.TrimSpace(strings.TrimPrefix(line, "- ")))
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key, val := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		listKey = ""
		switch key {
		case "id":
			s.ID = val
		case "version":
			s.Version = val
		case "name":
			s.Name = val
		case "kind":
			s.Kind = strings.Trim(strings.TrimSpace(val), "\"'")
		case "requires-core":
			s.RequiresCore = strings.Trim(strings.TrimSpace(val), "\"'")
		case "no-override":
			s.NoOverride = val == "true"
		case "provides":
			if strings.HasPrefix(val, "[") && strings.HasSuffix(val, "]") {
				for _, p := range strings.Split(strings.Trim(val, "[]"), ",") {
					if p = strings.TrimSpace(p); p != "" {
						s.Provides = append(s.Provides, p)
					}
				}
			} else {
				listKey = "provides"
			}
		}
	}
	if s.ID == "" {
		return s, fmt.Errorf("pack.yaml sem id")
	}
	if s.Version == "" {
		return s, fmt.Errorf("pack %s sem version", s.ID)
	}
	if s.Name == "" {
		return s, fmt.Errorf("pack %s sem name", s.ID)
	}
	if s.Kind == "" {
		s.Kind = KindOverlay
	}
	if s.Kind != KindOverlay && s.Kind != KindState {
		return s, fmt.Errorf("pack %s com kind desconhecido: %q — valores aceitos: overlay (padrão) | state", s.ID, s.Kind)
	}
	if s.Kind == KindState && len(s.Provides) > 0 {
		return s, fmt.Errorf("pack state %s não pode declarar provides — em state o conteúdo é o próprio working copy (exceto pack.yaml)", s.ID)
	}
	if s.Kind == KindOverlay && len(s.Provides) == 0 {
		return s, fmt.Errorf("pack %s sem provides", s.ID)
	}
	for i, p := range s.Provides {
		clean, err := normalizeProvidedPath(p)
		if err != nil {
			return s, err
		}
		s.Provides[i] = clean
	}
	return s, nil
}

func normalizeProvidedPath(raw string) (string, error) {
	value := strings.Trim(strings.TrimSpace(raw), "\"'")
	normalized := strings.ReplaceAll(filepath.ToSlash(value), "\\", "/")
	if normalized == "" || strings.HasPrefix(normalized, "/") ||
		(len(normalized) > 1 && normalized[1] == ':') {
		return "", fmt.Errorf("provides fora do escopo: %s", raw)
	}
	for _, part := range strings.Split(normalized, "/") {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("provides contém componente inseguro: %s", raw)
		}
	}
	if !strings.HasPrefix(normalized, "agents/") && !strings.HasPrefix(normalized, "skills/") {
		return "", fmt.Errorf("provides deve estar em agents/ ou skills/: %s", raw)
	}
	return normalized, nil
}

func loadSpec(root string) (Spec, error) {
	data, err := os.ReadFile(filepath.Join(root, "pack.yaml"))
	if err != nil {
		return Spec{}, fmt.Errorf("pack.yaml: %w", err)
	}
	return ParseSpec(data)
}

func Add(home, configRoot, engine, sourceURL string, force bool) (*Report, error) {
	return AddWithOptions(home, configRoot, engine, sourceURL, force, false)
}

// AddWithOptions aplica um pack; em dry-run só lê a fonte e calcula o plano.
// O arquétipo (kind do pack.yaml) escolhe o caminho: overlay segue o fluxo
// original (cache somente-leitura + provides); state materializa a partir do
// working copy bidirecional em <configRoot>/packs/<id>.
func AddWithOptions(home, configRoot, engine, sourceURL string, force, dryRun bool) (*Report, error) {
	spec, probeDir, cleanup, err := probeSource(sourceURL, filepath.Join(home, ".golmet", "packs"))
	if err != nil {
		return nil, err
	}
	defer cleanup()
	if spec.Kind == KindState {
		return addOrUpdateState(configRoot, sourceURL, spec, probeDir, force, dryRun)
	}
	// ---- fluxo overlay (comportamento original intocado) ----
	sync, err := source.SyncWithOptions(sourceURL, filepath.Join(home, ".golmet", "packs"), source.SyncOptions{DryRun: dryRun})
	if err != nil {
		return nil, err
	}
	spec, err = loadSpec(sync.Path)
	if err != nil {
		return nil, err
	}
	if err := ValidateRequiresCore(spec.RequiresCore, indexstate.KitVersion); err != nil {
		return nil, fmt.Errorf("pack %s: %w", spec.ID, err)
	}
	files := make([]content.OverlayFile, 0, len(spec.Provides))
	for _, rel := range spec.Provides {
		data, err := readProvidedFile(sync.Path, rel)
		if err != nil {
			return nil, fmt.Errorf("pack %s: arquivo %s: %w", spec.ID, rel, err)
		}
		files = append(files, content.OverlayFile{Rel: rel, Data: data})
	}
	rep, err := content.ApplyOverlay(configRoot, files, content.OverlayOptions{
		Layer: "pack:" + spec.ID, Source: source.Redact(sourceURL) + "@" + sync.SHA,
		NoOverride: spec.NoOverride, Force: force, DryRun: dryRun,
	})
	if err != nil {
		return nil, err
	}
	return &Report{Spec: spec, Sync: sync, Written: rep.Written, Preserved: rep.Preserved, Conflicts: rep.Conflicts}, nil
}

func readProvidedFile(root, rel string) ([]byte, error) {
	rootReal, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("raiz do pack ilegível: %w", err)
	}
	target := filepath.Join(root, filepath.FromSlash(rel))
	info, err := os.Stat(target)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("não é arquivo regular")
	}
	targetReal, err := filepath.EvalSymlinks(target)
	if err != nil {
		return nil, err
	}
	relReal, err := filepath.Rel(rootReal, targetReal)
	if err != nil || filepath.IsAbs(relReal) || relReal == ".." || strings.HasPrefix(relReal, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("caminho real escapa da raiz do pack")
	}
	return os.ReadFile(targetReal)
}

// ValidateRequiresCore aceita o requisito mínimo >=major.minor[.patch] e
// rejeita operadores/formatações desconhecidos para evitar instalar packs
// ambíguos — ranges (ex.: ">=0.1 <0.3") não são suportados; o erro cita o
// formato aceito.
func ValidateRequiresCore(requirement, current string) error {
	requirement = strings.TrimSpace(requirement)
	if requirement == "" {
		return nil
	}
	if !strings.HasPrefix(requirement, ">=") {
		return fmt.Errorf("requires-core não suportado: %q — formato aceito: \">=X.Y\" (ex.: \">=0.3\"); ranges (ex.: \">=X <Y\") não são suportados", requirement)
	}
	required, err := parseVersion(strings.TrimSpace(strings.TrimPrefix(requirement, ">=")))
	if err != nil {
		return fmt.Errorf("requires-core não suportado: %q — formato aceito: \">=X.Y\" (ex.: \">=0.3\"); ranges (ex.: \">=X <Y\") não são suportados", requirement)
	}
	actual, err := parseVersion(current)
	if err != nil || compareVersion(actual, required) < 0 {
		return fmt.Errorf("requires-core %q exige core >= %s (core atual %s)", requirement, formatVersion(required), current)
	}
	return nil
}

func parseVersion(raw string) ([3]int, error) {
	var out [3]int
	parts := strings.Split(strings.TrimPrefix(strings.TrimSpace(raw), "v"), ".")
	if len(parts) < 2 || len(parts) > 3 {
		return out, fmt.Errorf("versão inválida")
	}
	for i := range parts {
		value, err := strconv.Atoi(parts[i])
		if err != nil || value < 0 {
			return out, fmt.Errorf("versão inválida")
		}
		out[i] = value
	}
	return out, nil
}

func compareVersion(a, b [3]int) int {
	for i := range a {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

func formatVersion(version [3]int) string {
	return strconv.Itoa(version[0]) + "." + strconv.Itoa(version[1]) + "." + strconv.Itoa(version[2])
}

// Summary é a visão read-only de um pack presente no cache local (overlay)
// ou no working copy do config (state — com Path preenchido).
type Summary struct {
	Spec Spec   `json:"spec"`
	Slug string `json:"slug"`
	Path string `json:"path,omitempty"` // working copy (só em state packs)
}

// List lista apenas packs já materializados no cache; não sincroniza fontes.
func List(home string) ([]Summary, error) {
	root := filepath.Join(home, ".golmet", "packs")
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return []Summary{}, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Summary
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		spec, err := loadSpec(filepath.Join(root, entry.Name()))
		if err != nil {
			continue
		}
		out = append(out, Summary{Spec: spec, Slug: entry.Name()})
	}
	return out, nil
}
