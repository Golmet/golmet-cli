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

- Layout do workspace (pasta `quality/`): FORMAT §2
- RNs (fonte de contexto): FORMAT §7.2

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
- [ ] `golmet validate` verde no workspace (estado consistente com o FORMAT)
```

## Casos de Borda

- **Sem ACCEPTANCE_CRITERIA.md** — derive os níveis das RNs em REQUIREMENTS.md e sinalize que os critérios formais ainda não existem.
- **Projeto API-only (sem UI)** — marque E2E de interface como não aplicável; mantenha unitário, integração e API.
- **Stack não informada** — pergunte via `ask` antes de fixar ferramentas. Sem ask nativo: pergunte no chat e PARE — nunca adivinhe.
- **Workspace sem CI/CD definido** — registre a seção correspondente como "a definir" em vez de inventar pipeline.

---

compatibility: multi-engine

> **Degradação de checkpoints:** esta skill não declara checkpoints de kit — funciona integralmente sem o kit presente.
