package update

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Golmet/golmet-cli/internal/core/doctor"
	"github.com/Golmet/golmet-cli/internal/core/indexstate"
	"github.com/Golmet/golmet-cli/internal/core/scaffold"
)

// scaffoldWS prepara um workspace válido via o scaffold do kit.
func scaffoldWS(t *testing.T) string {
	t.Helper()
	ws := filepath.Join(t.TempDir(), "ws")
	if _, err := scaffold.Run(filepath.Dir(ws), filepath.Base(ws), scaffold.Options{}); err != nil {
		t.Fatalf("scaffold: %v", err)
	}
	return ws
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// O subset BASE embutido deve conter as 16 skills gm-* com front matter
// §10 parseável pelo subset §6.1 — qualquer desvio aborta o install.
func TestEmbeddedSkillsSixteenValid(t *testing.T) {
	skills, skillErrs, err := EmbeddedSkills()
	if err != nil {
		t.Fatalf("EmbeddedSkills: %v", err)
	}
	if len(skillErrs) > 0 {
		for _, se := range skillErrs {
			t.Errorf("skill inválida: %v", se)
		}
	}
	if len(skills) != 16 {
		t.Errorf("esperava 16 skills embutidas, veio %d", len(skills))
	}
	seen := map[string]bool{}
	for _, sk := range skills {
		if !strings.HasPrefix(sk.Name, "gm-") || sk.Name != sk.Dir {
			t.Errorf("skill com name/dir divergentes: %s vs %s", sk.Name, sk.Dir)
		}
		if sk.SHA256 == "" || len(sk.Content) == 0 {
			t.Errorf("skill %s sem conteúdo/hash", sk.Name)
		}
		seen[sk.Name] = true
	}
	for _, want := range []string{"gm-charter", "gm-requirements", "gm-clarify", "gm-taskspec"} {
		if !seen[want] {
			t.Errorf("skill %s ausente do subset embutido", want)
		}
	}
}

func TestValidateFrontMatterRejectsCorrupted(t *testing.T) {
	cases := []struct {
		name    string
		content string
		wantSub string
	}{
		{"sem front matter", "# skill solta\n", "front matter ausente"},
		{"sem name", "---\ndescription: x\nrequires: [file]\n---\n", "name ausente"},
		{"name sem gm-", "---\nname: charter\ndescription: x\n---\n", "prefixo gm-"},
		{"requires escalar", "---\nname: gm-x\nrequires: file\n---\n", "array inline"},
		{"multiline rejeitado", "---\nname: gm-x\ndescription: |\n  bloco\n---\n", "multiline"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, serr := validateFrontMatter("skills/gm-x/SKILL.md", []byte(tc.content))
			if serr == nil {
				t.Fatalf("esperava SkillError contendo %q", tc.wantSub)
			}
			if !strings.Contains(serr.Problem, tc.wantSub) {
				t.Errorf("problem = %q, want contendo %q", serr.Problem, tc.wantSub)
			}
		})
	}
	if _, serr := validateFrontMatter("skills/gm-x/SKILL.md",
		[]byte("---\nname: gm-x\nrequires: [file, ask]\ndescription: ok\n---\n")); serr != nil {
		t.Errorf("front matter válido rejeitado: %v", serr)
	}
}

func TestInstallCreatesSkillsManifestAndDoctorGreen(t *testing.T) {
	ws := scaffoldWS(t)
	rep, err := Install(ws)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if rep.Total != 16 || rep.Installed != 16 || len(rep.Preserved) != 0 {
		t.Errorf("install report = %+v", rep)
	}
	if rep.KitVersion != indexstate.KitVersion {
		t.Errorf("kit version no report = %q", rep.KitVersion)
	}

	mf, err := ParseManifest(ws)
	if err != nil || mf == nil {
		t.Fatalf("manifest ausente/ilegível: %v", err)
	}
	if mf.Version != indexstate.KitVersion || mf.InstalledAt == "" || len(mf.Skills) != 16 {
		t.Errorf("manifest inválido: %+v", mf)
	}
	for _, en := range mf.Skills {
		data, rerr := os.ReadFile(filepath.Join(ws, ".golmet", filepath.FromSlash(en.File)))
		if rerr != nil {
			t.Errorf("skill %s não instalada: %v", en.Name, rerr)
			continue
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != en.SHA256 {
			t.Errorf("hash do manifest diverge do conteúdo instalado: %s", en.Name)
		}
	}

	// doctor fica verde na zona do kit pós-update (antes: warn "não instalado")
	drep, derr := doctor.Run(ws, doctor.Options{})
	if derr != nil {
		t.Fatalf("doctor: %v", derr)
	}
	var manifestCheck *doctor.Check
	for i := range drep.Checks {
		if drep.Checks[i].Item == "manifest" {
			manifestCheck = &drep.Checks[i]
		}
	}
	if manifestCheck == nil || manifestCheck.Status != doctor.OK {
		t.Errorf("doctor manifest pós-update = %+v, want ok", manifestCheck)
	}
}

func TestInstallIdempotent(t *testing.T) {
	ws := scaffoldWS(t)
	if _, err := Install(ws); err != nil {
		t.Fatalf("install 1: %v", err)
	}
	before := readFile(t, filepath.Join(ws, ".golmet", "skills", "gm-charter", "SKILL.md"))
	rep2, err := Install(ws)
	if err != nil {
		t.Fatalf("install 2: %v", err)
	}
	// byte-idêntico = no-op: re-install idempotente não conta escrita
	if rep2.Installed != 0 || len(rep2.Preserved) != 0 {
		t.Errorf("re-install deveria ser idempotente (0 escritas): %+v", rep2)
	}
	after := readFile(t, filepath.Join(ws, ".golmet", "skills", "gm-charter", "SKILL.md"))
	if before != after {
		t.Errorf("re-install alterou o conteúdo da skill (deveria ser idempotente)")
	}
}

func TestInstallPreservesLocallyModified(t *testing.T) {
	ws := scaffoldWS(t)
	if _, err := Install(ws); err != nil {
		t.Fatalf("install 1: %v", err)
	}
	target := filepath.Join(ws, ".golmet", "skills", "gm-charter", "SKILL.md")
	modified := "---\nname: gm-charter\ndescription: adaptação local\nrequires: [file]\n---\n# local\n"
	if err := os.WriteFile(target, []byte(modified), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := Install(ws)
	if err != nil {
		t.Fatalf("install 2: %v", err)
	}
	if len(rep.Preserved) != 1 || rep.Preserved[0] != "gm-charter" {
		t.Errorf("skill modificada deveria ser preservada (DR-008): %+v", rep.Preserved)
	}
	// as outras 15 estão byte-idênticas — no-op, nada a (re)escrever
	if rep.Installed != 0 {
		t.Errorf("skills idênticas não deveriam contar escrita: installed=%d", rep.Installed)
	}
	if got := readFile(t, target); got != modified {
		t.Errorf("modificação local foi perdida no re-install")
	}
}

func TestCheckSemEscrita(t *testing.T) {
	ws := scaffoldWS(t)

	// sem manifest com skills → NotInstalled, nada escrito
	rep, err := Check(ws)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !rep.NotInstalled {
		t.Errorf("sem manifest deveria reportar NotInstalled: %+v", rep)
	}
	// scaffold cria a zona vazia com manifest vazio — o assertion correto
	// é: nenhum SKILL.md instalado e manifest intocado
	matches, _ := filepath.Glob(filepath.Join(ws, ".golmet", "skills", "*", "SKILL.md"))
	if len(matches) != 0 {
		t.Errorf("--check não deveria instalar skills: %v", matches)
	}
	manifestBefore := readFile(t, filepath.Join(ws, ".golmet", "manifest.json"))

	// instalado → conforme
	if _, err := Install(ws); err != nil {
		t.Fatalf("install: %v", err)
	}
	if len(rep.Faltantes) != 0 || len(rep.Extras) != 0 {
		t.Errorf("check pós-scaffold deveria apenas reportar NotInstalled: %+v", rep)
	}
	// o manifest PRÉ-install (vazio) não deveria ter sido tocado pelo check
	if got := readFile(t, filepath.Join(ws, ".golmet", "manifest.json")); got == manifestBefore {
		t.Errorf("install deveria ter reescrito o manifest com as skills")
	}
	sentinel := readFile(t, filepath.Join(ws, ".golmet", "manifest.json"))
	rep2, err := Check(ws)
	if err != nil {
		t.Fatalf("check 2: %v", err)
	}
	if rep2.NotInstalled || len(rep2.Faltantes) > 0 || len(rep2.Extras) > 0 || len(rep2.Modificados) > 0 {
		t.Errorf("pós-install deveria estar conforme: %+v", rep2)
	}
	if got := readFile(t, filepath.Join(ws, ".golmet", "manifest.json")); got != sentinel {
		t.Errorf("--check alterou o manifest (nunca deve escrever)")
	}

	// skill modificada + faltante + extra + drift de versão
	os.WriteFile(filepath.Join(ws, ".golmet", "skills", "gm-charter", "SKILL.md"),
		[]byte("---\nname: gm-charter\ndescription: local\n---\n"), 0o644)
	os.Remove(filepath.Join(ws, ".golmet", "skills", "gm-clarify", "SKILL.md"))
	rep3, err := Check(ws)
	if err != nil {
		t.Fatalf("check 3: %v", err)
	}
	if len(rep3.Modificados) != 1 || rep3.Modificados[0] != "gm-charter" {
		t.Errorf("modificados = %+v", rep3.Modificados)
	}
	if len(rep3.Faltantes) != 1 || rep3.Faltantes[0] != "gm-clarify" {
		t.Errorf("faltantes = %+v", rep3.Faltantes)
	}
}

func TestCheckExtrasForeignSkill(t *testing.T) {
	ws := scaffoldWS(t)
	if _, err := Install(ws); err != nil {
		t.Fatalf("install: %v", err)
	}
	// skill estranha no disco + entrada no manifest (hand-edited — o kit
	// é o único escritor; o check REPORTA, não conserta)
	mfPath := filepath.Join(ws, ".golmet", "manifest.json")
	mf := readFile(t, mfPath)
	mf = strings.Replace(mf, `"skills": [`, `"skills": [
    {
      "name": "gm-estranha",
      "file": "skills/gm-estranha/SKILL.md",
      "sha256": "x"
    },`, 1)
	os.WriteFile(mfPath, []byte(mf), 0o644)
	os.MkdirAll(filepath.Join(ws, ".golmet", "skills", "gm-estranha"), 0o755)
	os.WriteFile(filepath.Join(ws, ".golmet", "skills", "gm-estranha", "SKILL.md"), []byte("x"), 0o644)

	rep, err := Check(ws)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if len(rep.Extras) != 1 || rep.Extras[0] != "gm-estranha" {
		t.Errorf("extras = %+v, want [gm-estranha]", rep.Extras)
	}
}
