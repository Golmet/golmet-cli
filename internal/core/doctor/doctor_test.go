package doctor

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

const testdata = "../../testdata"

func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	err := filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func gitAvailable(t *testing.T) bool {
	t.Helper()
	return exec.Command("git", "--version").Run() == nil
}

func checkOf(t *testing.T, rep *Report, item string) *Check {
	t.Helper()
	for i := range rep.Checks {
		if rep.Checks[i].Item == item {
			return &rep.Checks[i]
		}
	}
	t.Fatalf("check %q ausente do relatório: %+v", item, rep.Checks)
	return nil
}

func makeRepoTwoCommits(t *testing.T, dir string) (string, string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	os.WriteFile(filepath.Join(dir, "src", "Agenda.cs"), []byte("v1\n"), 0o644)
	run("init", "-b", "main")
	run("add", ".")
	run("-c", "user.email=kit@golmet.dev", "-c", "user.name=golmet", "commit", "-m", "v1")
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	old := strings.TrimSpace(string(out))
	os.WriteFile(filepath.Join(dir, "src", "Agenda.cs"), []byte("v2\n"), 0o644)
	run("add", ".")
	run("-c", "user.email=kit@golmet.dev", "-c", "user.name=golmet", "commit", "-m", "v2")
	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err = cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	return old, strings.TrimSpace(string(out))
}

func TestDoctorValidWorkspace(t *testing.T) {
	tmp := t.TempDir()
	ws := filepath.Join(tmp, "ws")
	copyDir(t, filepath.Join(testdata, "valid-workspace"), ws)

	rep, err := Run(ws, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if c := checkOf(t, rep, "golmet.yaml"); c.Status != OK {
		t.Errorf("golmet.yaml = %s (%s)", c.Status, c.Detail)
	}
	// índice ausente → warn com remediação (validate cria baseline)
	if c := checkOf(t, rep, "indice"); c.Status != Warn {
		t.Errorf("indice = %s, want warn (ausente)", c.Status)
	}
	// zona do kit ausente (workspace não scaffoldado) → warn com remediação
	if c := checkOf(t, rep, "manifest"); c.Status != Warn {
		t.Errorf("manifest = %s, want warn (zona .golmet ausente)", c.Status)
	}
	// zona vazia mas presente (scaffold do init) → ok informativo
	os.MkdirAll(filepath.Join(ws, ".golmet", "skills"), 0o755)
	os.WriteFile(filepath.Join(ws, ".golmet", "manifest.json"),
		[]byte("{\"version\":\"0.2.0\",\"skills\":[]}\n"), 0o644)
	rep2, err := Run(ws, Options{})
	if err != nil {
		t.Fatalf("Run 2: %v", err)
	}
	if c := checkOf(t, rep2, "manifest"); c.Status != OK {
		t.Errorf("manifest vazio presente = %s (%s), want ok", c.Status, c.Detail)
	}
	if c := checkOf(t, rep2, "capacidades"); c.Status != OK {
		t.Errorf("capacidades = %s (%s)", c.Status, c.Detail)
	}
	// sem skill requerendo kit.jev/kit.graph → ok informativo
	if c := checkOf(t, rep, "kit.jev"); c.Status != OK {
		t.Errorf("kit.jev = %s (%s)", c.Status, c.Detail)
	}
	if c := checkOf(t, rep, "kit.graph"); c.Status != OK {
		t.Errorf("kit.graph = %s (%s)", c.Status, c.Detail)
	}
	if gitAvailable(t) {
		if c := checkOf(t, rep, "git"); c.Status != OK {
			t.Errorf("git = %s", c.Status)
		}
	}
}

func TestDoctorJevRequiredSimulatedEnv(t *testing.T) {
	tmp := t.TempDir()
	ws := filepath.Join(tmp, "ws")
	copyDir(t, filepath.Join(testdata, "valid-workspace"), ws)

	// skill requerendo kit.jev — sem key → S5 warn
	skillDir := filepath.Join(ws, ".golmet", "skills", "gm-jev")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	skill := "---\nname: gm-jev\ndescription: classificação\ndegrades:\n  jev: chat-stop\nrequires: [file, kit.jev]\n---\n# gm-jev\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skill), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TYPESAFE_API_KEY", "") // ausente — default offline

	rep, err := Run(ws, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if c := checkOf(t, rep, "kit.jev"); c.Status != Warn {
		t.Errorf("kit.jev sem key requerida = %s, want warn", c.Status)
	}
	// key presente simulada + --no-network → "smoke não executado" (o
	// default agora RODA o smoke quando a key existe)
	t.Setenv("TYPESAFE_API_KEY", "test-key-123")
	rep2, err := Run(ws, Options{NoNetwork: true})
	if err != nil {
		t.Fatalf("Run 2: %v", err)
	}
	if c := checkOf(t, rep2, "kit.jev"); c.Status != OK || !strings.Contains(c.Detail, "smoke não executado") {
		t.Errorf("kit.jev com key (--no-network) = %s (%s)", c.Status, c.Detail)
	}
}

// key + smoke OK (endpoint fake 200) → ✅ operacional — o default com key
// é o smoke (o caso do bug 404: "key presente" enganava por semanas).
func TestDoctorJevSmokeOK(t *testing.T) {
	tmp := t.TempDir()
	ws := filepath.Join(tmp, "ws")
	copyDir(t, filepath.Join(testdata, "valid-workspace"), ws)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("TYPESAFE_API_KEY", "test-key-123")
	t.Setenv("TYPESAFE_API_URL", srv.URL)

	rep, err := Run(ws, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	c := checkOf(t, rep, "kit.jev")
	if c.Status != OK || !strings.Contains(strings.ToLower(c.Detail), "operacional") {
		t.Errorf("kit.jev com smoke ok = %s (%s), want OK operacional", c.Status, c.Detail)
	}
}

// key + smoke falha de rede (endpoint inacessível) → ⚠️ NUNCA ❌ (health é
// soft) — o texto cita o endpoint.
func TestDoctorJevSmokeNetFailWarn(t *testing.T) {
	tmp := t.TempDir()
	ws := filepath.Join(tmp, "ws")
	copyDir(t, filepath.Join(testdata, "valid-workspace"), ws)

	t.Setenv("TYPESAFE_API_KEY", "test-key-123")
	t.Setenv("TYPESAFE_API_URL", "http://127.0.0.1:1") // porta morta

	rep, err := Run(ws, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	c := checkOf(t, rep, "kit.jev")
	if c.Status == Fail {
		t.Errorf("kit.jev com rede falhando NUNCA é Fail: %s (%s)", c.Status, c.Detail)
	}
	if c.Status != Warn || !strings.Contains(c.Detail, "não respondeu") {
		t.Errorf("kit.jev com rede falhando = %s (%s), want warn 'não respondeu'", c.Status, c.Detail)
	}
}

// key + smoke 401 (rejeitada) → ⚠️ com texto da rejeição (nunca ❌).
func TestDoctorJevSmokeRejectedWarn(t *testing.T) {
	tmp := t.TempDir()
	ws := filepath.Join(tmp, "ws")
	copyDir(t, filepath.Join(testdata, "valid-workspace"), ws)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("TYPESAFE_API_KEY", "test-key-123")
	t.Setenv("TYPESAFE_API_URL", srv.URL)

	rep, err := Run(ws, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	c := checkOf(t, rep, "kit.jev")
	if c.Status == Fail {
		t.Errorf("kit.jev rejeitada NUNCA é Fail: %s (%s)", c.Status, c.Detail)
	}
	if c.Status != Warn || !strings.Contains(strings.ToLower(c.Detail), "rejeitada") {
		t.Errorf("kit.jev rejeitada = %s (%s), want warn 'rejeitada'", c.Status, c.Detail)
	}
}

func TestDoctorManifestDrift(t *testing.T) {
	tmp := t.TempDir()
	ws := filepath.Join(tmp, "ws")
	copyDir(t, filepath.Join(testdata, "valid-workspace"), ws)

	skillsDir := filepath.Join(ws, ".golmet", "skills", "gm-a")
	os.MkdirAll(skillsDir, 0o755)
	os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte("---\nname: gm-a\nrequires: [file]\n---\n"), 0o644)
	// manifest lista gm-a e gm-b; gm-b não existe em disco
	os.WriteFile(filepath.Join(ws, ".golmet", "manifest.json"),
		[]byte("{\"version\":\"0.2.0\",\"skills\":[{\"name\":\"gm-a\"},{\"name\":\"gm-b\"}]}\n"), 0o644)

	rep, err := Run(ws, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	c := checkOf(t, rep, "manifest")
	if c.Status != Warn || !strings.Contains(c.Detail, "gm-b") {
		t.Errorf("manifest drift = %s (%s), want warn citando gm-b", c.Status, c.Detail)
	}

	// skill em disco sem manifest → warn
	os.Remove(filepath.Join(ws, ".golmet", "manifest.json"))
	os.MkdirAll(filepath.Join(ws, ".golmet", "skills", "gm-b"), 0o755)
	rep2, err := Run(ws, Options{})
	if err != nil {
		t.Fatalf("Run 2: %v", err)
	}
	c2 := checkOf(t, rep2, "manifest")
	if c2.Status != Warn || !strings.Contains(c2.Detail, "ausente") {
		t.Errorf("manifest ausente = %s (%s), want warn", c2.Status, c2.Detail)
	}
}

func TestDoctorIndexAgeStale(t *testing.T) {
	tmp := t.TempDir()
	ws := filepath.Join(tmp, "ws")
	copyDir(t, filepath.Join(testdata, "valid-workspace"), ws)

	// índice gerado "agora" mas prosa com mtime futuro → desatualizado
	idx := "{\"format\":1,\"generated_at\":\"" + time.Now().UTC().Format(time.RFC3339) + "\",\"workspace\":\"ws\",\"entities\":{},\"counts\":{}}\n"
	os.WriteFile(filepath.Join(ws, "golmet.index.json"), []byte(idx), 0o644)
	future := time.Now().Add(1 * time.Hour)
	reqPath := filepath.Join(ws, "REQUIREMENTS.md")
	os.Chtimes(reqPath, future, future)

	rep, err := Run(ws, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if c := checkOf(t, rep, "indice-idade"); c.Status != Warn {
		t.Errorf("indice-idade = %s, want warn (prosa mais nova que o índice)", c.Status)
	}

	// prosa mais velha que o índice → em dia
	os.Chtimes(reqPath, time.Now().Add(-1*time.Hour), time.Now().Add(-1*time.Hour))
	rep2, err := Run(ws, Options{})
	if err != nil {
		t.Fatalf("Run 2: %v", err)
	}
	if c := checkOf(t, rep2, "indice-idade"); c.Status != OK {
		t.Errorf("indice-idade = %s, want ok", c.Status)
	}
}

func TestDoctorS2WorklistRequiresGit(t *testing.T) {
	if !gitAvailable(t) {
		t.Skip("git indisponível")
	}
	tmp := t.TempDir()
	ws := filepath.Join(tmp, "ws")
	repo := filepath.Join(tmp, "MeuApp-API")
	oldHash, _ := makeRepoTwoCommits(t, repo)

	copyDir(t, filepath.Join(testdata, "valid-workspace"), ws)
	// evidência aponta para o hash do 1º commit — existe no repo mas ≠ HEAD
	err := filepath.WalkDir(ws, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		return os.WriteFile(p, []byte(strings.ReplaceAll(string(data), "a1b2c3d", oldHash[:7])), 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}

	rep, err := Run(ws, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	c := checkOf(t, rep, "worklist-s2")
	if c.Status != Warn || !strings.Contains(c.Detail, oldHash[:7]) {
		t.Errorf("worklist-s2 = %s (%s), want warn citando o hash stale", c.Status, c.Detail)
	}
}

func TestDoctorInvalidWorkspace(t *testing.T) {
	rep, err := Run(filepath.Join(t.TempDir(), "nada"), Options{})
	if err == nil {
		t.Errorf("workspace inexistente deveria falhar (exit 2 no CLI)")
	}
	if rep == nil || len(rep.Checks) == 0 {
		t.Errorf("o check falho de golmet.yaml deveria vir no relatório")
	}
	ws := t.TempDir()
	os.WriteFile(filepath.Join(ws, "golmet.yaml"), []byte("format: |\n  x\n"), 0o644)
	rep2, err2 := Run(ws, Options{})
	if err2 == nil {
		t.Errorf("golmet.yaml ilegível deveria falhar")
	}
	if c := checkOf(t, rep2, "golmet.yaml"); c.Status != Fail {
		t.Errorf("golmet.yaml ilegível = %s, want fail", c.Status)
	}
}

// ---- agregação de capacidades (S3 — apresentação do doctor) ----

const skillAskDegraded = "---\nname: %s\ndescription: usa ask com degradação\nrequires: [file, ask]\ndegrades:\n  ask: chat-stop\n---\n# %s\n"

func writeDoctorSkill(t *testing.T, ws, name, fm string) {
	t.Helper()
	dir := filepath.Join(ws, ".golmet", "skills", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(fm), 0o644); err != nil {
		t.Fatal(err)
	}
}

func capacityChecks(t *testing.T, rep *Report) []Check {
	t.Helper()
	var out []Check
	for _, c := range rep.Checks {
		if c.Item == "capacidades" {
			out = append(out, c)
		}
	}
	return out
}

// 12 skills com ask ausente (todas com degradação declarada) → 1 Check
// agregado com contagem, degradação e nomes truncados.
func TestDoctorCapabilitiesAggregatedByDegradation(t *testing.T) {
	ws := t.TempDir()
	cfg := "format: 1\n"
	os.WriteFile(filepath.Join(ws, "golmet.yaml"), []byte(cfg), 0o644)
	names := []string{}
	for i := 1; i <= 12; i++ {
		name := "gm-skill" + strings.Repeat("x", i-1)
		if i < 10 {
			name = fmt.Sprintf("gm-skill%02d", i)
		}
		names = append(names, name)
		writeDoctorSkill(t, ws, name, fmt.Sprintf(skillAskDegraded, name, name))
	}
	sort.Strings(names)

	rep, err := Run(ws, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	caps := capacityChecks(t, rep)
	if len(caps) != 1 {
		t.Fatalf("esperava 1 Check agregado, veio %d: %+v", len(caps), caps)
	}
	c := caps[0]
	if c.Status != Warn {
		t.Errorf("status = %s, want warn", c.Status)
	}
	if !strings.Contains(c.Detail, "12 skill(es) usam ask") {
		t.Errorf("contagem ausente: %s", c.Detail)
	}
	if !strings.Contains(c.Detail, "degradação declarada: chat-stop") {
		t.Errorf("degradação ausente: %s", c.Detail)
	}
	// nomes: 6 exibidos + ellipsis com o restante
	if !strings.Contains(c.Detail, "… +6") {
		t.Errorf("truncamento ausente: %s", c.Detail)
	}
	for _, n := range names[:6] {
		if !strings.Contains(c.Detail, n) {
			t.Errorf("nome %s não listado: %s", n, c.Detail)
		}
	}
	for _, n := range names[6:] {
		if strings.Contains(c.Detail, "skills: "+n+",") && !strings.Contains(c.Detail, "…") {
			t.Errorf("nomes além do truncamento vazaram: %s", c.Detail)
		}
	}
	if !strings.Contains(c.Fix, "degradam conforme declarado") {
		t.Errorf("fix agregado ausente: %s", c.Fix)
	}
}

// capacidades DIFERENTES ausentes → 1 Check por capacidade, nunca misturadas.
func TestDoctorCapabilitiesDistinct(t *testing.T) {
	ws := t.TempDir()
	os.WriteFile(filepath.Join(ws, "golmet.yaml"), []byte("format: 1\n"), 0o644)
	// grupo ask: 3 skills com degradação declarada
	for _, n := range []string{"gm-a1", "gm-a2", "gm-a3"} {
		writeDoctorSkill(t, ws, n, fmt.Sprintf(skillAskDegraded, n, n))
	}
	// grupo mcp: 2 skills exigem mcp (sem degradação declarada) → individuais
	for _, n := range []string{"gm-m1", "gm-m2"} {
		writeDoctorSkill(t, ws, n,
			"---\nname: "+n+"\nrequires: [file, mcp]\ndegrades:\n  ask: chat-stop\n---\n")
	}
	rep, err := Run(ws, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	caps := capacityChecks(t, rep)
	// 1 agregado (ask) + 2 individuais (mcp) = 3
	if len(caps) != 3 {
		t.Fatalf("esperava 3 checks de capacidades (1 agregado + 2 individuais), veio %d: %+v", len(caps), caps)
	}
	var agg *Check
	individuals := 0
	for i := range caps {
		if strings.Contains(caps[i].Detail, "usam ask") {
			agg = &caps[i]
		} else {
			individuals++
			if !strings.Contains(caps[i].Detail, "exige mcp") || !strings.Contains(caps[i].Detail, "sem degradação declarada") {
				t.Errorf("individual malformado: %+v", caps[i])
			}
		}
	}
	if agg == nil || individuals != 2 {
		t.Fatalf("agregação por capacidade não separou os grupos: %+v", caps)
	}
	if !strings.Contains(agg.Detail, "3 skill(es) usam ask") {
		t.Errorf("contagem do agregado ask errada: %s", agg.Detail)
	}
	// capacidades nunca se misturam na mesma linha
	if strings.Contains(agg.Detail, "mcp") {
		t.Errorf("agregado de ask menciona mcp: %s", agg.Detail)
	}
}

// zero capacidades ausentes → nenhuma linha de aviso (como hoje).
func TestDoctorCapabilitiesAllPresent(t *testing.T) {
	ws := t.TempDir()
	os.WriteFile(filepath.Join(ws, "golmet.yaml"), []byte("format: 1\n"), 0o644)
	writeDoctorSkill(t, ws, "gm-ok",
		"---\nname: gm-ok\nrequires: [file, shell, kit]\ndegrades:\n  ask: chat-stop\n---\n")
	rep, err := Run(ws, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	caps := capacityChecks(t, rep)
	if len(caps) != 1 || caps[0].Status != OK {
		t.Fatalf("esperava 1 check OK de capacidades: %+v", caps)
	}
}
