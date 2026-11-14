# Tour do Workflow Golmet

> Conteúdo de onboarding carregado lazy — lido apenas quando o operador
> escolhe a Área 2 (Tour) do `gm-onboarding-orchestrator`. Vive na própria
> config (portável, sem resolução cross-repo).

## Primary Agents (seleção via Tab)

| Primary | Domínio | Quando usar |
|---------|---------|-------------|
| `gm-req` | Requirements — descoberta de negócio, REQUIREMENTS.md, escopo, CRs | Reuniões, requisitos, mudanças de escopo |
| `gm-doc` | Documentation — engenharia reversa multi-repo, specs, fechamento da DAG | Documentar sistema existente |
| `gm-dev` | Development — implementação de código a partir de tasks aprovadas | Escrever código nos repos |
| `gm-devops` | DevOps — infraestrutura e deploy: VPS, Docker, GHCR, Cloudflare, CI/CD | Setup de servidor, pipeline de deploy, troubleshooting |

## Vocabulário essencial

- **RN-NNN** — Regra de Negócio numerada em `REQUIREMENTS.md`. Toda regra só existe formalmente assim.
- **PEND-XXX** — Ambiguidade aguardando resposta do cliente (score de clareza < 70). Nunca ancora feature.
- **CR-XXX** — Change Request: mudança formal em requisito já confirmado.
- **DAG** — grafo de dependência dos artefatos: REQUIREMENTS → specs → features → código.
- **IMP-XXX** — número de melhoria do próprio workflow, citado nos docs para rastreabilidade.

## Ciclo RN → task → dev

1. Reunião/transcrição → `gm-clarify` (score de ambiguidade; < 70 vira PEND-XXX).
2. `gm-requirements` formaliza **RN-NNN** em `REQUIREMENTS.md` (raiz da DAG).
3. Tradução técnica: DATA_MODEL, API_CONTRACTS, UI_SPEC, ARCHITECTURE (domínio traducao).
4. `gm-taskspec` gera `analysis/RN-NNN — <título>/` + `delivery/<grupo>/RN-NNN — <título> — Feature.md`.
5. Issues no GitHub/Azure → `gm-dev` implementa task a task → PR vinculado.

Regra de ouro: toda regra de negócio só existe formalmente como **RN-NNN**;
PEND nunca ancora feature (IMP-094).

## Exemplo ponta a ponta

1. Cliente descreve a regra numa reunião → `gm-clarify` pergunta o que falta (score ≥ 70).
2. `gm-requirements` registra **RN-001**: "Agendamento só é confirmado com pagamento antecipado".
3. Tradução técnica deriva tabelas, contratos de API e telas a partir da RN-001.
4. `gm-taskspec` gera `analysis/RN-001 — Agendamento/` + `delivery/<grupo>/RN-001 — Agendamento — Feature.md`.
5. Issue no GitHub/Azure → `gm-dev` implementa a task → PR vinculado à issue.

## Mapa de artefatos (IMP-011)

| Caminho | Conteúdo |
|---------|----------|
| raiz do módulo | PROJECT_CHARTER.md, CONSTITUTION.md, REQUIREMENTS.md, DECISIONS.md |
| `specs/` | DATA_MODEL, API_CONTRACTS, ERROR_CATALOG, UI_SPEC, ARCHITECTURE, SECURITY, ACCEPTANCE_CRITERIA |
| `quality/` | TEST_STRATEGY, DEFINITION_OF_DONE, TRACEABILITY, VALIDATION_REPORT, PROPAGATION_STATUS |
| `delivery/` | `RN-NNN — <título> — Feature.md` (issues/tasks), SCOPE_AGREEMENT, HANDOVER_LOG |
| `support/` | LEGACY_CONTEXT, KNOWN_ERRORS, MIGRATION, CHANGE_REQUESTS |
| `glossaries/` | DEV/BUSINESS/USER_GLOSSARY |

## Navegação no dia a dia

- "O que fazer agora?" → `@gm-next` | Status → `@gm-progress` |
  Assumir módulo → `@gm-handover` | Tasks pendentes → `@gm-tasks-pending`.
