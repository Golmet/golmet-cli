// versioning.go — os 2 MODELOS DE VERSIONAMENTO DE ORGANIZAÇÃO (FORMAT
// §2.3 — M29): o golmet.yaml do container declara `versioning:` e todo o
// kit (scaffold do produto, zona do kit, gitignore) respeita o modelo
// declarado. Fonte única consumida pelo scaffold (init --into), pelo
// update (a exceção do gitignore na zona de kit da org) e pelo doctor.
//
//   - org-repo (a realidade da TechMinds — o TM.RDD.DOCS é um repo Git e
//     o Groom é pasta dele): a ORGANIZAÇÃO é o repo — o .gitignore é da
//     raiz da org, a zona do kit é da ORG (opcional: o essencial org
//     instala as skills BASE na zona da org — a equipe herda ao clonar) e
//     o workspace-produto nasce SEM .golmet/ própria (o versionamento da
//     org já cobre);
//   - produtos-repo (o modelo DR-008 clássico): cada produto é repo
//     próprio — a .golmet/ de cada produto é o kit do repo DELE
//     (versionada, a equipe do produto herda), o .gitignore é de cada
//     produto.
package container

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Golmet/golmet-cli/internal/core/frontmatter"
)

func gitignorePath(root string) string {
	return filepath.Join(root, ".gitignore")
}

func readFileForGitignore(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil // sem .gitignore — EnsureKitZoneException cria
	}
	return data, err
}

func containsKitZoneException(content string) bool {
	return strings.Contains(content, "!.golmet/skills/")
}

func splitLines(s string) []string { return strings.Split(s, "\n") }

func joinLines(lines []string) string { return strings.Join(lines, "\n") }

func trimTrailingSpace(s string) string { return strings.TrimRight(s, " \t\r") }

func osWriteGitignore(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

// Os modelos declarados no `versioning:` do golmet.yaml do container.
const (
	VersioningOrgRepo      = "org-repo"      // a ORG é o repo — produtos são pastas dela
	VersioningProductsRepo = "produtos-repo" // cada produto é repo próprio (DR-008)
)

// VersioningOf — o modelo declarado; SEM o campo = produtos-repo (o
// modelo DR-008 clássico, o default retroativo dos containers anteriores
// a M29).
func VersioningOf(fields map[string]frontmatter.Value) string {
	if v := fields["versioning"].Str; v == VersioningOrgRepo {
		return VersioningOrgRepo
	}
	return VersioningProductsRepo
}

// orgGitignoreContent — o .gitignore da ORGANIZAÇÃO que o `init --org`
// cria (M29): a zona de trabalho do kit é gitignored (`.golmet/*` —
// checkpoints/histórico) e os básicos de SO/editor; a EXCEÇÃO da zona de
// kit da org (`!.golmet/skills/`) entra quando o update essencial roda na
// raiz (EnsureKitZoneException) — sem update, a org fica sem zona de kit
// (o estado real do TM.RDD.DOCS). `.golmet/*` (o CONTEÚDO, não o dir — a
// negação git não re-inclui sob um dir excluído).
const orgGitignoreContent = `# Zona de trabalho do kit no container (FORMAT §2.3/M19) - checkpoints e
# histórico de processo, nunca versionados
.golmet/*

# Sistema/editor
.DS_Store
Thumbs.db
desktop.ini
*.swp
*.swo
*~
.idea/
.vscode/
`

// OrgGitignore — o conteúdo do .gitignore da organização (init --org).
func OrgGitignore() string { return orgGitignoreContent }

// orgKitExceptionBlock — a exceção que o update essencial acrescenta
// quando instala a zona do kit da ORG.
const orgKitExceptionBlock = "\n# Zona do kit da ORG (M29) - as skills BASE são versionadas (a equipe herda ao clonar)\n!.golmet/skills/\n"

// EnsureKitZoneException — a EXCEÇÃO `!.golmet/skills/` no .gitignore da
// org quando o update essencial instala a zona do kit na raiz da org
// (org-repo). Idempotente: já presente = no-op; sem .gitignore = cria com
// o conteúdo padrão + a exceção. O `.golmet/` DIR-level (containers
// criados antes da M29) é convertido cirurgicamente para `.golmet/*` — a
// negação git não re-inclui sob um dir excluído; só o CONTEÚDO é
// ignorado, a exceção passa a funcionar.
func EnsureKitZoneException(root string) error {
	path := gitignorePath(root)
	data, err := readFileForGitignore(path)
	if err != nil {
		return err
	}
	if data == nil {
		// sem .gitignore na org (não deveria — o init --org cria): cria
		// com o conteúdo padrão + a exceção
		return osWriteGitignore(path, OrgGitignore()+orgKitExceptionBlock)
	}
	content := string(data)
	if containsKitZoneException(content) {
		return nil // já presente — idempotente
	}
	lines := splitLines(content)
	for i, l := range lines {
		if trimTrailingSpace(l) == ".golmet/" {
			lines[i] = ".golmet/*" // dir-level impede a negação — converte
		}
	}
	content = joinLines(lines) + orgKitExceptionBlock
	return osWriteGitignore(path, content)
}
