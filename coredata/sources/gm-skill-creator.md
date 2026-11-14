# gm-skill-creator — Criação de Skill

Cria `skills/<nome>/SKILL.md` (no diretório de config do engine) — uma
skill completa, pronta para ser carregada via capacidade `skill`,
seguindo o padrão das skills existentes do workflow (gm-handover,
gm-charter, etc.).

## Diferenciação

| Skill | Cria |
|-------|------|
| `gm-subagent-creator` | Subagent (`mode: subagent`) — tarefa pontual, invocado via `@nome` |
| `gm-primary-agent-creator` | Primary agent (`mode: primary`) — sessão principal, permissão ampla |
| `gm-skill-creator` | Skill (`SKILL.md`) — conhecimento de referência sob demanda |

## Lê (upstream)

- `AGENTS.md` (config do engine) — convenções do workflow
- `ARCHITECTURE.md` (config do engine) — mapa de delegação (domínios e orquestradores — para registrar a skill no domínio certo)
- `skills/<exemplo>/SKILL.md` — padrões de estilo (ex: gm-handover, gm-charter)
- Skill `customize-opencode` — schema do front matter de skill (name, description, metadata)

## Gera (downstream)

- `skills/<nome>/SKILL.md` — a skill
- Atualiza `ARCHITECTURE.md` — diagrama (feito pelo `gm-workflow-builder`)

## Nunca edita

- Nenhum artefato de módulo, nenhum agente, nenhum código.

## Tools

- `ask` (coletar requisitos, confirmar plano)
- `glob` (verificar colisão de nome)
- `file` (exemplos, AGENTS.md, customize-opencode)
- `write` (criar o SKILL.md)
- `skill` (carregar customize-opencode)

## Modo de erro

`strict` — skill com front matter inválido (name ≠ pasta, description
ausente) não é carregada pelo engine.

## Questionário de requisitos

### Bloco 1 — Identificação (1 pergunta via `ask`)

- **Nome** (ex: `gm-coverage-audit`). Regras: lowercase-hyphen, ≤ 64
  chars, prefixo `gm-` (workflow) ou prefixo do projeto, único (verificar
  com `glob` no diretório de skills da config).
- **Propósito** (1-2 frases: o que a skill faz).

### Bloco 2 — Domínio e fluxo (1 pergunta)

- **Domínio/fase** do pipeline: Intake, Requisitos, Tradução Técnica,
  Qualidade, Suporte, Governança, Azure, Figma, Fábrica externa, Handover,
  ou transversal. (Skills de pipeline são acessadas via orquestradores de
  fase; skills transversais via demanda direta.)
- **Upstreams** (o que a skill lê) e **downstreams** (o que gera).
- **Pertence a um orchestrator?** Se sim, qual `@gm-*-orchestrator` o
  registra (o orchestrator precisa ser atualizado depois para listar a
  nova skill).

### Bloco 3 — Comportamento (1 pergunta)

- **Modo de erro**: `strict` (falha bloqueia o pipeline) vs `warn`
  (falha parcial não bloqueia). Exemplos: gm-charter é strict;
  gm-handover é warn.
- **Formato de commit** preferido (ex: `docs: ...` para artefatos).

## Template — skills/<nome>/SKILL.md

```markdown
---
name: gm-<nome>
description: >-
  <descrição em 1-3 linhas: o que a skill faz + quando usá-la. Front-load
  as keywords/nomes de arquivos que o usuário provavelmente vai dizer.>
---
# gm-<nome> — <título>

<parágrafo curto: o que a skill faz e para que responde>

## Diferenciação

<se houver skills parecidas, tabela comparando — senão omita>

## Lê (upstream)

- <o que a skill lê — arquivos e formatos>

## Gera (downstream)

- <o que a skill gera/atualiza>

## Nunca edita

- <o que está fora do escopo>

## Tools

- `glob` (<para quê>)
- `read` (<para quê>)
- `grep` (<para quê>)
- `ask` (<para quê>)
- `write` (<para quê>)

> **Escopo de tools:** cite apenas capacidades que a skill verifica
> disponíveis na sessão; capacidade indisponível ganha caminho alternativo
> (fallback) no Algoritmo — nunca há passos que só funcionam com a
> capacidade habilitada.

## Modo de erro

`<strict|warn>` — <o que acontece se falhar>

## Formato de commit

```
<exemplo de mensagem de commit>
```

## Algoritmo (ou Workflow)

### Passo 1 — <nome do passo>

<instrução detalhada com critérios verificáveis>

### Passo 2 — <nome do passo>

<instrução detalhada>

## Template de saída (se aplicável)

```<formato>
<template do documento/artefato gerado>
```

## Casos de Borda

- **<caso>**: <comportamento>
```

### Regras do template

1. **Front matter**: `name` (obrigatório, lowercase-hyphen, ≤ 64 chars,
   **deve casar com o nome da pasta**), `description` (obrigatória — skill
   sem description é filtrada e nunca é descoberta). Opcionais: `license`,
   `compatibility`, `metadata`.
2. **Estrutura**: use as seções do template acima (Lê/Gera/Nunca edita/
   Tools/Modo de erro/Formato de commit/Algoritmo ou Workflow/Template/
   Casos de Borda). PT-BR.
3. **Algoritmo verificável**: cada passo tem critério concreto — nunca
   instrução vaga.
4. **Nunca colocar `prompt:` no front matter** — o corpo do arquivo é o
   conteúdo da skill.
5. **Escopo de tools:** a skill direciona SOMENTE capacidades cuja
   disponibilidade ela mesma verifica na sessão — toda tool/MCP citado nas
   seções Tools e Algoritmo tem caminho alternativo documentado quando
   indisponível (cláusula de fallback padrão). A skill nunca direciona
   para uma call desabilitada.

## Workflow

1. Carregue `customize-opencode` (skill) para validar o shape do front matter.
2. Colete requisitos com o questionário (3 blocos, 1 pergunta cada).
3. Verifique colisão de nome com `glob` no diretório de skills da config.
4. Crie a pasta `skills/<nome>/` e escreva `SKILL.md`
   com o template completo (preenchido, não placeholder).
5. Valide o front matter: `name` == nome da pasta, `description` presente
   e com "Use quando..."/triggers.
6. **Escopo de tools:** confira que toda tool/MCP citada no
   SKILL.md tem guard de disponibilidade com caminho alternativo quando não
   garantida pela sessão (MCP opcional → cláusula de fallback no Algoritmo);
   tool hard-dependency sem guard → corrija antes de fechar.
7. **Fechamento com o orchestrator (obrigatório)**: se a skill pertence a
   um domínio com orchestrator (consultar `ARCHITECTURE.md` §3-§4):
   - Verifique se o orchestrator lista a skill na tabela "Skills
     disponíveis" do prompt dele (`agents/<orchestrator>.md`).
   - Se não lista, **atualize o prompt do orchestrator** (adicionar linha na
     tabela) — não apenas oriente o usuário.
   - Se o orchestrator não existe, reporte como "orchestrator a criar" —
     o `gm-workflow-builder` decide se encadeia o `gm-subagent-creator`.
8. Exiba o conteúdo para revisão do usuário.
9. **Gate de validação (obrigatório)** — execute a "Checklist para Nova
   Skill" do CHECKLISTS.md do workflow contra o SKILL.md criado. Todos os
   itens obrigatórios marcados, ou exceções justificadas explicitamente. O
   resultado (PASS + itens verificados / FAIL + itens violados) vai na
   seção "Validação" do IMP que esta criação registrará.
   Item obrigatório violado bloqueia o fechamento da criação.
10. Informe que o `gm-workflow-builder` fará o registro no
   `registry.yaml` e `ARCHITECTURE.md` e que o componente só valerá
   após reiniciar o engine.

## Formato de commit

```
chore(engine): add gm-<nome> skill
```

## Passo final — onde o componente vai viver

Depois de criar e validar a skill, apresente ao usuário as 3 histórias
legítimas de versionamento — o builder NÃO escolhe sozinho:

- **Opção A — engine pessoal (default):** a skill fica no config do engine
  onde foi criada (ex.: `~/.config/opencode/skills/<nome>/`). Uso pessoal
  imediato; versionamento só se você versionar o config por conta própria.
- **Opção B — zona do projeto (`.golmet/`):** mova para a zona `.golmet/`
  do workspace golmet e commite — o git do PROJETO passa a versionar e a
  equipe inteira herda ao clonar. Indicado quando a skill é específica
  daquele produto.
- **Opção C — pack (distribuição):** quando quiser levar a skill para
  outros projetos/máquinas/pessoas, empacote: sem pack ainda → execute a
  skill `gm-pack-scaffold` (cria o esqueleto e o remote via `gh repo
  create`); com pack existente → mova o componente para os `provides` do
  pack e publique com `gm-pack-publish`. Descoberta de packs da
  comunidade: `golmet pack search <termo>` (tópico GitHub `golmet-pack`).

**Regra de roteamento:** apresente as 3 opções com as consequências e
pergunte ao usuário (capacidade `ask` quando disponível; opções numeradas
em texto puro como alternativa) ANTES de mover/copiar qualquer arquivo.
Nunca mova sem confirmação.

## Casos de Borda

- **Nome já existe**: informe e pergunte — sobrescrever, renomear, cancelar.
- **Skill que pertence a orchestrator**: além do SKILL.md, atualize a
  tabela "Skills disponíveis" do orchestrator (passo 7 do Workflow) — o
  fechamento é obrigatório, não opcional.
- **Orchestrator do domínio não existe**: reporte como "orchestrator a
  criar" — o `gm-workflow-builder` encadeia o `gm-subagent-creator`
  (bottom-up) antes de registrar a skill no mapa.
- **Skill transversal**: sem orchestrator — acessada diretamente pelo
  primary agent (ex: gm-handover); registre no mapa §3 como acessada via
  `gm-req` ou `gm-doc` (demanda do usuário).
- **Skill de projeto específico**: use o prefixo do projeto (ex:
  `<projeto>-test-generator`) e não registre na seção do workflow no AGENTS.md.
