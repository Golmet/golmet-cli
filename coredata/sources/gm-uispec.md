---
name: gm-uispec
kind: full-f3   # edição FULL convergida com a twin BASE (FONTES gm-*)
---

---
name: gm-uispec
description: Gera specs/UI_SPEC.md a partir das RNs em REQUIREMENTS.md — telas, componentes, fluxos de navegação, validações de frontend e regras de exibição condicional.
requires: [file, ask]
degrades:
  ask: chat-stop
---
# gm-uispec — Especificação de UI

Gera `<Produto>/specs/UI_SPEC.md` com a especificação de interface
derivada das regras de negócio e casos de uso.

## Referências

- Entidade RN (seções `## RN-NNN — <título>`): FORMAT §7.2
- Layout do workspace (pasta `specs/`): FORMAT §2

## Lê (upstream)

- `<Produto>/REQUIREMENTS.md` (essencial — §2 Atores, §5 RNs, §7 Casos de Uso, §8 Saídas)
- `<Produto>/specs/API_CONTRACTS.md` (se existir, para campos e validações; fallback: raiz do workspace)
- `<Produto>/specs/DATA_MODEL.md` (se existir, para tabelas e colunas envolvidas; fallback: raiz do workspace)
- `<Produto>/glossaries/` (se existir, para consistência terminológica; fallback: raiz do workspace)

## Gera (downstream)

- `<Produto>/specs/UI_SPEC.md`

## Nunca edita

- `REQUIREMENTS.md`
- `API_CONTRACTS.md`
- `DATA_MODEL.md`
- `golmet.index.json` (único escritor é o kit)

## Capacidades

- `file` — ler REQUIREMENTS.md, API_CONTRACTS.md, DATA_MODEL.md e escrever UI_SPEC.md
- `ask` — para ambiguidades de layout ou comportamento visual (sem ask nativo: pergunte no chat e PARE — nunca adivinhe)

## Modo de erro

`strict` — se a UI spec não for gerada, o time de frontend não tem input.

## Propagação de confiança (legado)

- Afirmações derivadas de RN induzida de engenharia reversa **herdam a marca de confiança da RN de origem** (campo `**Confiança:**` em REQUIREMENTS.md).
- Escala: Alta → 🟢 CONFIRMADO · Média → 🟡 INFERIDO · Baixa → 🔴 GAP.
- **Regra de ouro:** na dúvida, nível mais baixo.
- Quando disponível, anexe a evidência da análise reversa (`[arquivo:linha @hash]`) — **afirmação sem evidência nunca recebe 🟢**.

## Formato de commit

```
docs: add ui spec for <produto>
```

## Workflow

1. Leia `REQUIREMENTS.md §2 (Atores e Permissões)` — identifique perfis de usuário
2. Leia `REQUIREMENTS.md §5 (Regras de Negócio)` — identifique regras de exibição condicional
3. Leia `REQUIREMENTS.md §7 (Casos de Uso)` — identifique fluxos de tela
4. Leia `REQUIREMENTS.md §8 (Saídas e Integrações)` — identifique outputs visuais
5. Leia `API_CONTRACTS.md` (se existir) — identifique campos e validações a espelhar no frontend
6. Para cada ator, descreva:
   - Telas disponíveis
   - Fluxo de navegação entre telas
   - Permissões por tela/ação
7. Para cada fluxo, descreva:
   - Componentes de tela (formulários, tabelas, modais)
   - Campos com tipo de input (text, date, select, checkbox)
   - Regras de exibição condicional (exibir/ocultar campo baseado em valor)
   - Validações de frontend (derivadas de RNs)
   - Estados (carregando, vazio, erro, sucesso)
8. Escreva `specs/UI_SPEC.md`
9. Aponte o próximo passo natural: fechado o conjunto de specs do workspace, a camada seguinte é a de features — proponha gerá-la com `gm-taskspec` (análise `analysis/RN-NNN — <título>/` + feature `delivery/<grupo>/RN-NNN — <título> — Feature.md`) ou confirme a ordem com o roteamento do perfil (se disponível).

## Template — UI_SPEC.md

```markdown
# UI Specification — [Nome do Sistema]

> **Derivado de:** REQUIREMENTS.md (RN-001, RN-002)
> **Última atualização:** DD/MM/AAAA

---

## 1. Atores e Telas

| Ator | Telas disponíveis | Fluxo inicial |
|---|---|---|
| [papel] | [lista de telas] | [tela de entrada] |

---

## 2. [Tela 1 — Nome]

### 2.1 Acesso

**Permissão:** [papel]
**URL:** /[path]

### 2.2 Layout

[Descrição textual do layout, seções da tela]

### 2.3 Componentes

| Componente | Tipo | Campos | Regras |
|---|---|---|---|
| Formulário de cadastro | Form | campo1, campo2 | RN-001 |
| Tabela de listagem | Table | coluna1, coluna2 | RN-002 |

### 2.4 Campos do Formulário

| Campo | Input | Obrigatório | Validação | RN |
|---|---|---|---|---|
| nome | text | Sim | max 100 chars | RN-001 |
| data | date | Sim | >= data mínima | RN-002 |
| tipo | select | Sim | options: [A, B, C] | RN-003 |

### 2.5 Regras de Exibição Condicional

| Condição | Ação | RN |
|---|---|---|
| Se tipo = A | Exibir campo X | RN-001 |
| Se propriedade habilitada | Exibir aba Detalhes | RN-002 |

### 2.6 Estados

| Estado | Comportamento |
|---|---|
| Carregando | Skeleton/spinner |
| Vazio | Mensagem "Nenhum registro encontrado" |
| Erro | Toast com mensagem do erro |
| Sucesso | Redirecionar para listagem |

---

## 3. [Tela 2 — Nome]

...

---

## 4. Fluxo de Navegação

```
[Tela 1] → [Tela 2] → [Tela 3]
    ↓                      ↓
[Erro]                [Sucesso]
```

---

## 5. Regras de UI Globais

| Regra | Descrição | RN |
|---|---|---|
| Formatação de data | DD/MM/AAAA | — |
| Moeda | Formato local da região (ex.: 1.234,56) | — |
| Confirmação antes de excluir | Modal "Tem certeza?" | — |
```

## Casos de Borda

- **Sem API_CONTRACTS.md** — derive campos e validações apenas de REQUIREMENTS.md e sinalize a ausência no cabeçalho (`Derivado de:`).
- **Projeto API-only (sem UI)** — informe que a skill não se aplica ao workspace e encerre sem gerar arquivo.
- **RN ambígua para layout/comportamento visual** — use `ask` antes de especificar (não invente interação). Sem ask nativo: pergunte no chat e PARE — nunca adivinhe.
- **Regra de exibição condicional sem RN associada** — pergunte qual RN origina a regra; toda linha das seções de exibição deve referenciar uma RN-XXX.
- **Ator sem permissão definida em REQUIREMENTS.md** — pergunte antes de criar tela para o ator.

---

compatibility: multi-engine

> **Degradação de checkpoints:** esta skill não declara checkpoints de kit — funciona integralmente sem o kit presente.
## Convergência com a twin BASE (FULL)

A twin BASE `gm-uispec` publicada no kit cobre o fluxo essencial. Esta edição
FULL preserva a twin INTEGRAL (nenhum passo BASE removido) e adiciona os
extras FULL do pipeline (seções marcadas "FULL" neste documento). Seções BASE e FULL coincidem neste item — a BASE já consolidava o conteúdo FULL da fonte.
