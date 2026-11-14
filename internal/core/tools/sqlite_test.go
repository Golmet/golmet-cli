package tools

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// serveSqliteRelease monta um httptest com o asset zip (conteúdo dado) e
// o checksums.txt correspondente (ou divergente, para o caso de falha).
func serveSqliteRelease(t *testing.T, exeContent []byte, corruptChecksum bool) (*httptest.Server, string) {
	t.Helper()
	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)
	w, err := zw.Create(SqliteBinaryName())
	if err != nil {
		t.Fatal(err)
	}
	w.Write(exeContent)
	w2, err := zw.Create("README.md")
	if err != nil {
		t.Fatal(err)
	}
	w2.Write([]byte("docs"))
	zw.Close()

	sum := sha256.Sum256(zipBuf.Bytes())
	checksumLine := hex.EncodeToString(sum[:]) + "  " + SqliteAssetName()
	if corruptChecksum {
		// hash falsificado mantendo o FORMATO (2 fields: hash de 64 + nome)
		checksumLine = "deadbeef" + checksumLine[8:64] + "  " + SqliteAssetName()
	}
	mux := http.NewServeMux()
	// BaseURL do sqlite JÁ inclui a versão (…/download/v0.0.3) — os paths
	// do mux são os assets diretos
	mux.HandleFunc("/"+SqliteAssetName(), func(w http.ResponseWriter, r *http.Request) {
		w.Write(zipBuf.Bytes())
	})
	mux.HandleFunc("/checksums.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(checksumLine + "\n"))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, checksumLine
}

// download + SHA-256 do checksums.txt + extração do binário → ok.
func TestInstallSqliteOK(t *testing.T) {
	exe := []byte("fake mcp-sqlite binary")
	srv, _ := serveSqliteRelease(t, exe, false)
	dest := t.TempDir()

	rep, err := InstallSqlite(context.Background(), SqliteOptions{
		BaseURL: srv.URL,
		DestDir: dest,
	})
	if err != nil {
		t.Fatalf("InstallSqlite: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dest, SqliteBinaryName()))
	if err != nil {
		t.Fatalf("binário não instalado: %v", err)
	}
	if !bytes.Equal(got, exe) {
		t.Errorf("binário instalado diverge do conteúdo do zip")
	}
	if !strings.Contains(rep.Attestation, "sem attestation") {
		t.Errorf("ausência de attestation deveria ser declarada: %s", rep.Attestation)
	}
	if len(rep.Warnings) == 0 {
		t.Errorf("aviso declarado da ausência de attestation ausente")
	}
}

// SHA-256 divergente do checksums.txt → FALHA FECHADA, nada instalado.
func TestInstallSqliteHashMismatch(t *testing.T) {
	srv, _ := serveSqliteRelease(t, []byte("fake"), true)
	dest := t.TempDir()

	resp, _ := http.Get(srv.URL + "/checksums.txt")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	t.Logf("checksums servido: %q", strings.TrimSpace(string(body)))

	if _, err := InstallSqlite(context.Background(), SqliteOptions{
		BaseURL: srv.URL,
		DestDir: dest,
	}); err == nil || !strings.Contains(err.Error(), "SHA-256 divergente") {
		t.Fatalf("hash divergente deveria falhar fechado: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, SqliteBinaryName())); !os.IsNotExist(err) {
		t.Errorf("NADA deveria ser instalado com hash divergente")
	}
}

// asset inexistente na release (404) → falha com instrução manual.
func TestInstallSqliteAssetMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	dest := t.TempDir()

	_, err := InstallSqlite(context.Background(), SqliteOptions{
		BaseURL: srv.URL,
		DestDir: dest,
	})
	if err == nil {
		t.Fatal("asset 404 deveria falhar")
	}
	if !strings.Contains(err.Error(), "release") {
		t.Errorf("erro sem a instrução da release: %v", err)
	}
}

// nomes por plataforma: asset/zip/binary com GOOS/GOARCH.
func TestSqliteNames(t *testing.T) {
	if !strings.Contains(SqliteAssetName(), "mcp-sqlite_0.0.3_") {
		t.Errorf("asset sem a versão validada: %s", SqliteAssetName())
	}
	if !strings.Contains(SqliteAssetName(), ".zip") {
		t.Errorf("asset sem .zip: %s", SqliteAssetName())
	}
	if !strings.HasPrefix(SqliteBinaryName(), "mcp-sqlite") {
		t.Errorf("binary name inesperado: %s", SqliteBinaryName())
	}
}
