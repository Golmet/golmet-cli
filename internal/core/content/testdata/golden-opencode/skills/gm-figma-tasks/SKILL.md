---
name: gm-figma-tasks
description: >-
   Extrai o design real de um frame Figma via MCP (figma-developer-mcp), cruza com os docs do módulo e gera o delta de specs/UI_SPEC.md + propostas de RNs/CRs de UI + handoff de features candidatas (RN-âncora + RNs do escopo) para o gm-taskspec. Direção Figma → módulo — inversa da gm-figma-prompt. Fallback: notas do operador.
requires: [file, edit, write, glob, grep, ask]
degrades: [shell, webfetch]
---

# gm-figma-tasks — Design Figma → delta de UI_SPEC + handoff de features

Fecha o ciclo inverso do domínio Figma: em vez de gerar prompt PARA o Figma Make,
lê um design REAL (frame/arquivo no Figma) via MCP e o traduz para os artefatos
do módulo — delta de UI_SPEC.md, propostas de RNs/CRs de UI e um handoff de features
candidatas no formato que o `gm-taskspec` consome (RN-âncora + RNs do escopo).
A geração da análise (`analysis/RN-NNN — <título>/`) e das tasks
(`delivery/<grupo>/RN-NNN — <título> — Feature.md`) permanece papel exclusivo do
`gm-taskspec` (reuso — esta skill NÃO gera analysis/ nem delivery/).

## Diferenciação

| Componente | Direção | Função |
|---|---|---|
| `gm-figma-prompt` | módulo → Figma | Prompt textual para protótipo no Figma Make |
| `gm-design-brief` | módulo → Figma | Contexto de produto (identidade, estilo) para o Figma Make |
| **`gm-figma-tasks`** (esta) | **Figma → módulo** | Design real → delta de UI_SPEC + propostas RN/CR + handoff de features |
| `gm-taskspec` | docs → analysis/ + delivery | Gera `analysis/RN-NNN — <título>/` + `delivery/<grupo>/RN-NNN — <título> — Feature.md` (IMP-093; consumidor do handoff) |
| `gm-mcp-figma-setup` | infra | Wizard de token/conexão do MCP Figma |

## Pré-requisito — MCP Figma

- Server `figma` (`figma-developer-mcp`) configurado e habilitado — wizard: skill `gm-mcp-figma-setup`.
- Tools disponíveis com prefixo `figma_`: `figma_get_metadata`, `figma_get_code`,
  `figma_get_screenshot`, `figma_get_variable_defs`.
- **Fallback (MCP indisponível):** aceitar do operador um resumo do design em texto
  (colado no chat ou em `FIGMA_DESIGN_NOTES.md` na raiz do módulo) — sinalizar no
  handoff que a fonte foi "notas do operador" e não o design extraído via MCP.

## Resolução de caminhos (IMP-011)

- **Raiz:** PROJECT_CHARTER.md, CONSTITUTION.md, REQUIREMENTS.md, DECISIONS.md, CHANGELOG.md
- **`specs/`:** DATA_MODEL.md, API_CONTRACTS.md, UI_SPEC.md, ARCHITECTURE.md, FIGMA_PROMPTS.md, COMPLIANCE.md
- **`docs/`:** FIGMA_TASKS_HANDOFF.md e documentos auxiliares
- **`analysis/`:** pastas `RN-NNN — <título>/` (RN-NNN — Analise-Funcional.md +
  RN-NNN — Analise-Tecnica.md) — camada de análise gerada SOMENTE pelo
  gm-taskspec (IMP-092)
- **`delivery/<grupo>/`:** arquivos `RN-NNN — <título> — Feature.md` (IMP-093;
  grupo conforme o `delivery_style` da CONSTITUTION.md §9) — gerados SOMENTE
  pelo gm-taskspec
- Fallback de leitura: caminho novo primeiro; se não existir, caminho legado na raiz do módulo

## Lê (upstream)

- **MCP Figma** — a partir do link do node fornecido pelo operador (`https://www.figma.com/design/<fileKey>/<Nome>?node-id=<x-y>`):
  - `figma_get_metadata` — frames, nome do arquivo, estrutura de páginas
  - `figma_get_code` — composição/estrutura das telas (componentes, hierarquia)
  - `figma_get_screenshot` — referência visual dos estados
  - `figma_get_variable_defs` — tokens de design (cores, espaçamentos, tipografia)
- `<modulo>/<projeto>/specs/UI_SPEC.md` (fallback: raiz) — telas/componentes/estados já especificados
- `<modulo>/<projeto>/REQUIREMENTS.md` — RNs confirmadas (tag `[confirmado]`), seções de UI
- `<modulo>/<projeto>/specs/DATA_MODEL.md` (opcional) — campos para enriquecer o delta
- `<modulo>/<projeto>/specs/API_CONTRACTS.md` (opcional) — validações de frontend existentes
- `<modulo>/<projeto>/PROJECT_CHARTER.md` — repositórios do módulo (para o handoff indicar repo frontend)
- Fallback MCP: `FIGMA_DESIGN_NOTES.md` (raiz do módulo) ou resumo colado pelo operador

## Gera (downstream)

- **Delta de `specs/UI_SPEC.md`** — telas/componentes/estados a adicionar ou alterar; aplicado via `edit` SOMENTE após aprovação explícita do operador (permissão `ask` reforça)
- **`<modulo>/<projeto>/docs/FIGMA_TASKS_HANDOFF.md`** — append-only, com timestamp: delta proposto (quando não aplicado), propostas de RNs/CRs de UI e seção `## Features candidatas` no formato de handoff do gm-taskspec (que gerará a análise `analysis/RN-NNN — <título>/` + o arquivo de feature `delivery/<grupo>/RN-NNN — <título> — Feature.md` — IMP-092/IMP-093)
- **Relatório no chat** — resumo do cruzamento + próximo passo (invocar `gm-taskspec` com o handoff)

## Nunca edita

- **REQUIREMENTS.md** — propostas de RN/CR vivem APENAS no handoff; registro formal é papel do `gm-requirements` / `gm-change-request`
- **Código de aplicações** — esta skill é de documentação
- **O design no Figma** — somente leitura via MCP
- **Config do engine** (skills, agents, config JSON) — setup é papel da `gm-mcp-figma-setup`
- **O token FIGMA_API_KEY** — nunca logar, nunca escrever em arquivo
- **`gm-taskspec`, analysis/ e delivery/** — a geração de análise e tasks é
  dele (`analysis/RN-NNN — <título>/` + `delivery/<grupo>/RN-NNN — <título> — Feature.md`,
  IMP-092); esta skill só entrega o handoff
- **UI_SPEC.md sem aprovação** — o delta é proposto no chat/arquivo; aplicação só com confirmação explícita do operador

## Tools disponíveis

- `figma_get_metadata` / `figma_get_code` / `figma_get_screenshot` / `figma_get_variable_defs` — extrair o design do node
- `file` / `glob` / `grep` — ler os documentos do módulo
- `ask` — confirmar escopo (frame(s) alvo), aprovação do delta, RN-âncora da feature
- `write` — gravar `docs/FIGMA_TASKS_HANDOFF.md` (novo)
- `edit` — aplicar delta aprovado em `specs/UI_SPEC.md` (arquivo existente)

## Modo de erro

`warn` — a extração do design é a parte crítica: se o MCP falhar, a skill NÃO aborta —
cai para o fallback de notas do operador (com sinalização no handoff). Já a inconsistência
grave de requisitos (design extrapola RNs confirmadas) NÃO bloqueia: vira proposta de RN/CR
no handoff, para decisão do operador. Abortar (strict) apenas quando: nem MCP nem notas de
design existem, ou nenhum documento do módulo existe para cruzar.

## Formato de commit

```
docs(tasks): handoff de features do Figma (delta UI_SPEC + features candidatas) do módulo <nome>
```

## Workflow

### Fase 0 — Pré-requisitos

1. Confirmar que as tools `figma_*` estão disponíveis (server habilitado). Se não:
   informar e aplicar fallback (notas do operador) ou abortar com instrução de rodar `gm-mcp-figma-setup`.
2. Localizar o módulo: glob `**/REQUIREMENTS.md` (ou perguntar se houver mais de um).
3. Verificar `specs/UI_SPEC.md` (fallback: raiz) — existe ou será criação inicial?

### Fase 1 — Extrair o design

1. Pedir ao operador o link do node Figma (URL com `node-id`).
2. Extrair na ordem: `figma_get_metadata` (estrutura) → `figma_get_code` (composição) →
   `figma_get_screenshot` (visual) → `figma_get_variable_defs` (tokens).
3. Se o link vier sem `node-id`: pedir o frame específico (nunca extrair o arquivo inteiro sem confirmar — contexto).

### Fase 2 — Inventário do design

Montar o inventário a partir da extração:

| Item | Fonte MCP |
|---|---|
| Telas/frames | `figma_get_metadata` |
| Componentes e hierarquia | `figma_get_code` |
| Campos, textos, estados visuais | `figma_get_code` + `figma_get_screenshot` |
| Tokens (cores, espaçamento, tipografia) | `figma_get_variable_defs` |

### Fase 3 — Cruzar com os docs

1. **vs UI_SPEC.md:** classificar cada tela/componente do design como `novo`, `alterado` ou `sem mudança` → formação do delta.
2. **vs REQUIREMENTS.md:** para cada delta, identificar a RN de UI relacionada (`§5` RNs de UI / casos de uso). Design extrapola requisitos → marcar como `proposta de RN` (com pergunta de negócio que a originaria).
3. **vs DATA_MODEL.md / API_CONTRACTS.md:** enriquecer campos e validações de frontend do delta.
4. RNs confirmadas citadas pelo design mas sem tela no Figma → registrar como `gap` no handoff (requisito sem design).

### Fase 4 — Delta de UI_SPEC (com aprovação)

1. Apresentar o delta no chat, por tela: blocos a adicionar/alterar em UI_SPEC.md.
2. Perguntar ao operador: aplicar agora (edit) ou deixar apenas proposto no handoff.
3. Aplicar via `edit` SOMENTE o aprovado — preservar o restante do arquivo intacto.

### Fase 5 — Propostas de RN/CR

1. Para cada item marcado como `proposta de RN`: redigir a regra em linguagem de negócio + a pergunta de negócio que a originaria.
2. Se existir RN confirmada sendo ALTERADA pelo design → propor `CR-XXX` (papel formal do `gm-change-request`; aqui só a proposta).
3. Nunca numerar RN nova nem editar REQUIREMENTS.md — a numeração sequencial é do `gm-requirements`.

### Fase 6 — Features candidatas (handoff)

Agrupar os deltas aprovados/propostos em features candidatas, cada uma com:

- **Título da feature** (derivado do design/discussão; sanitizado — IMP-037)
- **RN-âncora sugerida**: primeira RN do escopo — PREFERIR RN confirmada existente; se o escopo depende de RN proposta, indicar a dependência explicitamente
- **RNs do escopo**: âncora + demais (confirmadas e propostas, sinalizadas)
- **Repositório frontend** (do PROJECT_CHARTER.md) e telas afetadas
- **Status**: `pronto para gm-taskspec` (só RNs confirmadas) ou `bloqueado — registrar RNs primeiro` (depende de proposta)
- **Saída que o gm-taskspec gerará (IMP-092)**: análise em
  `analysis/RN-NNN — <título>/` (Analise-Funcional + Analise-Tecnica) + tasks no
  arquivo `delivery/<grupo>/RN-NNN — <título> — Feature.md` — o handoff informa
  RN-âncora + RNs do escopo, nunca o caminho final dos artefatos

### Fase 7 — Gravar handoff + relatório

1. Gravar/atualizar `docs/FIGMA_TASKS_HANDOFF.md` (append-only: nova entrada timestampada por execução).
2. Relatório no chat: resumo do delta, propostas, features candidatas e próximo passo —
   invocar `gm-taskspec` (via `gm-quality-orchestrator` ou demanda direta) usando o handoff como insumo.
