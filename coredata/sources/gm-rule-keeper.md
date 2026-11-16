# gm-rule-keeper — Guardião de Normas do Pack (T3 — rules/)

Cria e edita **normas da organização** em `packs/<pack-ativo>/rules/` com o
FORMAT canônico e o fluxo seguro de escrita e publicação. É a skill
escritora da store T3 (o destino que `gm-knowledge-route` aponta quando o
conteúdo é norma que muda comportamento).

## Gatilho

- "Criar/editar uma norma da org", "nova regra para o pack"
- "Documentar como a organização faz X" (com gatilho de leitura acionável)
- `gm-knowledge-route`/`gm-knowledge-router` roteou para T3

## Fronteira — a skill NÃO decide a norma

O **conteúdo** da norma é decisão do operador/da org (o IMP ou a decisão de
origem é do processo próprio — `gm-decision-log` quando formalizar): esta
skill **garante o FORMAT e o fluxo seguro** — classificação de tema,
estrutura com gatilho, lazy index e publicação. Norma sem decisão de origem
clara → pergunte quem decide antes de escrever (norma é comportamento
obrigatório da org, não anotação).

## Lê (upstream)

- `packs/<pack-ativo>/rules/` — normas existentes (temas e seus gatilhos)
- O índice lazy do AGENTS.md do operador — gatilhos já
  referenciados (editar norma existente = gatilho existente; novo tema =
  linha nova no lazy index)
- O conteúdo da norma ditado pelo operador + a decisão/IMP de origem
- Bloco `<!-- gm-pack-accumulation -->` do AGENTS.md global — pack ativo

## Gera (downstream)

- `packs/<pack-ativo>/rules/<assunto>.md` — norma nova (arquivo novo) ou
  edit/append no arquivo do tema existente
- Linha do lazy index no AGENTS.md do operador (sob `ask` — pessoal-por-design)
- Lembrete de `golmet pack publish` (publicação é do operador — push manual
  por design; a skill NÃO publica sem confirmação)

## Nunca edita

- Norma existente sem confirmação do operador (sobrescrever/divergir de
  norma publicada é mudança de comportamento da org — sempre `ask`)
- Outras pastas do pack (`domains/`, `accumulation/**`) — escritores
  canônicos próprios
- `AGENTS.md` sem `ask` explícito — é pessoal-por-design; o kit não edita
  sem confirmação
- Componentes do workflow (`registry.yaml`, `ARCHITECTURE.md`)

## Tools

- `file`/`glob`/`grep` — ler normas existentes e o lazy index (allow)
- `write`/`edit` — restritos ao working copy `packs/<pack-ativo>/rules/**`
  (allow) + `AGENTS.md` do operador **sob `ask`** (única escrita fora do
  pack, sempre confirmada)
- `shell` — `golmet pack publish` com confirmação do operador (nunca
  executa o publish por conta própria)

## Workflow

### Passo 1 — Classificar o assunto

Globe `packs/<pack-ativo>/rules/*.md` e decida:

- **Norma existente cobre o tema** → edit/append no arquivo do tema (ex.:
  `org.md`, `deploy.md`, `vps.md`, `observability.md` — um arquivo por
  assunto). Preserve os itens existentes (norma é merge incremental —
  remover item de norma é decisão de org, com `ask`).
- **Tema novo** → crie `packs/<pack-ativo>/rules/<assunto>.md`
  (assunto em kebab-case, nome estável — o lazy index referencia o path).
- **NORMA NACIONAL — uma norma por arquivo, nunca monólito**: a lição da
  org é o rule monolítico de 544 linhas que precedeu a divisão — virou
  manutenção inviável e gatilho de leitura não-acionável. Se o tema novo
  "cabe" num arquivo existente só por conveniência, prefira arquivo novo
  com gatilho próprio.

### Passo 2 — Escrever no FORMAT canônico

```markdown
# <assunto> — <o que a norma governa>

> Gatilho: AO <fazer X / decidir Y>, leia este arquivo ANTES de agir
> (quando invocado: <cenas de uso concretas>).

<conteúdo prescritivo — como a organização faz; imperativo, não descritivo>

<exceções e limites explícitos>

## Referências cruzadas

- <norma irmã> (<path relativo>) — <quando a pair dela se aplica>
```

Regras do formato:

- **Header H1** com o tema + **gatilho explícito** ("AO fazer X, leia
  isto") — o gatilho É a norma: sem ele o arquivo é docs, não rule (nada
  o carrega na hora certa)
- **Conteúdo prescritivo** — imperativo ("use", "nunca", "sempre"), não
  descritivo ("a org costuma")
- **Referências cruzadas** a normas irmãs — o lazy index aponta o entry
  point; a norma conecta os vizinhos
- Uma norma por arquivo (Passo 1) — gatilho acionável por tema

### Passo 3 — Lazy index (sob `ask`)

O AGENTS.md do operador é **pessoal-por-design** — o kit não edita sem
confirmação:

1. Verifique se a linha do lazy index de `rules/` já referencia o arquivo.
2. Existe e o gatilho continua válido → nada a fazer.
3. Não existe (norma nova) → proponha a linha (path + gatilho "AO ...") e
   edite o AGENTS.md **só após confirmação** do operador.
4. Sem confirmação → devolva a linha pronta no relatório (pendência
   declarada — a norma sem lazy index não é carregada na hora certa).

### Passo 4 — Publicar (com confirmação)

`golmet pack publish` **com confirmação explícita do operador** — a
publicação é push manual por design (a skill não empurra por conta
própria). Lembre: o que não publica não alcança a org.

### Passo 5 — Reportar

```text
═══ gm-rule-keeper ═══
norma: <path no pack> | <arquivo do tema editado>
ação: criada | editada (o que mudou — 1 linha)
gatilho: <AO ...>
lazy index: atualizado (confirmado) | linha proposta (pendente — sem ask ok)
decisão de origem: <IMP/DEC/pergunta respondida pelo operador>
publish: executado (confirmado) | lembrete emitido
```

## Modo de erro

`continue` — falha de escrita/publicação não bloqueia o fluxo: a norma
redigida segue com o operador (devolva o conteúdo + o passo pendente).

- **Norma existente divergente**: NUNCA sobrescreve sem confirmação —
  mostre o delta (o que muda de comportamento) e `ask`; sem ok, devolve a
  proposta como diff.
- **Sem decisão de origem clara** (quem decidiu a norma?): pergunte antes
  de escrever — a skill formata e publica, não legisla.
- **Gatilho fraco/genérico** ("leia quando relevante"): reescreva com o
  operador em forma "AO fazer X" — gatilho não-acionável vira rule morta.
- **Pack ativo não configurado**: pergunte; sem resposta → devolva a
  norma pronta com o destino relativo sugerido.

## Casos de Borda

- **Norma que conflita com norma irmã** (org.md vs deploy.md): NÃO edite
  nenhuma — reporte o conflito ao operador para decidir qual muda (norma é
  decisão de org; o keeper executa a decisão, não a toma).
- **Conteúdo que parece norma mas é fundamento teórico** ("como funciona o
  CAP") → roteia para T1 (`gm-domain-research`) — rule é comportamento
  prescritivo com gatilho, não conhecimento perene (ver `gm-knowledge-route`).
- **Norma temporária/experimental** → marque no header ("experimental até
  <data>/decisão") — o lazy index só ganha linha permanente quando
  promovida.
- **Segredo/credencial no conteúdo da norma** → NUNCA; a norma descreve o
  comportamento ("token no ADO_PAT_B64, nunca versionado"), o valor vive
  na env var do operador.
- **Duas orgs/packs na máquina**: confirme o pack ativo — norma escrita no
  pack errado é pior que norma não escrita.
