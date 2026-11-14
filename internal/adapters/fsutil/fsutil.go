// Package fsutil caminha o workspace ignorando zonas não-documentais
// (.git/, node_modules/, packs/ e a zona do kit .golmet/) e prepara
// linhas com fences blankados para scan sem falsos positivos.
package fsutil

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var excludedDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"packs":        true,
	".golmet":      true, // zona do kit (vendored) — lida diretamente pelas regras S3/S5
}

// ExcludedDir reporta se a pasta é ignorada pelo walk do kit (mesma
// lista do Walk — usada pelo doctor para o drift-check de idade).
func ExcludedDir(name string) bool { return excludedDirs[name] }

// NewestMDMod devolve o mtime mais recente entre os .md documentais do
// workspace (mesmas exclusões do Walk) — usado pelo drift-check de idade
// do índice (doctor, next).
func NewestMDMod(root string) (time.Time, error) {
	var newest time.Time
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != root && excludedDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(p), ".md") {
			return nil
		}
		info, ierr := d.Info()
		if ierr != nil {
			return nil
		}
		if info.ModTime().After(newest) {
			newest = info.ModTime()
		}
		return nil
	})
	return newest, err
}

// FileDoc é um .md do workspace com linhas originais e a variante plain
// (fences blankados preservando a numeração de linhas).
type FileDoc struct {
	Rel   string // caminho relativo com separador '/'
	Abs   string
	Lines []string // original (normalizado \n, sem BOM)
	Plain []string // fences blankados
}

// Walk retorna todos os .md do workspace em ordem determinística.
func Walk(root string) ([]FileDoc, error) {
	var out []FileDoc
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != root && excludedDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(p), ".md") {
			return nil
		}
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return nil // ilegível: best-effort
		}
		content := strings.TrimPrefix(strings.ReplaceAll(string(data), "\r\n", "\n"), "\uFEFF")
		lines := strings.Split(content, "\n")
		rel, rerr := filepath.Rel(root, p)
		if rerr != nil {
			return nil
		}
		out = append(out, FileDoc{
			Rel:   filepath.ToSlash(rel),
			Abs:   p,
			Lines: lines,
			Plain: BlankFenced(lines),
		})
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Rel < out[j].Rel })
	return out, err
}

// BlankFenced substitui linhas dentro de blocos ``` / ~~~ por espaços,
// preservando a contagem de linhas.
func BlankFenced(lines []string) []string {
	out := make([]string, len(lines))
	inFence := false
	var marker string
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if !inFence && (strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~")) {
			inFence = true
			marker = t[:3]
			out[i] = strings.Repeat(" ", len(line))
			continue
		}
		if inFence {
			if strings.HasPrefix(t, marker) {
				inFence = false
			}
			out[i] = strings.Repeat(" ", len(line))
			continue
		}
		out[i] = line
	}
	return out
}
