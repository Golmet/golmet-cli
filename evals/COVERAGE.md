# Cobertura de evals — corpus do kit

> Este documento é o mapa de cobertura dos evals do kit. Complementa o
> `README.md` (protocolo de paridade). Atualizado a cada lote de fixtures.

## Gate de CI

`go test ./internal/core/eval/ -run TestCorpusSelfScore` — **toda fixture é
auto-pontuada** sem engine externo: o teste mescla `input/` +
`output-expected/` em um workspace temporário e pontua com o mesmo checker do
`golmet eval` (`RunFixture`). Qualquer check error-severity quebrando falha o
teste. Fixtures sem baseline são pontuadas contra o workspace input-only.
Exceção legítima exige marcador `<fixture>/.self-score-skip` com motivo
non-empty (coletado e exibido; skip sem motivo falha o gate).

**Estado: 61/61 fixtures PASS, 0 puladas** (2026-11-13 — inclui o lote
packs deste corte: gm-pack-scaffold/gm-pack-publish/gm-pack-update
fixture-001, e do lote paralelo a gm-dba/fixture-002, a
gm-validate/fixture-001 e 3 fixtures extras).

## Baselines: política

- **CLI-gerada** (preferível): quando a skill tem comando determinístico
  (`golmet adopt`, `golmet db drift`), a baseline é o output real do CLI.
- **Autorada (contrato de saída)**: para skills AI-authored, a baseline é o
  contrato do que a skill produz, derivado do corpo da skill
  (`coredata/sources/gm-<skill>.md`). Execuções reais que divergirem da
  baseline viram issue (ou o criteria evolui com decisão registrada) —
  divergência nunca é resolvida silenciosamente.

## Cobertura: 49 de 68 skills do registry (72%)

> **Universo e regra de contagem (rederivados programaticamente em
> 2026-11-13):** skills = itens do `coredata/registry.yaml` com `kind` ≠
> `primary`/`subagent` — 68 (o registry tem 93 itens = 68 skills + 4
> primary + 21 subagents). **A soma fecha: 49 cobertas + 19 excluídas = 68;
> pendente declarada: 0.** Fora do universo, com linha marcada
> na tabela: `gm-adopt` (kit-nativa — CLI `golmet adopt`, corpo em
> `internal/cli/`, não é item do registry) e `gm-validate` (skill do
> workflow ainda sem item no registry — fixture-001 aportada por lote
> paralelo (lote negativas); linha marcada até virar item do registry). A contagem anterior
> ("46 de 65") contava gm-adopt como skill e omitia a linha do
> gm-datamodel, que tem fixture.

| Skill | Fixture(s) | Baseline |
|---|---|---|
| gm-adopt † | 001 | CLI-gerada |
| gm-agent-health-monitor | 001 | autorada |
| gm-agents-md-sync | 001 | autorada |
| gm-apicontracts | 001 | autorada |
| gm-arch | 001 | autorada |
| gm-init | 001 | autorada |
| gm-change-request | 001 | autorada |
| gm-charter | 001 | autorada |
| gm-cicd-workflow | 001 | autorada |
| gm-clarify | 001–003 | autorada |
| gm-client-report | 001 | autorada |
| gm-clientreq | 001 | autorada |
| gm-compliance | 001 | autorada |
| gm-criteria | 001 | autorada |
| gm-datamodel | 001 | autorada |
| gm-dba | 001 (drift, CLI-gerada) + 002 (performance, autorada) | mista (CLI + autorada) |
| gm-decision-log | 001 | autorada |
| gm-design-brief | 001 | autorada |
| gm-domain-research | 001 | autorada |
| gm-drift-detector | 001 | autorada |
| gm-figma-prompt | 001 | autorada |
| gm-glossary | 001 | autorada |
| gm-handover | 001 | autorada |
| gm-howto | 001 | autorada |
| gm-incident-postmortem | 001 | autorada |
| gm-knowledge-auditor | 001 | autorada |
| gm-known-error | 001 | autorada |
| gm-legacy-guide | 001 | autorada |
| gm-metrics-collector | 001 | autorada |
| gm-migration | 001 | autorada |
| gm-pack-publish | 001 | autorada |
| gm-pack-scaffold | 001 | autorada |
| gm-pack-update | 001 | autorada |
| gm-pattern-extractor | 001 | autorada |
| gm-primary-agent-creator | 001 | autorada |
| gm-project-retrospective | 001 | autorada |
| gm-quality-gates | 001 | autorada |
| gm-registry-render | 001 | autorada |
| gm-requirements | 001–003 | autorada |
| gm-restructure | 001 | autorada |
| gm-reverse | 001 | autorada |
| gm-scope | 001 | autorada |
| gm-security-audit | 001 | autorada |
| gm-skill-creator | 001 | autorada |
| gm-skill-evolution | 001 | autorada |
| gm-subagent-creator | 001 | autorada |
| gm-taskspec | 001–003 | autorada |
| gm-teststrategy | 001 | autorada |
| gm-traceability | 001 | autorada |
| gm-uispec | 001 | autorada |
| gm-validate ‡ | 001 (lote negativas — paralelo) | autorada |

† kit-nativa (CLI do kit) — fora do universo do registry; não entra na fração.
‡ fixture-001 aportada por lote paralelo (2026-11-13); fora do universo do
registry até virar item.

## Excluídas por natureza (19 skills + 3 serviços advisory)

> **Regra de contagem (declarada):** esta tabela conta SKILLS e, como
> sub-entrada, SERVIÇOS de uma mesma skill quando têm naturezas de saída
> distintas (hoje: os serviços advisory do gm-dba). A linha gm-stack-N
> agrupa 4 skills. As 19 skills fecham o universo com as 49 cobertas e a
> pendente declarada: 0 (gm-init coberto pelo lote negativas/fecha-soma).

Skills/serviços cuja saída não é um artefato de workspace deterministicamente
comparável. Reavaliar se ganharem artefato congelável.

| Skill / serviço | Motivo |
|---|---|
| gm-stack-angular / -dotnet-csharp / -java-quarkus / -rust (4 skills) | Playbooks de referência — conteúdo instalado/lido, sem output de workspace |
| gm-stack-boilerplate | Scaffold a partir de repositórios externos do perfil |
| gm-gh-issue-creator / gm-gh-pr-creator | Saída = API do GitHub (tracker FULL-only) |
| gm-azure-workitem-creator | Saída = API do Azure DevOps (tracker FULL-only) |
| gm-figma-tasks | Requer MCP Figma com design real |
| gm-ui-capture | Requer Playwright + sistema legado em execução |
| gm-vps-setup / gm-vps-troubleshoot / gm-observability | Operação guiada em servidor real (linha dura: nunca produção) |
| gm-typesafe-ai | Advisory/read-only — docs vivas são a fonte |
| gm-prompt-gate | Saída conversacional (classificação) |
| gm-detect-onboarding | Diagnóstico conversacional |
| gm-split-commits-by-diff | Mutação de histórico de repo real |
| gm-perf-auditor | Requer SQLite runtime local do engine |
| gm-knowledge-graph | Artefato visual HTML — não-comparável byte a byte |
| gm-dba · modelar / migrar / operar (3 serviços advisory) | Parecer consultivo sem artefato de workspace — diferem de drift (`DB_DRIFT.md` CLI-gerado) e performance (`DB_PERFORMANCE.md`), ambos avaliáveis e com fixture |

## Histórico de lotes

- Corpus original (16 skills, 22 fixtures) — protocolo de paridade do piloto F3.
- Lote core-docs: traceability, change-request, decision-log, handover, howto, known-error.
- Lote qualidade: drift-detector, agent-health-monitor, migration, incident-postmortem, security-audit, quality-gates.
- Lote meta/workflow: skill-creator, subagent-creator, primary-agent-creator, domain-research, skill-evolution, registry-render.
- Lote autorável-cliente: cicd-workflow, agents-md-sync, figma-prompt, design-brief, client-report, pattern-extractor.
- Lote CLI+resto: adopt (baseline CLI), dba (baseline CLI), legacy-guide, restructure, knowledge-auditor, project-retrospective, metrics-collector.
- Lote negativas (2026-11-13): gm-validate fixture-001 — eval negativo/report, baseline autorada (aportada por lote paralelo, fora do registry). O mesmo lote paralelo aportou fixtures extras: gm-adopt 002, gm-charter 002, gm-requirements 004.
- Lote dba-performance (2026-11-13): gm-dba fixture-002 — serviço performance, baseline AUTORADA (não há CLI: só drift tem comando determinístico; modelar/migrar/operar movidos para exclusões como serviços advisory do gm-dba).
- Lote packs (2026-11-13): gm-pack-scaffold/gm-pack-publish/gm-pack-update fixture-001 — ciclo completo de packs L2 (criar → publicar → consumir), baseline AUTORADA; aporta junto as 4 componentes novas no registry (3 skills pack + gm-pack-orchestrator — 93 itens = 68 skills + 4 primary + 21 subagents), com evals listados nos itens de skill.

