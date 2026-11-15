// Checklist do CONTAINER de organização (FORMATO §2): hierarquia
// products ↔ disco, base canônica e artefatos org-level soltos. Substitui
// os checks de workspace-produto na raiz de um container (índice, scan de
// entidades, zona .golmet e capacidades vendored não se aplicam — o
// container não roda o pipeline de RNs); checks de AMBIENTE DE MÁQUINA
// (git, kit.jev, kit.graph, engine-config) continuam no Run.
package doctor

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Golmet/golmet-cli/internal/core/container"
	"github.com/Golmet/golmet-cli/internal/core/jev"
	"github.com/Golmet/golmet-cli/internal/core/rules"
)

// addMachineChecks — checks 6-9: ambiente de MÁQUINA (kit.jev, kit.graph,
// git, engine-config) — comum a workspace-produto e container (o container
// chama com skills nil: não há skills vendored na raiz dele).
func (r *Report) addMachineChecks(opt Options, gitOK bool, skills []rules.SkillInfo) {
	// kit.jev — TYPESAFE_API_KEY: key presente → smoke POR DEFAULT (1
	// POST pequeno — o aprendizado REAL da sessão: "key presente" enganou
	// o operador por semanas com o endpoint errado; o doctor checa se o Jev
	// FUNCIONA, não só se a key existe). Resultado é ⚠️ NUNCA ❌ (health é
	// soft). --no-network pula o smoke (CI/offline); --network é alias de
	// compatibilidade (o smoke já é default — sem efeito extra).
	jevRequired := false
	for _, sk := range skills {
		if sk.RequiresKit("jev") {
			jevRequired = true
		}
	}
	key := os.Getenv("TYPESAFE_API_KEY")
	switch {
	case key == "" && jevRequired:
		r.add("kit.jev", Warn, "TYPESAFE_API_KEY ausente — skill requer kit.jev", "configure o runtime Typesafe/Jev (DR-015)")
	case key == "":
		r.add("kit.jev", OK, "offline (não requerido por nenhuma skill)", "")
	case opt.NoNetwork:
		r.add("kit.jev", OK, "TYPESAFE_API_KEY presente (smoke não executado — --no-network)", "verifique a key em console.typesafe.ai/keys")
	default:
		switch s, smoke := jev.SmokeHealth(context.Background()); s {
		case jev.SmokeOK:
			r.add("kit.jev", OK, smoke+" — key presente e runtime operacional", "")
		case jev.SmokeRejected:
			r.add("kit.jev", Warn, "key presente mas REJEITADA pelo runtime — "+smoke, "verifique a key em console.typesafe.ai/keys e o endpoint")
		default:
			r.add("kit.jev", Warn, "key presente mas o runtime não respondeu — verifique o endpoint ("+smoke+")", "verifique a key em console.typesafe.ai/keys")
		}
	}

	// kit.graph — binário DeusData em ~/.golmet/tools/codegraph/ (bloco 3 instala)
	graphRequired := false
	for _, sk := range skills {
		if sk.RequiresKit("graph") {
			graphRequired = true
		}
	}
	graphInstalled := detectCodeGraph()
	switch {
	case !graphInstalled && graphRequired:
		r.add("kit.graph", Warn, "binário DeusData ausente em ~/.golmet/tools/codegraph/ — skill requer kit.graph", "instale o grafo de código do kit (DR-016)")
	case !graphInstalled:
		r.add("kit.graph", OK, "não instalado (não requerido por nenhuma skill)", "")
	default:
		r.add("kit.graph", OK, "binário detectado em ~/.golmet/tools/codegraph/", "")
	}

	// git disponível
	if gitOK {
		r.add("git", OK, "git disponível", "")
	} else {
		r.add("git", Warn, "git indisponível — H7/S2 degradam", "instale o git no PATH")
	}

	// estado do engine-config — cache reconstrutível vs dado do usuário
	// (customização em gerido, sem proveniência, layer local removida).
	// Roda contra o CONFIG ROOT do engine, não contra o workspace; vazio =
	// check não roda (compatibilidade total com chamadas workspace-only).
	if opt.EngineConfigRoot != "" {
		r.runEngineConfig(opt.EngineConfigRoot)
	}
}

func (r *Report) runContainerChecks(root string, products []string) {
	st := container.Inspect(root, products)

	// products declarados vs disco — declarado ausente = fail (stale)
	switch {
	case len(st.MissingProducts) > 0:
		r.add("container-products", Fail,
			"produto(s) declarado(s) no products: ausente(s) no disco — "+strings.Join(st.MissingProducts, ", "),
			"remova do products: do golmet.yaml (o produto não existe mais) ou restaure o diretório")
	case len(products) == 0:
		r.add("container-products", OK,
			"nenhum produto declarado ainda no products:",
			"crie o primeiro com 'golmet init <produto> --into <container>'")
	default:
		r.add("container-products", OK,
			fmt.Sprintf("%d produto(s) declarado(s) e presentes no disco", len(st.DeclaredPresent)), "")
	}

	// dirs de produto na raiz fora do products: (workspace órfão ou pasta
	// perdida — só o container sabe qual); docs/ ganha mensagem própria
	// (não é estrutura canônica do container — M18)
	var otherDirs []string
	for _, d := range st.UndeclaredDirs {
		if d == "docs" {
			r.add("container-estrutura", Warn,
				"docs/ não é estrutura canônica do container",
				"docs de produto vão no produto; docs do kit vão no repo do kit; históricos em migrations/")
			continue
		}
		otherDirs = append(otherDirs, d)
	}
	if len(otherDirs) > 0 {
		r.add("container-products", Warn,
			"dir(s) de produto na raiz fora do products: declarado — "+strings.Join(otherDirs, ", "),
			"declare no golmet.yaml do container ('golmet init <produto> --into <container>') ou mova para migrations/ (histórico) se não for produto")
	}

	// base canônica ausente (INDEX.md é a navegação do container — IMP-012)
	if len(st.MissingCanonical) > 0 {
		r.add("container-estrutura", Warn,
			"arquivo(s) canônico(s) do container ausente(s) — "+strings.Join(st.MissingCanonical, ", "),
			"crie o arquivo (o stub vem do 'golmet init --org' — idempotente, nada existente é sobrescrito)")
	}

	// artefatos org-level soltos na raiz
	if len(st.LooseMDs) > 0 {
		r.add("container-raiz", Warn,
			"artefato(s) org-level na raiz do container — "+strings.Join(st.LooseMDs, ", "),
			"mova para migrations/ (histórico da org) ou declare allowlist")
	}
}
