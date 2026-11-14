package tools

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// assetName devolve o nome canônico do asset para a plataforma do host
// de teste (hífens — padrão oficial da release; mesma lógica de pickAsset).
func assetName(ext string) string {
	return "codebase-memory-mcp-" + runtime.GOOS + "-" + runtime.GOARCH + ext
}

func fakeBinary() []byte {
	return []byte("#!/usr/bin/env fake-codegraph\n" + strings.Repeat("x", 512))
}

func zipBytes(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	w.Write(content)
	// distrator: install.ps1 presente mas nunca extraído
	if w2, err := zw.Create("install.ps1"); err == nil {
		w2.Write([]byte("evil"))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func tarGzBytes(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(content))}); err != nil {
		t.Fatal(err)
	}
	tw.Write(content)
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func releaseJSON(tag, asset, host string) string {
	return fmt.Sprintf(`{"tag_name": %q, "assets": [{"name": %q, "browser_download_url": "%s/download/%s/%s"}]}`,
		tag, asset, host, tag, asset)
}

func testOptions(srvURL, dest string) Options {
	return Options{
		APIURL:  srvURL + "/releases/latest",
		BaseURL: srvURL + "/download",
		DestDir: dest,
		// sem gh na máquina (injeção) — attestation degrada com aviso
		LookPath: func(string) (string, error) { return "", exec.ErrNotFound },
	}
}

func TestInstallCodegraphOK(t *testing.T) {
	content := fakeBinary()
	asset := assetName(".zip")
	payload := zipBytes(t, BinaryName(), content)
	checksum := sha256Hex(payload) + "  " + asset + "\n"

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/releases/latest":
			w.Write([]byte(releaseJSON("v0.0.1-test", asset, srv.URL)))
		case r.URL.Path == "/download/v0.0.1-test/"+asset:
			w.Write(payload)
		case r.URL.Path == "/download/v0.0.1-test/checksums.txt":
			w.Write([]byte(checksum))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dest := t.TempDir()
	rep, err := InstallCodegraph(context.Background(), testOptions(srv.URL, dest))
	if err != nil {
		t.Fatalf("InstallCodegraph: %v", err)
	}
	if rep.Version != "v0.0.1-test" || rep.SHA256 != sha256Hex(payload) || rep.Bytes != int64(len(payload)) {
		t.Errorf("report = %+v", rep)
	}
	if rep.Attestation != "skipped (gh ausente)" || len(rep.Warnings) == 0 {
		t.Errorf("attestation deveria degradar com aviso declarado: %+v", rep)
	}
	got, rerr := os.ReadFile(filepath.Join(dest, BinaryName()))
	if rerr != nil || !bytes.Equal(got, content) {
		t.Errorf("binário instalado diverge do asset: %v", rerr)
	}
	if strings.Contains(string(got), "evil") || fileExists(filepath.Join(dest, "install.ps1")) {
		t.Errorf("install.ps1 não deveria nunca ser extraído")
	}
}

func TestInstallCodegraphTarGzVariant(t *testing.T) {
	content := fakeBinary()
	// release publica APENAS .tar.gz para a plataforma — pickAsset cai na 2ª opção
	asset := assetName(".tar.gz")
	payload := tarGzBytes(t, BinaryName(), content)
	checksum := sha256Hex(payload) + "  " + asset + "\n"

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/releases/latest":
			w.Write([]byte(releaseJSON("v0.0.2-test", asset, srv.URL)))
		case r.URL.Path == "/download/v0.0.2-test/"+asset:
			w.Write(payload)
		case r.URL.Path == "/download/v0.0.2-test/checksums.txt":
			w.Write([]byte(checksum))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dest := t.TempDir()
	_, err := InstallCodegraph(context.Background(), testOptions(srv.URL, dest))
	if err != nil {
		t.Fatalf("InstallCodegraph tar.gz: %v", err)
	}
	got, rerr := os.ReadFile(filepath.Join(dest, BinaryName()))
	if rerr != nil || !bytes.Equal(got, content) {
		t.Errorf("binário tar.gz diverge: %v", rerr)
	}
}

func TestInstallCodegraphChecksumMismatchFailsClosed(t *testing.T) {
	content := fakeBinary()
	asset := assetName(".zip")
	payload := zipBytes(t, BinaryName(), content)
	wrong := sha256Hex([]byte("outra coisa")) + "  " + asset + "\n"

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/releases/latest":
			w.Write([]byte(releaseJSON("v0.0.3-test", asset, srv.URL)))
		case r.URL.Path == "/download/v0.0.3-test/"+asset:
			w.Write(payload)
		case r.URL.Path == "/download/v0.0.3-test/checksums.txt":
			w.Write([]byte(wrong))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dest := t.TempDir()
	_, err := InstallCodegraph(context.Background(), testOptions(srv.URL, dest))
	if err == nil || !strings.Contains(err.Error(), "SHA-256 divergente") {
		t.Fatalf("esperava falha fechada de checksum, veio %v", err)
	}
	if fileExists(filepath.Join(dest, BinaryName())) {
		t.Errorf("binário NÃO deveria ser instalado com hash divergente")
	}
}

func TestInstallCodegraphAssetNotListedInChecksums(t *testing.T) {
	content := fakeBinary()
	asset := assetName(".zip")
	payload := zipBytes(t, BinaryName(), content)

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/releases/latest":
			w.Write([]byte(releaseJSON("v0.0.4-test", asset, srv.URL)))
		case r.URL.Path == "/download/v0.0.4-test/"+asset:
			w.Write(payload)
		case r.URL.Path == "/download/v0.0.4-test/checksums.txt":
			w.Write([]byte("deadbeef  outro-arquivo.zip\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	_, err := InstallCodegraph(context.Background(), testOptions(srv.URL, t.TempDir()))
	if err == nil || !strings.Contains(err.Error(), "não listado no checksums.txt") {
		t.Fatalf("esperava erro de asset ausente do checksums, veio %v", err)
	}
}

func TestInstallCodegraphPlatformAssetMissing(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/releases/latest" {
			// apenas um asset de outra plataforma
			w.Write([]byte(`{"tag_name": "v0.0.5-test", "assets": [{"name": "codebase-memory-mcp-solaris-sparc.tar.gz", "browser_download_url": "x"}]}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	_, err := InstallCodegraph(context.Background(), testOptions(srv.URL, t.TempDir()))
	if err == nil || !strings.Contains(err.Error(), "asset indisponível para "+runtime.GOOS+"/"+runtime.GOARCH) {
		t.Fatalf("esperava erro de plataforma sem asset, veio %v", err)
	}
}

func TestInstallCodegraphAttestationFailFailsClosed(t *testing.T) {
	content := fakeBinary()
	asset := assetName(".zip")
	payload := zipBytes(t, BinaryName(), content)
	checksum := sha256Hex(payload) + "  " + asset + "\n"

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/releases/latest":
			w.Write([]byte(releaseJSON("v0.0.6-test", asset, srv.URL)))
		case r.URL.Path == "/download/v0.0.6-test/"+asset:
			w.Write(payload)
		case r.URL.Path == "/download/v0.0.6-test/checksums.txt":
			w.Write([]byte(checksum))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	opt := testOptions(srv.URL, t.TempDir())
	opt.LookPath = func(string) (string, error) { return "/usr/bin/gh", nil }
	opt.RunAttestation = func(ctx context.Context, file string) error {
		return fmt.Errorf("attestation verify saiu 1 — provenance não confere")
	}

	_, err := InstallCodegraph(context.Background(), opt)
	if err == nil || !strings.Contains(err.Error(), "attestation falhou") {
		t.Fatalf("esperava falha fechada de attestation, veio %v", err)
	}
}

func TestInstallCodegraphAttestationVerified(t *testing.T) {
	content := fakeBinary()
	asset := assetName(".zip")
	payload := zipBytes(t, BinaryName(), content)
	checksum := sha256Hex(payload) + "  " + asset + "\n"

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/releases/latest":
			w.Write([]byte(releaseJSON("v0.0.7-test", asset, srv.URL)))
		case r.URL.Path == "/download/v0.0.7-test/"+asset:
			w.Write(payload)
		case r.URL.Path == "/download/v0.0.7-test/checksums.txt":
			w.Write([]byte(checksum))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	opt := testOptions(srv.URL, t.TempDir())
	opt.LookPath = func(string) (string, error) { return "/usr/bin/gh", nil }
	opt.RunAttestation = func(ctx context.Context, file string) error { return nil }

	rep, err := InstallCodegraph(context.Background(), opt)
	if err != nil {
		t.Fatalf("InstallCodegraph: %v", err)
	}
	if rep.Attestation != "verified" || len(rep.Warnings) != 0 {
		t.Errorf("attestation deveria estar verified sem avisos: %+v", rep)
	}
}

func TestInstallCodegraphNoNetwork(t *testing.T) {
	opt := Options{DestDir: t.TempDir(), Client: &http.Client{Timeout: 1}}
	// URL inalcançável — erro claro com fix
	_, err := InstallCodegraph(context.Background(), opt)
	if err == nil {
		t.Fatalf("esperava erro de rede")
	}
	if !strings.Contains(err.Error(), "verifique conectividade") {
		t.Errorf("erro sem rede deveria sugerir o fix manual: %v", err)
	}
}

func TestList(t *testing.T) {
	lines := List()
	if len(lines) != 1 {
		t.Fatalf("List = %v", lines)
	}
	if !strings.Contains(lines[0], "codegraph") {
		t.Errorf("linha do List não menciona codegraph: %s", lines[0])
	}
}
