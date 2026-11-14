---
name: gm-glossary
kind: full-f3   # edição FULL convergida com a twin BASE (fontes gm-*)
---

---
name: gm-glossary
description: Gera ou atualiza os 3 glossários do workspace (DEV, BUSINESS, USER) a partir do mesmo insumo — merge incremental, nunca deleta termos existentes.
requires: [file, ask]
degrades:
  ask: chat-stop
---
# gm-glossary — Glossários (3 níveis)

Gera 3 arquivos de glossário a partir do mesmo insumo, cada um para uma
audiência diferente. Faz merge incremental com termos existentes.

## Referências

- Layout do workspace (pasta `glossaries/`): FORMATO §2

## Resolução de caminhos

Os artefatos do workspace seguem o layout do FORMATO §2:

- **Raiz do workspace:** PROJECT_CHARTER.md, REQUIREMENTS.md, DECISIONS.md, CHANGE_REQUESTS.md
- **`specs/`:** DATA_MODEL.md, API_CONTRACTS.md, ERROR_CATALOG.md, UI_SPEC.md, ARCHITECTURE.md, SECURITY.md, ACCEPTANCE_CRITERIA.md, COMPLIANCE.md
- **`quality/`:** TEST_STRATEGY.md, DEFINITION_OF_DONE.md, TRACEABILITY.md
- **`delivery/`:** Features (`<grupo>/RN-NNN — <título> — Feature.md`), SCOPE_AGREEMENT.md, CLIENT_REQUIREMENTS.md
- **`glossaries/`:** DEV_GLOSSARY.md, BUSINESS_GLOSSARY.md, USER_GLOSSARY.md, GLOSSARY.md
- **`meetings/`:** checkpoints

**Regra de fallback (transição):** ao **LER** um artefato, verifique primeiro
o caminho novo (ex: `specs/DATA_MODEL.md`); se não existir, use o caminho
legado na raiz do workspace (ex: `DATA_MODEL.md`). Ao **ESCREVER/GERAR**, use
sempre o caminho novo. Workspaces ainda não adotados continuam legíveis
durante a transição.

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

- `<Produto>/REQUIREMENTS.md` (se existir)
- `<Produto>/glossaries/DEV_GLOSSARY.md` (se existir, para merge; fallback: raiz)
- `<Produto>/glossaries/BUSINESS_GLOSSARY.md` (se existir, para merge; fallback: raiz)
- `<Produto>/glossaries/USER_GLOSSARY.md` (se existir, para merge; fallback: raiz)
- Transcrição de reunião ou termos soltos

## Gera (downstream)

- `<Produto>/glossaries/DEV_GLOSSARY.md`
- `<Produto>/glossaries/BUSINESS_GLOSSARY.md`
- `<Produto>/glossaries/USER_GLOSSARY.md`

## Nunca edita

- `REQUIREMENTS.md`
- Qualquer outro documento de requisito
- `golmet.index.json` (único escritor é o kit)

## Capacidades

- `file` — ler arquivos existentes e escrever os glossários
- `ask` — para ambiguidades (sem ask nativo: pergunte no chat e PARE — nunca adivinhe)

## Modo de erro

`continue` — se falhar, loga warning e segue. Glossários são enriquecimento.

## Algoritmo

1. Leia `REQUIREMENTS.md` (se existir) e a transcrição/termos fornecidos — extraia os termos candidatos.
2. Leia os 3 glossários existentes em `glossaries/` (fallback: raiz) para o merge incremental.
3. Para cada termo novo, classifique a audiência (DEV/BUSINESS/USER) e aplique o formato e a regra de idioma do glossário correspondente.
4. Aplique as regras de merge incremental: adicione apenas termos novos; em definição conflitante, use `ask`.
5. Escreva/atualize `DEV_GLOSSARY.md`, `BUSINESS_GLOSSARY.md` e `USER_GLOSSARY.md` em `glossaries/`.

## Formato de commit

```
docs: add/update glossary terms for <produto>
```

## Regras de merge incremental

1. Leia os glossários existentes antes de gerar
2. Adicione apenas termos **novos** (não presentes no existente)
3. **Nunca delete** ou sobrescreva termos existentes
4. Se um termo existente tiver definição conflitante, use `ask` para perguntar

## 1. DEV_GLOSSARY.md — audiência técnica

**Formato por termo:**
```markdown
## [Nome do Termo]

**Definição:** [definição técnica precisa]
**Classificação DDD:** [Aggregate Root, Value Object, Entity, Domain Event]
**Relações:** [termos relacionados]
**Não confundir com:** [termos parecidos]
```

**Regra de idioma:** o título é o termo técnico (jargão da área, frequentemente em inglês — ex.: `## Backfill`). Não traduza o título para DEV; a tradução/contexto vai na definição.

## 2. BUSINESS_GLOSSARY.md — audiência negócio

**Formato por termo:**
```markdown
## [Termo na língua do negócio (sigla/expansão)]

[2-4 frases, linguagem clara, sem jargão técnico]
```

**Regra de idioma:** glossário 100% na língua da audiência de negócio (padrão PT-BR). **Nunca use termos em língua estrangeira ao negócio** (nem no título, nem entre parênteses). Parênteses apenas para siglas ou expansões no idioma local. Se o conceito só existir em outra língua, traduza o título e explique na definição.

## 3. USER_GLOSSARY.md — audiência usuário final

**Formato por termo:**
```markdown
## O que é [nome]

[2-4 frases, linguagem coloquial]
```

**Regra de idioma:** `[nome]` na língua do usuário final, linguagem coloquial. Ex.: `## O que é Carga Inicial`.

## Casos de Borda

- **Termo existente com definição conflitante**: não sobrescreva — use `ask` para decidir entre manter, atualizar ou registrar as duas acepções.
- **Termo que só existe em outra língua**: no DEV_GLOSSARY mantenha o título técnico original; no BUSINESS e no USER traduza o título e explique na definição (nunca língua estrangeira nesses dois).
- **Mesmo termo serve a várias audiências**: gere a entrada nos 3 glossários, cada uma no formato e idioma da sua audiência — não reutilize o texto do DEV no BUSINESS.
- **Glossários ainda não existem**: crie os 3 arquivos em `glossaries/` — o merge incremental passa a valer nas execuções seguintes.
- **Termo sem definição clara no insumo**: pergunte via `ask` antes de escrever — não publique definição genérica. Sem ask nativo: pergunte no chat e PARE — nunca adivinhe.

---

compatibility: multi-engine

> **Degradação de checkpoints:** esta skill não declara checkpoints de kit — funciona integralmente sem o kit presente.

## Convergência com a twin BASE (FULL)

A twin BASE `gm-glossary` publicada no kit cobre o fluxo essencial (regra de
fallback de caminhos legado preservada inline). Esta edição FULL adiciona a
seção explícita de resolução de caminhos (mapa de pastas do módulo) e
mantém TODO o conteúdo BASE sem remoções.