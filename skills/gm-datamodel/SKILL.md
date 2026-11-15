---
name: gm-datamodel
description: Gera specs/DATA_MODEL.md a partir das RNs em REQUIREMENTS.md (§6 Dados e Campos) — tabelas, colunas, tipos, PKs, FKs, constraints, índices e decisão de armazenamento.
requires: [file, ask]
degrades:
  ask: chat-stop
kit:
  checkpoints: post-write validate
---
# gm-datamodel - Modelo de Dados (Tabelas e Relacionamentos)

Gera `<Produto>/specs/DATA_MODEL.md` com o modelo de dados focado em
**tabelas do banco** (não entidades ORM ou DTOs). Colunas, tipos de banco,
PKs, FKs, check constraints, índices e decisão de armazenamento.
Quando o agregado unitário cresce além do limiar de quebra, a skill também
executa o **Split por micro-contexto** — transforma `specs/DATA_MODEL.md`
em `specs/data_model/` (INDEX.md + escopos auto-contidos; ver seção própria).

## Resolução de caminhos

Os artefatos do workspace seguem o layout do FORMATO §2 (§7.7.1 para os
agregados de `specs/`) — a saída fica em `specs/DATA_MODEL.md` (formato
**unitário**) ou em `specs/data_model/` (formato **quebrado**:
`INDEX.md` obrigatório + `<escopo>.md` auto-contidos); o contexto de legado
fica em `reverse/LEGACY_CONTEXT.md`. As duas formas são canônicas
simultaneamente — nenhum workspace é obrigado a quebrar.

**Regra de fallback (transição):** ao **LER** um artefato, verifique primeiro
o caminho novo (ex: `specs/DATA_MODEL.md`); se não existir, use o caminho
legado na raiz do workspace (ex: `DATA_MODEL.md`). Ao **ESCREVER/GERAR**, use
sempre o caminho novo. Workspaces ainda não adotados continuam legíveis
durante a transição.

## Lê (upstream)

- `<Produto>/REQUIREMENTS.md` (essencial — §6 Dados e Campos, §9 Persistência e Auditoria)
- `<Produto>/DECISIONS.md` (se existir — decisões de armazenamento, ADRs)
- `<Produto>/reverse/LEGACY_CONTEXT.md` (essencial — tabelas, colunas e tipos do banco legado; fallback: raiz do workspace)
- `CONSTITUTION.md` do container de organização (se presente — convenções de banco: prefixo de tabelas, colunas de auditoria, schema padrão; sem container, pergunte o padrão ao responsável)

## Gera (downstream)

- `<Produto>/specs/DATA_MODEL.md` — formato **unitário** (agregado único)
- `<Produto>/specs/data_model/` — formato **quebrado**: `INDEX.md` (mestre:
  visão geral, ER global, índice dos escopos) + `<escopo>.md`
  micro-contextos auto-contidos — gerado pelo Split por micro-contexto
  (seção abaixo); o padrão é genérico (`specs/api_contracts/`,
  `specs/ui_spec/`), a implantação começa no DATA_MODEL

## Nunca edita

- `REQUIREMENTS.md` (fonte primária não pode ser alterada por downstream)
- `DECISIONS.md`
- `golmet.index.json` (único escritor é o kit)

## Capacidades

- `file` — ler REQUIREMENTS.md, DECISIONS.md, LEGACY_CONTEXT.md e escrever DATA_MODEL.md
- `ask` — para ambiguidades de tipo, nome de tabela ou decisão de armazenamento (sem ask nativo: pergunte no chat e PARE — nunca adivinhe)

## Modo de erro

`strict` — se o modelo de dados não for gerado, o fluxo de API contracts não pode prosseguir.
Se `LEGACY_CONTEXT.md` não existir e o workspace for `legado: true`, abortar (não é possível documentar tabelas legado sem o contexto).

## Propagação de confiança (legado)

- Afirmações derivadas de RN induzida de engenharia reversa **herdam a marca de confiança da RN de origem** (campo `**Confiança:**` em REQUIREMENTS.md).
- Escala: Alta → 🟢 CONFIRMADO · Média → 🟡 INFERIDO · Baixa → 🔴 GAP.
- **Regra de ouro:** na dúvida, nível mais baixo.
- Quando disponível, anexe a evidência da análise reversa (`[arquivo:linha @hash]`) — **afirmação sem evidência nunca recebe 🟢**.

## Formato de commit

```
docs: add data model for <produto>
```

## Workflow

1. **Convenções de banco (CONSTITUTION.md do container, se presente)** — leia o prefixo obrigatório de tabelas (se houver), as colunas de auditoria exigidas (se houver) e o schema padrão. **Nomes de colunas sempre em MAIÚSCULAS.** O modelo de dados deve seguir essas convenções. Tabelas reaproveitadas do legado mantêm suas colunas originais.
2. **REQUIREMENTS.md §6 (Dados e Campos)** — extraia campos do formulário, tipos, obrigatoriedade
3. **REQUIREMENTS.md §9 (Persistência e Auditoria)** — extraia tabelas envolvidas, finalidade, rastreabilidade
4. **DECISIONS.md** — identifique ADRs sobre armazenamento (reuso de tabela existente vs criar nova)
5. **LEGACY_CONTEXT.md** — identifique tabelas legadas, colunas, tipos de banco (ex.: NUMBER, VARCHAR2, DATE), PKs, FKs, constraints, índices
6. Para cada tabela:
   - Se for **reaproveitada** (ex.: `TABELA_EXISTENTE`): documentar TODAS as colunas da tabela legada, marcando as não usadas pelo escopo atual com "(não usado)"
   - Se for **nova**: derivar colunas de REQUIREMENTS.md §6 e tipos de banco apropriados. **Incluir colunas de auditoria conforme as convenções** (se exigidas). **Nomes de colunas sempre em MAIÚSCULAS.**
   - **As colunas de auditoria também devem constar no diagrama ER (mermaid) da §1** — não apenas na tabela de colunas da §2, para que fiquem visíveis na visão gráfica do modelo.
   - Documentar: nome real da tabela, PK, FKs, check constraints, índices
7. Para cada coluna de valor fixo (ex.: `TP_ACAO` com valores `1, 2`), documentar os valores no comentário da coluna
8. Criar tabela de **"Uso no escopo atual — colunas preenchidas"** mapeando campo do request → coluna da tabela
9. Se houver ADR conflitante (reuso vs tabela nova), resolver com `ask` ou documentar a decisão final na seção "Decisão de armazenamento"
10. Escreva `specs/DATA_MODEL.md`
11. **Limiar de quebra (warning L7):** se o agregado unitário passar de
    **800 linhas** (constante `DataModelSplitLines` declarada no lint —
    FORMATO §7.7.1), proponha o **Split por micro-contexto** ao responsável
    (seção abaixo). Em workspace **já quebrado**, escreva direto no formato
    quebrado seguindo o merge incremental (seção abaixo)
12. Rode `golmet validate` (checkpoint post-write) — ou anote a pendência se o kit não estiver disponível

> **Regra de idioma:** Documentos no idioma do cliente (padrão PT-BR). Títulos, descrições de colunas, comentários no idioma combinado. Tipos de banco (`NUMBER`, `VARCHAR2`, `DATE`) e identificadores de coluna (`NRO_ID_REGISTRO`) mantêm o nome real do banco. **Nomes de colunas sempre em MAIÚSCULAS** (ex.: `CTR_DTH_INC`, `TXT_STATUS`).

## Split por micro-contexto (unitário → quebrado)

Artefatos agregados crescem indefinidamente e quem implementa uma RN carrega
o arquivo inteiro quando precisa de um escopo — contexto desnecessário derruba
qualidade. O FORMATO §7.7.1 define o formato **quebrado**: `specs/data_model/`
com `INDEX.md` OBRIGATÓRIO + `<escopo>.md` auto-contidos. Gatilhos: o unitário
passou do limiar de **800 linhas** (warning do lint — constante
`DataModelSplitLines`) **ou** o responsável pede o split.

**Procedimento:**

1. **Proponha os escopos** — agregados/domínios identificados no conteúdo do
   unitário (ex.: agendamento, financeiro, cadastros) — **com confirmação do
   responsável** antes de quebrar (corte de escopo é decisão de conteúdo,
   nunca mecânica)
2. **Gere `specs/data_model/INDEX.md`** — o mestre: visão geral do modelo,
   ER global agregado (mermaid com as tabelas de todos os escopos) e índice
   dos escopos com links — é por ele que todo leitor começa
3. **Distribua o conteúdo** nos `<escopo>.md` **auto-contidos** — entidades
   (tabelas, colunas, PKs/FKs, constraints, índices), relacionamentos do
   escopo e as RNs daquele escopo; cada arquivo se lê SOZINHO — nenhuma
   leitura cruzada obrigatória (repita a tabela compartilhada ou resuma-a
   com referência explícita "detalhe em `<outro-escopo>.md`")
4. **Remova o `specs/DATA_MODEL.md` original** — o mestre passa a ser o
   INDEX (decisão do responsável); deixe os dois = split incompleto

**Merge incremental NO quebrado** (o fluxo normal passa a manter a estrutura
quebrada):

- **Entidade/tabela nova** → entra no escopo certo (agregado/domínio do
  conteúdo); se couber em mais de um, pergunte via `ask`
- **Escopo novo** (domínio sem arquivo) → arquivo `<escopo>.md` novo +
  entrada correspondente no INDEX.md
- **Nada é deletado sem substituição explícita** — conteúdo movido de escopo
  sai do arquivo antigo apenas quando entra no novo (mesma disciplina de
  merge incremental do unitário)
- O lint valida a forma: quebrado sem `INDEX.md` = **error L7** com
  remediação apontando este Split

**Traceability:** com o agregado quebrado, a matriz de rastreabilidade mapeia
RN → caminho de escopo específico (ex.: `RN-012 → data_model/agendamento.md`)
— finura maior que o agregado inteiro (ver gm-traceability).

## Template — DATA_MODEL.md

```markdown
# Modelo de Dados — [Nome do Sistema]

> **Derivado de:** LEGACY_CONTEXT.md §3 (tabelas legado), REQUIREMENTS.md (RN-001, RN-002)
> **Última atualização:** DD/MM/AAAA

---

## 1. Diagrama Entidade-Relacionamento

```mermaid
erDiagram
    TABELA {
        Tipo coluna PK
        Tipo coluna FK
        Tipo coluna
    }

    TABELA2 {
        Tipo coluna PK
    }

    TABELA }o--|| TABELA2 : "fk"
```

- Tabelas externas marcadas como "apenas referência"

---

## 2. Tabelas

### 2.1 [NOME_DA_TABELA]

[Descrição: se é tabela existente reaproveitada ou nova, qual o critério de segregação]

**Identificação:** PK via sequence (nominal `SEQ_NOME` ou similar)

| Coluna | Tipo | Obrigatório | PK/FK | Descrição |
|---|---|---|---|---|
| NOME_COLUNA | NUMBER(10) | Sim | PK | ID (sequence) |
| NOME_COLUNA | VARCHAR2(100) | Sim | — | Descrição |
| NOME_COLUNA | NUMBER(5) | Não | FK → TABELA | Referência externa (não usado no escopo atual) |
| CTR_DTH_INC | DATE | Sim | — | Data/hora de criação (auditoria) |
| CTR_DTH_ATU | DATE | Não | — | Data/hora da última atualização (auditoria) |
| CTR_USU_INC | NUMBER(8) | Sim | — | ID do usuário que criou (auditoria) |
| CTR_USU_ATU | NUMBER(8) | Não | — | ID do usuário da última atualização (auditoria) |
| CTR_NRO_IP_INC | VARCHAR2(15) | Sim | — | IP de origem da criação (auditoria) |
| CTR_NRO_IP_ATU | VARCHAR2(15) | Não | — | IP de origem da última atualização (auditoria) |
| ... | ... | ... | ... | ... |

**Check constraints:**
- `COLUNA IN (1, 2, 3)` — com descrição dos valores
- `CAMPO1 >= CAMPO2`

**Índices relevantes:**
- `IDX_NOME` (COLUNA)
- `IDX_NOME2` (COLUNA1, COLUNA2)

**Uso no escopo atual — colunas preenchidas:**

| Coluna | Valor | Origem |
|---|---|---|
| PK | Sequence | PK |
| CAMPO1 | Do request | Request |
| CAMPO2 | Valor fixo | Regra de negócio |
| ... | ... | ... |

---

### 2.2 [NOME_DA_TABELA2]

[Tabela catálogo, externa, etc.]

| Coluna | Tipo | Obrigatório | PK/FK | Descrição |
|---|---|---|---|---|
| PK | NUMBER(5) | Sim | PK | ID |
| NOME | VARCHAR2(100) | Sim | — | Nome |
| FLAG | CHAR(1) | Não | — | S/N — descrição |

**Relacionamentos:**
- 1:N com TABELA_PRINCIPAL (via FK)

---

## 3. Decisão de armazenamento

| Opção | Status | Motivo |
|---|---|---|
| Reutilizar `TABELA_LEGADA` com filtro | **Adotado** | Motivo da decisão. Ver ADR-001. |
| Criar `TABELA_NOVA` | Rejeitado | Motivo da rejeição. Ver ADR-002. |

Colunas marcadas como "não usado" podem ser nulas. O escopo atual só preenche as colunas indicadas na tabela de uso.
```

## Formato — o que este template documenta (e o que não)

O formato documenta **APENAS tabelas de banco** — um único nível. Não mistura
tabela com entidade ORM (campos da linguagem) nem com DTO (campos de request):

| Aspecto | Fora do formato | Dentro do formato |
|---|---|---|
| Nomes | Entidade da aplicação | Tabela real do banco |
| Tipos | Linguagem (Long, LocalDate) | Banco (NUMBER, DATE) |
| Colunas não usadas | Omitidas | Listadas com "(não usado)" |
| Constraints | Apenas CK | CK + índices |
| Legado | Seção própria de mapeamento | Fonte é LEGACY_CONTEXT.md (sem redundância) |
| Value Objects (Enums) | No modelo | Vão para API_CONTRACTS.md |
| Decisão de armazenamento | Ausente | Seção §3 |
| Uso de colunas | Ausente | Tabela "Uso no escopo atual" |

## Casos de Borda

- **RN §6 sem campos suficientes para derivar a tabela**: complete com §9 (Persistência e Auditoria) e LEGACY_CONTEXT.md; o que restar ambíguo, pergunte via `ask` — não invente coluna. Sem ask nativo: pergunte no chat e PARE — nunca adivinhe.
- **Tabela reaproveitada do LEGACY_CONTEXT.md**: documente TODAS as colunas legadas (marcando "(não usado)" as não usadas) e mantenha os tipos originais do banco — não aplique as convenções de tabela nova.
- **Tipo de coluna ambíguo** (ex.: texto que pode ser VARCHAR2 ou NUMBER): pergunte via `ask` antes de escrever; nunca derive tipo da linguagem de aplicação — só tipos de banco.
- **ADR conflitante (reuso vs tabela nova)**: registre as duas opções na seção de decisão de armazenamento e resolva com `ask` ou documente a decisão final com referência ao ADR.
- **LEGACY_CONTEXT.md ausente em workspace `legado: true`**: aborte (modo `strict`) - não é possível documentar tabelas legado sem o contexto.
- **Split solicitado sem escopos claros**: se os agregados/domínios não emergem do conteúdo (tabelas todas muito entrelaçadas), pergunte via `ask` com uma proposta concreta — nunca quebre por contagem mecânica de seções; sem consenso, mantenha o unitário (as duas formas são válidas simultaneamente).
- **Split parcial (unitário + quebrado coexistindo)**: é split incompleto — termine a distribuição e remova o `DATA_MODEL.md` original; o mestre passa a ser o `specs/data_model/INDEX.md`.
- **Workspace já quebrado**: NÃO recrie o `DATA_MODEL.md` unitário — escreva no escopo certo (merge incremental) e atualize o INDEX quando criar escopo novo.

---

compatibility: multi-engine

> **Degradação de checkpoints:** o checkpoint `post-write: validate` depende do kit golmet disponível. Sem o kit, a skill continua funcionando — o checkpoint vira "rode `golmet validate` manualmente" após a escrita.
