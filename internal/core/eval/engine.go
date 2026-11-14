// engine.go — os 15 tipos de check, um método por tipo, semântica fiel
// ao runner de referência: contains/not_contains/section_exists = substring
// EXATA (não regex); paths de diretório (terminam em "/" ou existem como
// pasta) varrem *.md recursivo; mermaid_valid é heurístico (bloco fechado,
// keyword válida, chaves balanceadas exceto linhas de relacionamento com
// "--"); id_sequence usa o 1º capture group com dedupe por primeira
// aparição; expect: fail inverte o resultado (IMP-100 — evals negativos).
package eval

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// targetInfo resolve o path do check contra o workspace.
type targetInfo struct {
	abs   string
	rel   string // como escrito no criteria (separador '/')
	isDir bool
}

func resolveTarget(workspace, rel string) targetInfo {
	abs := workspace
	if rel != "" {
		abs = filepath.Join(workspace, filepath.FromSlash(rel))
	}
	isDir := strings.HasSuffix(rel, "/") || strings.HasSuffix(rel, "\\")
	if !isDir {
		if st, err := os.Stat(abs); err == nil && st.IsDir() {
			isDir = true
		}
	}
	return targetInfo{abs: abs, rel: rel, isDir: isDir}
}

// readDoc lê o arquivo UTF-8 (nil se ausente — mesmo contrato do Read-Doc
// do runner de referência: contains em arquivo ausente = FAIL;
// not_contains em ausente = PASS).
func readDoc(p string) (string, bool) {
	data, err := os.ReadFile(p)
	if err != nil {
		return "", false
	}
	return strings.TrimPrefix(strings.ReplaceAll(string(data), "\r\n", "\n"), "\uFEFF"), true
}

// mdFiles lista os *.md recursivos de um diretório (ordem determinística).
func mdFiles(dir string) []string {
	var out []string
	filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(p), ".md") {
			out = append(out, p)
		}
		return nil
	})
	return out
}

// concatDirDocs junta os *.md recursivos (cross_ref/contains em dir).
func concatDirDocs(dir string) string {
	var parts []string
	for _, f := range mdFiles(dir) {
		if d, ok := readDoc(f); ok {
			parts = append(parts, d)
		}
	}
	return strings.Join(parts, "\n")
}

// numList extrai os números do 1º capture group na ordem de primeira
// aparição, com dedupe (Get-NumList do runner).
func numList(text, pattern string) []int {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil
	}
	var out []int
	seen := map[int]bool{}
	for _, m := range re.FindAllStringSubmatch(text, -1) {
		if len(m) < 2 {
			continue
		}
		v, perr := strconv.Atoi(m[1])
		if perr != nil {
			continue
		}
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

// countLines conta as linhas do arquivo (0 se ausente).
func countLines(p string) int {
	d, ok := readDoc(p)
	if !ok {
		return 0
	}
	return len(strings.Split(strings.TrimRight(d, "\n"), "\n"))
}

// checkOutcome é o resultado bruto de um check antes do expect:fail.
type checkOutcome struct {
	ok     bool
	detail string
	fix    string
}

// runCheck avalia um check. Skipped = tipo desconhecido.
func runCheck(workspace, fixtureDir string, chk *Check) (status, detail, fix string) {
	var oc checkOutcome
	skipped := false
	switch chk.Type {
	case "file_exists":
		tg := resolveTarget(workspace, chk.Path)
		oc = checkFileExists(tg)
	case "not_file_exists":
		tg := resolveTarget(workspace, chk.Path)
		oc = checkNotFileExists(tg)
	case "dir_has_files":
		tg := resolveTarget(workspace, chk.Path)
		oc = checkDirHasFiles(tg)
	case "frontmatter_valid":
		tg := resolveTarget(workspace, chk.Path)
		oc = checkFrontmatterValid(tg)
	case "contains":
		tg := resolveTarget(workspace, chk.Path)
		oc = checkContains(tg, chk.Pattern)
	case "not_contains":
		tg := resolveTarget(workspace, chk.Path)
		oc = checkNotContains(tg, chk.Pattern)
	case "no_todos":
		tg := resolveTarget(workspace, chk.Path)
		oc = checkNoTodos(tg, chk.Patterns)
	case "section_exists":
		tg := resolveTarget(workspace, chk.Path)
		oc = checkSectionExists(tg, chk.Section)
	case "section_order":
		tg := resolveTarget(workspace, chk.Path)
		oc = checkSectionOrder(tg, chk.Sections)
	case "line_count_range":
		tg := resolveTarget(workspace, chk.Path)
		oc = checkLineCountRange(tg, *chk.Min, *chk.Max)
	case "id_sequence":
		tg := resolveTarget(workspace, chk.Path)
		oc = checkIDSequence(tg, chk.Pattern, allowGaps(chk))
	case "cross_ref":
		tg := resolveTarget(workspace, chk.Path)
		oc = checkCrossRef(workspace, tg, chk.Pattern, chk.Sources)
	case "mermaid_valid":
		tg := resolveTarget(workspace, chk.Path)
		oc = checkMermaidValid(tg)
	case "table_has_columns":
		tg := resolveTarget(workspace, chk.Path)
		oc = checkTableHasColumns(tg, chk.Columns)
	case "reference_match":
		oc = checkReferenceMatch(workspace, fixtureDir, chk)
	default:
		skipped = true
	}
	if skipped {
		return StatusSkipped, "tipo desconhecido: " + chk.Type, ""
	}
	if chk.Expect == "fail" {
		// IMP-100 — eval negativo: o check DEVE falhar; PASS = runner não
		// discriminou = FAIL do eval
		if oc.ok {
			return StatusFail, "runner não discriminou (esperado fail, obteve PASS)", ""
		}
		return StatusPass, "prato pego: " + oc.detail, ""
	}
	if oc.ok {
		return StatusPass, oc.detail, ""
	}
	if chk.Severity == SeverityWarning {
		return StatusWarn, oc.detail, oc.fix
	}
	return StatusFail, oc.detail, oc.fix
}

// allowGaps — default FALSE (exige +1 consecutivo); true = só ascendente.
func allowGaps(chk *Check) bool {
	return chk.AllowGaps != nil && *chk.AllowGaps
}

// ---- checks ----

func checkFileExists(tg targetInfo) checkOutcome {
	_, err := os.Stat(tg.abs)
	return checkOutcome{ok: err == nil, detail: "arquivo ausente: " + tg.rel,
		fix: "gere o artefato em " + tg.rel}
}

func checkNotFileExists(tg targetInfo) checkOutcome {
	_, err := os.Stat(tg.abs)
	return checkOutcome{ok: os.IsNotExist(err), detail: "arquivo existe e não deveria: " + tg.rel,
		fix: "remova " + tg.rel + " (contrato de abort/não-invenção)"}
}

func checkDirHasFiles(tg targetInfo) checkOutcome {
	entries, err := os.ReadDir(tg.abs)
	if err != nil {
		return checkOutcome{ok: false, detail: "diretório ausente: " + tg.rel,
			fix: "gere o diretório " + tg.rel + " com ao menos 1 arquivo"}
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() {
			n++
		}
	}
	return checkOutcome{ok: n >= 1, detail: fmt.Sprintf("arquivos=%d", n),
		fix: "gere ao menos 1 arquivo em " + tg.rel}
}

func checkFrontmatterValid(tg targetInfo) checkOutcome {
	doc, ok := readDoc(tg.abs)
	if !ok {
		return checkOutcome{ok: false, detail: "arquivo ausente: " + tg.rel,
			fix: "gere " + tg.rel + " com front matter --- … ---"}
	}
	valid := strings.HasPrefix(doc, "---\n") || strings.HasPrefix(doc, "---\r\n")
	return checkOutcome{ok: valid, detail: "front matter ausente (esperado --- inicial)",
		fix: "prefixe o front matter --- … --- em " + tg.rel}
}

func checkContains(tg targetInfo, pattern string) checkOutcome {
	var doc string
	var ok bool
	if tg.isDir {
		doc = concatDirDocs(tg.abs)
		ok = true
	} else {
		doc, ok = readDoc(tg.abs)
	}
	found := ok && strings.Contains(doc, pattern)
	return checkOutcome{ok: found, detail: fmt.Sprintf("pattern %q não encontrado em %s", pattern, tg.rel),
		fix: "gere o conteúdo esperado (" + pattern + ") em " + tg.rel}
}

func checkNotContains(tg targetInfo, pattern string) checkOutcome {
	var doc string
	var ok bool
	if tg.isDir {
		doc = concatDirDocs(tg.abs)
		ok = true
	} else {
		doc, ok = readDoc(tg.abs)
	}
	found := ok && strings.Contains(doc, pattern)
	return checkOutcome{ok: !found, detail: fmt.Sprintf("pattern proibido %q encontrado em %s", pattern, tg.rel),
		fix: "remova o conteúdo proibido (" + pattern + ") de " + tg.rel}
}

func checkNoTodos(tg targetInfo, patterns []string) checkOutcome {
	doc, _ := readDoc(tg.abs)
	var hits []string
	for _, p := range patterns {
		if strings.Contains(doc, p) {
			hits = append(hits, p)
		}
	}
	return checkOutcome{ok: len(hits) == 0, detail: "marcadores encontrados: " + strings.Join(hits, ","),
		fix: "remova os marcadores (" + strings.Join(hits, ", ") + ") de " + tg.rel}
}

func checkSectionExists(tg targetInfo, section string) checkOutcome {
	doc, ok := readDoc(tg.abs)
	found := ok && strings.Contains(doc, section)
	return checkOutcome{ok: found, detail: fmt.Sprintf("seção %q ausente em %s", section, tg.rel),
		fix: "gere a seção " + section + " em " + tg.rel}
}

func checkSectionOrder(tg targetInfo, sections []string) checkOutcome {
	doc, ok := readDoc(tg.abs)
	if !ok {
		return checkOutcome{ok: false, detail: "arquivo ausente: " + tg.rel, fix: "gere " + tg.rel}
	}
	var idxs []int
	var missing []string
	for _, s := range sections {
		i := strings.Index(doc, s)
		if i < 0 {
			missing = append(missing, s)
			continue
		}
		idxs = append(idxs, i)
	}
	ordered := true
	for i := 1; i < len(idxs); i++ {
		if idxs[i] <= idxs[i-1] {
			ordered = false
		}
	}
	if len(missing) > 0 {
		return checkOutcome{ok: false, detail: "seções ausentes: " + strings.Join(missing, "; "),
			fix: "gere as seções ausentes em " + tg.rel}
	}
	return checkOutcome{ok: ordered, detail: "ordem incorreta das seções (primeira aparição deve ser crescente)",
		fix: "reordene as seções conforme o criteria em " + tg.rel}
}

func checkLineCountRange(tg targetInfo, min, max int) checkOutcome {
	total := 0
	if tg.isDir {
		for _, f := range mdFiles(tg.abs) {
			total += countLines(f)
		}
	} else {
		total = countLines(tg.abs)
	}
	ok := total >= min && total <= max
	return checkOutcome{ok: ok, detail: fmt.Sprintf("linhas=%d (faixa %d-%d)", total, min, max),
		fix: mapBool(!ok && total < min, "conteúdo truncado — gere o documento completo em "+tg.rel,
			"conteúdo inflado — revise "+tg.rel)}
}

func checkIDSequence(tg targetInfo, pattern string, gaps bool) checkOutcome {
	doc, ok := readDoc(tg.abs)
	if !ok {
		return checkOutcome{ok: false, detail: "arquivo ausente: " + tg.rel, fix: "gere " + tg.rel}
	}
	ids := numList(doc, pattern)
	asc := true
	for i := 1; i < len(ids); i++ {
		if ids[i] < ids[i-1] {
			asc = false
		}
	}
	if !gaps {
		for i := 1; i < len(ids); i++ {
			if ids[i] != ids[i-1]+1 {
				asc = false
			}
		}
	}
	var ss []string
	for _, v := range ids {
		ss = append(ss, strconv.Itoa(v))
	}
	return checkOutcome{ok: asc, detail: "primeira aparição: " + strings.Join(ss, "->"),
		fix: "numeração fora de sequência em " + tg.rel}
}

func checkCrossRef(workspace string, tg targetInfo, pattern string, sources []string) checkOutcome {
	var doc string
	if tg.isDir {
		doc = concatDirDocs(tg.abs)
	} else {
		doc, _ = readDoc(tg.abs)
	}
	ids := numList(doc, pattern)
	var srcIDs []int
	for _, s := range sources {
		sd, ok := readDoc(filepath.Join(workspace, filepath.FromSlash(s)))
		if ok {
			srcIDs = append(srcIDs, numList(sd, pattern)...)
		}
	}
	srcSet := map[int]bool{}
	for _, v := range srcIDs {
		srcSet[v] = true
	}
	var hallucinated []int
	for _, v := range ids {
		if !srcSet[v] {
			hallucinated = append(hallucinated, v)
		}
	}
	if len(hallucinated) > 0 {
		var ss []string
		for _, v := range hallucinated {
			ss = append(ss, strconv.Itoa(v))
		}
		return checkOutcome{ok: false, detail: "IDs alucinados (não existem nas fontes): " + strings.Join(ss, ","),
			fix: "remova ou corrija os IDs citados em " + tg.rel}
	}
	return checkOutcome{ok: true}
}

// mermaidTypes — keywords válidas na 1ª linha do bloco.
var mermaidTypes = []string{
	"erDiagram", "flowchart", "graph", "sequenceDiagram", "classDiagram",
	"stateDiagram", "gantt", "pie", "mindmap", "timeline", "journey", "gitGraph",
}

// checkMermaidValid — HEURÍSTICO (documentado): ≥1 bloco ```mermaid;
// corpo não-vazio; 1ª linha começa com keyword válida; chaves {} e []
// balanceadas EXCETO em linhas de relacionamento (contêm "--").
func checkMermaidValid(tg targetInfo) checkOutcome {
	doc, ok := readDoc(tg.abs)
	if !ok {
		return checkOutcome{ok: false, detail: "arquivo ausente: " + tg.rel, fix: "gere " + tg.rel}
	}
	blocks := mermaidBlockRe.FindAllStringSubmatch(doc, -1)
	if len(blocks) == 0 {
		return checkOutcome{ok: false, detail: "nenhum bloco ```mermaid encontrado em " + tg.rel,
			fix: "inclua o diagrama mermaid em " + tg.rel}
	}
	for _, b := range blocks {
		body := strings.TrimLeft(b[1], " \t")
		if body == "" {
			return checkOutcome{ok: false, detail: "bloco mermaid vazio em " + tg.rel,
				fix: "gere o diagrama em " + tg.rel}
		}
		first := strings.Split(body, "\n")[0]
		first = strings.TrimSpace(first)
		validType := false
		for _, t := range mermaidTypes {
			if strings.HasPrefix(first, t) {
				validType = true
				break
			}
		}
		if !validType {
			return checkOutcome{ok: false, detail: "bloco mermaid sem tipo válido na 1ª linha: " + first,
				fix: "declare o tipo do diagrama (erDiagram, flowchart, …) em " + tg.rel}
		}
		var code []string
		for _, line := range strings.Split(body, "\n") {
			if !strings.Contains(line, "--") {
				code = append(code, line)
			}
		}
		codeStr := strings.Join(code, "\n")
		if strings.Count(codeStr, "{") != strings.Count(codeStr, "}") ||
			strings.Count(codeStr, "[") != strings.Count(codeStr, "]") {
			return checkOutcome{ok: false, detail: "chaves/colchetes desbalanceados no bloco mermaid",
				fix: "feche as chaves/colchetes do diagrama em " + tg.rel}
		}
	}
	return checkOutcome{ok: true}
}

var mermaidBlockRe = regexp.MustCompile(`(?s)` + "```mermaid\\s*\\r?\\n(.*?)```")

func checkTableHasColumns(tg targetInfo, columns []string) checkOutcome {
	doc, ok := readDoc(tg.abs)
	if !ok {
		return checkOutcome{ok: false, detail: "arquivo ausente: " + tg.rel, fix: "gere " + tg.rel}
	}
	for _, line := range strings.Split(doc, "\n") {
		trimmed := strings.TrimLeft(line, " \t")
		if !strings.HasPrefix(trimmed, "|") {
			continue
		}
		all := true
		for _, col := range columns {
			if !strings.Contains(line, col) {
				all = false
				break
			}
		}
		if all {
			return checkOutcome{ok: true}
		}
	}
	return checkOutcome{ok: false, detail: fmt.Sprintf("nenhuma tabela com as colunas %v em %s", columns, tg.rel),
		fix: "gere a tabela com as colunas esperadas em " + tg.rel}
}

// checkReferenceMatch — comparação linha a linha após normalização
// (datas/horas → placeholders, trailing spaces, blank lines colapsadas).
// Ordem do runner de referência: expected é relativo à FIXTURE (baseline
// congelado); ausente → fallback para o path do check no workspace
// (baseline-mode); se NEHUM existir → FAIL "baseline ausente".
func checkReferenceMatch(workspace, fixtureDir string, chk *Check) checkOutcome {
	expPath := filepath.Join(fixtureDir, filepath.FromSlash(chk.Expected))
	if _, err := os.Stat(expPath); err != nil {
		expPath = filepath.Join(workspace, filepath.FromSlash(chk.Path))
		if _, err := os.Stat(expPath); err != nil {
			return checkOutcome{ok: false, detail: "baseline ausente: " + chk.Expected,
				fix: "congele o output de referência em " + chk.Expected + " na fixture"}
		}
	}
	expDoc, ok := readDoc(expPath)
	if !ok {
		expDoc = "" // arquivo presente mas ilegível: compara vazio (fiel ao Read-Doc)
	}
	doc, _ := readDoc(filepath.Join(workspace, filepath.FromSlash(chk.Path)))
	return checkOutcome{ok: normalizeDoc(doc) == normalizeDoc(expDoc),
		detail: "output diverge do baseline além das tolerâncias (datas/horas, espaços à direita, blank lines)",
		fix:    "compare com " + path.Join("output-expected", filepath.Base(chk.Expected))}
}

var (
	rxDateISO  = regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)
	rxDateBR   = regexp.MustCompile(`\d{2}/\d{2}/\d{4}`)
	rxTime     = regexp.MustCompile(`\d{2}:\d{2}`)
	rxTrailing = regexp.MustCompile(`[ \t]+$`)
)

// normalizeDoc — mesma normalização do runner (datas ISO/BR → <DATE>,
// hh:mm → <TIME>, trailing spaces, blank lines colapsadas em 1, join \n).
func normalizeDoc(s string) string {
	s = rxDateISO.ReplaceAllString(s, "<DATE>")
	s = rxDateBR.ReplaceAllString(s, "<DATE>")
	s = rxTime.ReplaceAllString(s, "<TIME>")
	lines := strings.Split(s, "\n")
	var out []string
	blank := false
	for _, l := range lines {
		l = rxTrailing.ReplaceAllString(l, "")
		if strings.TrimSpace(l) == "" {
			if !blank {
				out = append(out, l)
			}
			blank = true
			continue
		}
		out = append(out, l)
		blank = false
	}
	return strings.Join(out, "\n")
}

func mapBool(cond bool, ifTrue, ifFalse string) string {
	if cond {
		return ifTrue
	}
	return ifFalse
}
