// Package install implementa `golmet install` — o onboarding do kit em
// um comando: engine + skills + MCPs + doctor final. O kit ESCREVE os
// blocos de config dos MCPs escolhidos (decisão do operador: config auto
// completa; a gestão contínua continua da runtime — DR-016). Nunca apaga
// nem desativa nada que já exista (coexistência é estado normal).
package install

import (
	"fmt"
	"sort"
	"strings"
)

// MCPType: stdio (command/args) ou remote (url + OAuth pelo engine).
const (
	MCPTypeStdio  = "stdio"
	MCPTypeRemote = "remote"
)

// MCPDef é uma entrada do catálogo embutido. Fontes citadas por entrada —
// nada inventado: o que não tem comando verificado não entra (ficou
// pendência documentada no relatório do bloco).
//
// Param (genérico, hoje só azure-devops usa): nome do parâmetro
// obrigatório interpolado no Command/Args pelo token "<Param>" — ex.:
// Command com "<ORG>" e Param "ORG". Sem o valor → pendência (config
// quebrada nunca).
type MCPDef struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Type        string   `json:"type"` // stdio | remote
	Command     string   `json:"command,omitempty"`
	Args        []string `json:"args,omitempty"`
	EnvRequired []string `json:"env_required,omitempty"`
	// EnvOptional: env que MELHORA o serviço (rate limits, quota) mas não
	// bloqueia o registro — contrário de EnvRequired, cuja ausência vira
	// pendência (config quebrada nunca). O install registra igual sem a
	// env; a orientação de cadastro fica na skill de setup do MCP.
	EnvOptional []string `json:"env_optional,omitempty"`
	URL         string   `json:"url,omitempty"`
	Param       string   `json:"param,omitempty"` // parâmetro obrigatório (token <Param> no Command/Args)
	Required    bool     `json:"required"`        // golmet — sempre registrado
	Recommended bool     `json:"recommended,omitempty"`
}

// Catalog devolve o catálogo de MCPs do v0.1 (ordem determinística).
//
// Fontes dos comandos (verificadas nas skills de setup do runtime e/ou
// em configs reais de máquina do operador — citadas por entrada):
//   - golmet: o próprio exe (`golmet mcp` — bloco 5).
//   - codegraph: binário instalado por `golmet tools install codegraph`
//     (DeusData/codebase-memory-mcp — bloco 3).
//   - figma: skill de setup do Figma ("npx", "-y", "figma-developer-mcp",
//     "--stdio") + FIGMA_API_KEY.
//   - playwright: skill de setup do Playwright ("npx", "-y",
//     "@playwright/mcp") — o contrato do install fixa @latest.
//   - postgres: skill de setup do Postgres ("npx", "-y",
//     "@microsoft/postgres-mcp", "run") — a senha é gerida por PROFILE do
//     próprio pacote (`connection set-password`), NÃO por env: sem
//     env_required (sigo a fonte, não o palpite do enunciado).
//   - hostinger: remote https://mcp.hostinger.com com OAuth pelo engine
//     (skill de setup do Hostinger).
//   - azure-devops (stdio): config REAL do operador, decodificado —
//     "npx", "-y", "@azure-devops/mcp", <ORG>, "--authentication", "pat"
//     com o PAT da env ADO_PAT_B64; <ORG> é Param do usuário.
//   - github (remote): github.blog changelog 2025-06-12 + README de
//     github/github-mcp-server — https://api.githubcopilot.com/mcp/,
//     OAuth 2.0 pelo engine (PAT opcional — sem env obrigatória).
//   - supabase (remote): config REAL do operador —
//     https://mcp.supabase.com/mcp, OAuth pelo engine.
//   - cloudflare (remote): config REAL do operador —
//     https://mcp.cloudflare.com/mcp, OAuth pelo engine.
//   - aws (stdio): config REAL do operador (AWS Agent Toolkit oficial) —
//     "uvx", "mcp-proxy-for-aws-cli@latest",
//     "https://aws-mcp.us-east-1.api.aws/mcp" + env AWS_PROFILE
//     (obrigatória — sem ela o proxy aborta; valor literal no config).
//   - sqlite (stdio, binário local): config REAL do operador
//     (rusq/mcp-sqlite v0.0.3) — binário em
//     %LOCALAPPDATA%\Programs\mcp-sqlite\, sem env, sem banco fixo.
//     Sem npx — resolveDef detecta o binário: presente registra o path
//     absoluto; ausente → pendência citando as releases do rusq/mcp-sqlite
//     (o kit NÃO instala esse no v0.1 — instalador próprio, diferente do
//     codegraph).
//   - context7 (remote): context7.com/docs (revisado 11/2026) —
//     https://mcp.context7.com/mcp, Streamable HTTP que funciona SEM
//     credenciais (initialize + tools/list OK; rate limits baixos).
//     CONTEXT7_API_KEY é OPCIONAL e recomendada (grátis no dashboard,
//     keys prefixo ctx7sk; header Authorization Bearer) — por isso fica
//     em EnvOptional, NUNCA em EnvRequired (registro funciona sem key).
func Catalog() []MCPDef {
	return []MCPDef{
		{
			Name:        "golmet",
			Description: "O próprio GOLMET ao alcance do agente: valida o projeto, mostra a saúde e indica o próximo passo",
			Type:        MCPTypeStdio,
			Command:     "@self", // resolvido no install para o exe corrente
			Args:        []string{"mcp"},
			Required:    true,
		},
		{
			Name:        "codegraph",
			Description: "O agente entende o seu código como um mapa: funções, relações e arquitetura sem procurar à mão",
			Type:        MCPTypeStdio,
			Command:     "@codegraph", // resolvido no install para ~/.golmet/tools/codegraph
			Args:        nil,
			Recommended: true,
		},
		{
			Name:        "azure-devops",
			Description: "Conecta o agente ao Azure DevOps: tarefas, repos, pipelines e PRs da organização",
			Type:        MCPTypeStdio,
			Command:     "npx",
			Args:        []string{"-y", "@azure-devops/mcp", "<ORG>", "--authentication", "pat"},
			Param:       "ORG",
			EnvRequired: []string{"ADO_PAT_B64"},
		},
		{
			Name:        "github",
			Description: "Conecta o agente ao GitHub: cria issues e pull requests, revisa código e acompanha actions",
			Type:        MCPTypeRemote,
			URL:         "https://api.githubcopilot.com/mcp/",
		},
		{
			Name:        "supabase",
			Description: "Conecta o agente ao Supabase: bancos, tabelas e login dos seus projetos",
			Type:        MCPTypeRemote,
			URL:         "https://mcp.supabase.com/mcp",
		},
		{
			Name:        "cloudflare",
			Description: "Conecta o agente à Cloudflare: domínios, DNS, workers e publicações",
			Type:        MCPTypeRemote,
			URL:         "https://mcp.cloudflare.com/mcp",
		},
		{
			Name:        "aws",
			Description: "Conecta o agente à AWS com as suas credenciais locais: consulta e usa serviços da nuvem",
			Type:        MCPTypeStdio,
			Command:     "uvx",
			Args:        []string{"mcp-proxy-for-aws-cli@latest", "https://aws-mcp.us-east-1.api.aws/mcp"},
			EnvRequired: []string{"AWS_PROFILE"},
		},
		{
			Name:        "sqlite",
			Description: "Deixa o agente abrir e consultar bancos SQLite locais (arquivos .db) quando precisar",
			Type:        MCPTypeStdio,
			Command:     "@sqlite", // resolvido no install para %LOCALAPPDATA%\Programs\mcp-sqlite
		},
		{
			Name:        "figma",
			Description: "O agente lê seus designs no Figma: telas, componentes e medidas para implementar o visual",
			Type:        MCPTypeStdio,
			Command:     "npx",
			Args:        []string{"-y", "figma-developer-mcp", "--stdio"},
			EnvRequired: []string{"FIGMA_API_KEY"},
		},
		{
			Name:        "playwright",
			Description: "Dá um navegador ao agente: ele abre páginas, testa telas e tira screenshots do que construiu",
			Type:        MCPTypeStdio,
			Command:     "npx",
			Args:        []string{"-y", "@playwright/mcp@latest"},
		},
		{
			Name:        "postgres",
			Description: "O agente explora bancos PostgreSQL: vê tabelas, schemas e roda consultas",
			Type:        MCPTypeStdio,
			Command:     "npx",
			Args:        []string{"-y", "@microsoft/postgres-mcp", "run"},
		},
		{
			Name:        "hostinger",
			Description: "Conecta o agente à Hostinger: servidores VPS, hospedagem e domínios da sua conta",
			Type:        MCPTypeRemote,
			URL:         "https://mcp.hostinger.com",
		},
		{
			Name:        "context7",
			Description: "O agente consulta documentação atualizada de bibliotecas e frameworks antes de responder",
			Type:        MCPTypeRemote,
			URL:         "https://mcp.context7.com/mcp",
			EnvOptional: []string{"CONTEXT7_API_KEY"},
		},
	}
}

// Lookup devolve a definição pelo nome (case-insensitive).
func Lookup(name string) (MCPDef, bool) {
	for _, def := range Catalog() {
		if strings.EqualFold(def.Name, name) {
			return def, true
		}
	}
	return MCPDef{}, false
}

// Names devolve os nomes do catálogo: kit primeiro (golmet, codegraph),
// depois serviços em ordem alfabética (ordem do --list-mcps).
func Names() []string {
	var kit, services []string
	for i, def := range Catalog() {
		if i < 2 {
			kit = append(kit, def.Name) // golmet, codegraph (ordem do catálogo)
			continue
		}
		services = append(services, def.Name)
	}
	sort.Strings(services)
	return append(kit, services...)
}

// MCPAttributes devolve ATÉ 2 atributos compactos do MCP (para a coluna
// dim da linha do wizard — nunca a frase de descrição):
//
//	1º tag (obrigatório/recomendado); 2º/3º env/param; além disso
//	remoto/binário local. Sem atributo → vazio (playwright/postgres nus).
func MCPAttributes(def MCPDef) []string {
	var out []string
	switch {
	case def.Required:
		out = append(out, "obrigatório")
	case def.Recommended:
		out = append(out, "recomendado")
	}
	if len(def.EnvRequired) == 1 {
		out = append(out, "env "+def.EnvRequired[0])
	}
	if def.Param != "" {
		out = append(out, "param "+def.Param)
	}
	if def.Type == MCPTypeRemote {
		out = append(out, "remoto")
	}
	if def.Command == "@sqlite" {
		out = append(out, "binário local")
	}
	if len(out) > 2 {
		out = out[:2]
	}
	return out
}

// ParseMCPSelection interpreta as entradas de --mcp: "nome" ou
// "nome=param" (o param é obrigatório para entradas com Param definido,
// ex.: azure-devops=MinhaOrg). Devolve as entradas normalizadas
// (golmet sempre presente) e erro para nome desconhecido ou = em MCP
// que não tem Param.
func ParseMCPSelection(selected []string) ([]SelectedMCP, error) {
	seen := map[string]bool{}
	var out []SelectedMCP
	for _, s := range selected {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		name, param := s, ""
		i := strings.Index(s, "=")
		if i >= 0 {
			name, param = s[:i], s[i+1:]
		}
		def, ok := Lookup(strings.TrimSpace(name))
		if !ok {
			return nil, fmt.Errorf("mcp %q não existe no catálogo — disponíveis: %s", s, strings.Join(Names(), ", "))
		}
		if def.Param != "" {
			// sem "=" → forma incorreta na flag (erro de uso; o wizard usa
			// "nome=" com valor vazio, que vira PENDÊNCIA no relatório)
			if i < 0 {
				return nil, fmt.Errorf("mcp %s exige o parâmetro %q — sintaxe: --mcp %s=<%s> (ex.: --mcp %s=MinhaOrg)",
					def.Name, def.Param, def.Name, def.Param, def.Name)
			}
			_ = param // valor vazio permitido aqui — Run classifica como pendência
		} else if param != "" {
			return nil, fmt.Errorf("mcp %s não aceita parâmetro (= %q) — use apenas o nome", def.Name, param)
		}
		if !seen[def.Name] {
			seen[def.Name] = true
			out = append(out, SelectedMCP{Name: def.Name, ParamValue: param})
		}
	}
	if !seen["golmet"] {
		out = append([]SelectedMCP{{Name: "golmet"}}, out...)
	}
	return out, nil
}

// SelectedMCP é uma entrada de --mcp normalizada (com o valor do Param
// quando a entrada exige).
type SelectedMCP struct {
	Name       string
	ParamValue string
}
