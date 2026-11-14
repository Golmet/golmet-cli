package doctor

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ---- fixtures ----

// shaOf — hex sha256 (mesma função do materialize sobre o conteúdo canônico).
func shaOf(data string) string {
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

type manifestEntryFixture struct {
	SHA   string
	Layer string // core | local | pack:<nome>
}

// writeContentManifestFixture grava um manifest de conteúdo no formato que
// o materialize escreve (<root>/.golmet/manifest.json).
func writeContentManifestFixture(t *testing.T, root string, entries map[string]manifestEntryFixture) {
	t.Helper()
	type entry struct {
		File   string `json:"file"`
		SHA256 string `json:"sha256"`
		Layer  string `json:"layer"`
	}
	type manifest struct {
		Version   string  `json:"version"`
		UpdatedAt string  `json:"updated_at"`
		Files     []entry `json:"files"`
	}
	m := manifest{Version: "test", UpdatedAt: "2026-01-01T00:00:00Z"}
	for f, e := range entries {
		m.Files = append(m.Files, entry{File: f, SHA256: e.SHA, Layer: e.Layer})
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".golmet"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".golmet", "manifest.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// writePersonalFixture grava as designações pessoal-por-design
// (<root>/.golmet/personal.json).
func writePersonalFixture(t *testing.T, root string, paths []string) {
	t.Helper()
	type personal struct {
		Version int      `json:"version"`
		Paths   []string `json:"paths"`
	}
	data, err := json.Marshal(personal{Version: 1, Paths: paths})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".golmet"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".golmet", "personal.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func minWorkspace(t *testing.T) string {
	t.Helper()
	ws := filepath.Join(t.TempDir(), "ws")
	if err := os.MkdirAll(ws, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws, "golmet.yaml"), []byte("format: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return ws
}

func gitInitCommit(t *testing.T, dir string) bool {
	t.Helper()
	if !gitAvailable(t) {
		return false
	}
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-b", "main")
	run("add", "-A")
	run("-c", "user.email=kit@golmet.dev", "-c", "user.name=golmet", "commit", "-m", "config")
	return true
}

func containsPath(list []string, want string) bool {
	for _, p := range list {
		if p == want {
			return true
		}
	}
	return false
}

// ---- gerido-igual + customização + não-reconstrutível + local-removido →
// achados completos e categorizados (config não-git) ----

func TestDoctorEngineConfigCategorized(t *testing.T) {
	ws := minWorkspace(t)
	root := filepath.Join(t.TempDir(), "configroot")

	canonicalA := "---\ndescription: a\n---\nbody a\n"
	canonicalX := "---\ndescription: x\n---\nbody x\n"
	writeFile(t, root, "skills/gm-a/SKILL.md", canonicalA)             // gerido-igual
	writeFile(t, root, "agents/gm-x.md", "CUSTOM do usuário\nlinha\n") // gerido divergente
	for i := 1; i <= 3; i++ {
		writeFile(t, root, fmt.Sprintf("plugins/p%d.js", i), "plugin\n") // sem proveniência
	}
	for i := 1; i <= 12; i++ {
		writeFile(t, root, fmt.Sprintf("knowledge/bd/rn-%02d.md", i), "conhecimento\n")
	}
	writeContentManifestFixture(t, root, map[string]manifestEntryFixture{
		"skills/gm-a/SKILL.md":    {SHA: shaOf(canonicalA), Layer: "core"},
		"agents/gm-x.md":          {SHA: shaOf(canonicalX), Layer: "core"},
		"skills/gm-gone/SKILL.md": {SHA: shaOf("ausente\n"), Layer: "core"}, // ausente core → kit rematerializa, não reporta
		"rules/velho.md":          {SHA: shaOf("v\n"), Layer: "local"},      // ausente local → perdido
	})

	rep, err := Run(ws, Options{EngineConfigRoot: root})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	c := checkOf(t, rep, "engine-config")
	if c.Status != Warn {
		t.Errorf("engine-config = %s (%s), want warn", c.Status, c.Detail)
	}
	sec := rep.EngineConfig
	if sec == nil {
		t.Fatalf("seção engine_config ausente do relatório")
	}
	if sec.ManagedOK != 1 {
		t.Errorf("ManagedOK = %d, want 1", sec.ManagedOK)
	}
	if len(sec.Customized) != 1 || sec.Customized[0] != "agents/gm-x.md" {
		t.Errorf("Customized = %v, want [agents/gm-x.md]", sec.Customized)
	}
	if sec.UntrackedTotal != 15 {
		t.Errorf("UntrackedTotal = %d, want 15", sec.UntrackedTotal)
	}
	if len(sec.Untracked) != 15 {
		t.Errorf("lista untracked = %d itens, want 15 (abaixo do cap)", len(sec.Untracked))
	}
	wantDirs := map[string]int{"plugins": 3, "knowledge/bd": 12}
	if len(sec.UntrackedDirs) != 2 {
		t.Fatalf("UntrackedDirs = %+v, want 2 grupos", sec.UntrackedDirs)
	}
	for _, d := range sec.UntrackedDirs {
		if wantDirs[d.Dir] != d.Count {
			t.Errorf("grupo %q = %d, want %d", d.Dir, d.Count, wantDirs[d.Dir])
		}
	}
	if len(sec.RemovedLocal) != 1 || sec.RemovedLocal[0] != "rules/velho.md" {
		t.Errorf("RemovedLocal = %v, want [rules/velho.md]", sec.RemovedLocal)
	}
	if sec.GitRepo {
		t.Errorf("GitRepo = true, want false (config não versionado)")
	}
	if sec.AtRisk() != 17 { // 1 customização + 15 não-reconstrutíveis + 1 removido
		t.Errorf("AtRisk = %d, want 17", sec.AtRisk())
	}
	// remediação: config não versionado sugere versionar ou mover para zona/pack
	if !strings.Contains(c.Fix, "git init && git add -A && git commit") || !strings.Contains(c.Fix, "pack") {
		t.Errorf("fix sem versionamento/mova: %s", c.Fix)
	}
	// render textual: paths e resumo por diretório visíveis; core ausente NÃO vira achado
	human := rep.Human()
	if !strings.Contains(human, "customização local: agents/gm-x.md") {
		t.Errorf("human sem a customização:\n%s", human)
	}
	if !strings.Contains(human, "knowledge/bd/: 12 arquivo(s)") || !strings.Contains(human, "plugins/: 3 arquivo(s)") {
		t.Errorf("human sem o resumo por diretório:\n%s", human)
	}
	if !strings.Contains(human, "local removido (dado perdido): rules/velho.md") {
		t.Errorf("human sem o local removido:\n%s", human)
	}
	if strings.Contains(human, "gm-gone") {
		t.Errorf("core ausente foi reportado (kit rematerializa):\n%s", human)
	}
}

// ---- git repo limpo + tudo gerido-igual → sem achados ----

func TestDoctorEngineConfigCleanGitRepo(t *testing.T) {
	ws := minWorkspace(t)
	root := filepath.Join(t.TempDir(), "configroot")
	canonical := "---\ndescription: a\n---\nbody\n"
	writeFile(t, root, "skills/gm-a/SKILL.md", canonical)
	writeFile(t, root, "agents/gm-b.md", canonical)
	writeContentManifestFixture(t, root, map[string]manifestEntryFixture{
		"skills/gm-a/SKILL.md": {SHA: shaOf(canonical), Layer: "core"},
		"agents/gm-b.md":       {SHA: shaOf(canonical), Layer: "core"},
	})
	if !gitInitCommit(t, root) {
		t.Skip("git indisponível")
	}

	rep, err := Run(ws, Options{EngineConfigRoot: root})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	c := checkOf(t, rep, "engine-config")
	if c.Status != OK {
		t.Errorf("engine-config = %s (%s), want ok", c.Status, c.Detail)
	}
	sec := rep.EngineConfig
	if !sec.GitRepo || sec.GitDirty != 0 {
		t.Errorf("git = repo %v dirty %d, want repo limpo", sec.GitRepo, sec.GitDirty)
	}
	if sec.AtRisk() != 0 || sec.ManagedOK != 2 {
		t.Errorf("AtRisk = %d ManagedOK = %d, want 0/2", sec.AtRisk(), sec.ManagedOK)
	}
	if !strings.Contains(c.Detail, "cache reconstrutível") {
		t.Errorf("detail sem 'cache reconstrutível': %s", c.Detail)
	}
}

// ---- EOL/BOM-only em gerido → NÃO é customização ----

func TestDoctorEngineConfigEOLNotCustomization(t *testing.T) {
	ws := minWorkspace(t)
	root := filepath.Join(t.TempDir(), "configroot")
	canonical := "---\ndescription: a\n---\ncorpo\n"
	crlf := strings.ReplaceAll(canonical, "\n", "\r\n")
	bom := "\xef\xbb\xbf" + canonical
	writeFile(t, root, "skills/gm-a/SKILL.md", crlf)
	writeFile(t, root, "skills/gm-b/SKILL.md", bom)
	writeContentManifestFixture(t, root, map[string]manifestEntryFixture{
		"skills/gm-a/SKILL.md": {SHA: shaOf(canonical), Layer: "core"},
		"skills/gm-b/SKILL.md": {SHA: shaOf(canonical), Layer: "core"},
	})

	rep, err := Run(ws, Options{EngineConfigRoot: root})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	c := checkOf(t, rep, "engine-config")
	if c.Status != OK {
		t.Errorf("engine-config = %s (%s), want ok (EOL/BOM não é customização)", c.Status, c.Detail)
	}
	if len(rep.EngineConfig.Customized) != 0 || rep.EngineConfig.ManagedOK != 2 {
		t.Errorf("Customized = %v ManagedOK = %d, want 0/2", rep.EngineConfig.Customized, rep.EngineConfig.ManagedOK)
	}
}

// ---- exclusões (.git/.golmet) respeitadas ----

func TestDoctorEngineConfigExclusions(t *testing.T) {
	ws := minWorkspace(t)
	root := filepath.Join(t.TempDir(), "configroot")
	canonical := "---\ndescription: a\n---\nbody\n"
	writeFile(t, root, "skills/gm-a/SKILL.md", canonical)
	// arquivos dentro de .git/ e .golmet/ NUNCA são estado do usuário
	writeFile(t, root, ".git/hooks/pre-commit.sample", "sample\n")
	writeFile(t, root, ".golmet/manifest.json", "{}\n")
	writeFile(t, root, ".golmet/manifest-content.json", "{}\n")
	writeContentManifestFixture(t, root, map[string]manifestEntryFixture{
		"skills/gm-a/SKILL.md": {SHA: shaOf(canonical), Layer: "core"},
	})

	rep, err := Run(ws, Options{EngineConfigRoot: root})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	sec := rep.EngineConfig
	if sec.UntrackedTotal != 0 || len(sec.Untracked) != 0 {
		t.Errorf("untracked = %d (%v), want 0 — .git/.golmet deveriam ser skipados", sec.UntrackedTotal, sec.Untracked)
	}
	if rep.EngineConfig.ManagedOK != 1 {
		t.Errorf("ManagedOK = %d, want 1", sec.ManagedOK)
	}
}

// ---- cap da listagem de não-reconstrutíveis ----

func TestDoctorEngineConfigCapList(t *testing.T) {
	ws := minWorkspace(t)
	root := filepath.Join(t.TempDir(), "configroot")
	for i := 0; i < 25; i++ {
		writeFile(t, root, fmt.Sprintf("plugins/p%02d.js", i), "plugin\n")
	}
	rep, err := Run(ws, Options{EngineConfigRoot: root})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	c := checkOf(t, rep, "engine-config")
	if c.Status != Warn {
		t.Fatalf("engine-config = %s, want warn", c.Status)
	}
	sec := rep.EngineConfig
	if sec.UntrackedTotal != 25 {
		t.Errorf("UntrackedTotal = %d, want 25", sec.UntrackedTotal)
	}
	if len(sec.Untracked) != untrackedListCap {
		t.Errorf("lista = %d itens, want cap %d", len(sec.Untracked), untrackedListCap)
	}
	if len(sec.UntrackedDirs) != 1 || sec.UntrackedDirs[0].Count != 25 {
		t.Errorf("resumo por diretório = %+v, want plugins: 25", sec.UntrackedDirs)
	}
	if !strings.Contains(c.Detail, "plugins/ 25") {
		t.Errorf("detail sem o resumo agregado: %s", c.Detail)
	}
}

// ---- Options vazio → check não roda (compatibilidade total) ----

func TestDoctorEngineConfigSkippedWithoutRoot(t *testing.T) {
	ws := minWorkspace(t)
	rep, err := Run(ws, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, c := range rep.Checks {
		if c.Item == "engine-config" {
			t.Errorf("check engine-config rodou sem EngineConfigRoot: %+v", c)
		}
	}
	if rep.EngineConfig != nil {
		t.Errorf("seção engine_config presente sem EngineConfigRoot: %+v", rep.EngineConfig)
	}
}

// raiz inexistente → ok informativo (nada gerido, nada a perder)
func TestDoctorEngineConfigRootMissing(t *testing.T) {
	ws := minWorkspace(t)
	rep, err := Run(ws, Options{EngineConfigRoot: filepath.Join(t.TempDir(), "nada")})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	c := checkOf(t, rep, "engine-config")
	if c.Status != OK || !strings.Contains(c.Detail, "raiz do config inexistente") {
		t.Errorf("engine-config = %s (%s), want ok 'raiz inexistente'", c.Status, c.Detail)
	}
}

// manifest ilegível → warn com remediação (nunca pânico)
func TestDoctorEngineConfigManifestIlegivel(t *testing.T) {
	ws := minWorkspace(t)
	root := filepath.Join(t.TempDir(), "configroot")
	writeFile(t, root, "skills/gm-a/SKILL.md", "x\n")
	writeFile(t, root, ".golmet/manifest.json", "{ quebrado")
	rep, err := Run(ws, Options{EngineConfigRoot: root})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	c := checkOf(t, rep, "engine-config")
	if c.Status != Warn || !strings.Contains(c.Detail, "manifest de conteúdo ilegível") {
		t.Errorf("engine-config = %s (%s), want warn manifest ilegível", c.Status, c.Detail)
	}
}

// arquivos soltos na raiz do config agrupam sob (raiz)
func TestDoctorEngineConfigUntrackedRootFiles(t *testing.T) {
	ws := minWorkspace(t)
	root := filepath.Join(t.TempDir(), "configroot")
	writeFile(t, root, "settings.local.json", "{}\n")
	writeFile(t, root, "notes.md", "n\n")
	rep, err := Run(ws, Options{EngineConfigRoot: root})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	sec := rep.EngineConfig
	if len(sec.UntrackedDirs) != 1 || sec.UntrackedDirs[0].Dir != dirRootLabel || sec.UntrackedDirs[0].Count != 2 {
		t.Fatalf("UntrackedDirs = %+v, want (raiz): 2", sec.UntrackedDirs)
	}
	if !strings.Contains(rep.Human(), "(raiz): 2 arquivo(s)") {
		t.Errorf("human sem o grupo (raiz):\n%s", rep.Human())
	}
}

// ---- pessoal-por-design: personal.json isenta 2 paths exatos + 1 prefixo
// de diretório → Personal, fora do Untracked e fora do AtRisk ----

func TestDoctorEngineConfigPersonalDesignated(t *testing.T) {
	ws := minWorkspace(t)
	root := filepath.Join(t.TempDir(), "configroot")
	canonical := "---\ndescription: a\n---\nbody\n"
	writeFile(t, root, "skills/gm-a/SKILL.md", canonical)
	writeFile(t, root, "AGENTS.md", "agents com delta do operador\n")  // designação exata
	writeFile(t, root, "rules/client-profile.md", "dado do cliente\n") // designação exata
	for i := 1; i <= 3; i++ {                                          // designação por prefixo
		writeFile(t, root, fmt.Sprintf("evals/reports/r%d.md", i), "report\n")
	}
	writeFile(t, root, "plugins/nao-designado.js", "x\n") // SEM designação → untracked de verdade
	writePersonalFixture(t, root, []string{"AGENTS.md", "rules/client-profile.md", "evals/reports/"})
	writeContentManifestFixture(t, root, map[string]manifestEntryFixture{
		"skills/gm-a/SKILL.md": {SHA: shaOf(canonical), Layer: "core"},
	})

	rep, err := Run(ws, Options{EngineConfigRoot: root})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	c := checkOf(t, rep, "engine-config")
	sec := rep.EngineConfig
	if sec == nil {
		t.Fatalf("seção engine_config ausente")
	}
	if sec.PersonalTotal != 5 { // 2 exatos + 3 do prefixo
		t.Errorf("PersonalTotal = %d, want 5", sec.PersonalTotal)
	}
	if len(sec.Personal) != 5 {
		t.Fatalf("Personal = %v, want 5 itens", sec.Personal)
	}
	for _, want := range []string{"AGENTS.md", "rules/client-profile.md", "evals/reports/r1.md", "evals/reports/r2.md", "evals/reports/r3.md"} {
		if !containsPath(sec.Personal, want) {
			t.Errorf("Personal sem %q: %v", want, sec.Personal)
		}
	}
	if sec.UntrackedTotal != 1 || sec.Untracked[0] != "plugins/nao-designado.js" {
		t.Errorf("Untracked = %d (%v), want só plugins/nao-designado.js", sec.UntrackedTotal, sec.Untracked)
	}
	if len(sec.UntrackedDirs) != 1 || sec.UntrackedDirs[0].Dir != "plugins" {
		t.Errorf("UntrackedDirs = %+v, want só plugins", sec.UntrackedDirs)
	}
	if sec.AtRisk() != 1 {
		t.Errorf("AtRisk = %d, want 1 (pessoal-por-design NÃO conta)", sec.AtRisk())
	}
	// remediação: só o AtRisk entra na contagem — pessoal-por-design não
	if !strings.Contains(c.Fix, "1 arquivo(s) não-reconstrutíveis") {
		t.Errorf("fix deveria contar só o AtRisk (1), não o pessoal: %s", c.Fix)
	}
	// summary e render carregam a parte isenta
	if !strings.Contains(c.Detail, "5 pessoal-por-design (isento)") {
		t.Errorf("detail sem a parte isenta: %s", c.Detail)
	}
	human := rep.Human()
	if !strings.Contains(human, "pessoal-por-design (isento): evals/reports/r1.md") ||
		!strings.Contains(human, "pessoal-por-design (isento): AGENTS.md") {
		t.Errorf("human sem os itens isentos:\n%s", human)
	}
	if strings.Contains(human, "AGENTS.md\n") && !strings.Contains(human, "pessoal-por-design (isento): AGENTS.md") {
		t.Errorf("AGENTS.md vazou de outra categoria:\n%s", human)
	}
}

// ---- sem personal.json → comportamento idêntico ao anterior (regressão) ----

func TestDoctorEngineConfigWithoutPersonalFile(t *testing.T) {
	ws := minWorkspace(t)
	root := filepath.Join(t.TempDir(), "configroot")
	writeFile(t, root, "plugins/a.js", "x\n")
	writeFile(t, root, "AGENTS.md", "agents\n") // sem designação → untracked normal
	rep, err := Run(ws, Options{EngineConfigRoot: root})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	sec := rep.EngineConfig
	if sec.PersonalTotal != 0 || len(sec.Personal) != 0 {
		t.Errorf("Personal = %d (%v), want 0/nil sem personal.json", sec.PersonalTotal, sec.Personal)
	}
	if sec.UntrackedTotal != 2 || !containsPath(sec.Untracked, "AGENTS.md") {
		t.Errorf("Untracked = %d (%v), want 2 com AGENTS.md incluso", sec.UntrackedTotal, sec.Untracked)
	}
	if sec.AtRisk() != 2 {
		t.Errorf("AtRisk = %d, want 2", sec.AtRisk())
	}
}

// ---- designação NÃO isenta gerido divergente (customização segue AtRisk) ----

func TestDoctorEngineConfigPersonalNotExemptManaged(t *testing.T) {
	ws := minWorkspace(t)
	root := filepath.Join(t.TempDir(), "configroot")
	canonicalX := "---\ndescription: x\n---\noriginal\n"
	writeFile(t, root, "agents/gm-x.md", "CUSTOM do usuário\n") // GERIDO divergente
	writePersonalFixture(t, root, []string{"agents/"})          // prefixo engloba o gerido
	writeContentManifestFixture(t, root, map[string]manifestEntryFixture{
		"agents/gm-x.md": {SHA: shaOf(canonicalX), Layer: "core"},
	})

	rep, err := Run(ws, Options{EngineConfigRoot: root})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	c := checkOf(t, rep, "engine-config")
	sec := rep.EngineConfig
	if sec.PersonalTotal != 0 || len(sec.Personal) != 0 {
		t.Errorf("Personal = %d (%v), want 0 — designação não isenta gerido", sec.PersonalTotal, sec.Personal)
	}
	if len(sec.Customized) != 1 || sec.Customized[0] != "agents/gm-x.md" {
		t.Errorf("Customized = %v, want [agents/gm-x.md]", sec.Customized)
	}
	if sec.AtRisk() != 1 {
		t.Errorf("AtRisk = %d, want 1 (customização em gerido continua em risco)", sec.AtRisk())
	}
	if c.Status != Warn {
		t.Errorf("engine-config = %s, want warn", c.Status)
	}
}

// ---- personal.json corrompido → Unreadable registra e a classificação segue ----

func TestDoctorEngineConfigPersonalCorrompido(t *testing.T) {
	ws := minWorkspace(t)
	root := filepath.Join(t.TempDir(), "configroot")
	writeFile(t, root, "plugins/a.js", "x\n")
	writeFile(t, root, "AGENTS.md", "agents\n") // designado, mas o personal.json está quebrado
	writeFile(t, root, ".golmet/personal.json", "{ quebrado")
	rep, err := Run(ws, Options{EngineConfigRoot: root})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	sec := rep.EngineConfig
	if !containsPath(sec.Unreadable, personalManifestRel) {
		t.Errorf("Unreadable = %v, want %q registrado", sec.Unreadable, personalManifestRel)
	}
	if sec.PersonalTotal != 0 {
		t.Errorf("PersonalTotal = %d, want 0 (sem designações legíveis)", sec.PersonalTotal)
	}
	if sec.UntrackedTotal != 2 || !containsPath(sec.Untracked, "AGENTS.md") {
		t.Errorf("Untracked = %d (%v), want classificação sem designações", sec.UntrackedTotal, sec.Untracked)
	}
}

// ---- só pessoal-por-design → ok com a parte isenta (sem ruído de risco) ----

func TestDoctorEngineConfigPersonalOnly(t *testing.T) {
	ws := minWorkspace(t)
	root := filepath.Join(t.TempDir(), "configroot")
	writeFile(t, root, "AGENTS.md", "agents\n")
	writeFile(t, root, "rules/client-profile.md", "cliente\n")
	writePersonalFixture(t, root, []string{"AGENTS.md", "rules/client-profile.md"})
	rep, err := Run(ws, Options{EngineConfigRoot: root})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	c := checkOf(t, rep, "engine-config")
	if c.Status != OK {
		t.Errorf("engine-config = %s (%s), want ok (pessoal é isento)", c.Status, c.Detail)
	}
	if !strings.Contains(c.Detail, "2 pessoal-por-design (isento)") {
		t.Errorf("detail sem a parte isenta: %s", c.Detail)
	}
	if rep.EngineConfig.AtRisk() != 0 {
		t.Errorf("AtRisk = %d, want 0", rep.EngineConfig.AtRisk())
	}
	if !strings.Contains(rep.Human(), "pessoal-por-design (isento): rules/client-profile.md") {
		t.Errorf("human sem o item isento:\n%s", rep.Human())
	}
}

// ---- state pack working copy: versionado no repo do pack = SEGURO; só o
// WIP não-commitado conta como risco (eixo IMP — dado acumulado pelo
// operador vive em packs/<id>/ do config do engine) ----

func TestDoctorEngineConfigStatePackWorkingCopy(t *testing.T) {
	ws := minWorkspace(t)
	root := filepath.Join(t.TempDir(), "configroot")
	writeFile(t, root, "plugins/loose.js", "solto sem proveniência\n") // ruído fora do pack

	// working copy de state pack: conteúdo commitado + 1 arquivo em WIP
	packDir := filepath.Join(root, "packs", "techminds")
	writeFile(t, packDir, "pack.yaml", "id: techminds\nversion: 1.0.0\nname: TechMinds pack\nkind: state\n")
	writeFile(t, packDir, "skills/gm-x/SKILL.md", "skill do pack\n")
	writeFile(t, packDir, "knowledge/banco/indices.md", "conhecimento\n")
	if !gitInitCommit(t, packDir) {
		t.Skip("git indisponível")
	}
	writeFile(t, packDir, "accumulation/patterns/wip.md", "acumulação não commitada\n")

	rep, err := Run(ws, Options{EngineConfigRoot: root})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	c := checkOf(t, rep, "engine-config")
	if c.Status != Warn {
		t.Errorf("engine-config = %s (%s), want warn (há WIP em risco)", c.Status, c.Detail)
	}
	sec := rep.EngineConfig
	if sec == nil {
		t.Fatalf("seção engine_config ausente")
	}
	if sec.PackVersioned != 3 { // pack.yaml + SKILL.md + indices.md
		t.Errorf("PackVersioned = %d, want 3", sec.PackVersioned)
	}
	if sec.PackDirty != 1 || len(sec.PackDirtyDirs) != 1 || sec.PackDirtyDirs[0] != "packs/techminds" {
		t.Errorf("PackDirty = %d (%v), want 1 ([packs/techminds])", sec.PackDirty, sec.PackDirtyDirs)
	}
	if sec.UntrackedTotal != 1 { // só plugins/loose.js — nada do pack vira untracked
		t.Errorf("UntrackedTotal = %d, want 1 (working copy do pack é seguro)", sec.UntrackedTotal)
	}
	if sec.AtRisk() != 2 { // 1 não-reconstrutível + 1 WIP do state pack
		t.Errorf("AtRisk = %d, want 2", sec.AtRisk())
	}
	if !strings.Contains(c.Fix, "pack publish") {
		t.Errorf("fix sem a remediação do publish: %s", c.Fix)
	}
	human := rep.Human()
	if !strings.Contains(human, "versionado no repo do pack (working copy): 3 arquivo(s)") {
		t.Errorf("human sem o working copy seguro:\n%s", human)
	}
	if !strings.Contains(human, "state pack com mudanças não-commitadas: packs/techminds") {
		t.Errorf("human sem o WIP do state pack:\n%s", human)
	}
}

// state pack LIMPO (tudo commitado) não gera achado nem ruído
func TestDoctorEngineConfigStatePackClean(t *testing.T) {
	ws := minWorkspace(t)
	root := filepath.Join(t.TempDir(), "configroot")
	packDir := filepath.Join(root, "packs", "techminds")
	writeFile(t, packDir, "pack.yaml", "id: techminds\nversion: 1.0.0\nname: TechMinds pack\nkind: state\n")
	writeFile(t, packDir, "skills/gm-x/SKILL.md", "skill do pack\n")
	if !gitInitCommit(t, packDir) {
		t.Skip("git indisponível")
	}
	rep, err := Run(ws, Options{EngineConfigRoot: root})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	c := checkOf(t, rep, "engine-config")
	if c.Status != OK {
		t.Errorf("engine-config = %s (%s), want ok (working copy commitado é seguro)", c.Status, c.Detail)
	}
	if rep.EngineConfig.AtRisk() != 0 {
		t.Errorf("AtRisk = %d, want 0", rep.EngineConfig.AtRisk())
	}
	if !strings.Contains(c.Detail, "state pack (seguro)") {
		t.Errorf("detail sem o working copy seguro: %s", c.Detail)
	}
}
