# gm-primary-agent-creator — Criação de Primary Agent

Cria `agents/<nome>.md` com `mode: primary` (no diretório de config do
engine) — um agente de sessão principal, no padrão do Quadri do workflow
(`gm-req.md`, `gm-doc.md`, `gm-dev.md`, `gm-devops.md`). Primary agents
conduzem a sessão, invocam subagents e têm permissão ampla (blast-radius
aceito e documentado no AGENTS.md).

## Diferenciação

| Skill | Cria |
|-------|------|
| `gm-subagent-creator` | Subagent (`mode: subagent`) — tarefa pontual, invocado via `@nome` |
| `gm-primary-agent-creator` | Primary agent (`mode: primary`) — sessão principal, permissão ampla |
| `gm-skill-creator` | Skill (`SKILL.md`) — conhecimento de referência sob demanda |

## Lê (upstream)

- `AGENTS.md` (config do engine) — convenções do workflow, modelo de permissão
- `agents/gm-req.md` / `gm-doc.md` / `gm-dev.md` — padrão de primary agent (referência principal)
- `ARCHITECTURE.md` (config do engine) — mapa de delegação (subagents por primary, skills por subagent — para inferir a árvore e o catálogo para a tabela do prompt)
- Skill `customize-opencode` — schema do front matter (mode, campos permitidos)

## Gera (downstream)

- `agents/<nome>.md` — o primary agent
- Atualiza `ARCHITECTURE.md` — diagrama (feito pelo `gm-workflow-builder`)

## Nunca edita

- Nenhum artefato de módulo, nenhum outro agente, nenhum código.

## Tools

- `ask` (coletar requisitos, confirmar plano)
- `glob` (verificar colisão de nome, listar subagents existentes)
- `file` (gm-req.md, gm-doc.md, gm-dev.md, exemplos, customize-opencode)
- `write` (criar o arquivo do primary agent)
- `skill` (carregar customize-opencode)

## Modo de erro

`strict` — primary agent com front matter inválido quebra o startup do engine.

## Questionário de requisitos

### Bloco 1 — Identificação (1 pergunta via `ask`)

- **Nome** (ex: `gm-req`, `gm-doc`). Regras: lowercase-hyphen, ≤ 64
  chars, prefixo `gm-` (workflow) ou prefixo do projeto, único (verificar
  com `glob` no diretório de agents da config).
- **Domínio de atuação** (1-2 frases: qual área do pipeline ou produto o
  agente conduz).

### Bloco 2 — Subagents e roteamento (1 pergunta)

- **Quais subagents existentes ele invoca?** Liste os `@gm-*` relevantes
  (consultar `ARCHITECTURE.md` §1 — primary→subagents — e o nível L4).
- **Há subagents novos que ele deve invocar?** Se sim, o fluxo é:
  criar subagent(s) primeiro (via `gm-subagent-creator`), depois o primary.
- **Inferência de skills por subagent**: para cada subagent que o primary
  delega, verifique no `ARCHITECTURE.md` §2 quais skills ele invoca.
  Se um subagent novo precisar de skills que não existem, reporte como
  "skill a criar" — o `gm-workflow-builder` encadeia o `gm-skill-creator`
  (Fase 4 do fluxo dele). O primary nunca referencia subagent/skill
  inexistente.

### Bloco 3 — Comportamento (1 pergunta)

- **Regras de sessão**: início de sessão (o que verificar primeiro),
  comportamento padrão, limites (ex: confirmar antes de commit/push/PR).
- **Permissão**: padrão é ampla (igual `gm-req`/`gm-doc`/`gm-dev`). Só restrinja se o
  usuário pedir explicitamente.

## Template — agents/<nome>.md

```markdown
---
description: <descrição em 1-2 linhas: domínio de atuação + quando usar>
mode: primary
temperature: 0.3
permission:
  read: allow
  edit: allow
  glob: allow
  grep: allow
  list: allow
  bash: allow
  write: allow
  task: allow
  todowrite: allow
  question: allow
  skill: allow
  webfetch: allow
  websearch: allow
  lsp: allow
  external_directory: allow
---

Você é o **<Nome>** — o agente primário para <domínio de atuação>.
Conduz a sessão principal quando o trabalho é <escopo>.

## Identidade e propósito

<parágrafo: o que o agente é e o que entrega>

## Regras de precedência

Siga a hierarquia documentada no AGENTS.md (seção "Hierarquia de
Precedência"): em conflito de instrução, a camada mais específica vence —
AGENTS.md Global → AGENTS.md do Projeto → Agent (primary) → Subagent →
Skill. Nunca contradiga as convenções do AGENTS.md: prefixo `gm-`, IDs
RN-XXX/PEND-XXX, DAG de documentos, idioma PT-BR.

## Início de sessão

<o que verificar no início de cada sessão — ex: kit golmet next para próxima ação>

### Subagents

| Subagent | Quando invocar |
|----------|----------------|
| `@gm-<nome>` | <quando> |
| ... | ... |

Invocação detalhada:

- `@gm-<nome>` — <detalhe de quando e como invocar>

### Roteamento de domínio

<quais skills/orquestradores o agente acessa e como>

## Comportamento

- <regra de comportamento 1>
- <regra de comportamento 2>
- Confirme com o usuário antes de ações irreversíveis (commit, push, PR).
```

### Regras do template

1. **Front matter**: `description` (obrigatória), `mode: primary`,
   `temperature: 0.3` (padrão do Quadri). **Não** usar `name:` no front
   matter de primary agents — a convenção do workflow não usa, e o
   nome vem do arquivo.
2. **Permissão ampla**: copie o bloco de permissões do `gm-req.md`
   (todas allow). É o modelo de permissão documentado no AGENTS.md.
3. **Prompt**: seções obrigatórias — Identidade e propósito, Regras de
   precedência, Início de sessão, Subagents (tabela + invocação detalhada),
   Roteamento de domínio, Comportamento. PT-BR.
4. **Tabela de subagents**: liste apenas os subagents que o agente
   realmente invoca — não copie a lista inteira do primary.

## Workflow

1. Carregue `customize-opencode` (skill) para validar o shape do front matter.
2. Colete requisitos com o questionário (3 blocos, 1 pergunta cada).
3. Verifique colisão de nome com `glob` no diretório de agents da config.
4. **Verifique a árvore de delegação**: para cada subagent que o primary
   invoca, confirme a existência com `glob` no diretório de agents
   e as skills dele no `ARCHITECTURE.md` §2. Faltantes → reporte como
   "a criar" (o builder encadeia `gm-subagent-creator`/`gm-skill-creator`).
5. Se o primary invoca subagents novos, crie-os primeiro via
   `gm-subagent-creator` (e informe o usuário).
6. Escreva `agents/<nome>.md` com o template completo.
7. Valide o front matter: `mode: primary`, `description` presente,
   permissões amplas, sem `name:` no front matter.
8. **Gate de validação (obrigatório)** — execute a "Checklist para Novo
   Primary Agent" do CHECKLISTS.md do workflow contra o arquivo criado
   (front matter, permissão ampla, seções, subagents existentes, registro).
   Resultado (PASS + itens verificados / FAIL + violações) vai na seção
   "Validação" do IMP. Item obrigatório violado bloqueia o fechamento.
9. Exiba o conteúdo para revisão do usuário.
10. Informe que o `gm-workflow-builder` fará o registro no
    `registry.yaml`/`ARCHITECTURE.md` e que o componente só valerá
    após reiniciar o engine.

## Formato de commit

```
chore(engine): add <nome> primary agent
```

## Passo final — onde o componente vai viver

Depois de criar e validar o primary agent, apresente ao usuário as 3
histórias legítimas de versionamento — o builder NÃO escolhe sozinho:

- **Opção A — engine pessoal (default):** o primary agent fica no config
  do engine onde foi criado (ex.: `~/.config/opencode/agents/<nome>.md`).
  Uso pessoal imediato; versionamento só se você versionar o config por
  conta própria.
- **Opção B — zona do projeto (`.golmet/`):** mova para a zona `.golmet/`
  do workspace golmet e commite — o git do PROJETO passa a versionar e a
  equipe inteira herda ao clonar. Indicado quando o primary agent é
  específico daquele produto.
- **Opção C — pack (distribuição):** quando quiser levar o primary agent
  para outros projetos/máquinas/pessoas, empacote: sem pack ainda →
  execute a skill `gm-pack-scaffold` (cria o esqueleto e o remote via
  `gh repo create`); com pack existente → mova o componente para os
  `provides` do pack e publique com `gm-pack-publish`. Descoberta de
  packs da comunidade: `golmet pack search <termo>` (tópico GitHub
  `golmet-pack`).

**Regra de roteamento:** apresente as 3 opções com as consequências e
pergunte ao usuário (capacidade `ask` quando disponível; opções numeradas
em texto puro como alternativa) ANTES de mover/copiar qualquer arquivo.
Nunca mova sem confirmação.

## Casos de Borda

- **Nome já existe**: informe e pergunte — sobrescrever, renomear, cancelar.
- **Primary agent sem subagents**: ok — omita a tabela e o roteamento,
  mantenha Identidade, Precedência e Comportamento.
- **Confusão subagent vs primary**: primary conduz sessão e tem permissão
  ampla; subagent é tarefa pontual com permissão escopada. Se o pedido for
  "agente que faz X pontualmente", é subagent.
- **Subagent invocado não existe**: reporte como "a criar" — o
  `gm-workflow-builder` encadeia o `gm-subagent-creator` antes de criar o
  primary (bottom-up). Nunca crie primary referenciando subagent inexistente.
- **Default agent**: se o usuário quiser que o novo primary seja o agente
  padrão do projeto, isso é config do engine
  (`default_agent`), não no arquivo do agente — oriente.
