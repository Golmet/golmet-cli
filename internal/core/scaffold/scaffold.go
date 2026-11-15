// Package scaffold implementa o `golmet init` (FORMATO §2/§3): esqueleto
// do workspace de produto (golmet.yaml, charter, entidades, zonas) ou
// container de organização (--org), com idempotência — arquivo existente
// não-vazio NUNCA é sobrescrito (avisa e pula). O kit é o único escritor
// do índice; o init cria o baseline vazio (primeira execução sem história
// — H6/H8 não disparam).
package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/Golmet/golmet-cli/internal/core/frontmatter"
	"github.com/Golmet/golmet-cli/internal/core/indexstate"
)

// Options controla o init. Org e Into são exclusivos. CharterStatus é o
// status gravado no charter scaffoldado (default draft; confirmed é
// usado pelo `golmet adopt --apply --charter-status confirmed`).
type Options struct {
	Org           bool   // cria container de organização (container: true)
	Into          string // cria o produto dentro de um container existente
	CharterStatus string // draft (default) | confirmed (§7.1)
}

// userHomeDir injetável (testes) — a guarda do home não pode exigir o
// home real da máquina de teste.
var userHomeDir = os.UserHomeDir

// FileResult registra o destino de um arquivo do scaffold.
type FileResult struct {
	Path    string // relativo à raiz criada, separador '/'
	Created bool
	Skipped bool   // existia não-vazio — preservado (IMP-080)
	Note    string // motivo do skip (erro de escrita), exibido no CLI
}

// Report resume a execução do init.
type Report struct {
	Root              string // raiz absoluta do workspace/container
	Name              string
	Files             []FileResult
	Created, Skipped  int
	ContainerUpdated  bool   // products: do container ganhou o produto (--into)
	ContainerProducts string // lista products após a atualização (exibição)
}

// Run executa o scaffold. baseDir é onde criar (cwd normalmente); name
// vazio usa o próprio baseDir como workspace (nome = base do caminho).
func Run(baseDir, name string, opt Options) (*Report, error) {
	if opt.Org && opt.Into != "" {
		return nil, fmt.Errorf("flags --org e --into são exclusivas")
	}
	base, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("caminho inválido: %w", err)
	}
	root := base
	if opt.Into != "" {
		if name == "" {
			return nil, fmt.Errorf("--into exige o nome do produto: golmet init <produto> --into <container>")
		}
		container, cerr := filepath.Abs(opt.Into)
		if cerr != nil {
			return nil, fmt.Errorf("container inválido: %w", cerr)
		}
		if st, serr := os.Stat(container); serr != nil || !st.IsDir() {
			return nil, fmt.Errorf("container não encontrado: %s", container)
		}
		if err := requireContainer(container); err != nil {
			return nil, err
		}
		root = filepath.Join(container, name)
	} else if name != "" {
		root = filepath.Join(base, name)
	}
	if st, serr := os.Stat(root); serr == nil && !st.IsDir() {
		return nil, fmt.Errorf("%s existe e não é uma pasta", root)
	}
	// M22 — guarda do caso real: o init NUNCA cria na RAIZ do HOME do
	// operador (o ~/.golmet/ é a instalação global do kit — tools, packs;
	// um workspace na raiz do home mistura estado do kit com o workspace e
	// quase apagou o codegraph). Subpastas do home são legítimas.
	if home, herr := userHomeDir(); herr == nil && sameDirPath(root, home) {
		return nil, fmt.Errorf("raiz %s é o HOME do operador — o init nunca cria no home (a instalação global do kit vive em ~/.golmet/); rode o init dentro de uma subpasta", root)
	}
	wsName := filepath.Base(root)

	rep := &Report{Root: root, Name: wsName}
	write := func(rel, content string) {
		fr := FileResult{Path: filepath.ToSlash(rel)}
		fail := func(note string) {
			fr.Skipped = true
			fr.Note = note
			rep.Files = append(rep.Files, fr)
			rep.Skipped++
		}
		target := filepath.Join(root, filepath.FromSlash(rel))
		if data, err := os.ReadFile(target); err == nil && len(data) > 0 {
			fr.Skipped = true // existente não-vazio — nunca sobrescreve (IMP-080)
			rep.Files = append(rep.Files, fr)
			rep.Skipped++
			return
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			fail("erro ao criar a pasta: " + err.Error())
			return
		}
		if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
			fail("erro de escrita: " + err.Error())
			return
		}
		fr.Created = true
		rep.Files = append(rep.Files, fr)
		rep.Created++
	}
	mkdir := func(rel string) {
		_ = os.MkdirAll(filepath.Join(root, filepath.FromSlash(rel)), 0o755)
	}

	if opt.Org {
		write("golmet.yaml", "format: 1\ncontainer: true\nproducts: []\n")
		write("CONSTITUTION.md", "# Constituição — "+wsName+"\n\nPrincípios da organização (edite este stub).\n")
		write("INDEX.md", "# Índice — "+wsName+"\n\nProjetos e documentos do container (edite este stub).\n")
		return rep, nil
	}

	charterStatus := opt.CharterStatus
	if charterStatus == "" {
		charterStatus = "draft"
	}
	write("golmet.yaml", "format: 1\n")
	write("PROJECT_CHARTER.md", charterTemplate(wsName, charterStatus))
	write("REQUIREMENTS.md", "# Requisitos — "+wsName+"\n")
	write("CHANGE_REQUESTS.md", "# Change Requests — "+wsName+"\n")
	write("DECISIONS.md", "# Decisões — "+wsName+"\n")
	write("reverse/CANDIDATES.md", "# Candidates — "+wsName+"\n")
	write("specs/README.md", "# specs — downstream técnico (DATA_MODEL, API_CONTRACTS, UI_SPEC…)\n")
	mkdir("analysis")
	mkdir("delivery")
	mkdir(".golmet/skills")
	write(".golmet/manifest.json", manifestTemplate())

	// baseline do índice (§8) — workspace vazio, sem história; charter
	// status alinhado ao template para o primeiro validate não reescrever
	// o baseline (mesmo estado semântico).
	baseline := indexstate.Build(wsName, indexstate.Snapshot{
		CharterFound:  true,
		CharterStatus: charterStatus,
	})
	if data, err := baseline.Marshal(); err == nil {
		write("golmet.index.json", string(data))
	}

	if opt.Into != "" {
		container := filepath.Dir(root)
		products, err := addToContainer(container, wsName)
		if err != nil {
			return rep, err
		}
		rep.ContainerUpdated = true
		rep.ContainerProducts = strings.Join(products, ", ")
	}
	return rep, nil
}

// charterTemplate — §7.1: front matter com o status informado (draft |
// confirmed) + §1 com tabela de repos vazia.
func charterTemplate(name, status string) string {
	return "---\nstatus: " + status + "\n---\n" +
		"# Charter — " + name + "\n\n" +
		"## 1. Repos\n\n" +
		"| Repo | Branch | HEAD |\n" +
		"|---|---|---|\n"
}

func manifestTemplate() string {
	return "{\n  \"version\": \"" + indexstate.KitVersion + "\",\n  \"skills\": []\n}\n"
}

// requireContainer valida que dir é um container (golmet.yaml parseável,
// format suportado, container: true).
func requireContainer(dir string) error {
	cfgPath := filepath.Join(dir, "golmet.yaml")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("container sem golmet.yaml em %s", dir)
	}
	fields, perr := frontmatter.ParseBlock(string(data))
	if perr != nil {
		return fmt.Errorf("golmet.yaml do container ilegível (%s): %v", cfgPath, perr)
	}
	fv, ok := fields["format"]
	if !ok || fv.Str == "" {
		return fmt.Errorf("golmet.yaml do container sem format (DEVE ser 1)")
	}
	if fv.Str != "1" {
		return fmt.Errorf("formato %s não suportado — este kit suporta format: 1 (§12)", fv.Str)
	}
	if cv := fields["container"]; cv.Str != "true" {
		return fmt.Errorf("%s não é um container (container: true ausente no golmet.yaml)", dir)
	}
	return nil
}

// sameDirPath compara dois caminhos com tolerância do SO (Windows é case-
// insensitive; o root resolvido e o home do operador podem diferir em
// case/barras).
func sameDirPath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// addToContainer acrescenta o produto à lista products: do container com
// cirurgia de linha (preserva o restante do arquivo — o único ponto onde
// o init edita arquivo existente).
var productsLineRe = regexp.MustCompile(`(?m)^products:.*$`)

func addToContainer(dir, name string) ([]string, error) {
	cfgPath := filepath.Join(dir, "golmet.yaml")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("container sem golmet.yaml em %s", dir)
	}
	fields, perr := frontmatter.ParseBlock(string(data))
	if perr != nil {
		return nil, fmt.Errorf("golmet.yaml do container ilegível: %v", perr)
	}
	products := fields["products"].Arr
	for _, p := range products {
		if p == name {
			return products, nil // já registrado — idempotente
		}
	}
	products = append(products, name)
	line := "products: [" + strings.Join(products, ", ") + "]"
	content := string(data)
	if productsLineRe.MatchString(content) {
		content = productsLineRe.ReplaceAllString(content, line)
	} else {
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += line + "\n"
	}
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		return nil, fmt.Errorf("falha ao atualizar products do container: %w", err)
	}
	return products, nil
}
