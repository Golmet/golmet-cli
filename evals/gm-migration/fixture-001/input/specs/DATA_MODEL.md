---
project: MEUAPP
module: modelo-dados
status: draft
---

# Modelo de Dados — MeuApp (destino da migração)

> **Banco:** PostgreSQL 16 (DEC-001)

## 1. Diagrama Entidade-Relacionamento

```mermaid
erDiagram
    CLIENTE {
        NUMBER(10) COD_CLIENTE PK
        VARCHAR2(100) NOME
        VARCHAR2(14) CPF_CNPJ
        NUMBER(12,2) VAL_LIMITE_CREDITO
        NUMBER(1) FLAG_ATIVO
    }

    PEDIDO {
        NUMBER(12) COD_PEDIDO PK
        NUMBER(10) COD_CLIENTE FK
        DATE DTH_PEDIDO
        NUMBER(14,2) VAL_TOTAL
        VARCHAR2(10) STATUS
    }

    CLIENTE ||--o{ PEDIDO : "emite"
```

## 2. Tabelas

### 2.1 CLIENTE

| Coluna | Tipo | Obrigatório | PK/FK | Descrição |
|---|---|---|---|---|
| COD_CLIENTE | NUMBER(10) | Sim | PK | ID preservado do legado |
| NOME | VARCHAR2(100) | Sim | — | Padronizado em maiúsculas |
| CPF_CNPJ | VARCHAR2(14) | Sim | — | Apenas dígitos |
| VAL_LIMITE_CREDITO | NUMBER(12,2) | Não | — | Nulo legado vira zero |
| FLAG_ATIVO | NUMBER(1) | Sim | — | 1=ativo, 0=inativo |

Índices: `UK_CLIENTE_CPF_CNPJ` único.

### 2.2 PEDIDO

| Coluna | Tipo | Obrigatório | PK/FK | Descrição |
|---|---|---|---|---|
| COD_PEDIDO | NUMBER(12) | Sim | PK | ID preservado do legado |
| COD_CLIENTE | NUMBER(10) | Sim | FK | Cliente emissor |
| DTH_PEDIDO | DATE | Sim | — | Data de emissão |
| VAL_TOTAL | NUMBER(14,2) | Sim | — | Arredondado a 2 casas |
| STATUS | VARCHAR2(10) | Sim | — | Legado 'A' vira 'ABERTO' |
