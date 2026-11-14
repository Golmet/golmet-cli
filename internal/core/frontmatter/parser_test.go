package frontmatter

import (
	"strings"
	"testing"
)

func TestExtractValid(t *testing.T) {
	in := "---\nstatus: confirmed\nanchor: RN-001\nrequires: [file, ask, kit]\nquoted: \"draft 1\"\ncomments: draft # nota\n---\n# Corpo\n"
	doc, has, err := Extract(in)
	if err != nil || !has {
		t.Fatalf("Extract falhou: has=%v err=%v", has, err)
	}
	if got := doc.Fields["status"].Str; got != "confirmed" {
		t.Errorf("status = %q", got)
	}
	if got := doc.Fields["anchor"].Str; got != "RN-001" {
		t.Errorf("anchor = %q", got)
	}
	arr := doc.Fields["requires"]
	if !arr.IsArr || strings.Join(arr.Arr, ",") != "file,ask,kit" {
		t.Errorf("requires = %+v", arr)
	}
	if got := doc.Fields["quoted"].Str; got != "draft 1" {
		t.Errorf("quoted = %q", got)
	}
	if got := doc.Fields["comments"].Str; got != "draft" {
		t.Errorf("comentário não ignorado: %q", got)
	}
	if !strings.HasPrefix(doc.Body, "# Corpo") {
		t.Errorf("body = %q", doc.Body)
	}
	if doc.KeyLines["status"] != 2 {
		t.Errorf("KeyLines[status] = %d", doc.KeyLines["status"])
	}
}

func TestExtractMapOneLevel(t *testing.T) {
	in := "---\ndegrades:\n  ask: chat-stop\nname: gm-x\n---\n"
	doc, has, err := Extract(in)
	if err != nil || !has {
		t.Fatalf("Extract falhou: %v", err)
	}
	m := doc.Fields["degrades"]
	if !m.IsMap || m.Map["ask"] != "chat-stop" {
		t.Errorf("degrades = %+v", m)
	}
}

func TestNoFrontMatter(t *testing.T) {
	_, has, err := Extract("# só corpo\n")
	if has || err != nil {
		t.Errorf("has=%v err=%v", has, err)
	}
}

func TestExtractErrors(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"sem fechamento", "---\nstatus: draft\n", "fechamento"},
		{"multiline |", "---\nstatus: |\n  draft\n---\n", "multiline"},
		{"multiline >", "---\ndescription: >\n  texto\n---\n", "multiline"},
		{"anchor &", "---\nstatus: &x draft\n---\n", "anchor"},
		{"alias *", "---\nstatus: *x\n---\n", "alias"},
		{"tag !!", "---\nstatus: !!str draft\n---\n", "tag"},
		{"aspas abertas", "---\nstatus: \"draft\n---\n", "aspas"},
		{"array aberto", "---\ntags: [a, b\n---\n", "não fechado"},
		{"aninhamento 2 níveis", "---\na:\n  b:\n    c: 1\n---\n", "1 nível"},
		{"chave duplicada", "---\nstatus: draft\nstatus: done\n---\n", "duplicada"},
		{"linha sem chave", "---\nsó texto\n---\n", "chave: valor"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, has, err := Extract(tc.in)
			if !has {
				t.Fatalf("esperava FM detectado")
			}
			if err == nil {
				t.Fatalf("esperava erro, veio nil")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("erro %q não contém %q", err.Error(), tc.want)
			}
		})
	}
}

func TestParseBlock(t *testing.T) {
	fields, err := ParseBlock("format: 1\ncontainer: false\nproducts: [A, B]\n# comentário solto\n")
	if err != nil {
		t.Fatalf("ParseBlock: %v", err)
	}
	if fields["format"].Str != "1" {
		t.Errorf("format = %q", fields["format"].Str)
	}
	if !fields["products"].IsArr || len(fields["products"].Arr) != 2 {
		t.Errorf("products = %+v", fields["products"])
	}
}
