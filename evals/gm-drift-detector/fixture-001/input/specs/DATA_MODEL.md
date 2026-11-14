---
project: MEUAPP
module: modelo-dados
status: draft
---

# Modelo de Dados — MeuApp

> **Derivado de:** REQUIREMENTS.md (RN-001, RN-002, RN-003), DECISIONS.md

## 1. Diagrama Entidade-Relacionamento

```mermaid
erDiagram
    CLIENTE {
        NUMBER(10) COD_CLIENTE PK
        VARCHAR2(100) NOME
        VARCHAR2(14) CPF_CNPJ
        NUMBER(12,2) VAL_LIMITE_CREDITO
    }

    PEDIDO {
        NUMBER(10) COD_PEDIDO PK
        NUMBER(10) COD_CLIENTE FK
        DATE DTH_PEDIDO
        NUMBER(12,2) VAL_TOTAL
    }

    ITEM_PEDIDO {
        NUMBER(10) COD_ITEM PK
        NUMBER(10) COD_PEDIDO FK
        NUMBER(10) QUANTIDADE
    }

    CLIENTE ||--o{ PEDIDO : "emite"
    PEDIDO ||--o{ ITEM_PEDIDO : "contem"
```

## 2. Tabelas

### 2.1 CLIENTE

| Coluna | Tipo | Obrigatório | PK/FK | Descrição |
|---|---|---|---|---|
| COD_CLIENTE | NUMBER(10) | Sim | PK | ID do cliente (sequence) |
| NOME | VARCHAR2(100) | Sim | — | Nome do cliente |
| CPF_CNPJ | VARCHAR2(14) | Sim | — | CPF ou CNPJ |
| VAL_LIMITE_CREDITO | NUMBER(12,2) | Não | — | Limite de crédito |
| CTR_DTH_INC | DATE | Sim | — | Auditoria |

Índices: `UK_CLIENTE_CPF_CNPJ` único.

### 2.2 PEDIDO

| Coluna | Tipo | Obrigatório | PK/FK | Descrição |
|---|---|---|---|---|
| COD_PEDIDO | NUMBER(10) | Sim | PK | ID do pedido (sequence) |
| COD_CLIENTE | NUMBER(10) | Sim | FK | Cliente emissor |
| DTH_PEDIDO | DATE | Sim | — | Data de emissão |
| VAL_TOTAL | NUMBER(12,2) | Sim | — | Total do pedido |
| CTR_DTH_INC | DATE | Sim | — | Auditoria |

Índices: `FK_PEDIDO_CLIENTE` (COD_CLIENTE).

### 2.3 ITEM_PEDIDO

| Coluna | Tipo | Obrigatório | PK/FK | Descrição |
|---|---|---|---|---|
| COD_ITEM | NUMBER(10) | Sim | PK | ID do item (sequence) |
| COD_PEDIDO | NUMBER(10) | Sim | FK | Pedido proprietário |
| QUANTIDADE | NUMBER(10) | Sim | — | Quantidade vendida |

Nota de revisão: conforme DEC-009 o aggregate de vendas passou a ser
calculado sob demanda; o modelo não mantém tabela de apuração.
