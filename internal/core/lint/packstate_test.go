// L10 — estado interno dos state packs: A (normas sem gatilho), B
// (taxonomia PT de volta), C (domains sem contrato), D (working copy sem
// pack.yaml = error). Cada check com finding exato e sem duplicação.
package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestL10PackState(t *testing.T) {
	home := t.TempDir()
	cfg := filepath.Join(home, ".config", "opencode")
	pack := filepath.Join(cfg, "packs", "org-pack")
	os.MkdirAll(filepath.Join(pack, "rules"), 0o755)
	os.MkdirAll(filepath.Join(pack, "accumulation", "patterns", "process"), 0o755)
	os.MkdirAll(filepath.Join(pack, "accumulation", "patterns", "processo"), 0o755) // PT de volta → B
	os.MkdirAll(filepath.Join(pack, "domains", "deploy"), 0o755)                    // sem KNOWLEDGE.md → C
	os.WriteFile(filepath.Join(pack, "pack.yaml"), []byte("id: org-pack\nversion: 1\nname: Org\nkind: state\n"), 0o644)

	// A: norma COM gatilho (ok) + SEM gatilho (warning) + gatilho fora da
	// janela de 12 linhas (warning — o gatilho tem que estar no topo)
	os.WriteFile(filepath.Join(pack, "rules", "com-gatilho.md"),
		[]byte("# Deploy\n\n> Gatilho: AO fazer deploy, leia este arquivo ANTES de agir.\n"), 0o644)
	os.WriteFile(filepath.Join(pack, "rules", "sem-gatilho.md"),
		[]byte("# Norma sem marcador\n\nTexto prescritivo direto, sem linha de acionamento.\n"), 0o644)
	os.WriteFile(filepath.Join(pack, "rules", "gatilho-tarde.md"),
		[]byte(fillLines(15)+"\nAO fazer qualquer coisa depois da janela.\n"), 0o644)

	// D: working copy com .git SEM pack.yaml (error) + dir não-pack (sem L10)
	broken := filepath.Join(cfg, "packs", "broken")
	os.MkdirAll(filepath.Join(broken, ".git"), 0o755)
	junk := filepath.Join(cfg, "packs", "junk")
	os.MkdirAll(junk, 0o755)

	rep := runLint(t, t.TempDir(), home)
	l10 := findingsOf(rep, "L10")
	want := map[string]string{ // file → severidade esperada
		"packs/org-pack/rules/sem-gatilho.md":           "warning",
		"packs/org-pack/rules/gatilho-tarde.md":         "warning",
		"packs/org-pack/accumulation/patterns/processo": "warning",
		"packs/org-pack/domains":                        "warning",
		"packs/org-pack/domains/deploy":                 "warning",
		"packs/broken":                                  "error",
	}
	if len(l10) != len(want) {
		t.Fatalf("L10 = %d findings, want %d: %+v", len(l10), len(want), l10)
	}
	for _, f := range l10 {
		sev, ok := want[f.File]
		if !ok {
			t.Errorf("L10 finding inesperado em %s: %+v", f.File, f)
			continue
		}
		if f.Severity != sev {
			t.Errorf("%s = %s, want %s", f.File, f.Severity, sev)
		}
	}
	for _, f := range l10 {
		switch {
		case f.File == "packs/broken":
			if !strings.Contains(f.Description, "fonte da verdade do pack sumiu") {
				t.Errorf("D deveria citar a fonte da verdade: %+v", f)
			}
		case f.File == "packs/org-pack/rules/sem-gatilho.md":
			if !strings.Contains(f.Fix, "gm-rule-keeper") {
				t.Errorf("A deveria remeter ao gm-rule-keeper: %+v", f)
			}
		case f.File == "packs/org-pack/accumulation/patterns/processo":
			if !strings.Contains(f.Fix, "gm-pattern-extractor") {
				t.Errorf("B deveria remeter ao gm-pattern-extractor: %+v", f)
			}
		case f.File == "packs/org-pack/domains", f.File == "packs/org-pack/domains/deploy":
			if !strings.Contains(f.Fix, "gm-domain-research") {
				t.Errorf("C deveria remeter ao gm-domain-research: %+v", f)
			}
		}
	}
	// sem duplicação: a taxonomia PT não aparece mais no L5
	for _, f := range findingsOf(rep, "L5") {
		if f.File == "packs/org-pack/accumulation/patterns/processo" {
			t.Errorf("taxonomia deveria ser exclusiva do L10: %+v", f)
		}
	}
	// dir não-pack (junk, sem .git e sem pack.yaml) não gera L10
	for _, f := range l10 {
		if strings.Contains(f.File, "junk") {
			t.Errorf("dir não-reconhecível não é do L10: %+v", f)
		}
	}

	// correção: gatilho aplicado, tipo renomeado, contrato de domains
	// completo, broken restaurado → L10 zera
	os.WriteFile(filepath.Join(pack, "rules", "sem-gatilho.md"),
		[]byte("# Norma\n\n> Gatilho: AO criar credentials, leia este arquivo antes de agir.\n"), 0o644)
	os.Remove(filepath.Join(pack, "rules", "gatilho-tarde.md"))
	os.Rename(filepath.Join(pack, "accumulation", "patterns", "processo"), filepath.Join(pack, "accumulation", "patterns", "test"))
	os.WriteFile(filepath.Join(pack, "domains", "INDEX.md"), []byte("# Índice\n"), 0o644)
	os.WriteFile(filepath.Join(pack, "domains", "deploy", "KNOWLEDGE.md"), []byte("# Deploy\n"), 0o644)
	os.WriteFile(filepath.Join(broken, "pack.yaml"), []byte("id: broken\nversion: 1\nname: Broken\nkind: state\n"), 0o644)

	rep2 := runLint(t, t.TempDir(), home)
	if l102 := findingsOf(rep2, "L10"); len(l102) != 0 {
		t.Fatalf("L10 pós-correção deveria zerar: %+v", l102)
	}
}

// fillLines — N linhas de corpo antes do gatilho (para testar a janela).
func fillLines(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		s += "linha de corpo\n"
	}
	return s
}
