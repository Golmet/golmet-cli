// Package tools implementa `golmet tools` — instalação do tool nativo de
// grafo de código (DeusData/codebase-memory-mcp, DR-016) em
// ~/.golmet/tools/codegraph/, com verificação de supply-chain
// OBRIGATÓRIA: SHA-256 contra o checksums.txt oficial (falha fechada) e
// gh attestation verify quando o gh CLI está disponível (ausência =
// AVISO declarado, nunca silencioso). Referência de procedimento:
// skill guiada de setup do codegraph (release latest via API GitHub,
// extração só do binário — install.ps1 nunca executado).
package tools

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	// RepoOficial é o repo canônico — NUNCA espelhos/forks.
	RepoOficial = "DeusData/codebase-memory-mcp"
	// APIURLDefault descobre a release latest (tag + assets).
	APIURLDefault = "https://api.github.com/repos/" + RepoOficial + "/releases/latest"
	// BaseURLDefault é o prefixo de download dos assets da release.
	BaseURLDefault = "https://github.com/" + RepoOficial + "/releases/download"
	// AttestTimeout limita o gh attestation verify.
	AttestTimeout = 90 * time.Second
	// httpTimeout limita cada request (release JSON, asset, checksums).
	httpTimeout = 120 * time.Second
)

// BinaryName é o nome do binário dentro do asset (por GOOS).
func BinaryName() string {
	if runtime.GOOS == "windows" {
		return "codebase-memory-mcp.exe"
	}
	return "codebase-memory-mcp"
}

// Options controla o instalador — campos com override para testes.
type Options struct {
	APIURL  string // default APIURLDefault
	BaseURL string // default BaseURLDefault
	DestDir string // default ~/.golmet/tools/codegraph
	Client  *http.Client

	// LookPath/RunAttestation são injetáveis para testes (default:
	// exec.LookPath + gh attestation verify oficial).
	LookPath       func(string) (string, error)
	RunAttestation func(ctx context.Context, file string) error
}

func (o *Options) fill() error {
	if o.APIURL == "" {
		o.APIURL = APIURLDefault
	}
	if o.BaseURL == "" {
		o.BaseURL = BaseURLDefault
	}
	if o.DestDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("sem HOME para o destino do tool: %w", err)
		}
		o.DestDir = filepath.Join(home, ".golmet", "tools", "codegraph")
	}
	if o.Client == nil {
		o.Client = &http.Client{Timeout: httpTimeout}
	}
	if o.LookPath == nil {
		o.LookPath = exec.LookPath
	}
	if o.RunAttestation == nil {
		o.RunAttestation = runAttestation
	}
	return nil
}

func runAttestation(ctx context.Context, file string) error {
	cctx, cancel := context.WithTimeout(ctx, AttestTimeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, "gh", "attestation", "verify", file, "--repo", RepoOficial)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh attestation rejeitou o asset: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Release é o recorte do JSON da API de releases do GitHub.
type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Report resume a instalação do tool.
type Report struct {
	Tool        string   `json:"tool"`
	Version     string   `json:"version"`
	Binary      string   `json:"binary"`
	SHA256      string   `json:"sha256"`
	Bytes       int64    `json:"bytes"`
	Attestation string   `json:"attestation"` // verified | skipped (gh ausente)
	Warnings    []string `json:"warnings,omitempty"`
}

// DetectCodegraph reporta se o binário do grafo está instalado em
// ~/.golmet/tools/codegraph/ (mesmo check que o doctor consome).
func DetectCodegraph() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	return fileExists(filepath.Join(home, ".golmet", "tools", "codegraph", BinaryName()))
}

// List devolve o estado dos tools geridos pelo kit (tools list).
func List() []string {
	if DetectCodegraph() {
		return []string{"codegraph: instalado (" + RepoOficial + ")"}
	}
	return []string{"codegraph: não instalado — rode 'golmet tools install codegraph'"}
}

// InstallCodegraph baixa, verifica e instala o binário do grafo.
// Falha FECHADA em divergência de hash; attestation degrada para aviso
// apenas quando o gh não existe na máquina.
func InstallCodegraph(ctx context.Context, opt Options) (*Report, error) {
	if err := opt.fill(); err != nil {
		return nil, err
	}

	// 1. release latest → asset da plataforma
	rel, err := fetchRelease(ctx, opt)
	if err != nil {
		return nil, err
	}
	assetName, _, err := pickAsset(rel, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return nil, err
	}

	// 2. download do asset + checksums.txt para temp
	tmp, err := os.MkdirTemp("", "golmet-codegraph-")
	if err != nil {
		return nil, fmt.Errorf("falha ao criar o temp de download: %w", err)
	}
	defer os.RemoveAll(tmp)
	assetPath := filepath.Join(tmp, assetName)
	checksumsPath := filepath.Join(tmp, "checksums.txt")
	base := strings.TrimRight(opt.BaseURL, "/") + "/" + rel.TagName
	if err := download(ctx, opt.Client, base+"/"+assetName, assetPath); err != nil {
		return nil, fmt.Errorf("download do asset falhou — verifique conectividade ou instale manualmente em %s: %w", opt.DestDir, err)
	}
	if err := download(ctx, opt.Client, base+"/checksums.txt", checksumsPath); err != nil {
		return nil, fmt.Errorf("download do checksums.txt falhou — verifique conectividade ou instale manualmente em %s: %w", opt.DestDir, err)
	}

	// 3. SHA-256 vs checksums.txt — divergência = FALHA FECHADA
	sum, size, err := fileSHA256(assetPath)
	if err != nil {
		return nil, err
	}
	if err := verifyChecksums(checksumsPath, assetName, sum); err != nil {
		return nil, err
	}

	// 4. attestation SLSA — gh ausente degrada para AVISO declarado
	rep := &Report{
		Tool:    "codegraph",
		Version: rel.TagName,
		SHA256:  sum,
		Bytes:   size,
	}
	if _, err := opt.LookPath("gh"); err != nil {
		rep.Attestation = "skipped (gh ausente)"
		rep.Warnings = append(rep.Warnings,
			"gh CLI não encontrado — attestation SLSA NÃO verificado; verifique a provenance manualmente ou instale o gh e re rode o install")
	} else {
		if err := opt.RunAttestation(ctx, assetPath); err != nil {
			return nil, fmt.Errorf("attestation falhou — NÃO instalar: %w", err)
		}
		rep.Attestation = "verified"
	}

	// 5. extrair apenas o binário para o destino (install.ps1 nunca)
	binary, err := extractBinary(assetPath, assetName, opt.DestDir)
	if err != nil {
		return nil, err
	}
	rep.Binary = binary
	return rep, nil
}

func fetchRelease(ctx context.Context, opt Options) (*Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, opt.APIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("url da release inválida: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := opt.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("consulta da release latest falhou — verifique conectividade ou instale manualmente em %s: %w", opt.DestDir, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("consulta da release latest respondeu %d — verifique conectividade ou instale manualmente em %s", resp.StatusCode, opt.DestDir)
	}
	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("JSON da release ilegível: %w", err)
	}
	if rel.TagName == "" || len(rel.Assets) == 0 {
		return nil, fmt.Errorf("release latest sem tag/assets — verifique a publicação em %s", RepoOficial)
	}
	return &rel, nil
}

// pickAsset escolhe o asset da plataforma: codebase-memory-mcp-<goos>-<goarch>
// com extensão .zip (windows) ou .tar.gz (demais) — nomes oficiais da
// release (hífens; ver assets de v0.12.1).
func pickAsset(rel *Release, goos, goarch string) (string, string, error) {
	exts := []string{".zip", ".tar.gz"}
	if goos != "windows" {
		exts = []string{".tar.gz", ".zip"}
	}
	prefix := "codebase-memory-mcp-" + goos + "-" + goarch
	for _, ext := range exts {
		for _, a := range rel.Assets {
			if a.Name == prefix+ext {
				return a.Name, a.BrowserDownloadURL, nil
			}
		}
	}
	return "", "", fmt.Errorf("asset indisponível para %s/%s na release %s — confira as plataformas publicadas em %s",
		goos, goarch, rel.TagName, RepoOficial)
}

// verifyChecksums compara o hash do asset baixado contra o checksums.txt
// oficial (linhas "<sha256>  <arquivo>"). Divergência = erro estruturado.
func verifyChecksums(path, assetName, got string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("checksums.txt ilegível: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) != 2 {
			continue
		}
		if fields[1] != assetName && fields[1] != "*"+assetName {
			continue
		}
		want := strings.ToLower(fields[0])
		if want == got {
			return nil
		}
		return fmt.Errorf("SHA-256 divergente do checksums.txt oficial para %s — NÃO instalar (esperado %s, obtido %s)",
			assetName, want, got)
	}
	return fmt.Errorf("asset %s não listado no checksums.txt oficial — NÃO instalar", assetName)
}

// extractBinary extrai apenas o binário do asset para destDir (nenhum
// outro arquivo do zip é instalado — install.ps1 nunca é executado).
func extractBinary(assetPath, assetName, destDir string) (string, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("falha ao criar o destino %s: %w", destDir, err)
	}
	target := filepath.Join(destDir, BinaryName())
	var data []byte
	switch {
	case strings.HasSuffix(assetName, ".zip"):
		var err error
		data, err = readZipBinary(assetPath)
		if err != nil {
			return "", err
		}
	case strings.HasSuffix(assetName, ".tar.gz"):
		var err error
		data, err = readTarGzBinary(assetPath)
		if err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("formato de asset não suportado: %s", assetName)
	}
	if err := os.WriteFile(target, data, 0o755); err != nil {
		return "", fmt.Errorf("falha ao instalar o binário em %s: %w", target, err)
	}
	return target, nil
}

func readZipBinary(path string) ([]byte, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("zip do asset ilegível: %w", err)
	}
	defer r.Close()
	for _, f := range r.File {
		if filepath.Base(f.Name) != BinaryName() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("entrada %s ilegível: %w", f.Name, err)
		}
		defer rc.Close()
		return io.ReadAll(rc)
	}
	return nil, fmt.Errorf("binário %s não encontrado dentro do asset zip", BinaryName())
}

func readTarGzBinary(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("asset ilegível: %w", err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("gzip do asset ilegível: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tar do asset ilegível: %w", err)
		}
		if filepath.Base(hdr.Name) != BinaryName() {
			continue
		}
		return io.ReadAll(tr)
	}
	return nil, fmt.Errorf("binário %s não encontrado dentro do asset tar.gz", BinaryName())
}

func download(ctx context.Context, client *http.Client, url, dst string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("url inválida %q: %w", url, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download de %s respondeu %d", url, resp.StatusCode)
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func fileSHA256(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, fmt.Errorf("asset baixado ilegível: %w", err)
	}
	defer f.Close()
	h := sha256.New()
	size, err := io.Copy(h, f)
	if err != nil {
		return "", 0, fmt.Errorf("falha ao hashear o asset: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), size, nil
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}
