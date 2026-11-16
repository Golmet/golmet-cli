package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ---- M31 — o launcher npm (bin/golmet.js) ----

// nodeAvailable — os testes do launcher exigem node no PATH (o CI do kit
// roda em runner com node; sem node, skip — nunca falha a suite Go).
func nodeAvailable(t *testing.T) string {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node indisponível — os testes do launcher npm são skipped")
	}
	return node
}

// launcherSource — o caminho absoluto do launcher no repo.
func launcherSource(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("..", "..", "bin", "golmet.js"))
	if err != nil {
		t.Fatal(err)
	}
	if _, serr := os.Stat(p); serr != nil {
		t.Fatalf("bin/golmet.js ausente: %v", serr)
	}
	return p
}

// nodePlatform — o naming do Node (process.platform/arch) difere do Go
// (windows→win32, amd64→x64) — o sandbox usa o naming do NODE (o launcher
// detecta via process.platform).
func nodePlatform() string {
	switch runtime.GOOS {
	case "windows":
		return "win32"
	}
	return runtime.GOOS
}

func nodeArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "x64"
	}
	return runtime.GOARCH
}

// npmSandbox — o shape npm real: node_modules/golmet-cli/bin/golmet.js +
// node_modules/@golmet/cli-<plataforma REAL>/golmet[.exe] (o stub compilado,
// copiado do path dado).
func npmSandbox(t *testing.T, stubPath string) string {
	t.Helper()
	root := t.TempDir()
	cliDir := filepath.Join(root, "node_modules", "golmet-cli", "bin")
	if err := os.MkdirAll(cliDir, 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(launcherSource(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cliDir, "golmet.js"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	pkgDir := filepath.Join(root, "node_modules", "@golmet", "cli-"+nodePlatform()+"-"+nodeArch())
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"name":"x","version":"0.0.0-test"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	binName := "golmet"
	if nodePlatform() == "win32" {
		binName = "golmet.exe"
	}
	stubBytes, err := os.ReadFile(stubPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, binName), stubBytes, 0o755); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(root, "node_modules", "golmet-cli", "bin", "golmet.js")
}

// runLauncher — roda o launcher do sandbox com os args dados; devolve
// stdout+stderr e o exit code.
func runLauncher(t *testing.T, launcherPath string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(nodeAvailable(t), append([]string{launcherPath}, args...)...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("rodar o launcher: %v", err)
	}
	return buf.String(), code
}

// stubExe — compila um stub Go mínimo (imprime e sai com o code dado) —
// o "binário nativo" do sandbox.
func stubExe(t *testing.T, dir, message string, code int) string {
	t.Helper()
	src := filepath.Join(dir, "stub.go")
	out := filepath.Join(dir, "stub")
	if runtime.GOOS == "windows" {
		out += ".exe"
	}
	prog := "package main\n\nimport (\n\t\"fmt\"\n\t\"os\"\n)\n\nfunc main() {\n\tfmt.Println(" +
		"\"" + message + "\"" + ")\n\tos.Exit(" + itoaTest(code) + ")\n}\n"
	if err := os.WriteFile(src, []byte(prog), 0o644); err != nil {
		t.Fatal(err)
	}
	build := exec.Command("go", "build", "-o", out, src)
	if b, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build do stub: %v\n%s", err, b)
	}
	return out
}

func itoaTest(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}

// A versão DELEGA ao binário (o -v/--version passa direto).
func TestNPMLauncherDelegatesVersion(t *testing.T) {
	dir := t.TempDir()
	stub := stubExe(t, dir, "golmet 0.3.0 (stub)", 0)
	launcher := npmSandbox(t, stub)
	out, code := runLauncher(t, launcher, "--version")
	if code != 0 || !strings.Contains(out, "golmet 0.3.0 (stub)") {
		t.Errorf("versão delegada = exit %d out %q", code, out)
	}
}

// O exit code do binário PROPAGA (o contrato do CLI — 0/1/2).
func TestNPMLauncherExitCodePropagates(t *testing.T) {
	dir := t.TempDir()
	stub := stubExe(t, dir, "findings", 7)
	launcher := npmSandbox(t, stub)
	_, code := runLauncher(t, launcher)
	if code != 7 {
		t.Errorf("exit = %d, want 7 (propagado do binário)", code)
	}
}

// Pacote da plataforma NÃO instalado → erro claro com os dois caminhos
// (exit 2 — o fail-closed do launcher).
func TestNPMLauncherMissingPlatformPackage(t *testing.T) {
	node := nodeAvailable(t)
	root := t.TempDir()
	cliDir := filepath.Join(root, "node_modules", "golmet-cli", "bin")
	if err := os.MkdirAll(cliDir, 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(launcherSource(t))
	if err != nil {
		t.Fatal(err)
	}
	launcher := filepath.Join(cliDir, "golmet.js")
	if err := os.WriteFile(launcher, data, 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(node, launcher)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	rerr := cmd.Run()
	code := 0
	if ee, ok := rerr.(*exec.ExitError); ok {
		code = ee.ExitCode()
	}
	out := buf.String()
	if code != 2 {
		t.Errorf("exit = %d, want 2 (pacote ausente)", code)
	}
	for _, want := range []string{
		"não acompanhou a instalação",
		"npm install -g golmet-cli",
		"github.com/Golmet/golmet-cli/releases",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("mensagem de plataforma ausente sem %q:\n%s", want, out)
		}
	}
}

// A resolução da plataforma (o mock da convenção) — via node -e sobre o
// módulo exportado do launcher.
func TestNPMLauncherPlatformResolution(t *testing.T) {
	node := nodeAvailable(t)
	src := launcherSource(t)
	cases := []struct{ platform, arch, want, bin string }{
		{"darwin", "arm64", "@golmet/cli-darwin-arm64", "golmet"},
		{"win32", "x64", "@golmet/cli-win32-x64", "golmet.exe"},
		{"linux", "arm64", "@golmet/cli-linux-arm64", "golmet"},
	}
	for _, c := range cases {
		expr := "const l=require('" + strings.ReplaceAll(src, "\\", "/") +
			"'); console.log(l.platformPackage('" + c.platform + "','" + c.arch + "')+'|'+l.binaryName('" + c.platform + "'))"
		out, err := exec.Command(node, "-e", expr).CombinedOutput()
		if err != nil {
			t.Fatalf("node -e: %v\n%s", err, out)
		}
		if got := strings.TrimSpace(string(out)); got != c.want+"|"+c.bin {
			t.Errorf("resolução %s/%s = %q, want %q|%q", c.platform, c.arch, got, c.want, c.bin)
		}
	}
}

// ---- M31 — o shape dos pacotes npm ----

// O principal: version 0.3.0, bin golmet → ./bin/golmet.js, os 5
// optionalDependencies @ 0.3.0, engines node >=14, files com bin/.
func TestNPMPackageShape(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "package.json"))
	if err != nil {
		t.Fatalf("package.json ausente: %v", err)
	}
	var pkg struct {
		Name    string            `json:"name"`
		Version string            `json:"version"`
		Bin     map[string]string `json:"bin"`
		Engines struct {
			Node string `json:"node"`
		} `json:"engines"`
		Files        []string          `json:"files"`
		OptionalDeps map[string]string `json:"optionalDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		t.Fatalf("package.json ilegível: %v", err)
	}
	if pkg.Name != "golmet-cli" || pkg.Version != "0.3.0" {
		t.Errorf("name/version = %s/%s", pkg.Name, pkg.Version)
	}
	if pkg.Bin["golmet"] != "./bin/golmet.js" {
		t.Errorf("bin = %v", pkg.Bin)
	}
	if pkg.Engines.Node != ">=14" {
		t.Errorf("engines.node = %q, want >=14 (o launcher é Node >=14 sem deps)", pkg.Engines.Node)
	}
	hasBin := false
	for _, f := range pkg.Files {
		if f == "bin/" {
			hasBin = true
		}
	}
	if !hasBin {
		t.Errorf("files sem bin/: %v", pkg.Files)
	}
	want := map[string]string{
		"@golmet/cli-win32-x64":    "0.3.0",
		"@golmet/cli-linux-x64":    "0.3.0",
		"@golmet/cli-linux-arm64":  "0.3.0",
		"@golmet/cli-darwin-x64":   "0.3.0",
		"@golmet/cli-darwin-arm64": "0.3.0",
	}
	if len(pkg.OptionalDeps) != len(want) {
		t.Fatalf("optionalDependencies = %d, want 5", len(pkg.OptionalDeps))
	}
	for name, ver := range want {
		if got := pkg.OptionalDeps[name]; got != ver {
			t.Errorf("optionalDependencies[%s] = %q, want %q", name, got, ver)
		}
	}
}

// Os 5 sub-pacotes de plataforma: name/os/cpu coerentes com o dir e o
// binário declarado no files.
func TestNPMPlatformPackagesShape(t *testing.T) {
	cases := []struct{ dir, name, os, cpu, bin string }{
		{"win32-x64", "@golmet/cli-win32-x64", "win32", "x64", "golmet.exe"},
		{"linux-x64", "@golmet/cli-linux-x64", "linux", "x64", "golmet"},
		{"linux-arm64", "@golmet/cli-linux-arm64", "linux", "arm64", "golmet"},
		{"darwin-x64", "@golmet/cli-darwin-x64", "darwin", "x64", "golmet"},
		{"darwin-arm64", "@golmet/cli-darwin-arm64", "darwin", "arm64", "golmet"},
	}
	for _, c := range cases {
		data, err := os.ReadFile(filepath.Join("..", "..", "npm", c.dir, "package.json"))
		if err != nil {
			t.Fatalf("package.json de %s ausente: %v", c.dir, err)
		}
		var pkg struct {
			Name    string   `json:"name"`
			Version string   `json:"version"`
			OS      []string `json:"os"`
			CPU     []string `json:"cpu"`
			Files   []string `json:"files"`
		}
		if err := json.Unmarshal(data, &pkg); err != nil {
			t.Fatalf("package.json de %s ilegível: %v", c.dir, err)
		}
		if pkg.Name != c.name || pkg.Version != "0.3.0" {
			t.Errorf("%s: name/version = %s/%s", c.dir, pkg.Name, pkg.Version)
		}
		if len(pkg.OS) != 1 || pkg.OS[0] != c.os || len(pkg.CPU) != 1 || pkg.CPU[0] != c.cpu {
			t.Errorf("%s: os/cpu = %v/%v", c.dir, pkg.OS, pkg.CPU)
		}
		if len(pkg.Files) != 1 || pkg.Files[0] != c.bin {
			t.Errorf("%s: files = %v, want [%s]", c.dir, pkg.Files, c.bin)
		}
	}
}

// ---- M31 — o release.yml: a ordem npm (os 5 de plataforma ANTES do
// principal), o NPM_TOKEN e a tolerância a falha ----

func TestReleaseWorkflowNPMPublish(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "release.yml"))
	if err != nil {
		t.Fatalf("release.yml ausente: %v", err)
	}
	y := string(data)
	if strings.Contains(y, "Publicação npm NÃO faz parte") {
		t.Errorf("o release.yml ainda carrega o comentário placeholder da M4")
	}
	// a ordem: os 5 de plataforma ANTES do principal (o npm resolve os
	// opcionais na instalação do principal)
	idxPlatform := strings.Index(y, "Publish platform npm packages")
	idxMain := strings.Index(y, "Publish main npm package")
	if idxPlatform < 0 || idxMain < 0 || idxPlatform > idxMain {
		t.Errorf("ordem npm errada (plataforma=%d principal=%d) — os 5 de plataforma vêm ANTES do principal", idxPlatform, idxMain)
	}
	if !strings.Contains(y, "npm/win32-x64") || !strings.Contains(y, "npm/darwin-arm64") {
		t.Errorf("os 5 pacotes de plataforma ausentes do passo de publish")
	}
	// a tolerância a falha (decisão da M4): o release nunca quebra por npm
	if !strings.Contains(y, "continue-on-error: true") {
		t.Errorf("os passos npm devem ser tolerantes a falha (continue-on-error)")
	}
	if !strings.Contains(y, "NPM_TOKEN") {
		t.Errorf("o publish npm sem o NPM_TOKEN")
	}
	// a versão fixada pela tag
	if !strings.Contains(y, "GITHUB_REF_NAME#v") {
		t.Errorf("a versão dos pacotes deveria vir da tag (GITHUB_REF_NAME#v)")
	}
}
