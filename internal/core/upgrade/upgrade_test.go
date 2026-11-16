package upgrade

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		cur, latest string
		want        int
		wantErr     bool
	}{
		{"v1.2.3", "v1.2.3", 0, false},
		{"1.2.3", "v1.2.3", 0, false},      // prefixo v tolerado nos dois lados
		{"v1.2.3", "1.2.4", -1, false},     // patch menor
		{"v1.2.4", "v1.2.3", 1, false},     // patch maior
		{"v1.2.3", "v1.3.0", -1, false},    // minor menor
		{"v1.9.0", "v1.10.0", -1, false},   // comparação NUMÉRICA (não lexica)
		{"v2.0.0", "v1.99.99", 1, false},   // major maior
		{"v1.2.3-rc1", "v1.2.3", 0, false}, // sufixo pré-release ignorado
		{"dev", "v9.9.9", 0, true},         // build local não é semver
		{"", "v1.0.0", 0, true},
		{"abc", "v1.0.0", 0, true},
	}
	for _, tc := range cases {
		got, err := CompareVersions(tc.cur, tc.latest)
		if (err != nil) != tc.wantErr {
			t.Errorf("Compare(%q, %q) err = %v, wantErr %v", tc.cur, tc.latest, err, tc.wantErr)
			continue
		}
		if !tc.wantErr && got != tc.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", tc.cur, tc.latest, got, tc.want)
		}
	}
	// ErrNotSemver identificável
	if _, err := CompareVersions("dev", "v1.0.0"); err == nil || !strings.Contains(err.Error(), "semver") {
		t.Errorf("Compare(dev, ...) deveria devolver ErrNotSemver, got %v", err)
	}
}

func TestAssetForPlatform(t *testing.T) {
	if got := AssetForPlatform("windows", "amd64"); got != "golmet_windows_amd64.exe" {
		t.Errorf("windows asset = %q", got)
	}
	if got := AssetForPlatform("linux", "arm64"); got != "golmet_linux_arm64" {
		t.Errorf("linux asset = %q", got)
	}
	if got := AssetForPlatform("darwin", "amd64"); got != "golmet_darwin_amd64" {
		t.Errorf("darwin asset = %q", got)
	}
}

func TestParseChecksums(t *testing.T) {
	data := []byte(strings.Join([]string{
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  golmet_windows_amd64.exe",
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb *golmet_linux_amd64",
		"curta-linha-inválida",
		"",
		"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc golmet_darwin_arm64",
	}, "\n"))
	got := parseChecksums(data)
	if len(got) != 3 {
		t.Fatalf("parseChecksums = %d entradas, want 3", len(got))
	}
	if got["golmet_windows_amd64.exe"] != strings.Repeat("a", 64) {
		t.Errorf("entrada windows = %q", got["golmet_windows_amd64.exe"])
	}
	// formato *binário (binário mode do sha256sum) — asterisco removido
	if got["golmet_linux_amd64"] != strings.Repeat("b", 64) {
		t.Errorf("entrada linux = %q", got["golmet_linux_amd64"])
	}
	if _, ok := got["curta-linha-inválida"]; ok {
		t.Errorf("linha sem sha256 de 64 chars não deveria ser indexada")
	}
}

// fakeRelease monta um httptest que serve a release JSON + os assets —
// os checksums são reais (SHA-256 dos payloads gravados).
func fakeRelease(t *testing.T, tag, binaryPayload string, checksumBroken bool, missingChecksum, missingAsset bool) *httptest.Server {
	t.Helper()
	sum := sha256.Sum256([]byte(binaryPayload))
	binSum := hex.EncodeToString(sum[:])
	if checksumBroken {
		binSum = strings.Repeat("0", 64)
	}
	binName := AssetForPlatform(runtime.GOOS, runtime.GOARCH)
	sums := fmt.Sprintf("%s  %s\n", binSum, binName)
	if missingChecksum {
		sums = "9999999999999999999999999999999999999999999999999999999999999999  outro-asset\n"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/Golmet/golmet-cli/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "golmet-cli" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		// binName (closure externa) — mesmo alvo do /download/
		assets := []map[string]string{
			{"name": "checksums.txt", "browser_download_url": srvURL(r, "/download/"+tag+"/checksums.txt")},
		}
		if !missingAsset {
			assets = append(assets, map[string]string{
				"name": binName, "browser_download_url": srvURL(r, "/download/"+tag+"/"+binName),
			})
		}
		payload := map[string]any{"tag_name": tag, "assets": assets}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	})
	mux.HandleFunc("/download/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/checksums.txt"):
			_, _ = w.Write([]byte(sums))
		case strings.HasSuffix(r.URL.Path, "/"+binName):
			_, _ = w.Write([]byte(binaryPayload))
		default:
			http.NotFound(w, r)
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func srvURL(r *http.Request, path string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host + path
}

// upgradeSandbox cria um "binário corrente" fake num diretório próprio
// (mesmo volume do temp do SO — rename funciona) e devolve o caminho.
func upgradeSandbox(t *testing.T, content string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "golmet")
	if runtime.GOOS == "windows" {
		p += ".exe"
	}
	if err := os.WriteFile(p, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestRunCheckOnly(t *testing.T) {
	srv := fakeRelease(t, "v9.9.9", "novo-binario", false, false, false)
	var out strings.Builder
	res, err := Run(context.Background(), Options{
		CheckOnly: true, APIBase: srv.URL,
		ExecutablePath: upgradeSandbox(t, "antigo"),
		Out:            &out,
	}, "v1.0.0")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Applied {
		t.Errorf("--check aplicou o upgrade (read-only violado)")
	}
	if !res.Check.HasUpdate || res.Check.Latest != "v9.9.9" || res.Check.Current != "v1.0.0" {
		t.Errorf("check = %+v", res.Check)
	}
	if !strings.Contains(out.String(), "atualização disponível") {
		t.Errorf("saída sem o anúncio: %s", out.String())
	}
}

func TestRunAlreadyUpdated(t *testing.T) {
	srv := fakeRelease(t, "v1.0.0", "novo", false, false, false)
	var out strings.Builder
	res, err := Run(context.Background(), Options{
		APIBase: srv.URL, ExecutablePath: upgradeSandbox(t, "atual"), Out: &out,
	}, "v1.0.0")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Applied || res.Check.HasUpdate {
		t.Errorf("igual deveria ser no-op: %+v", res)
	}
	if !strings.Contains(out.String(), "já está atualizado") {
		t.Errorf("saída sem 'já está atualizado': %s", out.String())
	}
}

// current >= latest (semver maior local) → no-op
func TestRunLocalNewer(t *testing.T) {
	srv := fakeRelease(t, "v1.0.0", "novo", false, false, false)
	res, err := Run(context.Background(), Options{
		APIBase: srv.URL, ExecutablePath: upgradeSandbox(t, "atual"),
	}, "v2.0.0")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Applied || res.Check.HasUpdate {
		t.Errorf("local maior deveria ser no-op: %+v", res)
	}
}

// dev (não-semver) → update disponível, sem crash
func TestRunDevVersion(t *testing.T) {
	srv := fakeRelease(t, "v1.0.0", "novo", false, false, false)
	res, err := Run(context.Background(), Options{
		CheckOnly: true, APIBase: srv.URL, ExecutablePath: upgradeSandbox(t, "x"),
	}, "dev")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !res.Check.HasUpdate {
		t.Errorf("dev deveria reportar update disponível: %+v", res.Check)
	}
}

func TestRunUpgradeApplies(t *testing.T) {
	const oldPayload, newPayload = "binario-antigo-v1", "binario-novo-v999"
	srv := fakeRelease(t, "v9.9.9", newPayload, false, false, false)
	bin := upgradeSandbox(t, oldPayload)
	var out strings.Builder
	res, err := Run(context.Background(), Options{
		APIBase: srv.URL, ExecutablePath: bin, Out: &out,
	}, "v1.0.0")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !res.Applied {
		t.Fatalf("upgrade não aplicado: %+v (out: %s)", res, out.String())
	}
	got, rerr := os.ReadFile(bin)
	if rerr != nil || string(got) != newPayload {
		t.Errorf("binário após upgrade = %q (err %v), want %q", string(got), rerr, newPayload)
	}
	oldBytes, oerr := os.ReadFile(bin + ".old")
	if oerr != nil || string(oldBytes) != oldPayload {
		t.Errorf(".old = %q (err %v), want %q", string(oldBytes), oerr, oldPayload)
	}
	if res.BinPath != bin || res.BackupPath != bin+".old" {
		t.Errorf("paths do resultado = %s/%s", res.BinPath, res.BackupPath)
	}
	if runtime.GOOS != "windows" {
		info, serr := os.Stat(bin)
		if serr != nil || info.Mode().Perm()&0o100 == 0 {
			t.Errorf("binário sem permissão de execução pós-upgrade: %v (%v)", info, serr)
		}
	}
	if !strings.Contains(out.String(), "rollback") {
		t.Errorf("saída sem a dica de rollback: %s", out.String())
	}
}

// checksum divergente → ABORTA, binário intocado, sem .old
func TestRunChecksumMismatchAborts(t *testing.T) {
	srv := fakeRelease(t, "v9.9.9", "novo", true /*broken*/, false, false)
	bin := upgradeSandbox(t, "antigo-intacto")
	res, err := Run(context.Background(), Options{
		APIBase: srv.URL, ExecutablePath: bin,
	}, "v1.0.0")
	if err == nil {
		t.Fatalf("checksum divergente deveria abortar, got %+v", res)
	}
	if res.Applied {
		t.Errorf("aplicou com checksum divergente")
	}
	got, rerr := os.ReadFile(bin)
	if rerr != nil || string(got) != "antigo-intacto" {
		t.Errorf("binário FOI alterado: %q (err %v)", string(got), rerr)
	}
	if _, serr := os.Stat(bin + ".old"); !os.IsNotExist(serr) {
		t.Errorf(".old criado num upgrade abortado (err %v)", serr)
	}
	if !strings.Contains(err.Error(), "checksum") {
		t.Errorf("erro sem menção a checksum: %v", err)
	}
}

// checksums.txt sem o asset → aborta
func TestRunChecksumMissingAborts(t *testing.T) {
	srv := fakeRelease(t, "v9.9.9", "novo", false, true /*missing*/, false)
	bin := upgradeSandbox(t, "antigo-intacto")
	res, err := Run(context.Background(), Options{
		APIBase: srv.URL, ExecutablePath: bin,
	}, "v1.0.0")
	if err == nil || res.Applied {
		t.Fatalf("checksum ausente deveria abortar, got %+v err %v", res, err)
	}
	got, rerr := os.ReadFile(bin)
	if rerr != nil || string(got) != "antigo-intacto" {
		t.Errorf("binário FOI alterado: %q (err %v)", string(got), rerr)
	}
}

// asset da plataforma ausente na release → erro com a lista de disponíveis
func TestRunAssetMissing(t *testing.T) {
	srv := fakeRelease(t, "v9.9.9", "novo", false, false, true /*missing asset*/)
	bin := upgradeSandbox(t, "antigo")
	res, err := Run(context.Background(), Options{
		APIBase: srv.URL, ExecutablePath: bin,
	}, "v1.0.0")
	if err == nil || res.Applied {
		t.Fatalf("asset ausente deveria errar, got %+v err %v", res, err)
	}
	if !strings.Contains(err.Error(), "disponíveis") || !strings.Contains(err.Error(), "checksums.txt") {
		t.Errorf("erro sem a lista de assets disponíveis: %v", err)
	}
}

// sem release publicada → erro claro (404)
func TestRunNoRelease(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/Golmet/golmet-cli/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	_, err := Run(context.Background(), Options{CheckOnly: true, APIBase: srv.URL}, "v1.0.0")
	if err == nil || !strings.Contains(err.Error(), "nenhuma release publicada") {
		t.Errorf("erro de release ausente inadequado: %v", err)
	}
}
