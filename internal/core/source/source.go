// Package source gerencia fontes Git opcionais do GOLMET: core externo (L1)
// e packs (L2). O kit mantém cache local por slug e fixa o SHA resolvido no
// relatório/manifest; diretórios locais sem Git usam digest determinístico.
package source

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type SyncReport struct {
	Source       string `json:"source"`
	Path         string `json:"path"`
	SHA          string `json:"sha"`
	Cloned       bool   `json:"cloned"`
	Updated      bool   `json:"updated"`
	CacheWarning string `json:"cache_warning,omitempty"`
}

// SyncOptions controla efeitos colaterais da sincronização de uma fonte.
// DryRun nunca clona nem faz pull; quando a fonte não está disponível no
// cache, o chamador deve reportar que não foi possível montar o plano.
type SyncOptions struct {
	DryRun bool
}

var unsafeSlug = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
var urlToken = regexp.MustCompile(`(?i)[a-z][a-z0-9+.-]*://[^\s]+`)

// Redact remove userinfo e parâmetros de consulta potencialmente secretos de
// uma origem antes de persistir ou devolvê-la ao consumidor.
func Redact(input string) string {
	u, err := url.Parse(input)
	if err != nil {
		if strings.Contains(input, "://") {
			return "[redacted-source]"
		}
		return input
	}
	if u.Scheme == "" || u.Host == "" {
		return input
	}
	u.User = nil
	values := u.Query()
	for key := range values {
		lower := strings.ToLower(key)
		if strings.Contains(lower, "token") || strings.Contains(lower, "secret") ||
			strings.Contains(lower, "password") || strings.Contains(lower, "passwd") ||
			lower == "pass" || lower == "key" ||
			strings.Contains(lower, "api_key") || strings.Contains(lower, "apikey") ||
			strings.Contains(lower, "auth") || strings.Contains(lower, "credential") ||
			strings.HasSuffix(lower, "_key") {
			delete(values, key)
		}
	}
	u.RawQuery = values.Encode()
	u.Fragment = ""
	return u.String()
}

func redactText(input string) string {
	return urlToken.ReplaceAllStringFunc(input, Redact)
}

func Slug(input string) string {
	s := strings.TrimSuffix(filepath.Base(strings.TrimSuffix(input, "/")), ".git")
	s = unsafeSlug.ReplaceAllString(s, "-")
	if s == "" || s == "." || s == "-" {
		s = "source"
	}
	sum := sha1.Sum([]byte(input))
	return s + "-" + hex.EncodeToString(sum[:])[:10]
}

func run(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		safeArgs := append([]string(nil), args...)
		for i, arg := range safeArgs {
			safeArgs[i] = Redact(arg)
		}
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(safeArgs, " "), err, strings.TrimSpace(redactText(string(out))))
	}
	return out, nil
}

func head(dir string) (string, error) {
	out, err := run(dir, "rev-parse", "HEAD")
	return strings.TrimSpace(string(out)), err
}

func localDigest(dir string) (string, error) {
	h := sha256.New()
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if filepath.Base(path) == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if _, err := io.WriteString(h, filepath.ToSlash(rel)+"\x00"); err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(h, f)
		return err
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Sync materializa source em cacheRoot. Diretório local é aceito para testes
// e desenvolvimento; URLs Git são cloneadas/puxadas com --ff-only.
func Sync(sourceURL, cacheRoot string) (*SyncReport, error) {
	return SyncWithOptions(sourceURL, cacheRoot, SyncOptions{})
}

// SyncWithOptions sincroniza uma fonte, preservando Sync como wrapper
// compatível para o CLI existente.
func SyncWithOptions(sourceURL, cacheRoot string, opt SyncOptions) (*SyncReport, error) {
	if strings.TrimSpace(sourceURL) == "" {
		return nil, fmt.Errorf("source vazio")
	}
	local := sourceURL
	if st, err := os.Stat(local); err == nil && st.IsDir() {
		if _, err := os.Stat(filepath.Join(local, ".git")); err == nil {
			status, statusErr := run(local, "status", "--porcelain", "--untracked-files=all")
			if statusErr != nil {
				return nil, statusErr
			}
			if strings.TrimSpace(string(status)) != "" {
				return nil, fmt.Errorf("fonte Git local suja; commit ou descarte as alterações antes de sincronizar: %s", strings.TrimSpace(redactText(string(status))))
			}
		}
		sha, err := head(local)
		if err != nil {
			sha, err = localDigest(local)
		}
		if err != nil {
			return nil, err
		}
		return &SyncReport{Source: Redact(sourceURL), Path: local, SHA: sha}, nil
	}
	path := filepath.Join(cacheRoot, Slug(sourceURL))
	report := &SyncReport{Source: Redact(sourceURL), Path: path}
	if opt.DryRun {
		if _, err := os.Stat(filepath.Join(path, ".git")); err != nil {
			return nil, fmt.Errorf("dry-run não sincroniza cache Git e a fonte não está em cache: %s", Redact(sourceURL))
		}
		sha, err := head(path)
		if err != nil {
			return nil, err
		}
		report.SHA = sha
		report.CacheWarning = "dry-run: cache Git não foi sincronizado; SHA é o HEAD local"
		return report, nil
	}
	if err := os.MkdirAll(cacheRoot, 0o755); err != nil {
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
		status, err := run(path, "status", "--porcelain", "--untracked-files=all")
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(string(status)) != "" {
			return nil, fmt.Errorf("cache Git sujo para %s; limpe ou remova o cache antes de sincronizar: %s", Redact(sourceURL), path)
		}
		if _, err := run(path, "pull", "--ff-only"); err != nil {
			return nil, err
		}
		report.Updated = true
	} else {
		if _, err := os.Stat(path); err == nil {
			return nil, fmt.Errorf("cache existe mas não é um repo Git: %s", path)
		}
		if _, err := run(cacheRoot, "clone", "--depth", "1", "--", sourceURL, path); err != nil {
			return nil, err
		}
		report.Cloned = true
	}
	sha, err := head(path)
	if err != nil {
		return nil, err
	}
	report.SHA = sha
	return report, nil
}
