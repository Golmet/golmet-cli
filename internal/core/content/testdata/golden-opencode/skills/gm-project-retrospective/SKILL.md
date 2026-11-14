---
name: gm-project-retrospective
description: >-
   Retrospectiva pós-projeto — captura aprendizados, padrões e métricas de cada projeto concluído (RETROSPECTIVE.md), com Scorecard do Workflow (IMP-048 — 14 segmentos, evidência obrigatória). Invoca gm-pattern-extractor (padrões → T2) e gm-metrics-collector (métricas → T2). Acúmulo no pack ativo (IMP-061/112).
requires: [file, write, glob, grep, ask, skill]
degrades: [edit, shell, webfetch]
---

# gm-project-retrospective — Retrospectiva Pós-Projeto (T2 driver)

Captura aprendizados de cada projeto para alimentar o ciclo de melhoria
contínua do workflow. Cada retrospectiva vira um documento em
`<docs-do-projeto>/RETROSPECTIVE.md` que alimenta a pattern library (T2)
e o tracker de métricas. Invoca `gm-pattern-extractor` para extrair padrões
automaticamente e `gm-metrics-collector` para acumular métricas.

## Lê (upstream)

- Estado do projeto documentado — aprendizados, decisões, artefatos gerados
- `WORKFLOW_IMPROVEMENTS.md` — IMPs registrados durante o projeto
- `DECISIONS.md` — ADRs tomados durante o projeto
- `registry.yaml` — skills disponíveis
- Código do repo do projeto (se acessível) — para métricas

## Gera (downstream)

- `RETROSPECTIVE.md` do projeto — documento da retrospectiva
- Invoca `gm-pattern-extractor` — extrai padrões automaticamente
- Invoca `gm-metrics-collector` — acumula métricas
- Paths de patterns e METRICS = acúmulo do pack ativo (T2:
  `packs/<pack-ativo>/accumulation/` — bloco de acúmulo do
  AGENTS.md — IMP-061/IMP-112); resolução feita pelas skills invocadas

## Nunca edita

- Código-fonte do projeto
- Artefatos de spec do módulo
- `WORKFLOW_IMPROVEMENTS.md` (read-only aqui — IMPs são registrados
  pelo improver, não pela retrospectiva)

## Tools

- `file`/`glob`/`grep` — inspecionar estado do projeto
- `ask` — coletar feedback do usuário
- `write` — gerar RETROSPECTIVE.md
- `skill` — invocar gm-pattern-extractor e gm-metrics-collector

## Modo de erro

`warn` — se a pasta do projeto não existir, crie a estrutura.
Se o projeto não tiver IMPs ou ADRs registrados, pule essas seções
com aviso.

## Workflow

### Passo 1 — Identificar o projeto

Pergunte ao usuário qual projeto está sendo retrospectado.
O nome deve corresponder ao diretório documentado ou ao nome do repo.
Se o projeto é novo, crie a estrutura:

```
<docs-do-projeto>/
└── RETROSPECTIVE.md
```

### Passo 2 — Coletar dados automaticamente

Leia automaticamente:
1. `WORKFLOW_IMPROVEMENTS.md` — filtre IMPs do período do projeto
2. `DECISIONS.md` — filtre ADRs do período do projeto
3. `registry.yaml` — conta skills ativas
4. Relatórios de eval existentes

### Passo 3 — Coletar feedback do usuário

Pergunte (uma chamada agrupada):

1. **O que funcionou bem?** (padrões, ferramentas, processos)
2. **O que não funcionou?** (problemas, gargalos, decisões erradas)
3. **Padrões novos descobertos?** (soluções reutilizáveis)
4. **Skills que faltaram?** (necessidades não atendidas)
5. **Métricas subjetivas?** (satisfação, complexidade, risco)

### Passo 4 — Gerar RETROSPECTIVE.md

Gere o documento com o template abaixo:

```markdown
# Retrospectiva — <Nome do Projeto>

**Data:** DD/MM/AAAA
**Projeto:** <nome> (<repo>)
**Duração:** <estimativa ou "N/A">
**Skills usadas:** <lista>

## O que funcionou

- <padrão/pattern que deu certo>
- <ferramenta que surpreendeu>
- <processo que fluiu bem>

## O que não funcionou

- <problema recorrente>
- <decisão que precisou ser revertida>
- <gargalo identificado>

## Padrões extraídos

- <novo pattern descoberto> → ver `patterns/<nome>/PATTERN.md`
- <convenção que surgiu naturalmente>

## Métricas do projeto

| Métrica | Valor |
|---------|-------|
| Skills usadas | N |
| Issues de qualidade | N |
| RNs implementadas | N |
| Quality gate final | PASS/WARN/FAIL |
| IMPs registrados | N |
| ADRs tomados | N |

## Ações para o workflow

| Ação | Tipo | Prioridade |
|------|------|------------|
| <melhoria na skill X> | IMP | alta |
| <novo pattern> | pattern | média |
| <checklist a ajustar> | IMP | baixa |

## Scorecard do Workflow (IMP-048)

Nota 0-10 por segmento, **somente com evidência do período** (sem uso = N/A,
nunca chute). Toda nota cita a evidência. Taxonomia fixa:

**Fases:** Intake · Requisitos & cliente · Tradução técnica · Entrega/tasks ·
Desenvolvimento · Verificação & qualidade · Suporte & pós-produção

**Transversais:** Governança da config · Meta-melhoria · Observabilidade &
eficiência (latência/contexto/custo) · Padrões & aprendizado · Segurança ·
Integrações externas · Portabilidade & onboarding

| Segmento | Tipo | Nota | Evidência |
|----------|------|:----:|-----------|
| <14 linhas, uma por segmento> | fase/transversal | 0-10 ou N/A | <fato do período ou "sem uso"> |

**Regras:** nota geral = média das fases COM evidência (transversais listadas
à parte); tendência vs milestone anterior vai para o METRICS.md do pack
(§Scorecard do Workflow, append-only via `gm-metrics-collector`).
```

### Passo 4b — Scorecard do Workflow (IMP-048)

Preencha a seção `## Scorecard do Workflow` do RETROSPECTIVE.md:
1. Para cada um dos 14 segmentos, busque evidência no período (artefatos
   gerados, evals, métricas, incidentes). Sem evidência → N/A.
2. Nota geral = média das FASES com evidência; transversais informam contexto.
3. Append da linha do milestone na tabela `§Scorecard do Workflow` do
   METRICS.md do pack (via hand-off ao `gm-metrics-collector`, como o bloco
   de performance — IMP-041).

### Passo 5 — Extrair padrões automaticamente

Invocar `gm-pattern-extractor` para:
1. Ler a seção "Padrões extraídos" da retrospectiva
2. Criar/atualizar `accumulation/patterns/<tipo>/<nome>/PATTERN.md` para cada padrão (T2 — pack ativo)
3. Atualizar `accumulation/patterns/INDEX.md`

### Passo 6 — Coletar métricas

Invocar `gm-metrics-collector` para:
1. Extrair métricas da retrospectiva
2. Adicionar linha em `accumulation/metrics/METRICS.md` (T2 — pack ativo)
3. Calcular tendências vs projetos anteriores
