---
name: gm-traceability
description: >-
   Gera quality/TRACEABILITY.md — matriz de rastreabilidade entre RNs e documentos downstream (DATA_MODEL, API_CONTRACTS, UI_SPEC). Atualizado após cada skill de tradução técnica rodar. Consome pré-scan determinístico do kit golmet validate quando disponível; modo de erro skip (não bloqueia o fluxo).
requires: [file, grep, write, glob]
degrades: [edit, webfetch]
---

# gm-traceability — Rastreabilidade

Gera `<modulo>/<projeto>/quality/TRACEABILITY.md` — a matriz de rastreabilidade
que mapeia cada RN-XXX aos documentos downstream que a implementam.

## Referências

- IDs RN-XXX, CR-XXX, DEC-XXX: `docs/CONSTITUTION.md §1`
- Fonte primária de dados: `docs/CONSTITUTION.md §6`

## Resolução de caminhos

Os artefatos do módulo seguem a estrutura com subpastas:

- **Raiz:** PROJECT_CHARTER.md, CONSTITUTION.md, REQUIREMENTS.md, DECISIONS.md, CHANGELOG.md
- **`specs/`:** DATA_MODEL.md, API_CONTRACTS.md, ERROR_CATALOG.md, UI_SPEC.md, ARCHITECTURE.md, SECURITY.md, ACCEPTANCE_CRITERIA.md, USER_STORIES.md, US<NNN>.md, FIGMA_PROMPTS.md, COMPLIANCE.md — agregados técnicos também podem estar **quebrados por micro-contexto** (FORMAT §7.7.1): `specs/data_model/` com `INDEX.md` + escopos (idem `api_contracts/`, `ui_spec/`)
- **`quality/`:** TEST_STRATEGY.md, DEFINITION_OF_DONE.md, TRACEABILITY.md, VALIDATION_REPORT.md, PROPAGATION_STATUS.md, PROGRESS_REPORT.md
- **`delivery/`:** arquivos de feature/tasks, SCOPE_AGREEMENT.md, CLIENT_REQUIREMENTS.md, CLIENT_REPORT_*.md, HANDOVER_LOG.md
- **`support/`:** LEGACY_CONTEXT.md, KNOWN_ERRORS.md, MIGRATION.md, CHANGE_REQUESTS.md
- **`glossaries/`:** os 3 glossários (DEV, BUSINESS, USER) e variações
- **`docs/`:** howto/*.md e documentos auxiliares
- **`meetings/`, `contracts/`:** checkpoints (inalterados)

**Regra de fallback (transição):** ao **LER** um artefato, verifique primeiro
o caminho novo (ex: `specs/DATA_MODEL.md`); se não existir, use o caminho
legado na raiz do módulo (ex: `DATA_MODEL.md`). Ao **ESCREVER/GERAR**, use
sempre o caminho novo. Módulos ainda não migrados via `gm-restructure`
continuam legíveis durante a transição.

## Lê (upstream)

- `<modulo>/<projeto>/REQUIREMENTS.md` (para lista de RNs)
- `<modulo>/<projeto>/specs/DATA_MODEL.md` (se existir; fallback: raiz do módulo) — se o agregado estiver **quebrado** (`specs/data_model/`), comece pelo `INDEX.md` e leia só os escopos que citam RNs
- `<modulo>/<projeto>/specs/API_CONTRACTS.md` (se existir; fallback: raiz do módulo)
- `<modulo>/<projeto>/specs/ERROR_CATALOG.md` (se existir; fallback: raiz do módulo)
- `<modulo>/<projeto>/specs/UI_SPEC.md` (se existir; fallback: raiz do módulo)
- `<modulo>/<projeto>/specs/ARCHITECTURE.md` (se existir; fallback: raiz do módulo)
- `<modulo>/<projeto>/specs/SECURITY.md` (se existir; fallback: raiz do módulo)
- `<modulo>/<projeto>/support/CHANGE_REQUESTS.md` (se existir, para CRs implementados; fallback: raiz do módulo)
- `<modulo>/<projeto>/quality/TRACEABILITY.md` (se existir, para merge incremental; fallback: raiz do módulo)

## Gera (downstream)

- `<modulo>/<projeto>/quality/TRACEABILITY.md`

## Nunca edita

- Nenhum documento de origem (é sink — só consome, não produz)

## Tools

- `file` (ler todos os documentos do projeto)
- `grep` (buscar referências a RN-XXX nos documentos)
- `write` (escrever TRACEABILITY.md)

## Modo de erro

`skip` — se falhar, loga warning. A rastreabilidade não bloqueia o fluxo.

## Formato de commit

```
docs: update traceability matrix for <projeto>
```

## Workflow

0. **Pré-scan determinístico (determinístico > agêntico):** se `quality/scan.json` existir (gerado pelo pré-scan do kit golmet `validate`), consuma `coverage` e `coberturaCandidatos` — a matriz de presença RN × downstream dos passos 2–3 já está computada (com expansão de ranges). O julgamento permanece seu no merge (passo 6) e na narração dos gaps.
1. Leia `REQUIREMENTS.md` — extraia todas as RN-XXX
2. Leia cada documento downstream — identifique quais RNs cada um implementa; se o DATA_MODEL estiver **quebrado por micro-contexto** (FORMAT §7.7.1), o escopo `<escopo>.md` que cita a RN é o caminho do mapeamento
3. Para cada RN, marque se está coberta em cada documento — com o agregado quebrado, registre o **caminho de escopo específico** (ex.: `specs/data_model/agendamento.md`), não o agregado inteiro: quem implementa a RN lê só o escopo, não o arquivo todo (finura maior da matriz)
4. Identifique RNs sem cobertura (gap) — registre na seção de gaps
5. Identifique CRs aprovados e verifique se já foram refletidos nos downstreams
6. Faça merge incremental com TRACEABILITY.md existente (se houver)
7. Escreva `quality/TRACEABILITY.md`
8. Aponte o próximo passo natural: fechado o conjunto de specs do módulo, a camada seguinte da DAG é a de features — proponha gerá-la com `gm-taskspec` (análise `analysis/RN-NNN — <título>/` + `delivery/<grupo>/RN-NNN — <título> — Feature.md`) ou confirme a ordem com o kit golmet (`next`).

## Template — TRACEABILITY.md

```yaml
---
project: <NOME>
module: rastreabilidade
audience: [tecnica]
status: draft
---
```

```markdown
# Traceability Matrix — [Nome do Sistema]

> **Última atualização:** DD/MM/AAAA

---

## 1. Matriz RN vs Downstreams

| RN | Descrição | DATA_MODEL | API_CONTRACTS | ERROR_CATALOG | UI_SPEC | Status |
|---|---|---|---|---|---|---|
| RN-001 | Regra de exemplo | ✅ | ✅ | ✅ | ❌ | parcial |
| RN-002 | Outra regra | ✅ | ✅ | ✅ | ✅ | completo |
| RN-003 | Pendente | ❌ | ❌ | ❌ | ❌ | gap |

**Legenda:**
- ✅ = coberto, ❌ = gap, — = não se aplica

**Agregado quebrado (FORMAT §7.7.1):** na célula do agregado quebrado, registre o caminho do escopo específico — ex.: `✅ data_model/agendamento.md` — em vez de só `✅`; a matriz ganha finura de leitura (a RN aponta o micro-contexto que a implementa).

---

## 2. Gaps Detectados

| RN | Downstream faltante | Impacto |
|---|---|---|
| RN-001 | UI_SPEC | Frontend não tem especificação |

---

## 3. CRs Aprovados vs Implementados

| CR | RN afetada | Aprovado em | Refletido em downstreams |
|---|---|---|---|
| CR-001 | RN-001 | 10/07/2026 | ✅ DATA_MODEL, ❌ API_CONTRACTS |
```

Se não houver documentos downstream, a matriz fica vazia — isso é esperado para projetos novos.

## Casos de Borda

- **Sem documentos downstream** — matriz vazia é esperado em projeto novo; registre o fato, não invente cobertura.
- **RN sem cobertura (gap)** — registre com downstream faltante e impacto; nunca marque ✅ sem evidência no documento.
- **Merge incremental** — nunca remova linhas existentes da matriz; apenas atualize status e adicione RNs novas.
- **CR implementado** — verifique o reflexo nos downstreams e atualize o status por artefato.
- **Documento downstream existe mas não menciona a RN** — considere gap, mesmo com o arquivo presente.
- **Agregado quebrado sem INDEX** (`specs/data_model/` sem `INDEX.md`) — estrutura inválida (error L7); registre o gap e aponte o Split do gm-datamodel — não adivinhe o escopo de cada RN.
