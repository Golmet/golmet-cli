// Package upgrade implementa o self-update do binário golmet: consulta a
// última release do repo no GitHub, valida o asset da plataforma corrente
// contra o checksums.txt da release (SHA-256 — linha vermelha: checksum
// ausente ou divergente ABORTA sem tocar no binário corrente) e substitui o
// binário de forma Windows-safe (o corrente é renomeado para <bin>.old no
// mesmo volume e o novo move para o lugar — rollback manual pelo .old).
// A comparação de versões é semver tolerante ao prefixo v; build local
// ("dev") não é semver — o check reporta update disponível sem crash.
package upgrade

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// BinaryVersion é a versão corrente do binário — o package cli sincroniza
// com cli.Version (injetada via ldflags no release) num init(); default
// "dev" para build local (nunca comparável a semver — HasUpdate fica true).
var BinaryVersion = "dev"

// DefaultAPIBase é a base da API do GitHub — variável de pacote para os
// testes servirem uma release fake via httptest (padrão lookPathFn do kit).
var DefaultAPIBase = "https://api.github.com"

// ErrNotSemver sinaliza versão não comparável (ex.: "dev" de build local).
var ErrNotSemver = errors.New("versão não é semver comparável")

const (
	repoOwner = "Golmet"
	repoName  = "golmet-cli"
	dlTimeout = 15 * time.Minute // download generoso — binários ~10-20MB e redes lentas
)

// Options controla a execução. Campos vazios usam os defaults (produção);
// os overrides existem para os testes (httptest + sandbox de binário).
type Options struct {
	// CheckOnly: read-only — reporta {current, latest, has_update} sem baixar.
	CheckOnly bool
	// APIBase sobrescreve a base da API (default DefaultAPIBase).
	APIBase string
	// ExecutablePath sobrescreve o binário corrente (default os.Executable).
	ExecutablePath string
	// Out recebe o progresso curto (nil = descarta).
	Out io.Writer
}

// CheckResult é o resultado do check read-only.
type CheckResult struct {
	Current   string `json:"current"`
	Latest    string `json:"latest"`
	HasUpdate bool   `json:"has_update"`
}

// Result agrega o check + o resultado da aplicação.
type Result struct {
	Check      CheckResult `json:"check"`
	Applied    bool        `json:"applied"`
	BinPath    string      `json:"bin_path,omitempty"`
	BackupPath string      `json:"backup_path,omitempty"`
}

// Asset é um anexo da release.
type Asset struct {
	Name string
	URL  string
}

// Release é a última release publicada (tag + assets).
type Release struct {
	Tag    string
	Assets []Asset
}

// parseSemver converte "v1.2.3[-sufixo]" → tripla numérica (tolerante).
func parseSemver(s string) ([3]int, error) {
	var out [3]int
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	if i := strings.IndexByte(s, '-'); i >= 0 {
		s = s[:i] // sufixo pré-release ignorado na comparação
	}
	parts := strings.Split(s, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return out, fmt.Errorf("%w: %q", ErrNotSemver, s)
	}
	for i, p := range parts {
		if p == "" {
			return out, fmt.Errorf("%w: %q", ErrNotSemver, s)
		}
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, fmt.Errorf("%w: %q", ErrNotSemver, s)
		}
		out[i] = n
	}
	return out, nil
}

// CompareVersions compara current com latest (semver tolerante ao prefixo
// v; sufixo pré-release ignorado). Retorna -1/0/+1. current não-semver
// (ex.: "dev") devolve ErrNotSemver — o caller decide o fallback.
func CompareVersions(current, latest string) (int, error) {
	cur, err := parseSemver(current)
	if err != nil {
		return 0, err
	}
	lat, err := parseSemver(latest)
	if err != nil {
		return 0, err
	}
	for i := 0; i < 3; i++ {
		if cur[i] != lat[i] {
			if cur[i] < lat[i] {
				return -1, nil
			}
			return 1, nil
		}
	}
	return 0, nil
}

// AssetForPlatform devolve o nome do asset esperado para a plataforma.
func AssetForPlatform(goos, goarch string) string {
	name := "golmet_" + goos + "_" + goarch
	if goos == "windows" {
		name += ".exe"
	}
	return name
}

// parseChecksums lê um checksums.txt no formato sha256sum
// ("<hex>  <filename>" — 1+ espaços entre campos).
func parseChecksums(data []byte) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		sum := strings.ToLower(fields[0])
		if len(sum) != 64 {
			continue // não é um sha256 hex — linha ignorada
		}
		name := strings.TrimPrefix(strings.TrimPrefix(line[len(fields[0]):], " "), "*")
		out[strings.TrimSpace(name)] = sum
	}
	return out
}

// fetchLatest consulta a última release (não-prerelease) do repo.
func fetchLatest(ctx context.Context, apiBase string) (*Release, error) {
	url := strings.TrimRight(apiBase, "/") + "/repos/" + repoOwner + "/" + repoName + "/releases/latest"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "golmet-cli")
	req.Header.Set("Accept", "application/vnd.github+json")
	// token SE presente (rate limit) — nunca obrigatório
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	} else if tok := os.Getenv("GH_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	client := &http.Client{Timeout: dlTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("consulta à API do GitHub falhou (verifique rede/proxy HTTP(S)_PROXY): %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("nenhuma release publicada ainda em github.com/%s/%s", repoOwner, repoName)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API do GitHub respondeu %d — tente novamente ou configure GITHUB_TOKEN para o rate limit", resp.StatusCode)
	}
	var payload struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("resposta da API ilegível: %w", err)
	}
	if payload.TagName == "" {
		return nil, fmt.Errorf("release sem tag_name na resposta da API")
	}
	rel := &Release{Tag: payload.TagName}
	for _, a := range payload.Assets {
		rel.Assets = append(rel.Assets, Asset{Name: a.Name, URL: a.BrowserDownloadURL})
	}
	return rel, nil
}

// download grava url em destPath (proxy ambiente via DefaultTransport).
func download(ctx context.Context, url, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "golmet-cli")
	client := &http.Client{Timeout: dlTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download de %s falhou (verifique rede/proxy): %w", filepath.Base(url), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download de %s respondeu %d", filepath.Base(url), resp.StatusCode)
	}
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("download de %s interrompido: %w", filepath.Base(url), err)
	}
	return nil
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// Run executa o check e, salvo CheckOnly, o upgrade completo.
// current é a versão do binário (cli.Version — "dev" em build local).
func Run(ctx context.Context, opt Options, current string) (*Result, error) {
	out := opt.Out
	if out == nil {
		out = io.Discard
	}
	apiBase := opt.APIBase
	if apiBase == "" {
		apiBase = DefaultAPIBase
	}
	fmt.Fprintln(out, "consultando a última release de github.com/"+repoOwner+"/"+repoName+"...")
	rel, err := fetchLatest(ctx, apiBase)
	if err != nil {
		return nil, err
	}
	res := &Result{Check: CheckResult{Current: current, Latest: rel.Tag}}
	cmp, cerr := CompareVersions(current, rel.Tag)
	switch {
	case cerr != nil:
		// "dev"/não-semver: não há como comparar — reporta update disponível
		res.Check.HasUpdate = true
	default:
		res.Check.HasUpdate = cmp < 0
	}
	if opt.CheckOnly {
		if res.Check.HasUpdate {
			fmt.Fprintf(out, "atualização disponível: %s (local: %s)\n", rel.Tag, current)
		} else {
			fmt.Fprintf(out, "já está atualizado (%s)\n", current)
		}
		return res, nil
	}
	if !res.Check.HasUpdate {
		fmt.Fprintf(out, "já está atualizado (%s >= %s) — nada a fazer\n", current, rel.Tag)
		return res, nil
	}

	bin := opt.ExecutablePath
	if bin == "" {
		bin, err = os.Executable()
		if err != nil {
			return res, fmt.Errorf("não foi possível determinar o binário corrente — rode dentro de um golmet instalado: %w", err)
		}
	}
	bin, err = filepath.Abs(bin)
	if err != nil {
		return res, fmt.Errorf("não foi possível resolver o caminho do binário corrente: %w", err)
	}
	wantAsset := AssetForPlatform(runtime.GOOS, runtime.GOARCH)
	var asset, sums *Asset
	for i := range rel.Assets {
		switch rel.Assets[i].Name {
		case wantAsset:
			asset = &rel.Assets[i]
		case "checksums.txt":
			sums = &rel.Assets[i]
		}
	}
	if asset == nil {
		return res, fmt.Errorf("asset %s ausente na release %s — disponíveis: %s",
			wantAsset, rel.Tag, strings.Join(assetNames(rel.Assets), ", "))
	}
	if sums == nil {
		return res, fmt.Errorf("checksums.txt ausente na release %s — verificação SHA-256 impossível; ABORTANDO sem alterar o binário (linha vermelha)", rel.Tag)
	}

	tmpDir := filepath.Dir(bin) // MESMO volume — rename atômico sem cópia entre discos
	sumsPath := filepath.Join(tmpDir, ".golmet-upgrade-checksums.txt")
	defer os.Remove(sumsPath)
	if err := download(ctx, sums.URL, sumsPath); err != nil {
		return res, err
	}
	sumsData, err := os.ReadFile(sumsPath)
	if err != nil {
		return res, err
	}
	checksums := parseChecksums(sumsData)
	expected, ok := checksums[wantAsset]
	if !ok {
		return res, fmt.Errorf("checksum de %s ausente no checksums.txt da release — ABORTANDO sem alterar o binário (linha vermelha)", wantAsset)
	}

	fmt.Fprintf(out, "baixando %s...\n", wantAsset)
	tmpBin, err := os.CreateTemp(tmpDir, ".golmet-upgrade-*.tmp")
	if err != nil {
		return res, fmt.Errorf("falha ao criar o download temporário no volume do binário: %w", err)
	}
	tmpPath := tmpBin.Name()
	defer os.Remove(tmpPath)
	if err := tmpBin.Close(); err != nil {
		return res, err
	}
	if err := download(ctx, asset.URL, tmpPath); err != nil {
		return res, err
	}
	got, err := os.ReadFile(tmpPath)
	if err != nil {
		return res, err
	}
	if sha256Hex(got) != expected {
		_ = os.Remove(tmpPath)
		return res, fmt.Errorf("checksum SHA-256 divergente (esperado %s) — o download foi descartado e o binário atual NÃO foi alterado", expected)
	}

	old := bin + ".old"
	_ = os.Remove(old) // .old anterior de um upgrade passado — rollback dele não é mais possível
	if err := os.Rename(bin, old); err != nil {
		return res, fmt.Errorf("falha ao renomear o binário corrente para %s: %w", old, err)
	}
	if err := os.Rename(tmpPath, bin); err != nil {
		// restaura o corrente — nunca deixar o operador sem binário
		if rerr := os.Rename(old, bin); rerr != nil {
			return res, fmt.Errorf("falha ao mover o novo binário para %s E falha ao restaurar o antigo (%s ainda existe — restaure manualmente): %w", bin, old, err)
		}
		return res, fmt.Errorf("falha ao mover o novo binário para %s — o binário anterior foi restaurado: %w", bin, err)
	}
	if err := os.Chmod(bin, 0o755); err != nil && runtime.GOOS != "windows" {
		return res, fmt.Errorf("falha ao aplicar chmod 755 no novo binário: %w", err)
	}
	res.Applied = true
	res.BinPath = bin
	res.BackupPath = old
	fmt.Fprintf(out, "atualizado para %s — binário: %s\n", rel.Tag, bin)
	fmt.Fprintf(out, "rollback: o binário anterior está em %s (restaure renomeando de volta)\n", old)
	return res, nil
}

func assetNames(assets []Asset) []string {
	out := make([]string, 0, len(assets))
	for _, a := range assets {
		out = append(out, a.Name)
	}
	return out
}
