---
name: gm-mcp-context7-setup
description: >-
   Wizard de configuração do MCP Context7 no OpenCode — server REMOTE oficial (Streamable HTTP em https://mcp.context7.com/mcp), funciona SEM credenciais (initialize + tools/list OK; rate limits baixos); API key OPCIONAL e recomendada (grátis em context7.com/dashboard, prefixo ctx7sk) via header Authorization Bearer. Duas rotas: remota (recomendada — sem Node.js) e local (npx @upstash/context7-mcp, Node 20+, key via CONTEXT7_API_KEY); rota automática oficial `npx ctx7 setup --opencode` citada como alternativa. Expõe resolve-library-id e query-docs; a regra de COMPORTAMENTO (quando consultar docs de libs) vive em profile/context7.md do operador — esta skill cuida da INFRAESTRUTURA do server. Modo de erro continue (setup de MCP não bloqueia o fluxo). Idempotente e executado por demanda; delegado pelo gm-setup. Use quando o usuário pedir "configurar MCP Context7", "setup context7", "conectar context7 ao opencode" ou "docs de libs no agente".
requires: [file, edit, glob, grep, shell, ask]
---

# gm-mcp-context7-setup — Wizard de configuração do MCP Context7

Integra a documentação atualizada de bibliotecas ao OpenCode via MCP
**remoto oficial** (`https://mcp.context7.com/mcp`, Streamable HTTP) —
**funciona SEM credenciais** (initialize + tools/list OK, com rate limits
baixos); a API key é **OPCIONAL e recomendada** (grátis em
context7.com/dashboard, keys com prefixo `ctx7sk`) para rate limits
maiores, enviada como header `Authorization: Bearer <key>`. Ferramentas
expostas: `resolve-library-id` e `query-docs`.

Fronteira de responsabilidade: a regra de **COMPORTAMENTO** (quando
consultar docs de library/framework/SDK/API/CLI/cloud — ANTES de confiar
na memória do modelo) vive em `profile/context7.md` do operador (índice
lazy do AGENTS.md). Esta skill cuida da **INFRAESTRUTURA** do server —
config, key opcional e smoke test. Sem key, avise o operador sobre os
rate limits baixos (nunca é bloqueio — o server funciona sem key).

- Idempotente: pode ser reexecutado a qualquer momento — passos já
  satisfeitos são apenas confirmados.
- Escopo padrão: **GLOBAL** (`~/.config/opencode/opencode.jsonc`),
  convenção das skills irmãs de MCP setup (server agnóstico a repo).
  Escopo projeto só sob pedido explícito.
- Modo de erro: **continue** — setup de MCP não bloqueia o fluxo do
  workspace; falha vira pendência registrada no relatório.

## Por que remote (e as rotas alternativas)

| Opção | Veredito | Motivo |
|-------|----------|--------|
| Remote `https://mcp.context7.com/mcp` | ✅ **padrão** | Sem Node.js, sem binário local; funciona sem key (rate limits baixos); key opcional no header Bearer |
| Rota automática oficial `npx ctx7 setup --opencode` | alternativa | OAuth + key + instalação em 1 comando (`npx ctx7 remove` desfaz) — fora do wizard do kit, que mantém o controle/auditoria do opencode.jsonc; revisar o config resultante se usada |
| Local `npx -y @upstash/context7-mcp` | alternativa | Exige **Node.js ≥ 20**; key via `--api-key` ou env `CONTEXT7_API_KEY` — só em ambiente sem acesso ao remote |
| OAuth `mcp.context7.com/mcp/oauth` | alternativa | Fluxo OAuth do Context7 (só remote) — se preferir OAuth a key |

## Lê (upstream)

- `~/.config/opencode/opencode.jsonc` — config global (bloco
  `mcp.servers.context7` em V2 ou `mcp.context7` legado em V1 flat;
  checar duplicidade)
- Saída de `opencode mcp list` — server `context7` já existe?
- Env var `CONTEXT7_API_KEY` — apenas EXISTÊNCIA (para a rota local);
  NUNCA imprimir o valor
- `profile/context7.md` do operador — regra de comportamento (quando
  consultar docs; separada desta skill por design)

## Gera (downstream)

- Entrada `mcp.servers.context7` no opencode.jsonc global (V2) — com ou
  sem o header `Authorization` (key é opcional)
- Env var de usuário `CONTEXT7_API_KEY` (rota local, opcional — o valor
  vem do operador, nunca é ecoado)
- Pendência registrada se o smoke test falhar (modo continue — não
  bloqueia o restante do setup do workspace)

## Nunca edita

- Código de aplicações
- Entradas `mcp` de outros servers (azure-devops, playwright, figma,
  postgres, sqlite, codebase-memory-mcp, hostinger) — adiciona/ajusta
  SOMENTE o bloco `context7`
- `profile/context7.md` — regra de comportamento é do lazy index do
  operador; esta skill não a cria nem a edita
- **NENHUM arquivo com o valor da key** — proibido escrever
  `CONTEXT7_API_KEY` ou a key `ctx7sk...` em opencode.json(c) versionado,
  .env, docs, scripts, git. No remote, a key (se usada) fica no header do
  config GLOBAL pessoal; na rota local, EXCLUSIVAMENTE na env var do
  usuário. A skill NUNCA recebe, imprime, loga ou grava a key

## 1. Pré-requisitos

- **OpenCode V2** (CLI `opencode mcp` disponível)
- **Rede** para `https://mcp.context7.com/mcp`
- **(Rota local)** Node.js ≥ 20 (`node --version`)
- **(Key opcional)** conta no context7.com — a key é criada pelo
  OPERADOR no browser (Passo 4)
- **Reinício do OpenCode** após mudanças — config não é hot-reloaded

## 2. Verificar estado atual (idempotência)

```powershell
opencode mcp list
```

| Estado | Ação |
|--------|------|
| `context7` não aparece | Instalar (Passo 3) |
| `context7` presente e habilitado | Passo 4 (key opcional) + smoke test (Passo 6) |
| `context7` presente, desabilitado | Habilitar (Passo 3, target state) |

Checagem de duplicidade (V2 vs legado V1 no mesmo arquivo): se existirem
`mcp.servers.context7` **E** `mcp.context7`, manter UMA só (preferir a
V2) — server duplicado deixa o estado ambíguo.

## 3. Instalar o server — rota remota (recomendada, sem Node.js)

**Sem key (funciona imediatamente, rate limits baixos):**

```powershell
opencode mcp add context7 --global --url https://mcp.context7.com/mcp
```

**Target state no opencode.jsonc global (V2)** — o que o add grava (sem
key) ou o estado final esperado (com key, header do Passo 4):

```json
"mcp": {
  "servers": {
    "context7": {
      "type": "remote",
      "url": "https://mcp.context7.com/mcp",
      "headers": { "Authorization": "Bearer <key>" }
    }
  }
}
```

Em config V1 legado (flat), o formato equivalente é
`"mcp": { "context7": { "type": "remote", "url": ..., "headers": ...,
"enabled": true } }` — habilitado exige `"enabled": true` explícito.
Edite APENAS o bloco `context7` — demais servers intactos. Escopo
project-local só sob pedido explícito do operador.

## 4. Key OPCIONAL (recomendada) — passo do OPERADOR no browser

Sem key o server funciona com **rate limits baixos** — informe o
operador e deixe a decisão com ele (a skill nunca exige key para
concluir).

**Criação da key (OPERADOR, no browser — nunca pelo agente):**
1. Acessar `context7.com/dashboard` e criar a API key (prefixo
   `ctx7sk`)
2. Copiar a key e inseri-la em UMA das formas:
   - **Rota remota**: header `Authorization: Bearer <key>` do bloco
     `context7` no opencode.jsonc global (Passo 3) — colada PELO
     OPERADOR no editor
   - **Rota local**: env var de usuário `CONTEXT7_API_KEY` (Passo 5)

**Opção A — Interface gráfica (nada vai para histórico):**
`Win+R` → `sysdm.cpl` → Avançado → Variáveis de Ambiente → Novo →
Nome: `CONTEXT7_API_KEY`, Valor: cole a key → OK → reabra o terminal.

**Opção B — PowerShell (valor fora do histórico como literal):**

```powershell
$v = Read-Host -Prompt 'Cole a key context7 (entrada oculta nao existe no Read-Host; cole com cuidado)'
[Environment]::SetEnvironmentVariable('CONTEXT7_API_KEY', $v, 'User')
$v = $null  # descarta da memoria da sessao
```

**Proibições absolutas:**
- NUNCA `echo`, `Write-Host`, log ou screenshot da key
- NUNCA `setx CONTEXT7_API_KEY <valor>` com o valor inline
- NUNCA escrever a key em arquivo versionado, .env ou no chat
- Se o operador tentar COLAR a key no chat: recusar e orientar este passo

## 5. Rota local alternativa (npx, Node ≥ 20)

Só em ambiente sem acesso ao remote:

```json
"context7": {
  "type": "local",
  "command": ["npx", "-y", "@upstash/context7-mcp"],
  "enabled": true
}
```

A key vai via env var de usuário `CONTEXT7_API_KEY` (herdada pelo
processo do OpenCode — mesmo padrão de FIGMA_API_KEY) ou flag
`--api-key` do pacote (evite: grava a key no config). Requer
**Node.js ≥ 20**.

## 6. Smoke test (validação final)

Após **reiniciar o OpenCode** (config não é hot-reloaded):

1. **Tools na sessão** — as 2 tools do server aparecem no catálogo:
   `resolve-library-id` e `query-docs` (handshake do server OK).
2. **Consulta real** — `resolve-library-id` com o nome de uma lib
   popular (ex.: "react") → retorna o id da biblioteca; `query-docs`
   com o id + uma pergunta curta → retorna trecho da documentação.

**Fallback sem sessão de agente (conectividade):** o endpoint responde
a um `initialize` MCP sem key (HTTP 200 com corpo JSON-RPC):

```powershell
$body = '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"golmet-smoke","version":"0.0"}}}'
Invoke-RestMethod -Uri "https://mcp.context7.com/mcp" -Method Post -Body $body -ContentType "application/json"
```

Esperado: resposta JSON-RPC com `serverInfo` — sem 401/403 (sem key o
acesso é permitido; 429 é rate limit, não bloqueio de acesso).

## 7. Relatar estado final

Checklist de fechamento do wizard:

- [ ] `mcp.servers.context7` no opencode.jsonc global — demais servers intactos
- [ ] Sem duplicidade V1/V2 do server `context7`
- [ ] Key: decidida com o operador (com key no header Bearer / sem key
      com rate limits baixos avisados)
- [ ] Smoke test OK (2 tools na sessão ou initialize HTTP 200)
- [ ] **Reinício do OpenCode pendente?** — config não é hot-reloaded
- [ ] Falha em qualquer passo → pendência registrada, setup segue (modo continue)

## Ferramentas disponíveis

| Ferramenta | Uso | Exemplo |
|------------|-----|---------|
| `resolve-library-id` | Resolve o id da biblioteca no Context7 a partir do nome | nome: "react" → id da lib |
| `query-docs` | Consulta a documentação atualizada da lib (use a pergunta completa do usuário) | id da lib + pergunta → trechos de docs |

Padrão de uso: `resolve-library-id` → `query-docs`. O comportamento de
QUANDO usar (todo topic sobre library/framework/SDK/API/CLI/cloud,
mesmo em libs "conhecidas") está em `profile/context7.md` do operador —
esta skill não duplica a regra.

## Troubleshooting

| Erro | Causa | Solução |
|------|-------|---------|
| 401/403 na consulta | Key inválida/expirada no header ou env | OPERADOR regenera a key no dashboard e atualiza header/env (nunca colar a key no chat) |
| 429 (rate limit) | Uso sem key ou key gratuita atingiu quota | Orientar criar key no context7.com/dashboard (grátis) e inserir no header |
| Tools `context7` não aparecem | Sem reinício do OpenCode ou bloco desabilitado | Reiniciar; conferir o bloco no config (V1 flat exige `enabled: true`) |
| `npx: command not found` (rota local) | Node.js ausente/antigo | Instalar Node.js ≥ 20 — ou migrar para a rota remota (sem Node) |
| Timeout no add/initialize | Sem rota para o server | Verificar conectividade com `https://mcp.context7.com/mcp` (proxy corporativo → `HTTPS_PROXY`) |
| opencode.jsonc JSON inválido | Edição manual quebrou o config | Reverter o bloco `context7` e refazer o add via CLI |

## Modo de erro

- **Modo continue** — falha de setup de MCP NÃO bloqueia o fluxo do
  workspace: registrar pendência com o passo que falhou e seguir.
- **Operador tenta colar a key no chat/arquivo versionado: recusar e
  orientar** o Passo 4 (header do config global pessoal ou env var).
- **Server `context7` já existente:** idempotente — apenas validar
  estado (habilitado + smoke test); nunca duplicar a entrada.
- **opencode.jsonc corrompido (JSON inválido): abortar a edição e
  reportar** — nunca reescrever o config por conta própria.
- **Sem key:** prosseguir com rate limits baixos avisados — não é erro.

## Casos de borda

- **Rota automática oficial:** `npx ctx7 setup --opencode` faz OAuth +
  key + instalação em 1 comando (`npx ctx7 remove` desfaz). O wizard do
  kit prefere o caminho manual para manter o controle/auditoria do
  opencode.jsonc — se o operador usar o ctx7 CLI, revisar o config
  resultante (duplicidade, escopo) após rodar.
- **OAuth:** `mcp.context7.com/mcp/oauth` (só remote) — alternativa ao
  header Bearer para quem não quer key estática no config.
- **CI/automação:** key como secret do ambiente do runner (nunca em
  arquivo versionado); no remote, header injetado por env do runner.
- **Proxy corporativo:** o transporte remoto respeita `HTTPS_PROXY`;
  configurar env var de usuário se a rede exigir.
- **Troca de key:** OPERADOR substitui o valor no mesmo header/env —
  nunca criar segunda entrada MCP.

## Formato de commit sugerido

`feat(skills): add gm-mcp-context7-setup (MCP Context7 remote docs)`
