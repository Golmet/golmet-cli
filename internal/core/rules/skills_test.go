package rules

import (
	"os"
	"path/filepath"
	"testing"
)

func writeSkill(t *testing.T, root, name, frontMatter string) {
	t.Helper()
	dir := filepath.Join(root, ".golmet", "skills", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(frontMatter+"\n# corpo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParseDegrades(t *testing.T) {
	fm := `---
name: gm-x
description: teste
requires: [file, ask]
degrades:
  ask: chat-stop
kit:
  checkpoints: post-write validate
---
# corpo

  indented: não é do bloco — corpo
`
	sk := readSkillsWithContent(t, "gm-x", fm)
	if got := sk.DegradeFor("ask"); got != "chat-stop" {
		t.Errorf("DegradeFor(ask) = %q, want chat-stop", got)
	}
	if got := sk.DegradeFor("mcp"); got != "" {
		t.Errorf("DegradeFor(mcp) = %q, want vazio", got)
	}
	// corpo com linhas indentadas NÃO vaza para o bloco (o kit: checkpoints
	// de 1º nível fecha o bloco; conteúdo indentado do corpo não é lido)
	if _, ok := sk.Degrades["text"]; ok {
		t.Errorf("degrades vazou para o corpo do markdown: %+v", sk.Degrades)
	}
	if _, ok := sk.Degrades["checkpoints"]; ok {
		t.Errorf("kit.checkpoints não é degradação: %+v", sk.Degrades)
	}
}

func TestParseDegradesMultipleAndEmpty(t *testing.T) {
	fmMulti := `---
name: gm-y
requires: [file, ask, subagent]
degrades:
  ask: chat-stop
  subagent: inline sequencial
---
`
	sk := readSkillsWithContent(t, "gm-y", fmMulti)
	if sk.DegradeFor("ask") != "chat-stop" || sk.DegradeFor("subagent") != "inline sequencial" {
		t.Errorf("múltiplas degradações: %+v", sk.Degrades)
	}
	// sem bloco degrades → nil
	fmNo := "---\nname: gm-z\nrequires: [file]\n---\n"
	skz := readSkillsWithContent(t, "gm-z", fmNo)
	if skz.Degrades != nil {
		t.Errorf("skill sem degrades deveria ter nil: %+v", skz.Degrades)
	}
}

func readSkillsWithContent(t *testing.T, name, fm string) SkillInfo {
	t.Helper()
	ws := t.TempDir()
	writeSkill(t, ws, name, fm)
	sks := ReadSkills(ws)
	if len(sks) != 1 {
		t.Fatalf("ReadSkills = %d skills, want 1", len(sks))
	}
	return sks[0]
}

func TestReadSkillsRequiresErrAndDegrades(t *testing.T) {
	ws := t.TempDir()
	writeSkill(t, ws, "gm-quebrada", "---\nname: gm-quebrada\ndescription: sem requires\n---\n")
	sks := ReadSkills(ws)
	if len(sks) != 1 || !sks[0].RequiresErr {
		t.Errorf("skill sem requires deveria marcar RequiresErr: %+v", sks)
	}
}
