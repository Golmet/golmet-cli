package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
	stub := stubExe(t, dir, "golmet 1.0.0 (stub)", 0)
	launcher := npmSandbox(t, stub)
	out, code := runLauncher(t, launcher, "--version")
	if code != 0 || !strings.Contains(out, "golmet 1.0.0 (stub)") {
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

// O principal: version em semver X.Y.Z (o version workflow bumpa o repo a
// cada release — a tag aponta para o commit com a versão gravada), bin
// golmet → ./bin/golmet.js, os 5 optionalDependencies @ MESMA versão
// (a lição da issue do npm sem binário: optional resolvido por versão
// exata — divergência = launcher sem binário), engines node >=14, files
// com bin/.
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
	semver := regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	if pkg.Name != "golmet-cli" || !semver.MatchString(pkg.Version) {
		t.Errorf("name/version = %s/%s (version deve ser semver X.Y.Z gravado no repo)", pkg.Name, pkg.Version)
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
	want := map[string]bool{
		"@golmet/cli-win32-x64":    true,
		"@golmet/cli-linux-x64":    true,
		"@golmet/cli-linux-arm64":  true,
		"@golmet/cli-darwin-x64":   true,
		"@golmet/cli-darwin-arm64": true,
	}
	if len(pkg.OptionalDeps) != len(want) {
		t.Fatalf("optionalDependencies = %d, want 5", len(pkg.OptionalDeps))
	}
	for name := range want {
		if got := pkg.OptionalDeps[name]; got != pkg.Version {
			t.Errorf("optionalDependencies[%s] = %q, want %q (a versão da optional é a MESMA do principal — npm resolve por versão exata)", name, got, pkg.Version)
		}
	}
}

// Os 5 sub-pacotes de plataforma: name/os/cpu coerentes com o dir e o
// binário declarado no files; version SEMPRE = a do principal (os 6
// sincronizados pelo version workflow).
func TestNPMPlatformPackagesShape(t *testing.T) {
	root, err := os.ReadFile(filepath.Join("..", "..", "package.json"))
	if err != nil {
		t.Fatalf("package.json ausente: %v", err)
	}
	var rootPkg struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(root, &rootPkg); err != nil {
		t.Fatalf("package.json ilegível: %v", err)
	}
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
		if pkg.Name != c.name || pkg.Version != rootPkg.Version {
			t.Errorf("%s: name/version = %s/%s, want version %q (sincronizado com o principal)", c.dir, pkg.Name, pkg.Version, rootPkg.Version)
		}
		if len(pkg.OS) != 1 || pkg.OS[0] != c.os || len(pkg.CPU) != 1 || pkg.CPU[0] != c.cpu {
			t.Errorf("%s: os/cpu = %v/%v", c.dir, pkg.OS, pkg.CPU)
		}
		if len(pkg.Files) != 1 || pkg.Files[0] != c.bin {
			t.Errorf("%s: files = %v, want [%s]", c.dir, pkg.Files, c.bin)
		}
	}
}

// ---- M38 — o release.yml endurecido: a ordem npm (os 5 de plataforma
// ANTES do principal), o preflight do token, a idempotência e o VERMELHO
// no publish falho ----

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
	if !strings.Contains(y, "NPM_TOKEN") {
		t.Errorf("o publish npm sem o NPM_TOKEN")
	}
	// M38 — o fim da tolerância cega (a lição da v1.0.0: steps verdes com
	// publish morto): nenhum continue-on-error; o preflight valida o token
	// cedo e o gate final fecha VERMELHO com FAILED_PKGS
	if strings.Contains(y, "continue-on-error") {
		t.Errorf("o release.yml não deve carregar continue-on-error cego — preflight (npm whoami) falha rápido e o gate final fecha vermelho com FAILED_PKGS")
	}
	if !strings.Contains(y, "npm whoami") {
		t.Errorf("o preflight do token (npm whoami) ausente — sem ele o ENEEDAUTH só aparece no fim, no log")
	}
	if !strings.Contains(y, "FAILED_PKGS") {
		t.Errorf("o gate vermelho (FAILED_PKGS) ausente — publishes falhos ficariam verdes em silêncio")
	}
	// a idempotência: re-run não quebra (create → upload --clobber;
	// já-publicado → skip pelo npm view)
	if !strings.Contains(y, "--clobber") {
		t.Errorf("a GitHub Release idempotente (upload --clobber) ausente — o re-run quebrava no create com a release já existente")
	}
	if !strings.Contains(y, "já publicado — skip idempotente") {
		t.Errorf("o skip idempotente dos publishes ausente — o re-run de release completa falharia em 'cannot publish over'")
	}
	// a versão fixada pela tag — resolvida de push de tag OU do dispatch
	// do workflow version (a tag é a fonte da verdade)
	if !strings.Contains(y, "inputs.tag || github.ref_name") {
		t.Errorf("a versão deveria vir da tag (inputs.tag || github.ref_name — push de tag ou dispatch do workflow version)")
	}
	if !strings.Contains(y, "${VERSION#v}") {
		t.Errorf("a versão npm deveria ser o strip do prefixo v (${VERSION#v})")
	}
}

// ---- M38 — o version.yml: o bump por Conventional Commits em TODO merge
// na main — a fonte da verdade do versionamento ----

func TestVersionWorkflowBump(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "version.yml"))
	if err != nil {
		t.Fatalf("version.yml ausente: %v", err)
	}
	y := string(data)
	// o gatilho: TODO push na main (o merge é a fonte da versão)
	if !strings.Contains(y, "branches: [main]") {
		t.Errorf("o version.yml deve disparar no push para main (o merge é a fonte da versão)")
	}
	// as regras do bump — o contrato do semver por Conventional Commits
	if !strings.Contains(y, "'^[a-z]+(\\([^)]*\\))?!:'") {
		t.Errorf("a regra de major (o ! no subject, qualquer tipo) ausente")
	}
	if !strings.Contains(y, "'^feat(\\([^)]*\\))?:'") {
		t.Errorf("a regra de minor (feat) ausente")
	}
	if !strings.Contains(y, "'^fix(\\([^)]*\\))?:'") {
		t.Errorf("a regra de patch (fix) ausente")
	}
	if !strings.Contains(y, "BREAKING CHANGE:") {
		t.Errorf("a regra de major (footer BREAKING CHANGE) ausente")
	}
	// sem commit releasável → nenhuma tag (nenhuma release fantasma de docs)
	if !strings.Contains(y, "bump=none") {
		t.Errorf("o caminho sem bump ausente — merges de docs não devem gerar release")
	}
	// a tag anotada + o dispatch explícito da release (o push de tag feito
	// com o GITHUB_TOKEN não dispara workflows — o dispatch é a exceção)
	if !strings.Contains(y, "git tag -a") {
		t.Errorf("a tag anotada ausente")
	}
	if !strings.Contains(y, "gh workflow run release.yml") {
		t.Errorf("o dispatch da release ausente — o push de tag com GITHUB_TOKEN não dispara o release.yml")
	}
}

// ---- M38 — o commit de bump versionado: o version workflow grava a versão
// nos 6 package.json e empurra `chore(release): vX.Y.Z` na main ANTES de
// taggear — a tag aponta para o commit cuja árvore carrega a versão
// publicada (checkout da tag = versão certa). O push é por DEPLOY KEY: o
// GITHUB_TOKEN builtin não pode entrar no bypass do ruleset da main. ----

func TestVersionWorkflowCommitsBump(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "version.yml"))
	if err != nil {
		t.Fatalf("version.yml ausente: %v", err)
	}
	y := string(data)
	// o commit de bump é chore — chore não é releasável, então o commit
	// próprio nunca infla o bump seguinte (sem loop de versão)
	if !strings.Contains(y, "chore(release): ${VERSION}") {
		t.Errorf("o commit de bump ausente ou com type não-chore — chore(release) não pode contar como releasável no range seguinte")
	}
	// os 6 package.json gravados (raiz + 5 de plataforma) + as optionals
	for _, dir := range []string{"'.'", "'npm/win32-x64'", "'npm/darwin-arm64'"} {
		if !strings.Contains(y, dir) {
			t.Errorf("o stamp não cobre o dir %s (os 6 package.json são bumpados juntos)", dir)
		}
	}
	if !strings.Contains(y, "optionalDependencies[name]") {
		t.Errorf("as optionalDependencies não acompanham o stamp — npm resolve optional por versão exata")
	}
	// auto-assert de sincronia antes do push (o commit não passa por PR)
	if !strings.Contains(y, "versões divergentes após o stamp") {
		t.Errorf("o auto-assert de sincronia dos 6 ausente — commit de bump sem CI precisa validar dentro do passo")
	}
	// o push direto usa a deploy key (o GITHUB_TOKEN builtin não pode
	// receber bypass de ruleset — restrição do GitHub)
	if !strings.Contains(y, "BUMP_DEPLOY_KEY") {
		t.Errorf("o secret BUMP_DEPLOY_KEY ausente — sem ele o push do commit bate no ruleset (PR + 1 aprovação)")
	}
	if !strings.Contains(y, "git@github.com:${{ github.repository }}.git") {
		t.Errorf("o push via SSH da deploy key ausente — o remote https ainda usaria o GITHUB_TOKEN builtin")
	}
	// a tag aponta para a ponta da main (o commit de bump), não para o
	// checkout do run (merge commit com os arquivos antigos)
	if !strings.Contains(y, `git tag -a "${VERSION}" origin/main`) {
		t.Errorf("a tag deve nascer no commit de bump (origin/main pós-push) — tag no HEAD do checkout deixaria a árvore da tag com a versão antiga")
	}
}

// ---- M38 — o ci.yml: os gates de PR (conventional-commits +
// release-readiness) que impedem a main de quebrar a release ----

func TestCIWorkflowReleaseGates(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("ci.yml ausente: %v", err)
	}
	y := string(data)
	for _, job := range []string{"conventional-commits:", "release-readiness:"} {
		if !strings.Contains(y, job) {
			t.Errorf("o gate %q ausente do ci.yml", job)
		}
	}
	// o gate de commits valida o padrão CC nos subjects E no título do PR
	for _, want := range []string{"feat|fix|docs", "Título do PR"} {
		if !strings.Contains(y, want) {
			t.Errorf("a validação de Conventional Commits sem %q", want)
		}
	}
	// o gate de readiness: sync de versões, pack dry-run, cross-compile
	// dos 5 alvos e o contrato do launcher
	for _, want := range []string{"npm pack --dry-run", "darwin/arm64", "bin/golmet.js"} {
		if !strings.Contains(y, want) {
			t.Errorf("o release-readiness sem %q", want)
		}
	}
}
