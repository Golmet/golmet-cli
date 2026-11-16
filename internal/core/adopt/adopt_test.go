package adopt

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Golmet/golmet-cli/internal/core/indexstate"
)

// ---- helpers ----

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

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// snapshot captura conteúdo+mtime de todos os arquivos do workspace
// (sentinela de zero-escrita e de prosa intocada).
func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		info, _ := d.Info()
		out[strings.ReplaceAll(p, root, "")] = string(data) + "\x00" + info.ModTime().String()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

const reqLegacy = `# Requisitos — Legado

## RN-001 — Pedidos
> estado: active
Sem evidência canônica aqui.

## RN-002 — Pagamentos
Evidência legada: [src/pay.cs:10]

## RN-003 — Estoque A
primeira ocorrência

## RN-003 — Estoque B
duplicada

## RN-004 — Entrega
> estado: active
Com evidência: [src/d.cs:5 @a1b2c3d]

## PEND-001 — Frete
`

const charterLegacy = `# Charter — Legado

## 1. Repos
| Repo | Branch | HEAD |
|---|---|---|
| ../Legado-API | main | a1b2c3d |
`

// legacyWorkspace monta um workspace pré-FORMAT com todos os achados de
// julgamento do enunciado: charter sem FM, evidência sem hash, RN active
// sem evidência, ID duplicado, H7 (hash sem repo), layouts legados.
func legacyWorkspace(t *testing.T) string {
	t.Helper()
	ws := t.TempDir()
	writeFile(t, ws, "PROJECT_CHARTER.md", charterLegacy)
	writeFile(t, ws, "REQUIREMENTS.md", reqLegacy)
	writeFile(t, ws, "TASKS.md", "# Tasks legadas\n")
	writeFile(t, ws, "delivery/2026-11/RN-001 — Pedidos/RN-001 — Pedidos - Feature.md", "# Feature legada\n")
	writeFile(t, ws, "delivery/2026-11/RN-001 — Pedidos/RN-001 — Analise-Tecnica.md", "# Análise legada\n")
	writeFile(t, ws, "delivery/RN-009 — Avulso - Feature.md", "# feature na raiz do delivery\n")
	writeFile(t, ws, "docs/projects/Legado/ANTIGO.md", "# doc pré-FORMAT\n")
	return ws
}

func hasFinding(xs []Finding, sub string) bool {
	for _, f := range xs {
		if strings.Contains(f.Item+f.Detail+f.Fix+f.Rule, sub) {
			return true
		}
	}
	return false
}

// ---- testes ----

// Passo 1: leitura PURA — nenhum byte e nenhum mtime mudam.
func TestDryRunZeroWrites(t *testing.T) {
	ws := legacyWorkspace(t)
	before := snapshot(t, ws)

	rep, err := Run(ws, Options{})
	if err != nil {
		t.Fatalf("Run dry-run: %v", err)
	}
	if rep.Mode != "dry-run" {
		t.Errorf("mode = %q, want dry-run", rep.Mode)
	}
	after := snapshot(t, ws)
	for p, beforeEntry := range before {
		afterEntry, ok := after[p]
		if !ok {
			t.Fatalf("arquivo %s sumiu no dry-run", p)
		}
		if beforeEntry != afterEntry {
			t.Errorf("dry-run alterou %s (conteúdo/mtime)", p)
		}
	}
	if len(after) != len(before) {
		t.Errorf("dry-run criou arquivos: antes %d, depois %d", len(before), len(after))
	}
	if _, err := os.Stat(filepath.Join(ws, "golmet.yaml")); !os.IsNotExist(err) {
		t.Errorf("dry-run criou golmet.yaml")
	}
}

// Itens do relatório do passo 1, mapeados ao enunciado.
func TestDryRunReportItems(t *testing.T) {
	ws := legacyWorkspace(t)
	rep, err := Run(ws, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// charter ausente? não — sem front matter → kit resolve (prefixa)
	if !hasFinding(rep.KitResolve, "prefixar front matter do charter") {
		t.Errorf("kit_resolve deveria propor prefixar o front matter do charter: %+v", rep.KitResolve)
	}
	// evidência sem hash → julgamento (re-pin é do agente)
	if !hasFinding(rep.Julgamento, "evidência sem hash [src/pay.cs:10]") {
		t.Errorf("julgamento deveria listar a evidência sem hash: %+v", rep.Julgamento)
	}
	// RN active sem evidência parseável (H4)
	if !hasFinding(rep.Julgamento, "RN-001 active sem evidência parseável") {
		t.Errorf("julgamento deveria listar RN-001 (H4): %+v", rep.Julgamento)
	}
	// ID duplicado (H2)
	if !hasFinding(rep.Julgamento, "ID duplicado: RN-003") {
		t.Errorf("julgamento deveria listar RN-003 duplicado: %+v", rep.Julgamento)
	}
	// evidência com hash fora do charter §1 (H7) — repo ../Legado-API não existe
	if !hasFinding(rep.Julgamento, "@a1b2c3d") {
		t.Errorf("julgamento deveria listar a evidência H7 @a1b2c3d: %+v", rep.Julgamento)
	}
	// variantes legadas de layout (heurísticas)
	if !hasFinding(rep.Julgamento, "pasta própria delivery/2026-11/RN-001 — Pedidos") {
		t.Errorf("julgamento deveria detectar feature em pasta própria: %+v", rep.Julgamento)
	}
	if !hasFinding(rep.Julgamento, "docs/projects/") {
		t.Errorf("julgamento deveria detectar docs/projects/: %+v", rep.Julgamento)
	}
	if !hasFinding(rep.Julgamento, "TASKS.md") {
		t.Errorf("julgamento deveria detectar TASKS.md: %+v", rep.Julgamento)
	}
	if !hasFinding(rep.Julgamento, "raiz do delivery") {
		t.Errorf("julgamento deveria detectar RN-* flat na raiz do delivery: %+v", rep.Julgamento)
	}
	// estrutura §2 faltante → kit resolve
	if !hasFinding(rep.KitResolve, "criar golmet.yaml") {
		t.Errorf("kit_resolve deveria listar golmet.yaml: %+v", rep.KitResolve)
	}
	if !hasFinding(rep.KitResolve, "specs/") {
		t.Errorf("kit_resolve deveria listar specs/: %+v", rep.KitResolve)
	}
	if !hasFinding(rep.KitResolve, ".golmet/") {
		t.Errorf("kit_resolve deveria listar .golmet/: %+v", rep.KitResolve)
	}
	// RN-004 active COM evidência canônica não vira H4 (só H7)
	for _, f := range rep.Julgamento {
		if strings.Contains(f.Item, "RN-004 active sem evidência") {
			t.Errorf("RN-004 tem evidência canônica — não deveria gerar H4: %+v", f)
		}
	}
	// nenhuma escrita classificada como aplicada
	if len(rep.Applied) != 0 || len(rep.Worklist) != 0 {
		t.Errorf("dry-run não deveria ter applied/worklist: %+v %+v", rep.Applied, rep.Worklist)
	}
}

// Passo 4: apply cria golmet.yaml + índice + scaffold mínimo.
func TestApplyCreatesFiles(t *testing.T) {
	ws := legacyWorkspace(t)
	rep, err := Run(ws, Options{Apply: true})
	if err != nil {
		t.Fatalf("Run apply: %v", err)
	}
	if rep.Mode != "apply" {
		t.Errorf("mode = %q", rep.Mode)
	}
	if got := readFile(t, filepath.Join(ws, "golmet.yaml")); got != "format: 1\n" {
		t.Errorf("golmet.yaml = %q", got)
	}
	idx, perr := indexstate.Parse([]byte(readFile(t, filepath.Join(ws, "golmet.index.json"))))
	if perr != nil {
		t.Fatalf("índice ilegível: %v", perr)
	}
	// o índice é key-ID — a RN-003 duplicada colapsa em 1 entrada (o H2
	// reporta a duplicata; o índice nunca inventa chave)
	if idx.Counts["rn"] != 4 {
		t.Errorf("índice deveria ter 4 entradas de RN (RN-003 colapsa): %+v", idx.Counts)
	}
	for _, rel := range []string{
		"analysis", "delivery", "specs", "reverse/CANDIDATES.md",
		".golmet/manifest.json", "CHANGE_REQUESTS.md", "DECISIONS.md",
	} {
		if _, err := os.Stat(filepath.Join(ws, filepath.FromSlash(rel))); err != nil {
			t.Errorf("%s não criado no apply: %v", rel, err)
		}
	}
	if rep.Validate == nil {
		t.Fatalf("validate tolerant interno ausente do relatório")
	}
	// erros de validate NÃO falham a adoção — viram worklist (passo 6)
	if len(rep.Worklist) == 0 {
		t.Errorf("worklist deveria listar os findings do validate (H2/H4/H5…)")
	}
	if !hasFinding(rep.Worklist, "H2") {
		t.Errorf("worklist deveria conter o H2 (RN-003 duplicada): %+v", rep.Worklist)
	}
}

// item 3: front matter do charter — prefixo com --charter-status e default.
func TestApplyCharterFrontMatter(t *testing.T) {
	// (a) sem FM + --charter-status confirmed → prefixo confirmed, corpo intacto
	ws := t.TempDir()
	writeFile(t, ws, "PROJECT_CHARTER.md", charterLegacy)
	if _, err := Run(ws, Options{Apply: true, CharterStatus: "confirmed"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got := readFile(t, filepath.Join(ws, "PROJECT_CHARTER.md"))
	if !strings.HasPrefix(got, "---\nstatus: confirmed\n---\n") {
		t.Errorf("prefixo confirmed ausente:\n%s", got)
	}
	if !strings.Contains(got, "# Charter — Legado") || !strings.Contains(got, "| ../Legado-API | main | a1b2c3d |") {
		t.Errorf("corpo do charter foi alterado:\n%s", got)
	}

	// (b) default → draft
	ws2 := t.TempDir()
	writeFile(t, ws2, "PROJECT_CHARTER.md", charterLegacy)
	if _, err := Run(ws2, Options{Apply: true}); err != nil {
		t.Fatalf("Run 2: %v", err)
	}
	if !strings.HasPrefix(readFile(t, filepath.Join(ws2, "PROJECT_CHARTER.md")), "---\nstatus: draft\n---\n") {
		t.Errorf("default deveria ser draft:\n%s", readFile(t, filepath.Join(ws2, "PROJECT_CHARTER.md")))
	}

	// (c) FM presente sem status → chave inserida logo após o delimitador
	ws3 := t.TempDir()
	orig := "---\nowner: Victor\n---\n# Charter — Sem Status\n\ncorpo intocado.\n"
	writeFile(t, ws3, "PROJECT_CHARTER.md", orig)
	if _, err := Run(ws3, Options{Apply: true}); err != nil {
		t.Fatalf("Run 3: %v", err)
	}
	got3 := readFile(t, filepath.Join(ws3, "PROJECT_CHARTER.md"))
	want := "---\nstatus: draft\nowner: Victor\n---\n# Charter — Sem Status\n\ncorpo intocado.\n"
	if got3 != want {
		t.Errorf("inserção divergiu:\ngot:\n%s\nwant:\n%s", got3, want)
	}

	// (d) status presente → kit não altera (transição é gate §7.1)
	ws4 := t.TempDir()
	writeFile(t, ws4, "PROJECT_CHARTER.md", "---\nstatus: draft\n---\n# Charter — Draft\n")
	before := readFile(t, filepath.Join(ws4, "PROJECT_CHARTER.md"))
	rep, err := Run(ws4, Options{Apply: true, CharterStatus: "confirmed"})
	if err != nil {
		t.Fatalf("Run 4: %v", err)
	}
	if readFile(t, filepath.Join(ws4, "PROJECT_CHARTER.md")) != before {
		t.Errorf("apply alterou charter com status existente")
	}
	if !hasFinding(rep.Conforme, "charter com status: draft") {
		t.Errorf("charter draft deveria aparecer como conforme: %+v", rep.Conforme)
	}
}

// o kit NUNCA edita prosa: todos os .md de prosa ficam byte-a-byte iguais
// (o charter só ganha o prefixo de front matter — corpo preservado).
func TestApplyProsaByteAByte(t *testing.T) {
	ws := legacyWorkspace(t)
	reqBefore := readFile(t, filepath.Join(ws, "REQUIREMENTS.md"))
	tasksBefore := readFile(t, filepath.Join(ws, "TASKS.md"))
	featBefore := readFile(t, filepath.Join(ws, "delivery", "2026-11", "RN-001 — Pedidos", "RN-001 — Pedidos - Feature.md"))
	charterBefore := readFile(t, filepath.Join(ws, "PROJECT_CHARTER.md"))

	if _, err := Run(ws, Options{Apply: true}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := readFile(t, filepath.Join(ws, "REQUIREMENTS.md")); got != reqBefore {
		t.Errorf("REQUIREMENTS.md foi alterado pelo apply")
	}
	if got := readFile(t, filepath.Join(ws, "TASKS.md")); got != tasksBefore {
		t.Errorf("TASKS.md foi alterado pelo apply")
	}
	if got := readFile(t, filepath.Join(ws, "delivery", "2026-11", "RN-001 — Pedidos", "RN-001 — Pedidos - Feature.md")); got != featBefore {
		t.Errorf("Feature.md foi alterada pelo apply")
	}
	got := readFile(t, filepath.Join(ws, "PROJECT_CHARTER.md"))
	if !strings.HasSuffix(got, charterBefore) {
		t.Errorf("corpo do charter não é mais o original:\n%s", got)
	}
}

// --keep-legacy-layouts marca as variantes como mantidas por escolha;
// default deixa em julgamento (gate de normalização).
func TestApplyKeepLegacyLayouts(t *testing.T) {
	ws := legacyWorkspace(t)
	rep, err := Run(ws, Options{Apply: true, KeepLegacyLayouts: true})
	if err != nil {
		t.Fatalf("Run keep: %v", err)
	}
	if len(rep.Preserved) == 0 {
		t.Errorf("--keep-legacy-layouts deveria popular Preserved: %+v", rep.Preserved)
	}
	if !hasFinding(rep.Conforme, "mantido por escolha") {
		t.Errorf("layouts mantidos deveriam aparecer como conforme: %+v", rep.Conforme)
	}
	for _, f := range rep.Julgamento {
		if strings.Contains(f.Item, "variante legada") {
			t.Errorf("com keep, variantes não deveriam ficar em julgamento: %+v", f)
		}
	}
	// nenhum arquivo movido (normalização física não existe no v0)
	if _, err := os.Stat(filepath.Join(ws, "delivery", "2026-11", "RN-001 — Pedidos", "RN-001 — Pedidos - Feature.md")); err != nil {
		t.Errorf("keep não deveria mover arquivos: %v", err)
	}

	// default: variantes em julgamento, Preserved vazio
	ws2 := legacyWorkspace(t)
	rep2, err := Run(ws2, Options{Apply: true})
	if err != nil {
		t.Fatalf("Run default: %v", err)
	}
	if len(rep2.Preserved) != 0 {
		t.Errorf("sem keep, Preserved deveria ser vazio: %+v", rep2.Preserved)
	}
	if !hasFinding(rep2.Julgamento, "variante legada") {
		t.Errorf("sem keep, variantes deveriam ficar em julgamento: %+v", rep2.Julgamento)
	}
}

// apply é idempotente: re-run não duplica escritas nem reescreve o índice.
func TestApplyIdempotent(t *testing.T) {
	ws := legacyWorkspace(t)
	rep1, err := Run(ws, Options{Apply: true})
	if err != nil {
		t.Fatalf("apply 1: %v", err)
	}
	idxAfter1 := readFile(t, filepath.Join(ws, "golmet.index.json"))
	charterAfter1 := readFile(t, filepath.Join(ws, "PROJECT_CHARTER.md"))
	if len(rep1.Applied) == 0 {
		t.Fatalf("primeiro apply deveria registrar escritas: %+v", rep1.Applied)
	}

	rep2, err := Run(ws, Options{Apply: true})
	if err != nil {
		t.Fatalf("apply 2: %v", err)
	}
	if rep2.IndexWritten {
		t.Errorf("re-run deveria detectar estado igual e NÃO reescrever o índice")
	}
	if len(rep2.Skipped) == 0 {
		t.Errorf("re-run deveria pular os arquivos existentes: %+v", rep2.Skipped)
	}
	// nenhum reparo duplicado de front matter
	if got := readFile(t, filepath.Join(ws, "PROJECT_CHARTER.md")); got != charterAfter1 {
		t.Errorf("re-run alterou o charter:\n%s", got)
	}
	t.Logf("rep1.IndexWritten=%v rep2.IndexWritten=%v", rep1.IndexWritten, rep2.IndexWritten)
	if idx := readFile(t, filepath.Join(ws, "golmet.index.json")); idx != idxAfter1 {
		t.Errorf("re-run reescreveu o índice sem mudança semântica (diff gerado_at?\n%s vs\n%s)", idxAfter1, idx)
	}
	// applied do re-run não deve repetir o reparo do charter
	for _, a := range rep2.Applied {
		if strings.Contains(a, "front matter prefixado") || strings.Contains(a, "status inserido") {
			t.Errorf("re-run repetiu o reparo de front matter: %s", a)
		}
	}
}

// caso de borda da skill: workspace sem charter → scaffold mínimo nasce
// draft (ou com o status declarado), §1 com tabela vazia.
func TestApplyWithoutCharter(t *testing.T) {
	ws := t.TempDir()
	writeFile(t, ws, "REQUIREMENTS.md", reqLegacy)
	rep, err := Run(ws, Options{Apply: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got := readFile(t, filepath.Join(ws, "PROJECT_CHARTER.md"))
	if !strings.HasPrefix(got, "---\nstatus: draft\n---\n# Charter — ") {
		t.Errorf("charter scaffoldado inválido:\n%s", got)
	}
	if !strings.Contains(got, "|---|---|---|") {
		t.Errorf("§1 com tabela de repos vazia ausente:\n%s", got)
	}
	applied := strings.Join(rep.Applied, "; ")
	if !strings.Contains(applied, "PROJECT_CHARTER.md") {
		t.Errorf("applied deveria registrar o scaffold do charter: %+v", rep.Applied)
	}
}

// modo não-assistido: apply direto SEM agente funciona; julgamentos
// pendentes ficam no relatório (skill gm-adopt §"Modo não-assistido").
func TestUnattendedApply(t *testing.T) {
	ws := legacyWorkspace(t)
	reqBefore := readFile(t, filepath.Join(ws, "REQUIREMENTS.md"))
	rep, err := Run(ws, Options{Apply: true})
	if err != nil {
		t.Fatalf("apply não-assistido deveria funcionar: %v", err)
	}
	if len(rep.Julgamento) == 0 {
		t.Errorf("julgamentos pendentes deveriam permanecer no relatório: %+v", rep.Julgamento)
	}
	// o kit NUNCA edita a prosa para resolver julgamento: a evidência legada
	// [src/pay.cs:10] permanece sem hash (re-pin é do agente)
	if got := readFile(t, filepath.Join(ws, "REQUIREMENTS.md")); got != reqBefore {
		t.Errorf("o kit alterou a prosa no apply — re-pin é do agente")
	}
}

func TestOptionsValidation(t *testing.T) {
	if _, err := Run(t.TempDir(), Options{CharterStatus: "confirmed?"}); err == nil {
		t.Errorf("charter-status inválido deveria falhar")
	}
	// workspace inexistente → erro de uso
	if _, err := Run(filepath.Join(t.TempDir(), "nada"), Options{}); err == nil {
		t.Errorf("workspace inexistente deveria falhar")
	}
}

func TestReportJSONShape(t *testing.T) {
	ws := legacyWorkspace(t)
	rep, err := Run(ws, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	data, err := json.Marshal(rep)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var probe map[string]any
	if err := json.Unmarshal(data, &probe); err != nil {
		t.Fatalf("JSON inválido: %v", err)
	}
	for _, key := range []string{"workspace", "mode", "charter_status", "julgamento", "kit_resolve"} {
		if _, ok := probe[key]; !ok {
			t.Errorf("chave %q ausente do JSON", key)
		}
	}
}
