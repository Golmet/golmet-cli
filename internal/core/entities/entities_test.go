package entities

import (
	"strings"
	"testing"

	"github.com/Golmet/golmet-cli/internal/adapters/fsutil"
)

func docFrom(content string) *fsutil.FileDoc {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(content, "\n")
	return &fsutil.FileDoc{Rel: "x.md", Abs: "x.md", Lines: lines, Plain: fsutil.BlankFenced(lines)}
}

func TestParseDirectiveForms(t *testing.T) {
	cases := []struct {
		line, state, dest, reason string
	}{
		{"> estado: active", "active", "", ""},
		{"> estado: superseded → RN-142", "superseded", "RN-142", ""},
		{"> estado: resolved -> CR-013", "resolved", "CR-013", ""},
		{"> estado: stale · reason: CR-021", "stale", "", "CR-021"},
	}
	for _, tc := range cases {
		d := ParseDirective(tc.line, 1)
		if d == nil {
			t.Fatalf("linha %q não parseou", tc.line)
		}
		if d.State != tc.state || d.Dest != tc.dest || d.Reason != tc.reason {
			t.Errorf("linha %q → state=%q dest=%q reason=%q (quer %q/%q/%q)",
				tc.line, d.State, d.Dest, d.Reason, tc.state, tc.dest, tc.reason)
		}
	}
	if ParseDirective("linha comum", 1) != nil {
		t.Errorf("linha comum parseou como diretiva")
	}
}

func TestParseSections(t *testing.T) {
	content := "# Título\n\n## RN-001 — Primeira\n> estado: active\nevidência `[a:1 @a1b2c3d]`\n\n### subseção\ntexto\n\n## PEND-001 — Pendência\n> estado: open\n\n## RN-002 — Segunda\n"
	doc := docFrom(content)
	secs := ParseSections("x.md", doc.Plain, doc.Lines, func(id string, lvl int) bool {
		return lvl == 2 && strings.HasPrefix(id, "RN-")
	})
	if len(secs) != 2 {
		t.Fatalf("queria 2 seções RN, veio %d", len(secs))
	}
	s := secs[0]
	if s.ID != "RN-001" || s.Title != "Primeira" || s.Line != 3 {
		t.Errorf("seção 1 = %+v", s)
	}
	if s.State == nil || s.State.State != "active" {
		t.Errorf("diretiva = %+v", s.State)
	}
	// subseção ### pertence ao corpo da RN-001
	joined := strings.Join(s.Lines, "\n")
	if !strings.Contains(joined, "subseção") {
		t.Errorf("corpo não inclui ###: %q", joined)
	}
	// a seção termina antes do próximo ## (PEND não vaza)
	if strings.Contains(joined, "PEND-001") {
		t.Errorf("vazamento de seção: %q", joined)
	}
}

func TestHeadingInFenceIgnored(t *testing.T) {
	content := "## RN-001 — Real\n```\n## RN-999 — Dentro de fence\n```\n"
	doc := docFrom(content)
	secs := ParseSections("x.md", doc.Plain, doc.Lines, func(id string, lvl int) bool {
		return strings.HasPrefix(id, "RN-")
	})
	if len(secs) != 1 || secs[0].ID != "RN-001" {
		t.Errorf("heading em fence virou seção: %+v", secs)
	}
}

func TestParseCharterRepos(t *testing.T) {
	content := "---\nstatus: confirmed\n---\n# Charter\n\n## 1. Repos\n| Repo | Branch | HEAD |\n|---|---|---|\n| ../repo-a | main | a1b2c3d |\n| ../repo-b | dev | deadbee1 |\n\n## 2. Escopo\ntexto\n"
	root := `C:\ws`
	c := ParseCharter(docFrom(content), root)
	if !c.HasFM || c.FM["status"].Str != "confirmed" {
		t.Errorf("FM = %+v", c.FM)
	}
	if len(c.Repos) != 2 {
		t.Fatalf("repos = %+v", c.Repos)
	}
	r := c.Repos[0]
	if r.Raw != "../repo-a" || r.Branch != "main" || r.Head != "a1b2c3d" {
		t.Errorf("repo 1 = %+v", r)
	}
	if r.Abs == "" || r.Abs == root {
		t.Errorf("Abs não resolvido: %q", r.Abs)
	}
}

func TestParseFeatureTasks(t *testing.T) {
	content := "---\nstatus: ready\nanchor: RN-001\n---\n# Feature\n\n### RN-001-1. [Repo] Tarefa um\n> estado: done\n\n### RN-001-2. [Repo] Tarefa dois\n> estado: todo\n"
	doc := docFrom(content)
	f := ParseFeature(doc)
	if f.FM["anchor"].Str != "RN-001" || f.FM["status"].Str != "ready" {
		t.Errorf("FM = %+v", f.FM)
	}
	if len(f.Tasks) != 2 {
		t.Fatalf("tasks = %d", len(f.Tasks))
	}
	if f.Tasks[0].ID != "RN-001-1" || f.Tasks[0].State.State != "done" {
		t.Errorf("task 1 = %+v", f.Tasks[0])
	}
}

func TestParseSpec(t *testing.T) {
	content := "# RN-001 — Análise\n> estado: stale\n corpo"
	doc := docFrom(content)
	doc.Abs = `C:\ws\analysis\RN-001 — Fluxo\RN-001 — Analise-Tecnica.md`
	sp := ParseSpec(doc)
	if sp.ID != "RN-001" {
		t.Errorf("ID = %q", sp.ID)
	}
	if sp.Dir == nil || sp.Dir.State != "stale" {
		t.Errorf("Dir = %+v", sp.Dir)
	}
}

func TestIsFeatureFile(t *testing.T) {
	if !IsFeatureFile("delivery/2026-11/RN-001 — Agendamento — Feature.md") {
		t.Errorf("feature válida não detectada")
	}
	if IsFeatureFile("REQUIREMENTS.md") || IsFeatureFile("delivery/2026-11/notas.md") {
		t.Errorf("falso positivo em IsFeatureFile")
	}
}
