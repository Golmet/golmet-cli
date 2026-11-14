package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Golmet/golmet-cli/internal/core/update"
)

// Regra EOL-tolerante no BASE do engine: skills com o mesmo texto em CRLF
// no disco são CURADAS (rematerializadas em LF) contando como escrita —
// nunca "modificadas localmente".
func crlfify(b []byte) []byte { return []byte(strings.ReplaceAll(string(b), "\n", "\r\n")) }

// (c) writeEngineSkills: BASE 16 com gm-charter pré-semeada em CRLF →
// escrita (cura), disco LF; skill realmente modificada segue preservada.
func TestWriteEngineSkillsCuresEOL(t *testing.T) {
	noEngineCLI(t)
	skills, skillErrs, err := update.EmbeddedSkills()
	if err != nil || len(skillErrs) > 0 {
		t.Fatalf("EmbeddedSkills: %v %v", err, skillErrs)
	}
	skillsDir := filepath.Join(newHome(t), ".claude", "skills")
	// semeia TODAS em CRLF (mesmo texto)
	for _, sk := range skills {
		dir := filepath.Join(skillsDir, sk.Dir)
		os.MkdirAll(dir, 0o755)
		os.WriteFile(filepath.Join(dir, "SKILL.md"), crlfify(sk.Content), 0o644)
	}
	written, preserved, err := writeEngineSkills(skillsDir, skills, false)
	if err != nil {
		t.Fatalf("writeEngineSkills: %v", err)
	}
	if len(preserved) != 0 {
		t.Errorf("EOL-only não é customização — preserved deveria ser vazio: %v", preserved)
	}
	if len(written) != len(skills) {
		t.Fatalf("cura deveria escrever todas (%d), written=%d", len(skills), len(written))
	}
	for _, sk := range skills {
		data, rerr := os.ReadFile(filepath.Join(skillsDir, sk.Dir, "SKILL.md"))
		if rerr != nil {
			t.Fatalf("skill curada ausente: %v", rerr)
		}
		if strings.Contains(string(data), "\r") {
			t.Errorf("%s deveria ter sido curada para LF", sk.Dir)
		}
	}
	// regressão: customização REAL continua preservada sem --force
	charter := filepath.Join(skillsDir, "gm-charter", "SKILL.md")
	os.WriteFile(charter, []byte("---\nname: gm-charter\n---\n# adaptação local real\n"), 0o644)
	written2, preserved2, err := writeEngineSkills(skillsDir, skills, false)
	if err != nil {
		t.Fatalf("writeEngineSkills 2: %v", err)
	}
	if len(preserved2) != 1 || preserved2[0] != "gm-charter" {
		t.Errorf("customização real deveria preservar gm-charter: %v", preserved2)
	}
	for _, w := range written2 {
		if w == "gm-charter" {
			t.Errorf("customização real não deveria ser reescrita sem --force")
		}
	}
}

// (e) e2e do install em home com as 16 skills CRLF pré-semeadas →
// 0 "modificadas localmente", tudo escrito e curado para LF.
func TestRunEOLSeededSkillsAllWritten(t *testing.T) {
	noEngineCLI(t)
	home := newHome(t)
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	skills, skillErrs, err := update.EmbeddedSkills()
	if err != nil || len(skillErrs) > 0 {
		t.Fatalf("EmbeddedSkills: %v %v", err, skillErrs)
	}
	for _, sk := range skills {
		dir := filepath.Join(home, ".claude", "skills", sk.Dir)
		os.MkdirAll(dir, 0o755)
		os.WriteFile(filepath.Join(dir, "SKILL.md"), crlfify(sk.Content), 0o644)
	}

	res, err := Run(Options{EngineID: "claude-code", MCPs: []string{"golmet"}, Home: home})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Report.SkillsPreserved) != 0 {
		t.Errorf("EOL-only não deveria reportar modificadas localmente: %v", res.Report.SkillsPreserved)
	}
	if len(res.Report.SkillsWritten) != 16 {
		t.Fatalf("skills escritas = %d, want 16 (cura EOL)", len(res.Report.SkillsWritten))
	}
	data, rerr := os.ReadFile(filepath.Join(home, ".claude", "skills", "gm-charter", "SKILL.md"))
	if rerr != nil {
		t.Fatal(rerr)
	}
	if strings.Contains(string(data), "\r") {
		t.Errorf("skill deveria estar em LF após a cura")
	}

	// segunda execução: idempotente — 0 escritas, 0 preservadas
	res2, err := Run(Options{EngineID: "claude-code", MCPs: []string{"golmet"}, Home: home})
	if err != nil {
		t.Fatalf("Run 2: %v", err)
	}
	if len(res2.Report.SkillsWritten) != 0 || len(res2.Report.SkillsPreserved) != 0 {
		t.Errorf("re-run deveria ser idempotente: written=%v preserved=%v", res2.Report.SkillsWritten, res2.Report.SkillsPreserved)
	}
}
