---
name: gm-apicontracts
description: >-
   API_CONTRACTS.md (endpoints por grupo, convenções) e ERROR_CATALOG.md (validação 400 / negócio 422 / recurso 404) a partir do DATA_MODEL e das RNs.
requires: [file, edit, write]
---

# gm-apicontracts — Contratos de API

Gera `<Produto>/specs/API_CONTRACTS.md` e `<Produto>/specs/ERROR_CATALOG.md`
a partir do modelo de dados e das regras de negócio.

## Referências

- Entidade RN (seções `## RN-NNN — <título>`): FORMAT §7.2
- Layout do workspace (pasta `specs/`): FORMAT §2

## Resolução de caminhos

Os artefatos do workspace seguem o layout do FORMAT §2 — a saída fica em
`specs/API_CONTRACTS.md` e `specs/ERROR_CATALOG.md`.

**Regra de fallback (transição):** ao **LER** um artefato, verifique primeiro
o caminho novo (ex: `specs/DATA_MODEL.md`); se não existir, use o caminho
legado na raiz do workspace (ex: `DATA_MODEL.md`). Ao **ESCREVER/GERAR**, use
sempre o caminho novo. Workspaces ainda não adotados continuam legíveis
durante a transição.

## Lê (upstream)

- `<Produto>/specs/DATA_MODEL.md` (essencial — entidades, campos, tipos; fallback: raiz do workspace)
- `<Produto>/REQUIREMENTS.md` §5 (Regras de Negócio) e §7 (Casos de Uso)
- `<Produto>/reverse/LEGACY_CONTEXT.md` (se existir, para endpoints existentes; fallback: raiz do workspace)

## Gera (downstream)

- `<Produto>/specs/API_CONTRACTS.md`
- `<Produto>/specs/ERROR_CATALOG.md`

## Nunca edita

- `REQUIREMENTS.md`
- `DATA_MODEL.md`
- `golmet.index.json` (único escritor é o kit)

## Capacidades

- `file` — ler DATA_MODEL.md, REQUIREMENTS.md, LEGACY_CONTEXT.md e escrever API_CONTRACTS.md, ERROR_CATALOG.md
- `ask` — para ambiguidades de contrato (sem ask nativo: pergunte no chat e PARE — nunca adivinhe)

## Modo de erro

`strict` — se os contratos não forem gerados, o fluxo de UI spec não pode prosseguir.

## Propagação de confiança (legado)

- Afirmações derivadas de RN induzida de engenharia reversa **herdam a marca de confiança da RN de origem** (campo `**Confiança:**` em REQUIREMENTS.md).
- Escala: Alta → 🟢 CONFIRMADO · Média → 🟡 INFERIDO · Baixa → 🔴 GAP.
- **Regra de ouro:** na dúvida, nível mais baixo.
- Quando disponível, anexe a evidência da análise reversa (`[arquivo:linha @hash]`) — **afirmação sem evidência nunca recebe 🟢**.

## Formato de commit

```
docs: add api contracts and error catalog for <produto>
```

## Workflow

> **Regra de idioma:** Documentos no idioma do cliente (padrão PT-BR). Títulos de seção, descrições, mensagens de erro e nomes amigáveis no idioma combinado. Apenas identificadores técnicos (códigos HTTP, nomes de campos JSON, chaves de enum, notação Mermaid) permanecem no original.
>
> **Regra de nomenclatura de colunas de banco:** Ao documentar colunas de tabelas (ex.: em seções de tabela de auditoria, mapeamento de campos, ou referências a DATA_MODEL.md), os nomes das colunas DEVEM estar em **MAIÚSCULAS** conforme as convenções do container (ex.: `CTR_DTH_INC`, não `ctr_dth_inc` nem `dataHora`). Isso aplica-se a qualquer referência a coluna de banco no API_CONTRACTS.md ou ERROR_CATALOG.md.

### Fase 1 — Descobrir a API de destino

Antes de modelar endpoints, identifique **qual é o projeto de API correto** onde os endpoints serão implementados:

1. Leia `PROJECT_CHARTER.md` §1.1 — repos do produto com branch e HEAD (FORMAT §7.1)
2. Procure no sistema de arquivos por projetos de API relacionados aos repos registrados:
   - Busque em diretórios vizinhos pelo repo de API do produto (ex.: `<nome>-API/`)
   - Busque por marcadores da stack (arquivos de configuração de aplicação, definições de rota/route) para entender o padrão de URL
3. **Regra de precedência**: a API dedicada do produto é a fonte da verdade para base URL.
4. Use `LEGACY_CONTEXT.md` apenas como **referência** de endpoints que já existem no legado — não como padrão para a nova API

### Fase 1b — Aprender com workspaces irmãos

Antes de modelar, verifique se há outros workspaces do mesmo container de organização e aprenda suas convenções:

1. Busque por `specs/API_CONTRACTS.md` em workspaces existentes (fallback: raiz do workspace)
2. Extraia **convenções gerais** da organização:
   - Formato de URL (com/sem prefixo `/api`)
   - Envelope de erro (array vs objeto)
   - Paginação (offset vs cursor-based)
   - Formato de datas (ISO, MM/YYYY, contextual)
   - Códigos HTTP padrão
   - Headers customizados (Include-Fields, Exclude-Fields)
3. Documente essas convenções na seção "Convenções Gerais" do `API_CONTRACTS.md`
4. Se workspaces irmãos existirem e usarem padrões diferentes do que você planeja, priorize consistência com o projeto existente

### Fase 2 — Modelar contratos

5. Leia `DATA_MODEL.md` — tabelas, colunas, tipos, PKs, FKs, valores de enum (comentários nas colunas como `1=Bloquear, 2=Apenas Avisar`)
6. Leia `REQUIREMENTS.md §5` — RNs em EARS que viram validações de API
7. Leia `REQUIREMENTS.md §7` — casos de uso que viram endpoints
8. Leia `LEGACY_CONTEXT.md` (se existir) — endpoints existentes como referência apenas
9. Para cada endpoint, defina:
   - Método HTTP e path
   - Request body/params (campos, tipos, obrigatoriedade)
   - Response body (estrutura, campos, envelope)
   - Status codes
   - Validações de campo (derivadas de RNs EARS)
   - **"Usado em"** — qual tela/componente/funcionalidade consume o endpoint
10. Para cada coluna com valores fixos em DATA_MODEL.md, crie uma seção **Value Objects (Enums)** nas Convenções Gerais
11. Para cada validação de RN, registre o erro correspondente em ERROR_CATALOG.md
12. Escreva `specs/API_CONTRACTS.md` e `specs/ERROR_CATALOG.md`
13. Rode `golmet validate` (checkpoint post-write) — ou anote a pendência se o kit não estiver disponível

### Decisão de Paginação

Para cada endpoint de listagem, decidir entre:

| Estratégia | Quando usar |
|------------|-------------|
| **Sem paginação** (array direto) | Volumes pequenos e estáticos (< 100 registros, sem crescimento) |
| **Cursor-based** | Listas grandes, crescimento contínuo, scroll infinito, dados mutáveis (inserção/remoção frequente) |
| **Offset-based** (page/size) | Listas com paginação manual (controles "Anterior/Próximo"), dados estáveis durante a sessão |

Se o volume estimado não estiver claro no DATA_MODEL.md ou REQUIREMENTS.md, use `ask` para perguntar.

### Envelope de Resposta

| Tipo de resposta | Envelope |
|-----------------|----------|
| Recurso único | Retornado **diretamente**, sem envelope |
| Lista paginada | `{ content, nextCursor/offset, hasNext/totalPages }` |
| Lista não paginada | Array direto `[]` |

### Datas

O formato varia por contexto semântico do campo:

| Contexto | Formato | Exemplo |
|----------|---------|--------|
| Data de evento (criação, ocorrência de negócio) | `YYYY-MM-DD` | `2026-08-01` |
| Data de vigência (início, fim) | `YYYY-MM-DD` | `2026-12-31` |
| Data mês/ano (fabricação, validade) | `MM/YYYY` | `01/2026` |
| Timestamp (auditoria) | `YYYY-MM-DDTHH:mm:ssZ` | `2026-08-01T10:30:00Z` |

Mapear cada campo de data ao formato correspondente conforme seu contexto de negócio. NÃO usar ISO genérico para tudo.

## Template — API_CONTRACTS.md

```markdown
# Contratos de API — [Nome do Sistema]

> **Derivado de:** DATA_MODEL.md, REQUIREMENTS.md (RN-001, RN-002)
> **Última atualização:** DD/MM/AAAA

---

## Convenções Gerais

**Contrato parcial/revisão (FULL):** endpoints ou RNs ainda em revisão —
marque os campos não fechados como `a definir` no contrato (status do
endpoint em revisão), liste-os no checklist de pendências da execução e
NUNCA invente payload/código de erro por omissão.

### URL Base

[Base URL conforme padrão do projeto. Ex: `/api` sem prefixo de versão, ou outra convenção adotada.]

### Formato de Datas

| Contexto | Formato |
|----------|---------|
| Datas de evento | `YYYY-MM-DD` (ISO 8601) |
| Datas mês/ano | `MM/YYYY` |
| Timestamps | `YYYY-MM-DDTHH:mm:ssZ` |

### Envelope de Resposta — Listagens com Paginação

[Cursor-based ou offset, decidido conforme workflow de paginação]

```json
{
  "content": [...],
  "nextCursor": null,
  "hasNext": false
}
```

### Envelope de Resposta — Recurso Único

Recursos individuais são retornados diretamente, sem envelope:

```json
{ "id": 12345, "campo": "valor" }
```

### Envelope de Erro

Sempre um array, mesmo para erro único:

```json
[
  {
    "id": null,
    "message": "Descrição do erro",
    "field": "nomeDoCampo",
    "detail": null
  }
]
```

Para erros de validação com múltiplos campos:

```json
[
  { "id": null, "message": "Campo obrigatório", "field": "categoriaId", "detail": null },
  { "id": null, "message": "Data inválida", "field": "data", "detail": null }
]
```

### Paginação (Parâmetros)

| Parâmetro | Tipo | Padrão | Descrição |
|-----------|------|--------|-----------|
| `cursor` | string | `null` | Cursor para próxima página (Base64 do último ID) |
| `limit` | integer | `50` | Itens por página (max 100) |

### Códigos HTTP Utilizados

| Código | Situação |
|--------|----------|
| `200 OK` | Leitura bem-sucedida |
| `201 Created` | Recurso criado com sucesso |
| `204 No Content` | Exclusão bem-sucedida |
| `400 Bad Request` | Erro de validação dos dados enviados |
| `404 Not Found` | Recurso não encontrado |
| `409 Conflict` | Conflito de dados |
| `422 Unprocessable Entity` | Regra de negócio violada |
| `500 Internal Server Error` | Erro genérico do servidor |

### Value Objects (Enums)

Para cada campo do tipo enum/integer com valores fixos, documentar os valores possíveis e seus significados:

```markdown
#### TipoAcao

| Código | Descrição | Efeito |
|---|---|---|
| 1 | Bloquear | Impede a operação |
| 2 | Apenas Avisar | Não bloqueia, apenas registra observação |

#### TipoMovimentacao

| Código | Descrição |
|---|---|
| 1 | Entrada (Recebimento) |
| 2 | Saída (Envio) |
| 3 | Ambos (Entrada e Saída) |
```

**Como derivar:** Das colunas da DATA_MODEL.md com valores fixos comentados (ex: `1=Bloquear, 2=Apenas Avisar`) e dos conceitos de domínio em REQUIREMENTS.md §4.

### Glossário de Tipos

| Tipo | Significado | Uso |
|---|---|---|
| Long | Inteiro 64 bits | IDs de entidades |
| Integer | Inteiro 32 bits | IDs pequenos, enums, FKs |
| String(N) | Texto com N caracteres | Códigos, descrições |
| Boolean | Verdadeiro/Falso | Flags |

---

## 1. [Nome do Grupo: Dados Mestres / Operações / ...]

Todos os endpoints desta seção são `GET` sem corpo de requisição. Endpoints sem paginação retornam arrays completos (volumes pequenos e estáticos). Endpoints com paginação usam [cursor-based/offset].

---

### 1.1 [Nome do Recurso]

```
GET /[path]
```

**Paginação:** [cursor-based / offset / nenhuma]

**Query Params:**

| Parâmetro | Tipo | Obrigatório | Descrição |
|-----------|------|-------------|-----------|
| `param1` | string | Não | Filtro... |

**Header Params:**

| Parâmetro | Tipo | Obrigatório | Descrição |
|-----------|------|-------------|-----------|
| `Include-Fields` | string (CSV) | Não | Lista de campos a retornar |
| `Exclude-Fields` | string (CSV) | Não | Lista de campos a não retornar |

**Resposta `200 OK`:**

```json
[
  { "id": 1, "campo": "valor" }
]
```

**Usado em:** [Tela/Componente que consome este endpoint]

---

## 2. [Nome do Grupo: Operações]

### 2.1 [Método] /[path]

**RNs relacionadas:** RN-001, RN-002

**Request:**

```json
{
  "campo": "tipo (obrigatório/opcional)"
}
```

| Campo | Tipo | Obrigatório | Descrição |
|-------|------|-------------|-----------|
| campo | String | Sim | Descrição |

**Response 201:**

```json
{
  "id": 1,
  "campo": "valor"
}
```

**Status codes:**

| Código | Condição |
|--------|----------|
| 201 | Recurso criado com sucesso |
| 400 | Campo obrigatório ausente |
| 422 | Regra de negócio violada (ver ERROR_CATALOG.md) |

**Validações:**

| Condição | Erro | HTTP |
|----------|------|------|
| Campo obrigatório ausente | "Campo obrigatório: {field}" | 400 |
| Regra de negócio violada | Mensagem específica | 422 |

**Usado em:** [Tela/Componente]

---

## 3. Padrões Comuns

### 3.1 Datas

| Contexto | Formato |
|----------|---------|
| [contexto 1] | `YYYY-MM-DD` |
| [contexto 2] | `MM/YYYY` |
```

## Template — ERROR_CATALOG.md

```markdown
# Catálogo de Erros — [Nome do Sistema]

> Erros de API previstos em contrato. Incidentes de suporte imprevistos
> vão em KNOWN_ERRORS.md (se o workspace o mantiver — fronteira: contrato
> previsto vs incidente real).

---

## 1. Erros de Validação (400)

| Código | Campo | Mensagem | RN | HTTP |
|--------|-------|----------|----|------|
| ERR-001 | nome | Campo obrigatório | RN-001 | 400 |
| ERR-002 | email | Formato de email inválido | RN-002 | 400 |

## 2. Erros de Negócio (422)

| Código | Campo | Mensagem | RN | HTTP |
|--------|-------|----------|----|------|
| ERR-010 | dataVencimento | Registro vencido | RN-003 | 422 |
| ERR-011 | quantidade | Mínimo 1 quantidade > 0 | RN-004 | 422 |

## 3. Erros de Recurso (404)

| Código | Campo | Mensagem | RN | HTTP |
|--------|-------|----------|----|------|
| ERR-020 | id | Recurso não encontrado | — | 404 |
```

## Casos de Borda

- **Nenhuma API de destino encontrada**: pergunte via `ask` onde os endpoints serão implementados antes de modelar — não assuma o monorepo legado como base URL. Sem ask nativo: pergunte no chat e PARE — nunca adivinhe.
- **Workspaces irmãos com convenções conflitantes**: priorize a consistência com o projeto existente; persistindo o conflito, documente a decisão na seção "Convenções Gerais".
- **DATA_MODEL.md sem colunas com valores fixos comentados**: não invente enums — documente apenas os Value Objects deriváveis das colunas comentadas e dos conceitos de §4 de REQUIREMENTS.md.
- **Volume de listagem indefinido**: use `ask` para decidir a estratégia de paginação (sem paginação / cursor / offset) em vez de escolher arbitrariamente.
- **Endpoint existente só no legado**: documente-o como referência vinda de LEGACY_CONTEXT.md — não vira contrato da nova API.

---

compatibility: multi-engine

> **Degradação de checkpoints:** o checkpoint `post-write: validate` depende do kit golmet disponível. Sem o kit, a skill continua funcionando — o checkpoint vira "rode `golmet validate` manualmente" após a escrita.
## Convergência com a twin BASE (FULL)

A twin BASE `gm-apicontracts` publicada no kit cobre o fluxo essencial. Esta edição
FULL preserva a twin INTEGRAL (nenhum passo BASE removido) e adiciona os
extras FULL do pipeline (seções marcadas "FULL" neste documento). Seções BASE e FULL coincidem neste item — a BASE já consolidava o conteúdo FULL da fonte.
