# gm-subagent-creator — Criação de Subagent

Cria `agents/<nome>.md` (no diretório de config do engine) — um subagent
completo, pronto para uso, seguindo o padrão dos subagents do workflow
(next, progress, cr-propagation-guard, etc.).

## Diferenciação

| Skill | Cria |
|-------|------|
| `gm-subagent-creator` | Subagent (`mode: subagent`) — tarefa pontual, invocado via `@nome` |
| `gm-primary-agent-creator` | Primary agent (`mode: primary`) — sessão principal, permissão ampla |
| `gm-skill-creator` | Skill (`SKILL.md`) — conhecimento de referência sob demanda |

## Lê (upstream)

- `AGENTS.md` (config do engine) — convenções do workflow
- `ARCHITECTURE.md` (config do engine) — mapa de delegação (skills por domínio, para inferir dependências)
- `registry.yaml` — manifesto canônico
- `agents/<exemplo>.md` — padrões de estilo (ex: next.md, cr-propagation-guard.md)
- Skill `customize-opencode` — schema do front matter (campos permitidos, modos, permissões)

## Gera (downstream)

- `agents/<nome>.md` — o subagent
- Atualiza `registry.yaml` e primary agents (`gm-req.md`, `gm-doc.md`, `gm-dev.md`) — tabela de subagents (feito pelo `gm-workflow-builder`)
- Atualiza `ARCHITECTURE.md` — diagrama (feito pelo `gm-workflow-builder`)

## Nunca edita

- Nenhum artefato de módulo, nenhum outro agente, nenhum código.

## Tools

- `ask` (coletar requisitos, confirmar plano)
- `glob` (verificar colisão de nome)
- `file` (exemplos, AGENTS.md, customize-opencode)
- `write` (criar o arquivo do subagent)
- `skill` (carregar customize-opencode)

## Modo de erro

`strict` — subagent com front matter inválido quebra o startup do engine.

## Questionário de requisitos

### Bloco 1 — Identificação (1 pergunta via `ask`)

- **Nome** (ex: `gm-coverage-audit`). Regras: lowercase-hyphen, ≤ 64
  chars, prefixo `gm-` (workflow) ou prefixo do projeto, único (verificar
  com `glob` no diretório de agents da config).
- **Propósito** (1-2 frases: o que faz, para quê serve).

### Bloco 2 — Comportamento (1 pergunta)

- **O que faz**: entrada (lê o quê), saída (gera o quê), decisão (algoritmo).
- **Escopo de permissão** — escolha um dos perfis:

| Perfil | Uso típico | Permissão |
|--------|-----------|-----------|
| **Read-only** | Diagnóstico, recomendação, dashboards (next, progress) | read/glob/grep allow; edit/write/shell deny |
| **Escrita restrita** | Guard que persiste 1 arquivo específico (cr-propagation-guard) | edit/write apenas no caminho-alvo; shell deny |
| **Orquestrador** | Roteia para skills (gm-azure-orchestrator) | read/glob/grep allow; edit/write ask ou deny; shell ask |
| **Escrita em config** | Edita a configuração do próprio workflow (gm-workflow-improver) | edit/write restritos ao diretório de config do engine; shell deny |

- **Skills que o subagent invoca** (obrigatório para orquestradores e
  subagents que executam skills): liste as skills que o subagent vai
  carregar via capacidade `skill` para cumprir a função. Consulte o
  `ARCHITECTURE.md` §3 (skills por domínio) para selecionar as
  existentes. **Verifique a existência de cada uma com `glob`** no
  diretório de skills da config. Se alguma não existir, reporte como
  **"skill a criar"** — o `gm-workflow-builder` decide se encadeia o
  `gm-skill-creator` (Fase 4 do fluxo dele).

### Bloco 3 — Registro (1 pergunta)

- Confirmar se será registrado no `registry.yaml`, primary agents e no `ARCHITECTURE.md` (padrão: sim).

## Template — agents/<nome>.md

```markdown
---
name: gm-<nome>
description: >-
  <descrição em 1-3 linhas: o que faz + quando invocar. Front-load
  keywords e nomes de arquivos que o usuário provavelmente vai dizer.>
mode: subagent
temperature: 0.1
permission:
  read: allow
  glob: allow
  grep: allow
  skill: <allow se invoca skills; senão omita>
  edit: <allow|deny|ask|{caminho: allow, "*": deny}>
  write: <allow|deny|ask|{caminho: allow, "*": deny}>
  bash: <deny|ask>
---
Você é o **GOLMET <Nome Formatado>**. <propósito em 1-2 frases>.

## Princípios

1. <princípio 1>
2. <princípio 2>

## Lê (upstream)

- <o que este subagent lê — caminhos e formatos>

## Gera (downstream)

- <o que este subagent gera/persiste>

## Nunca edita

- <o que está fora do escopo>

## Skills disponíveis (se invoca skills)

| Skill | Quando usar |
|-------|-------------|
| `gm-<skill-1>` | <quando> |
| `gm-<skill-2>` | <quando> |

## Tools disponíveis

- `read` — <para quê>
- `glob` — <para quê>
- `grep` — <para quê>
- `skill` — <para quê: carregar as skills listadas acima>
- `ask` — <para quê>

## Algoritmo

### Passo 1 — <nome do passo>

<instrução detalhada com critérios verificáveis>

### Passo 2 — <nome do passo>

<instrução detalhada>

## Formato de saída

```text
═══ gm-<nome> — <título> ═══
<saída estruturada>
```

## Casos de Borda

- **<caso>**: <comportamento>
```

### Regras do template

1. **Front matter**: `name` (obrigatório, casa com o nome do arquivo),
   `description` (obrigatória, efetiva), `mode: subagent`, `temperature: 0.1`
   (subagents de precisão) ou 0.2 (criativos). Campos adicionais só se
   necessário (`model` para delegar a modelo específico).
2. **Permissões**: menor privilégio. `edit`/`write` **nunca** `allow` amplo
   para subagent — sempre `deny`, `ask` ou restrito a caminho.
3. **Front matter proibido**: NUNCA use `tools:` no front matter — este
   campo **não existe** na schema do engine. O campo correto é
   `permission:` com granularidade `allow`/`deny`/`ask` + paths. Se vir
   `tools:` em algum exemplo externo, ignore — é padrão de outro framework.
4. **Heading do corpo**: a seção de ferramentas no corpo do prompt DEVE ser
   `## Tools disponíveis` (informativo) — nunca `## Tools`. O enforcement
   real está no `permission:` do front matter.
5. **Prompt**: sempre nas seções do template (Princípios, Lê/Gera/Nunca
   edita, Tools disponíveis, Algoritmo, Formato de saída, Casos de Borda). PT-BR.
6. **Algoritmo verificável**: cada passo tem critério concreto — nunca
   instrução vaga tipo "documente bem".

## Workflow

1. Carregue `customize-opencode` (skill) para validar o shape do front matter.
2. Colete requisitos com o questionário (3 blocos, 1 pergunta cada — nunca
   campo por campo).
3. Verifique colisão de nome com `glob` no diretório de agents da config.
4. **Verifique as skills invocadas** (se o subagent invoca skills): `glob`
   no diretório de skills da config para cada skill listada no Bloco 2.
   Faltantes → reporte como "skill a criar" (o builder encadeia o
   `gm-skill-creator`). Se o subagent invoca skills, o front matter deve
   ter `skill: allow`.
5. Escreva `agents/<nome>.md` com o template completo
   (preenchido, não placeholder).
6. Valide o front matter: `name` ≤ 64 chars lowercase-hyphen; `description`
   presente; `mode: subagent`; permissões com menor privilégio; `skill:
   allow` se invoca skills.
7. **Gate de validação (obrigatório)** — execute a "Checklist para Novo
   Subagent" do CHECKLISTS.md do workflow contra o arquivo criado (front
   matter, permissões sem path de máquina, seções, skills invocadas
   existentes, registro). Resultado (PASS + itens verificados / FAIL +
   violações) vai na seção "Validação" do IMP. Item obrigatório violado
   bloqueia o fechamento.
8. Exiba o conteúdo para revisão do usuário (leitura + resumo na resposta).
9. Informe que o `gm-workflow-builder` fará o registro em
   `registry.yaml`, `ARCHITECTURE.md` e primary agents e que o
   componente só valerá após reiniciar o engine.

## Formato de commit

```
chore(engine): add gm-<nome> subagent
```

## Passo final — onde o componente vai viver

Depois de criar e validar o subagent, apresente ao usuário as 3 histórias
legítimas de versionamento — o builder NÃO escolhe sozinho:

- **Opção A — engine pessoal (default):** o subagent fica no config do
  engine onde foi criado (ex.: `~/.config/opencode/agents/<nome>.md`). Uso
  pessoal imediato; versionamento só se você versionar o config por conta
  própria.
- **Opção B — zona do projeto (`.golmet/`):** mova para a zona `.golmet/`
  do workspace golmet e commite — o git do PROJETO passa a versionar e a
  equipe inteira herda ao clonar. Indicado quando o subagent é específico
  daquele produto.
- **Opção C — pack (distribuição):** quando quiser levar o subagent para
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
- **Subagent sem skills**: função executável diretamente (lê artefatos,
  diagnostica) — omita a seção "Skills disponíveis" e não force `skill: allow`.
- **Skills invocadas inexistentes**: reporte como "skill a criar" — nunca
  registre referência quebrada; o `gm-workflow-builder` fecha a cadeia
  (bottom-up) antes do registro.
- **Perfil de permissão ambíguo**: se a função escreve artefatos de módulo,
  revalide — subagent de execução edita código do repo; subagent de
  workflow edita só config; quem escreve REQUIREMENTS/specs é skill de
  domínio, não creator.
