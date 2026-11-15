---
name: gm-nuance-collector
description: >-
   Coletor de nuances (T2 — acúmulo do pack) — registra descobertas e observações específicas DURANTE o trabalho ("descobrimos que X falha quando Y", "funcionou diferente com Z") na hora, com contexto, em packs/<pack-ativo>/accumulation/nuances/<tema>-nuances.md — um arquivo por TEMA; cada nuance = seção leve com contexto (projeto/o que estava sendo feito), observação, aplicação/prática e data_observacao; tema novo cria o arquivo. Append incremental — nunca deleta o histórico do tema. Diferença de rota explicitada: PATTERN é reutilizável com implementação compilável, formato pesado, nasce na retrospectiva (gm-pattern-extractor); NUANCE é observação específica, formato leve, imediata. Nuance que evolui para padrão → sugere promover via gm-pattern-extractor. Lembre golmet pack publish (push manual por design). Escrita restrita a nuances/** do pack ativo; modo de erro continue. Use quando o operador quiser registrar uma descoberta ou observação com contexto na hora.
requires: [file, write, edit, glob, grep, ask]
degrades: [shell, webfetch]
---

# gm-nuance-collector — Coletor de Nuances (T2 — acúmulo do pack)

Registra descobertas e observações específicas **DURANTE o trabalho** — na
hora, com contexto — no `accumulation/nuances/` do pack ativo da
organização. É o contraponto leve e imediato da pattern library: a nuance
nasce do dia-a-dia; o padrão nasce da retrospectiva.

## Gatilho

- "Descobrimos que X falha quando Y" — armadilha encontrada em execução
- "Funcionou diferente com Z" — comportamento inesperado de tool/processo
- Qualquer observação com contexto que vale registrar **agora** (não espera
  a retrospectiva do projeto)

## Diferença de rota (evitar conflito com a pattern library)

| | PATTERN (gm-pattern-extractor) | NUANCE (esta skill) |
|---|---|---|
| O que é | Solução reutilizável com implementação compilável | Observação específica com contexto |
| Formato | Pesado (template completo, tipo, maturidade, evidência) | Leve (seção curta por nuance) |
| Quando nasce | Retrospectiva de projeto concluído | Imediata, durante o trabalho |
| Store | `accumulation/patterns/<tipo>/<slug>/` | `accumulation/nuances/<tema>-nuances.md` |

Nuance que evolui para solução reutilizável → **sugira promover** via
`gm-pattern-extractor` (a nuance continua no histórico; o padrão referencia
a origem).

## Resolução de destino (IMP-061/IMP-111 — T2, genérico por organização)

Todos os `accumulation/nuances/` referidos nesta skill são relativos ao
**`accumulation/` do pack ativo da organização** (ex.:
`packs/<pack-ativo>/accumulation/` — resolução via `packs:` do
client-profile), sinalizado pelo bloco de acúmulo do AGENTS.md global:

1. Bloco presente (pack ativo configurado) → destino =
   `packs/<pack-ativo>/accumulation/nuances/<tema>-nuances.md`.
2. Bloco ausente → pergunte ao usuário qual é o pack ativo (instalado em
   `packs/<id>/`) e sugira configurar o bloco. **Nunca** use o diretório de
   docs da config do engine como destino — a config do workflow é
   genérica/portável, sem conteúdo de negócio.
3. Exceção sandbox de eval: resolva os paths relativos à raiz do sandbox.

## Lê (upstream)

- A observação/descoberta + contexto do pedido (projeto, o que estava
  sendo feito, evidência quando houver)
- `packs/<pack-ativo>/accumulation/nuances/` — temas existentes (ex.:
  `knowledge-management-nuances.md`, `reverse-engineering-nuances.md`) para
  decidir append em tema existente vs arquivo novo
- Bloco `<!-- gm-pack-accumulation -->` do AGENTS.md global — pack ativo

## Gera (downstream)

- Append em `packs/<pack-ativo>/accumulation/nuances/<tema>-nuances.md` —
  **um arquivo por TEMA**; cada nuance é uma seção nova ao final (append
  incremental — nunca reescreve nem remove o histórico do tema)
- Tema novo = cria o arquivo `<tema>-nuances.md` (tema em kebab-case)
- Lembrete ao operador: publique via `golmet pack publish` (o acúmulo é
  dado do working copy do state pack — sobe para a org só no publish)

## Nunca edita

- Histórico do tema — append incremental: nunca reescreve, move ou remove
  nuances existentes (nunca delete)
- Retrospectivas, código-fonte, artefatos de módulo (read-only)
- Outras pastas do pack (`domains/`, `accumulation/patterns/`,
  `accumulation/metrics/`, `rules/`) — escritores canônicos próprios
- Config do engine (destino de negócio é o pack, nunca a config)

## Tools

- `file`/`glob` — ler o tema existente e a pasta de nuances (allow)
- `write`/`edit` — restritos a `packs/<pack-ativo>/accumulation/nuances/**`
  (allow — é dado do working copy do pack; único destino de escrita)
- `ask` — tema existente vs novo quando ambíguo; pack ativo não configurado

## Workflow

### Passo 1 — Resolver destino (pack ativo + tema)

Bloco de acúmulo presente? Pack ativo configurado? Globe
`accumulation/nuances/*-nuances.md` e decida:

- **Tema existente encaixa** (o domínio da observação é o do arquivo) →
  append nele.
- **Tema novo** → crie `<tema>-nuances.md` com o header do template.
- Ambíguo entre dois temas → `ask` (never invente tema com nome próximo de
  um existente — temas duplicados fragmentam o histórico).

### Passo 2 — Redigir a nuance (formato leve)

```markdown
## <slug curto da observação> (<data AAAA-MM-DD>)

- **Contexto:** <projeto + o que estava sendo feito — 1-2 linhas>
- **Observação:** <o que foi descoberto/observado — o fato com dente>
- **Aplicação/prática:** <o que fazer diferente da próxima vez — acionável>
- **data_observacao:** <AAAA-MM-DD>
```

Regras do formato:

- Uma seção por nuance — a observação específica é o conteúdo, não generalize
  (generalização com implementação é caminho para a pattern library)
- `data_observacao` obrigatória (staleness por item, mesmo espírito do MP-03)
- Evidência citável? (arquivo/commit/erro) → inclua no contexto
- Append sempre ao FINAL do arquivo — histórico crescente

### Passo 3 — Checar promoção para padrão

A observação generaliza (resolve um cenário recorrente, tem implementação
reutilizável)? → **sugira** ao operador promover via `gm-pattern-extractor`
(o padrão novo referencia a nuance de origem; a nuance permanece).

### Passo 4 — Publicar

Lembre o operador: `golmet pack publish` — o acúmulo vive no working copy
do state pack e só alcança a org no publish (push manual por design; a
skill NÃO publica).

### Passo 5 — Reportar

```text
═══ gm-nuance-collector ═══
tema: <arquivo de destino>
ação: append | arquivo novo
nuances no tema: N (antes → depois)
promoção p/ padrão: sugerida | não aplicável
publish: lembrete emitido
```

## Modo de erro

`continue` — falha ao registrar não bloqueia o trabalho corrente: reporte e
siga (a observação fica com o operador para registrar depois).

- **Pack ativo não configurado**: pergunte; sem resposta → devolva a
  nuance redigida ao operador com o destino relativo sugerido
  (`packs/<id>/accumulation/nuances/<tema>-nuances.md`) — nada gravado.
- **Arquivo do tema ilegível/corrompido**: NÃO reescreva — reporte e
  devolva a nuance pronta para append manual.
- **Escrita falha** (permissão): reporte o path e a nuance pronta; modo
  continue.

## Casos de Borda

- **Observação que contradiz nuance existente do tema** → append a nova
  MESMO ASSIM (histórico é append-only) e marque na seção
  "supera a nuance de <data>" — a invalidação formal por contradição
  (P-05/MP-13) é da pattern library; em nuances, o registro cronológico
  com referência cruzada basta.
- **Mesma observação em dois temas** → escolha o tema dominante e cite o
  outro no contexto — nunca duplique a seção inteira.
- **Nuance sensível (credencial/segredo)** → NUNCA registre o valor;
  registre o comportamento ("o token do ambiente X expira em 1h") sem o
  segredo.
- **Tempestade de nuances no mesmo tema** (3+ no mesmo dia) → sinalize ao
  operador que o tema merece fundação T1 (`gm-domain-research`) ou padrão
  (`gm-pattern-extractor`) — o volume é o sintoma.
