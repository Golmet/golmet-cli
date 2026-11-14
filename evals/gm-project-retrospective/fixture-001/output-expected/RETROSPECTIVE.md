# Retrospectiva — MeuApp

**Data:** DD/MM/AAAA
**Projeto:** MeuApp (../MeuApp-API)
**Duração:** Setembro/2026
**Skills usadas:** gm-charter, gm-requirements, gm-datamodel, gm-uispec, gm-taskspec, gm-dev-backend, gm-quality-gates, gm-dba, gm-validate, gm-traceability

## O que funcionou

- Âncora RN → feature: cada Feature.md nasceu de uma RN confirmada, sem
  task órfã (delivery/2026-09 com RN-001 e RN-002 em [done])
- Quality gate no fim do projeto bloqueou a entrega antes do PASS
  (quality/QUALITY_GATE.md — cobertura 84%)
- Revisão técnica do gm-dba antes de cada migration evitou retrabalho de
  unique constraint (origem do IMP-002)
- Decisões rastreadas: ADR-001 (e-mail único) aceito e citado nas specs,
  sem divergência entre código e intenção

## O que não funcionou

- Estimativa de tasks subestimou o teste de rejeição da RN-002
  (RN-002-2 levou o dobro do estimado)
- O review apontou 2 issues de qualidade tardias — poderiam ter sido
  pegos no gate por qualidade antes do fim de projeto

## Padrões extraídos

- Paginação por cursor como padrão de listagem → ver
  `patterns/processo/api-pagination/PATTERN.md` (extraído do IMP-001)

## Métricas do projeto

| Métrica | Valor |
|---------|-------|
| Skills usadas | 10 |
| Issues de qualidade | 2 |
| Cobertura de testes | 84% |
| RNs implementadas | 2 |
| Quality gate final | PASS |
| IMPs registrados | 2 |
| ADRs tomados | 1 |

## Ações para o workflow

| Ação | Tipo | Prioridade |
|------|------|------------|
| Checklist de unique constraint no review de banco | IMP | alta |
| Estimar task de teste de rejeição junto com a task de dev | IMP | média |

## Scorecard do Workflow (IMP-048)

Nota 0-10 por segmento, somente com evidência do período (sem uso = N/A,
nunca chute). Nota geral = média das fases COM evidência.

| Segmento | Tipo | Nota | Evidência |
|----------|------|:----:|-----------|
| Intake | fase | 8 | PROJECT_CHARTER.md confirmado na 1ª semana |
| Requisitos & cliente | fase | 9 | RN-001..RN-003 confirmadas sem CR de retrabalho |
| Tradução técnica | fase | 8 | specs derivados das RNs sem divergência no validate |
| Entrega/tasks | fase | 7 | 2 features em [done] com 4 tasks concluídas |
| Desenvolvimento | fase | 8 | RN-001-1..RN-002-2 concluídas em ../MeuApp-API |
| Verificação & qualidade | fase | 9 | quality/QUALITY_GATE.md — gate PASS, cobertura 84% |
| Suporte & pós-produção | fase | N/A | sem uso no período |
| Governança da config | transversal | N/A | sem uso no período |
| Meta-melhoria | transversal | 7 | IMP-001 e IMP-002 registrados no período |
| Observabilidade & eficiência | transversal | N/A | sem medição no período |
| Padrões & aprendizado | transversal | 8 | padrão api-pagination extraído para a pattern library |
| Segurança | transversal | N/A | sem uso no período |
| Integrações externas | transversal | N/A | sem uso no período |
| Portabilidade & onboarding | transversal | N/A | sem uso no período |

**Nota geral (fases com evidência):** 8.2
