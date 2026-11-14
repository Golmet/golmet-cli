---
name: gm-teststrategy
kind: full-f3   # edição FULL convergida com a twin BASE (fontes gm-*)
---

---
name: gm-teststrategy
description: Gera quality/TEST_STRATEGY.md e quality/DEFINITION_OF_DONE.md a partir dos critérios de aceitação e arquitetura — níveis de teste, ferramentas e definição de done do workspace.
requires: [file, ask]
degrades:
  ask: chat-stop
---
# gm-teststrategy — Estratégia de Testes

Gera `<Produto>/quality/TEST_STRATEGY.md` e `<Produto>/quality/DEFINITION_OF_DONE.md`
com a estratégia de testes e a definição de done para o workspace.

## Referências

- Layout do workspace (pasta `quality/`): FORMATO §2
- RNs (fonte de contexto): FORMATO §7.2

## Resolução de caminhos (FULL)

- **Raiz:** PROJECT_CHARTER.md, CONSTITUTION.md, REQUIREMENTS.md, DECISIONS.md, CHANGELOG.md
- **`specs/`:** DATA_MODEL.md, API_CONTRACTS.md, ERROR_CATALOG.md, UI_SPEC.md, ARCHITECTURE.md, SECURITY.md, ACCEPTANCE_CRITERIA.md, USER_STORIES.md, US<NNN>.md, FIGMA_PROMPTS.md, COMPLIANCE.md
- **`quality/`:** TEST_STRATEGY.md, DEFINITION_OF_DONE.md, TRACEABILITY.md, VALIDATION_REPORT.md, PROPAGATION_STATUS.md, PROGRESS_REPORT.md
- **`delivery/`:** TASKS.md, TASKS_*.md, SCOPE_AGREEMENT.md, CLIENT_REQUIREMENTS.md, CLIENT_REPORT_*.md, HANDOVER_LOG.md
- **`support/`:** LEGACY_CONTEXT.md, KNOWN_ERRORS.md, MIGRATION.md, CHANGE_REQUESTS.md
- **`glossaries/`:** DEV_GLOSSARY.md, BUSINESS_GLOSSARY.md, USER_GLOSSARY.md, GLOSSARY.md
- **`docs/`:** howto/*.md e documentos auxiliares
- **`meetings/`, `contracts/`:** checkpoints (inalterados)

**Regra de fallback (transição):** ao **LER** um artefato, verifique primeiro
o caminho novo; se não existir, use o caminho legado na raiz do módulo. Ao
**ESCREVER/GERAR**, use sempre o caminho novo. Módulos ainda não migrados
continuam legíveis durante a transição.

## Lê (upstream)

- `<Produto>/specs/ACCEPTANCE_CRITERIA.md` (se existir; fallback: raiz)
- `<Produto>/specs/ARCHITECTURE.md` (se existir, para stack de teste; fallback: raiz)
- `<Produto>/specs/SECURITY.md` (se existir, para requisitos de segurança; fallback: raiz)
- `<Produto>/specs/API_CONTRACTS.md` (se existir, para testes de API; fallback: raiz)
- `<Produto>/specs/ERROR_CATALOG.md` (se existir, para casos de teste de erro; fallback: raiz)
- `<Produto>/REQUIREMENTS.md` (para contexto)

## Gera (downstream)

- `<Produto>/quality/TEST_STRATEGY.md`
- `<Produto>/quality/DEFINITION_OF_DONE.md`

## Nunca edita

- `ACCEPTANCE_CRITERIA.md`
- `ARCHITECTURE.md`
- `golmet.index.json` (único escritor é o kit)

## Capacidades

- `file` — ler documentos upstream e escrever TEST_STRATEGY.md, DEFINITION_OF_DONE.md
- `ask` — para decisões de estratégia (sem ask nativo: pergunte no chat e PARE — nunca adivinhe)

## Modo de erro

`skip` — tier completo, baixa frequência, não bloqueia.

## Algoritmo

1. Leia `specs/ACCEPTANCE_CRITERIA.md` (se existir) para derivar os níveis de teste necessários
2. Leia `specs/ARCHITECTURE.md` e `specs/SECURITY.md` (se existirem) para stack, ferramentas e requisitos de segurança testáveis
3. Leia `specs/API_CONTRACTS.md` e `specs/ERROR_CATALOG.md` (se existirem) para testes de contrato e de erro
4. Preencha o template de TEST_STRATEGY.md (níveis, ferramentas, dados de teste, CI/CD)
5. Derive o checklist de DEFINITION_OF_DONE.md a partir dos critérios de aceitação e da estratégia
6. Escreva ambos em `quality/`
7. Aponte o próximo passo natural: fechado o conjunto de specs do workspace, a camada seguinte é a de features — proponha gerá-la com `gm-taskspec` (análise `analysis/RN-NNN — <título>/` + feature `delivery/<grupo>/RN-NNN — <título> — Feature.md`) ou confirme a ordem com o roteamento do perfil (se disponível).

## Formato de commit

```
docs: add test strategy and dod for <produto>
```

## Template — TEST_STRATEGY.md

```markdown
# Test Strategy — [Nome do Sistema]

> **Última atualização:** DD/MM/AAAA

---

## 1. Níveis de Teste

| Nível | Responsável | Ferramenta | Cobertura esperada |
|---|---|---|---|
| Unitário | Dev | | > 80% |
| Integração | Dev | | Fluxos críticos |
| API (contrato) | Dev | | Todos endpoints |
| E2E | QA | | Fluxos principais |

## 2. Ferramentas

| Finalidade | Ferramenta |
|---|---|

## 3. Dados de Teste

| Origem | Estratégia |
|---|---|

## 4. CI/CD

| Pipeline | Gatilho | Ações |
|---|---|---|
```

## Template — DEFINITION_OF_DONE.md

```markdown
# Definition of Done — [Nome do Sistema]

> **Última atualização:** DD/MM/AAAA

---

## Checklist (todos devem ser true para considerar "done")

- [ ] Código revisado por pelo menos 1 par
- [ ] Testes unitários passando
- [ ] Testes de integração passando
- [ ] Documentação atualizada (REQUIREMENTS, API_CONTRACTS, UI_SPEC)
- [ ] Critérios de aceitação validados
- [ ] Sem alertas de segurança críticos
- [ ] CHANGELOG.md atualizado
- [ ] `golmet validate` verde no workspace (estado consistente com o FORMATO)
```

## Casos de Borda

- **Sem ACCEPTANCE_CRITERIA.md** — derive os níveis das RNs em REQUIREMENTS.md e sinalize que os critérios formais ainda não existem.
- **Projeto API-only (sem UI)** — marque E2E de interface como não aplicável; mantenha unitário, integração e API.
- **Stack não informada** — pergunte via `ask` antes de fixar ferramentas. Sem ask nativo: pergunte no chat e PARE — nunca adivinhe.
- **Workspace sem CI/CD definido** — registre a seção correspondente como "a definir" em vez de inventar pipeline.

---

compatibility: multi-engine

> **Degradação de checkpoints:** esta skill não declara checkpoints de kit — funciona integralmente sem o kit presente.

## Convergência com a twin BASE (FULL)

A twin BASE `gm-teststrategy` publicada no kit cobre o fluxo essencial (regra de
fallback de caminhos legado preservada inline). Esta edição FULL adiciona a
seção explícita de resolução de caminhos (mapa de pastas do módulo) e
mantém TODO o conteúdo BASE sem remoções.