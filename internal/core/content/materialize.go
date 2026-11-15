package content

import (
	"bytes"
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
)

// FileEntry registra proveniência e o hash que serviu de base para a última
// materialização. O arquivo local pode divergir sem ser destruído: o próximo
// update detecta o conflito comparando disco contra este hash (DR-021).
type FileEntry struct {
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
	Layer  string `json:"layer"` // core | local
}

type LayerInfo struct {
	Source string `json:"source"` // embedded | repo@sha futuro
	SHA    string `json:"sha,omitempty"`
}

type ContentManifest struct {
	Version   string               `json:"version"`
	UpdatedAt string               `json:"updated_at"`
	Layers    map[string]LayerInfo `json:"layers"`
	Files     []FileEntry          `json:"files"`
}

// MaterializeOptions controla a projeção do FULL no config do engine.
type MaterializeOptions struct {
	ConfigRoot string
	Engine     string // opencode | claude-code
	KitVersion string
	Source     string // embedded ou repo@sha
	Force      bool
	// DryRun calcula o plano sem criar, alterar ou remover arquivos, inclusive
	// o manifest de conteúdo.
	DryRun bool
	// ManagedHashes identifica arquivos BASE já geridos pelo kit; permite
	// promover BASE → FULL sem tratar o arquivo BASE como customização local.
	ManagedHashes map[string]string // caminho relativo do engine → SHA-256
	// ManifestPath é o caminho COMPLETO do manifest de conteúdo. Vazio =
	// default <root>/.golmet/manifest.json (config roots globais de engines
	// com skills nativas). O fallback do FULL na zona .golmet/ do workspace
	// usa nome distinto (manifest-content.json) porque o manifest.json dessa
	// pasta pertence ao BASE do update.Install — schema diferente no mesmo
	// diretório; um não pode sobrescrever nem ser parseado pelo outro.
	ManifestPath string
}

type MaterializeReport struct {
	Written   []string `json:"written,omitempty"`
	Preserved []string `json:"preserved,omitempty"`
	Conflicts []string `json:"conflicts,omitempty"`
	Total     int      `json:"total"`
}

func sha(data []byte) string { h := sha256.Sum256(data); return hex.EncodeToString(h[:]) }

// normalizeEOL converte CRLF e CR solitário para LF — o conteúdo canônico
// do kit é LF; diferença de EOL não é customização (resquício de editor
// PowerShell/migração), é estado a curar silenciosamente.
func normalizeEOL(b []byte) []byte {
	out := bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
	return bytes.ReplaceAll(out, []byte("\r"), []byte("\n"))
}

// stripBOM remove o BOM UTF-8 inicial — mesmo classe de resquício de
// editor (PowerShell lê/grava sem BOM) que o EOL; 10 das skills embutidas
// têm BOM e edições PowerShell as stripam, criando falso "modificado".
func stripBOM(b []byte) []byte {
	if len(b) >= 3 && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF {
		return b[3:]
	}
	return b
}

// SameContentNormalized compara dois conteúdos ignorando resquícios de
// editor (EOL e BOM inicial): iguais após normalizar = mesmo conteúdo —
// a cura é rematerializar o arquivo com o canônico do kit, sem flag de
// conflito e sem perder nada.
func SameContentNormalized(a, b []byte) bool {
	if bytes.Equal(a, b) {
		return true
	}
	return bytes.Equal(normalizeEOL(stripBOM(a)), normalizeEOL(stripBOM(b)))
}

// ContentManifestPath devolve o caminho do manifest de conteúdo da raiz.
// Override vazio = default <root>/.golmet/manifest.json (config roots
// globais). O fallback do FULL na zona do workspace passa override com o
// nome distinto manifest-content.json — manifest.json lá é do BASE.
func ContentManifestPath(root, override string) string {
	if override != "" {
		return override
	}
	return filepath.Join(root, ".golmet", "manifest.json")
}

// ReadContentManifest lê o manifest de conteúdo da raiz (default
// <root>/.golmet/manifest.json via ContentManifestPath). Manifest ausente →
// (nil, nil): raiz sem materialização prévia do kit NÃO é erro (consumidor
// trata tudo como sem proveniência). Ilegível → erro (consumidor decide).
// Leitor público para o doctor de engine-config (mesma fonte que o
// Materialize escreve — nada de parser duplicado).
func ReadContentManifest(root string) (*ContentManifest, error) {
	return readContentManifest(ContentManifestPath(root, ""))
}

// Normalize expõe a normalização de resquícios de editor (EOL → LF, BOM
// inicial removido) com a MESMA tolerância do install, para consumidores
// que comparam conteúdo em disco contra o hash do manifest (ex.: doctor de
// engine-config) sem reimplementar a regra — hash normalizado igual =
// gerido conforme, sem falso "customização" por linha.
func Normalize(b []byte) []byte {
	return normalizeEOL(stripBOM(b))
}

func readContentManifest(manifestPath string) (*ContentManifest, error) {
	data, err := os.ReadFile(manifestPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var m ContentManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("manifest de conteúdo ilegível: %w", err)
	}
	return &m, nil
}

func writeContentManifest(manifestPath string, m *ContentManifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	dir := filepath.Dir(manifestPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".manifest-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, manifestPath)
}

func previousEntries(m *ContentManifest) map[string]FileEntry {
	out := map[string]FileEntry{}
	if m == nil {
		return out
	}
	for _, e := range m.Files {
		out[e.File] = e
	}
	return out
}

// Materialize renderiza o core FULL e grava agents/skills no config do engine.
// Sem --force, qualquer arquivo divergente do último hash conhecido é
// preservado e listado como conflito; nunca sobrescreve customização local.
func Materialize(fsys fs.FS, reg *Registry, opt MaterializeOptions) (*MaterializeReport, error) {
	if opt.ConfigRoot == "" {
		return nil, fmt.Errorf("config root vazio")
	}
	rendered, missing := Render(fsys, reg, opt.Engine)
	if len(missing) > 0 {
		return nil, fmt.Errorf("%d corpos ausentes no core: %v", len(missing), missing)
	}
	for i := range rendered {
		rel, err := validateContentPath(rendered[i].Rel)
		if err != nil {
			return nil, err
		}
		rendered[i].Rel = rel
	}
	manifestPath := ContentManifestPath(opt.ConfigRoot, opt.ManifestPath)
	prev, err := readContentManifest(manifestPath)
	if err != nil {
		return nil, err
	}
	old := previousEntries(prev)
	rep := &MaterializeReport{Total: len(rendered)}
	source := opt.Source
	if source == "" {
		source = "embedded"
	}
	next := &ContentManifest{Version: opt.KitVersion, UpdatedAt: time.Now().UTC().Format(time.RFC3339), Layers: map[string]LayerInfo{
		"core": {Source: source},
	}}
	// Core update não pode apagar proveniência de packs (L2) ou arquivos
	// locais (L3) já registrados no mesmo manifest.
	if prev != nil {
		for layer, info := range prev.Layers {
			next.Layers[layer] = info
		}
		next.Layers["core"] = LayerInfo{Source: source}
	}
	fileMap := previousEntries(prev)
	for _, r := range rendered {
		target := filepath.Join(opt.ConfigRoot, filepath.FromSlash(r.Rel))
		data := []byte(r.Content)
		newHash := sha(data)
		rel := r.Rel
		existing, readErr := os.ReadFile(target)
		existingHash := ""
		sameNorm := false // mesmo conteúdo após normalizar EOL (CRLF/CR → LF)
		if readErr == nil {
			existingHash = sha(existing)
			sameNorm = SameContentNormalized(existing, data)
		}
		oldEntry, hadOld := old[rel]
		if hadOld && strings.HasPrefix(oldEntry.Layer, "pack:") {
			if !sameNorm {
				rep.Preserved = append(rep.Preserved, rel)
				if readErr != nil || existingHash != newHash {
					rep.Conflicts = append(rep.Conflicts, rel)
				}
				fileMap[rel] = oldEntry
				continue
			}
			// Conteúdo do pack idêntico ao canônico (só EOL diverge, quando
			// diverge): cura escrevendo o LF do kit SEM conflito e SEM
			// apagar a proveniência do pack.
			if !opt.DryRun && (readErr != nil || existingHash != newHash) {
				if err := ensureSafeTarget(opt.ConfigRoot, rel); err != nil {
					return nil, err
				}
				if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
					return nil, err
				}
				if err := os.WriteFile(target, data, 0o644); err != nil {
					return nil, fmt.Errorf("falha ao escrever %s: %w", rel, err)
				}
				rep.Written = append(rep.Written, rel)
			} else {
				rep.Preserved = append(rep.Preserved, rel)
			}
			fileMap[rel] = oldEntry
			continue
		}
		// Precedência local > full: arquivo registrado como local não é
		// sobrescrito sem --force (a exceção explícita do FORMATO §2.1) —
		// diferença só de EOL não é customização: cura escrevendo o LF.
		if hadOld && oldEntry.Layer == "local" && !opt.Force && !sameNorm {
			rep.Preserved = append(rep.Preserved, rel)
			rep.Conflicts = append(rep.Conflicts, rel)
			fileMap[rel] = oldEntry
			continue
		}
		managedBase := opt.ManagedHashes[rel]
		conflict := readErr == nil && !opt.Force && !sameNorm && ((hadOld && existingHash != oldEntry.SHA256) || (!hadOld && managedBase != "" && existingHash != managedBase))
		if conflict {
			rep.Preserved = append(rep.Preserved, rel)
			rep.Conflicts = append(rep.Conflicts, rel)
			if hadOld {
				fileMap[rel] = oldEntry
			} else {
				fileMap[rel] = FileEntry{File: rel, SHA256: existingHash, Layer: "local"}
			}
			continue
		}
		if readErr == nil && existingHash == newHash {
			// byte-a-byte idêntico — nada a fazer
			rep.Preserved = append(rep.Preserved, rel)
		} else {
			// novo ou EOL-only (mesmo conteúdo normalizado): rematerializa
			// com o canônico LF — Written cura o EOL e atualiza o tracker
			if !opt.DryRun {
				if err := ensureSafeTarget(opt.ConfigRoot, rel); err != nil {
					return nil, err
				}
				if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
					return nil, err
				}
				if err := os.WriteFile(target, data, 0o644); err != nil {
					return nil, fmt.Errorf("falha ao escrever %s: %w", rel, err)
				}
			}
			rep.Written = append(rep.Written, rel)
		}
		fileMap[rel] = FileEntry{File: rel, SHA256: newHash, Layer: "core"}
	}
	sort.Strings(rep.Written)
	sort.Strings(rep.Preserved)
	sort.Strings(rep.Conflicts)
	for _, entry := range fileMap {
		next.Files = append(next.Files, entry)
	}
	sort.Slice(next.Files, func(i, j int) bool { return next.Files[i].File < next.Files[j].File })
	if !opt.DryRun {
		if err := writeContentManifest(manifestPath, next); err != nil {
			return nil, fmt.Errorf("falha ao gravar manifest de conteúdo: %w", err)
		}
	}
	return rep, nil
}

// ensureSafeTarget rejeita destino não regular e caminhos cujos componentes
// (do destino em diante) resolvam, via symlink/junction, para fora da raiz.
// Windows junctions são resolvidos por EvalSymlinks — a checagem não depende
// do bit ModeSymlink.
func ensureSafeTarget(root, rel string) error {
	rootReal, err := filepath.EvalSymlinks(root)
	if err != nil {
		// Raiz ainda não existe (primeiro materialize): nada pré-existe para
		// escapar; os componentes criados abaixo dela são novos.
		if !os.IsNotExist(err) {
			return fmt.Errorf("raiz de conteúdo ilegível: %w", err)
		}
		rootReal = filepath.Clean(root)
	}
	target := filepath.Join(root, filepath.FromSlash(rel))
	if info, err := os.Lstat(target); err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("destino de conteúdo não é arquivo regular: %s", rel)
	}
	parts := strings.Split(rel, "/")
	cur := root
	for _, part := range parts {
		if part == "" {
			continue
		}
		cur = filepath.Join(cur, filepath.FromSlash(part))
		real, err := filepath.EvalSymlinks(cur)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("caminho de conteúdo ilegível: %s: %w", rel, err)
		}
		inside, err := filepath.Rel(rootReal, real)
		if err != nil || inside == ".." || filepath.IsAbs(inside) || strings.HasPrefix(inside, ".."+string(filepath.Separator)) {
			return fmt.Errorf("caminho de conteúdo escapa da raiz: %s", rel)
		}
	}
	return nil
}

// SafeDirFS expõe um diretório como fs.FS que rejeita leituras cujo caminho
// real (após symlink/junction) escape da raiz — fontes externas não são
// confiáveis.
func SafeDirFS(root string) (fs.FS, error) {
	rootReal, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("raiz da fonte ilegível: %w", err)
	}
	return safeDirFS{root: root, rootReal: rootReal}, nil
}

type safeDirFS struct{ root, rootReal string }

func (s safeDirFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	target := filepath.Join(s.root, filepath.FromSlash(name))
	real, err := filepath.EvalSymlinks(target)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}
	inside, err := filepath.Rel(s.rootReal, real)
	if err != nil || inside == ".." || filepath.IsAbs(inside) || strings.HasPrefix(inside, ".."+string(filepath.Separator)) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fmt.Errorf("caminho escapa da raiz da fonte")}
	}
	return os.Open(real)
}

// validateContentPath constrains renderer output to the three content roots
// (agents/, skills/, rules/) plus the root-level template asset
// (AGENTS.template.md). The check is deliberately independent of the host
// path separator so a malicious registry cannot exploit a different
// platform's syntax.
func validateContentPath(rel string) (string, error) {
	normalized := strings.ReplaceAll(filepath.ToSlash(rel), "\\", "/")
	if normalized == "" || strings.HasPrefix(normalized, "/") || filepath.IsAbs(rel) ||
		(len(normalized) > 1 && normalized[1] == ':') {
		return "", fmt.Errorf("caminho de conteúdo fora do escopo: %q", rel)
	}
	parts := strings.Split(normalized, "/")
	for _, part := range parts {
		if part == ".." || part == "" {
			return "", fmt.Errorf("caminho de conteúdo fora do escopo: %q", rel)
		}
	}
	if normalized == "AGENTS.template.md" {
		return normalized, nil
	}
	if !strings.HasPrefix(normalized, "agents/") && !strings.HasPrefix(normalized, "skills/") &&
		!strings.HasPrefix(normalized, "rules/") {
		return "", fmt.Errorf("caminho de conteúdo fora do escopo: %q", rel)
	}
	return normalized, nil
}
