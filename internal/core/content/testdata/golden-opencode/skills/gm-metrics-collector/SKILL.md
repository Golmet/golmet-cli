---
name: gm-metrics-collector
description: >-
   Coleta métricas de projetos e acumula no accumulation/metrics/ METRICS.md do pack ativo (T2 — genérico por organização): skills usadas, issues de qualidade, cobertura, duplicação, quality gate, Outcome Metrics (IMP-099 — fonte/período declarados, N/A nunca zero fingido). Hand-off do gm-perf-auditor (append é do coletor). Append incremental — nunca reescreve histórico.
requires: [file, write, glob, grep, ask]
degrades: [edit, shell, webfetch]
---

# gm-metrics-collector — Coletor de Métricas (T2 — acúmulo do pack)

Coleta métricas de cada projeto e acumula no `accumulation/metrics/METRICS.md`
do pack ativo da organização (ex.: `packs/<pack-ativo>/accumulation/metrics/`) para
rastrear tendências de qualidade ao longo do tempo.

## Resolução de destino (IMP-061/IMP-111 — T2, genérico por organização)

Todos os `accumulation/metrics/` referidos nesta skill são relativos ao
**`accumulation/` do pack ativo da organização** (ex.: `packs/<pack-ativo>/accumulation/`
— resolução via `packs:` do client-profile), sinalizado pelo bloco
de acúmulo do AGENTS.md global:

1. Bloco presente (pack ativo configurado) → destino =
   `packs/<pack-ativo>/accumulation/metrics/METRICS.md`.
2. Bloco ausente → pergunte ao usuário qual é o pack ativo
   (instalado em `packs/<id>/`) e sugira configurar o bloco. **Nunca** use
   o diretório de docs da config do engine como destino — a config do
   workflow é genérica/portável, sem conteúdo de negócio.
3. Exceção sandbox de eval: resolva os paths relativos à raiz do sandbox
   (fixtures usam `docs/` relativo).

## Lê (upstream)

- Retrospectivas de projetos — métricas do projeto
- Relatórios de eval (EVAL_REPORT_*.md)
- `quality/PERF_REPORT_*.md` no config do engine — roll-up por feature do gm-perf-auditor (IMP-099)
- `packs/<pack-ativo>/accumulation/metrics/METRICS.md` — histórico anterior

## Gera (downstream)

- `packs/<pack-ativo>/accumulation/metrics/METRICS.md` — métricas acumuladas atualizadas

## Nunca edita

- Retrospectivas (read-only)
- Relatórios de eval (read-only)

## Tools

- `file`/`glob`/`grep` — ler dados de entrada
- `write` — atualizar METRICS.md
- `ask` — perguntar qual projeto coletar (Passo 1)

## Modo de erro

`warn` — métrica indisponível entra como `N/A` na tabela; nunca bloqueia a
coleta das demais. `accumulation/metrics/METRICS.md` inexistente → crie com o
template do Passo 5. A atualização é sempre append incremental — nunca
reescreva ou remova linhas/seções existentes.

## Métricas coletadas

| Métrica | Fonte | Formato |
|---------|-------|---------|
| Skills usadas | Retrospectiva | N |
| Issues de qualidade | Retrospectiva / quality-gate | N |
| Cobertura de testes | Retrospectiva / relatório | N% |
| Duplicação de código | Retrospectiva / relatório | N% |
| Quality gate final | Retrospectiva | PASS/WARN/FAIL |
| IMPs registrados | WORKFLOW_IMPROVEMENTS.md | N |
| ADRs tomados | DECISIONS.md | N |
| Skills ativas | registry.yaml | N |

## Hand-off do gm-perf-auditor (contrato)

O `gm-perf-auditor` produz `quality/PERF_REPORT_<timestamp>.md` com um bloco
pronto da seção `## Performance do Runtime` e delega a escrita em
`accumulation/metrics/METRICS.md` do pack ativo a esta skill (o append é
responsabilidade do coletor). Ao ser invocada para esse hand-off:

1. Se a seção `## Performance do Runtime` não existe em
   `accumulation/metrics/METRICS.md` → crie-a ao final do arquivo.
2. Se existe → adicione nova linha na tabela (histórico crescente; nunca
   reescreva linhas antigas).
3. Preserve intactas todas as outras seções do METRICS.md.
4. Se o bloco não vier no prompt, extraia-o do PERF_REPORT mais recente no
   `quality/` do config do engine — o perf-auditor mantém o bloco íntegro no
   relatório quando o coletor não pôde ser invocado na hora.

## Outcome metrics (IMP-099)

Métricas de resultado (outcome) acumuladas em
`accumulation/metrics/METRICS.md` do pack ativo, seção
própria `## Outcome Metrics`. Cada métrica declara **fonte** e **período**
— métrica sem fonte declarada = não coletada (entrada `N/A (sem dados)`,
nunca zero fingido):

| Métrica | Fonte | Período |
|---------|-------|---------|
| Taxa de resolução de PENDs | kit golmet `progress` Bloco E (REQUIREMENTS.md §11) | janela da coleta |
| % de CRs por origem (retrabalho = fração `requisito-mal-capturado`) | kit golmet `progress` Bloco E (support/CHANGE_REQUESTS.md) | janela da coleta |
| Tempo/tokens médios por feature | gm-perf-auditor (roll-up por feature — join heurístico, métrica aproximada) | janela do PERF_REPORT |
| Features pendentes de drift-check pós-entrega (>14 dias) | kit golmet `progress` Bloco E (front matter `entregue_em`) | momento da coleta |

Template da seção (append incremental — nunca reescrever linhas):

```markdown
## Outcome Metrics

| Data | Resolução PENDs | CRs por origem (rmc/evo/bug/dec) | Retrabalho req. | Tempo médio/feature | Tokens médios/feature | Drift-check pendentes | Fonte/Período |
|------|-----------------|----------------------------------|-----------------|---------------------|-----------------------|-----------------------|----------------|
| DD/MM/AAAA | 27% | 2/3/0/1 | 33% | 12 min | 45000 | 1 | progress + PERF_REPORT <ts> / 30d |
```

Invariantes (IMP-099):

- Linha sem fonte/período declarados não é gravada — métrica sem fonte =
  `N/A (sem dados)`, nunca zero fingido.
- CR sem campo `Origem` conta como N/A (sem dados) — nunca inventar origem.
- Tempo/tokens por feature são aproximados (join heurístico do
  perf-auditor) — declarar a aproximação na seção.

## Workflow

### Passo 1 — Identificar projeto

Pergunte qual projeto está sendo coletado (ou detecte
automaticamente se invocado por `gm-project-retrospective`).

### Passo 2 — Coletar métricas

1. Ler a RETROSPECTIVE.md do projeto (se existir)
2. Ler os relatórios de eval do projeto
3. Contar IMPs e ADRs do período
4. Se invocada com hand-off do `gm-perf-auditor`, processe o bloco
   `## Performance do Runtime` conforme a seção Hand-off

### Passo 3 — Adicionar ao histórico

Leia `accumulation/metrics/METRICS.md` do pack ativo (se existir) e adicione nova linha na tabela.
Se não existir, crie com o template abaixo.

### Passo 4 — Calcular tendências

Compare com o projeto anterior (se existir):
- Delta de cobertura
- Delta de issues
- Delta de quality gate

### Passo 5 — Gerar/atualizar METRICS.md

```markdown
# Métricas Acumuladas — <pack-ativo> (accumulation/metrics)

> Atualizado automaticamente por `gm-metrics-collector`.
> Última atualização: DD/MM/AAAA

## Por Projeto

| Projeto | Data | Skills | Issues | Cobertura | Dupl. | Quality Gate | IMPs |
|---------|------|--------|--------|-----------|-------|--------------|------|
| AppIrmao | 2026-08 | 12 | 5 | 82% | 2.1% | PASS | 3 |
| MeuApp | 2026-09 | 15 | 3 | 85% | 1.8% | PASS | 2 |

## Tendências

| Métrica | Atual | Anterior | Delta | Trend |
|---------|-------|----------|-------|-------|
| Cobertura média | 85% | 82% | ↑ 3% | 📈 |
| Issues/projeto | 3 | 5 | ↓ 2 | 📈 |
| Duplication | 1.8% | 2.1% | ↓ 0.3% | 📈 |
| Skills ativas | 46 | 44 | ↑ 2 | 📈 |
| IMPs/mês | 4 | 3 | ↑ 1 | 📈 |

## Resumo Geral

| Métrica | Valor |
|---------|-------|
| Total de projetos | N |
| Cobertura média | N% |
| Issues totais | N |
| Skills mais usadas | <lista top 5> |
| Padrões extraídos | N |
```
