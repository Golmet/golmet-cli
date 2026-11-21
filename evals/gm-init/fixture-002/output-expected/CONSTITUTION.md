---
project: AGROCRED-DOCS
audience: [tecnica, negocio]
status: active
---

# Constituição do Repositório — AGROCRED-DOCS

> **Propósito:** Documentar decisões, requisitos e arquitetura dos projetos do
> cliente (perfil) neste repositório, com rastreabilidade de ponta a ponta.
> **Última atualização:** 17/11/2026 (init Portal do Cliente)

---

## 1. Produto

### Missão

Evoluir os sistemas do cliente (perfil) com documentação rastreável.

### Visão

Todo requisito de negócio tem um documento de origem, uma decisão registrada
e um artefato técnico correspondente.

---

> **Regra de idioma:** CONSTITUTION.md em português brasileiro (PT-BR). Títulos,
> descrições, princípios e seções em PT-BR. Apenas notação DAG, acrônimos
> (ADR, RN, DAG) e termos técnicos universais permanecem no original.

## 2. Objetivo do Produto

Documentar, modelar e especificar os módulos seguindo:

```
PROJECT_CHARTER → REQUIREMENTS → DATA_MODEL → API_CONTRACTS → UI_SPEC → TEST_STRATEGY
```

---

## 3. Princípios

| # | Princípio | Descrição |
|---|-----------|-----------|
| 1 | Rastreabilidade primeiro | Toda RN deve ter um artefato downstream |
| 2 | Decisão registrada | Toda decisão tem um ADR em DECISIONS.md |
| 3 | Legado explícito | Sistema existente documentado em LEGACY_CONTEXT.md antes de evoluir |
| 4 | Escopo delimitado | Cada módulo documenta seu escopo e o que está fora |
| 5 | Pendência visível | Tudo não definido fica como PEND no REQUIREMENTS.md |

---

## 4. Grafo de Documentos (DAG)

```
PROJECT_CHARTER.md
    │
    ▼
REQUIREMENTS.md
    │
    ├──▶ DATA_MODEL.md
    ├──▶ API_CONTRACTS.md
    ├──▶ UI_SPEC.md
    └──▶ SCOPE_AGREEMENT.md
    │
    ├──▶ TEST_STRATEGY.md
    └──▶ HOWTO.md
```

### Documentos transversais (raiz do projeto)

| Documento | Finalidade |
|-----------|------------|
| PROJECT_CHARTER.md | Termo de abertura do projeto |
| REQUIREMENTS.md | Regras de negócio (raiz da DAG) |
| DECISIONS.md | Decisões arquiteturais (ADR) |
| CHANGELOG.md | Histórico de mudanças |

### Estrutura de diretórios do módulo (IMP-011)

```
<projeto>/
├── PROJECT_CHARTER.md, REQUIREMENTS.md, DECISIONS.md, CHANGELOG.md
├── analysis/     # RN-NNN — <título>/ (Analise-Funcional.md + Analise-Tecnica.md pura — IMP-092)
├── specs/        # DATA_MODEL, API_CONTRACTS, ERROR_CATALOG, UI_SPEC, ARCHITECTURE, SECURITY, ACCEPTANCE_CRITERIA, USER_STORIES
├── quality/      # TEST_STRATEGY, DEFINITION_OF_DONE, TRACEABILITY, VALIDATION_REPORT, PROPAGATION_STATUS
├── delivery/     # <grupo-do-estilo>/RN-NNN — <título> — Feature.md (arquivo direto por feature — IMP-093)
├── support/      # LEGACY_CONTEXT, KNOWN_ERRORS, MIGRATION, CHANGE_REQUESTS
├── glossaries/   # DEV_GLOSSARY, BUSINESS_GLOSSARY, USER_GLOSSARY
├── docs/         # howto/ e documentos auxiliares
└── meetings/     # checkpoints de clarify
```

---

## 5. Ciclo de Vida dos Documentos

| Status | Significado |
|--------|-------------|
| draft | Em elaboração, pode mudar |
| review | Aguardando revisão |
| approved | Aguardando revisão do stakeholder |
| active | Vigente, fonte da verdade |
| superseded | Substituído |
| archived | Não é mais relevante |

---

## 6. Projetos

| Projeto | Repositórios | Status | Início | Última atualização |
|---------|-------------|--------|--------|-------------------|
| PortalCliente | — | draft | 17/11/2026 | 17/11/2026 |

> **Modo simplificado:** sistema existente (legado: true) — repos novos não
> declarados nesta fase; o sistema em produção é documentado em
> LEGACY_CONTEXT.md pelo gm-reverse antes de qualquer evolução.

---

## 7. Papéis e Responsabilidades

| Papel | Responsável | Responsabilidades |
|-------|-------------|-------------------|
| Lead Técnico | (a definir) | Decisões arquiteturais |
| Patrocinador (sponsor) | Gerência de atendimento (perfil) | Aprovação de escopo |
| Docência do workflow | engine (IA) | Geração de documentos do módulo |

---

## 8. Glossário do Projeto

| Termo | Definição |
|-------|-----------|
| Portal | Portal do Cliente — fachada de consulta self-service |
| Saldo | Valor disponível do crédito aprovado |
| Workflow GOLMET-like | Requisitos, Documentação e Desenvolvimento |
| RN | Regra de Negócio |
| PEND | Pendência a definir |
| ADR | Architecture Decision Record |
| DAG | Directed Acyclic Graph |

---

## 9. Organização de Delivery e Análise (IMP-092)

> Define onde vivem a análise e as tasks das features e como o delivery é
> agrupado. Editável por projeto; consultada pelo `gm-task-spec` e pelo
> `gm-restructure`. Configuração vigente deste repositório (defaults da skill):

```yaml
delivery_style: monthly
tracker: github
```

| Campo | Valor | Descrição |
|-------|-------|-----------|
| `delivery_style` | `monthly` | Estilo de agrupamento sob `delivery/`: `monthly` (`YYYY-MM`), `sprint` (`YYYY-MM-DD`), `release` (nome livre), `flow` (`backlog`), `horizon` (`now`/`next`/`later`), `epic` (nome do épico), `custom` (usa `custom_template`) |
| `sprint_weeks` | — | Duração da sprint em semanas (1..6) — usado somente quando `delivery_style: sprint` |
| `custom_template` | — | Template de caminho quando `delivery_style: custom`; placeholders resolvidos por pergunta na geração |
| `tracker` | `github` | Sistema de tracking das issues: `github` ou `azure` |
