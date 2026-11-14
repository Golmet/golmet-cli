// sqlite.go — instalação do MCP SQLite rusq/mcp-sqlite (`golmet tools
// install sqlite` e auto-install do `golmet install`). Procedimento da
// skill guiada gm-mcp-sqlite-setup: release v0.0.3 FIXA do repo oficial,
// asset zip por plataforma, SHA-256 OBRIGATÓRIO contra o checksums.txt do
// release (divergência = falha fechada), extração APENAS do binário para
// %LOCALAPPDATA%\Programs\mcp-sqlite\. O release NÃO publica attestation
// SLSA (diferente do codegraph) — a verificação é o checksum, e a ausência
// de attestation é AVISO declarado no report, nunca silencioso.
package tools

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	// SqliteRepoOficial é o repo canônico — NUNCA espelhos/forks.
	SqliteRepoOficial = "rusq/mcp-sqlite"
	// SqliteVersion é a versão VALIDADA pela skill de setup (release 2026-08).
	SqliteVersion = "v0.0.3"
	// SqliteBaseURLDefault é o prefixo de download dos assets da release.
	SqliteBaseURLDefault = "https://github.com/" + SqliteRepoOficial + "/releases/download/" + SqliteVersion
)

// SqliteBinaryName é o nome do binário dentro do asset (por GOOS).
func SqliteBinaryName() string {
	if runtime.GOOS == "windows" {
		return "mcp-sqlite.exe"
	}
	return "mcp-sqlite"
}

// SqliteAssetName é o asset zip da plataforma na release v0.0.3
// (mcp-sqlite_0.0.3_windows_amd64.zip / windows_arm64 / linux_amd64 / ...).
func SqliteAssetName() string {
	return "mcp-sqlite_0.0.3_" + runtime.GOOS + "_" + runtime.GOARCH + ".zip"
}

// SqliteOptions controla o instalador — campos com override para testes.
type SqliteOptions struct {
	BaseURL      string // default SqliteBaseURLDefault (httptest nos testes)
	DestDir      string // default %LOCALAPPDATA%\Programs\mcp-sqlite
	LocalAppData string // override p/ testes (default env LOCALAPPDATA)
	Client       *http.Client
}

func (o *SqliteOptions) fill() error {
	if o.BaseURL == "" {
		o.BaseURL = SqliteBaseURLDefault
	}
	if o.DestDir == "" {
		lad := o.LocalAppData
		if lad == "" {
			lad = os.Getenv("LOCALAPPDATA")
		}
		if lad == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("sem HOME/LOCALAPPDATA para o destino do tool: %w", err)
			}
			lad = filepath.Join(home, "AppData", "Local")
		}
		o.DestDir = filepath.Join(lad, "Programs", "mcp-sqlite")
	}
	if o.Client == nil {
		o.Client = &http.Client{Timeout: httpTimeout}
	}
	return nil
}

// SqliteReport resume a instalação (Attestation sempre declarado — o
// release do rusq não publica attestation SLSA; a verificação é o SHA-256).
type SqliteReport struct {
	Tool        string   `json:"tool"`
	Version     string   `json:"version"`
	Binary      string   `json:"binary"`
	SHA256      string   `json:"sha256"`
	Bytes       int64    `json:"bytes"`
	Attestation string   `json:"attestation"`
	Warnings    []string `json:"warnings,omitempty"`
}

// InstallSqlite baixa, verifica (SHA-256 obrigatório) e instala o binário
// do MCP SQLite. Falha FECHADA em divergência de hash ou asset fora do
// checksums.txt.
func InstallSqlite(ctx context.Context, opt SqliteOptions) (*SqliteReport, error) {
	if err := opt.fill(); err != nil {
		return nil, err
	}
	client := opt.Client
	if client == nil {
		client = &http.Client{Timeout: httpTimeout}
	}
	assetName := SqliteAssetName()

	// 1. download do asset + checksums.txt para temp
	tmp, err := os.MkdirTemp("", "golmet-sqlite-")
	if err != nil {
		return nil, fmt.Errorf("falha ao criar o temp de download: %w", err)
	}
	defer os.RemoveAll(tmp)
	assetPath := filepath.Join(tmp, assetName)
	checksumsPath := filepath.Join(tmp, "checksums.txt")
	base := strings.TrimRight(opt.BaseURL, "/")
	if err := download(ctx, client, base+"/"+assetName, assetPath); err != nil {
		return nil, fmt.Errorf("download do asset falhou — verifique conectividade ou instale manualmente em %s (release %s): %w", opt.DestDir, SqliteVersion, err)
	}
	if err := download(ctx, client, base+"/checksums.txt", checksumsPath); err != nil {
		return nil, fmt.Errorf("download do checksums.txt falhou — release %s: %w", SqliteVersion, err)
	}

	// 2. SHA-256 vs checksums.txt — divergência = FALHA FECHADA
	sum, size, err := fileSHA256(assetPath)
	if err != nil {
		return nil, err
	}
	if err := verifyChecksums(checksumsPath, assetName, sum); err != nil {
		return nil, err
	}

	// 3. attestation — o release do rusq NÃO publica SLSA: aviso declarado
	rep := &SqliteReport{
		Tool:        "sqlite",
		Version:     SqliteVersion,
		SHA256:      sum,
		Bytes:       size,
		Attestation: "skipped (release sem attestation SLSA — verificação por SHA-256 do checksums.txt)",
	}
	rep.Warnings = append(rep.Warnings,
		"release do rusq/mcp-sqlite sem attestation SLSA — integridade garantida pelo SHA-256 do checksums.txt oficial")

	// 4. extrair apenas o binário para o destino
	if err := os.MkdirAll(opt.DestDir, 0o755); err != nil {
		return nil, fmt.Errorf("falha ao criar o destino %s: %w", opt.DestDir, err)
	}
	target := filepath.Join(opt.DestDir, SqliteBinaryName())
	data, err := readZipEntry(assetPath, SqliteBinaryName())
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(target, data, 0o755); err != nil {
		return nil, fmt.Errorf("falha ao instalar o binário em %s: %w", target, err)
	}
	rep.Binary = target
	return rep, nil
}

// readZipEntry lê UMA entrada do zip pelo nome base (nenhum outro arquivo
// do zip é instalado; nada é executado).
func readZipEntry(path, name string) ([]byte, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("zip do asset ilegível: %w", err)
	}
	defer r.Close()
	for _, f := range r.File {
		if filepath.Base(f.Name) != name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("entrada %s ilegível: %w", f.Name, err)
		}
		defer rc.Close()
		return io.ReadAll(rc)
	}
	return nil, fmt.Errorf("binário %s não encontrado dentro do asset zip", name)
}
