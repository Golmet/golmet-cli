# INDEX — AGROCRED-DOCS

> **Manifesto de navegação do repositório** (padrão IMP-012).
> Contém apenas invariáveis: identidade, mapa de caminhos e desambiguação.
> Estado volátil (fase, pendências, status) → kit golmet (`next`) / HANDOVER_LOG.
> Atualizar somente quando a estrutura do repositório mudar.

## Identidade

| Campo | Valor |
|---|---|
| Sigla | AGROCRED-DOCS |
| Constituição | CONSTITUTION.md |
| Remote | `org-do-cliente/AGROCRED-DOCS` (perfil) |
| Cliente | (perfil do cliente) |

## Mapa de Projetos

| Projeto | Caminho | Repos | Status |
|---------|---------|-------|--------|
| PortalCliente | `./` (raiz do repositório) | — | draft |

## Estrutura Padrão por Projeto

```
<projeto>/
├── PROJECT_CHARTER.md      # Termo de abertura
├── REQUIREMENTS.md         # Regras de negócio (raiz da DAG)
├── DECISIONS.md            # Decisões arquiteturais (ADR)
├── CHANGELOG.md            # Histórico de mudanças
├── analysis/               # Análise por feature — analysis/RN-NNN — <título>/ (IMP-092)
├── specs/                  # Artefatos técnicos
├── quality/                # Qualidade e rastreabilidade
├── delivery/               # Entrega — delivery/<grupo-do-estilo>/RN-NNN — <título> — Feature.md (IMP-093)
├── support/                # Suporte e migração — LEGACY_CONTEXT.md (sistema existente)
├── glossaries/             # Glossários
├── docs/                   # Documentação auxiliar
└── meetings/               # Checkpoints de sessões
```

## Desambiguação

- "Portal" refere-se sempre ao Portal do Cliente (fachada de consulta) —
  nunca ao sistema de crédito em si.

## Fora da DAG

- `CONSTITUTION.md` — governança compartilhada de todos os projetos
- `INDEX.md` — este manifesto
