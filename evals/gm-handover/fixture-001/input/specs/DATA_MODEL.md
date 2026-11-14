---
project: MeuApp
module: modelo-dados
audience: [tecnica]
status: draft
---

# Modelo de Dados — MeuApp (módulo Relatórios)

> **Derivado de:** REQUIREMENTS.md (RN-001, RN-002)
> **Última atualização:** 01/10/2026

---

## 1. Diagrama Entidade-Relacionamento

```mermaid
erDiagram
    RELATORIO {
        NUMBER(10) COD_RELATORIO PK
        VARCHAR2(100) NOM_RELATORIO
    }

    EXPORTACAO {
        NUMBER(10) COD_EXPORTACAO PK
        NUMBER(10) COD_RELATORIO FK
        DATE DTH_PERIODO_INICIAL
        DATE DTH_PERIODO_FINAL
        VARCHAR2(500) ARQ_CAMINHO_PDF
        DATE DTH_EXPIRACAO
    }

    RELATORIO ||--o{ EXPORTACAO : "gera"
```

---

## 2. Tabelas

### 2.1 RELATORIO

Tabela **nova** para cadastro de relatórios disponíveis (RN-001).

### 2.2 EXPORTACAO

Tabela **nova** para histórico de exportações (RN-001, RN-002).
As colunas de período atendem o filtro de data inicial/data final (RN-002);
`ARQ_CAMINHO_PDF` aponta para o arquivo no armazenamento de objetos com
validade de 24 horas (RN-001).

---

## 3. Decisão de armazenamento

- Banco relacional do MeuApp.API (mesma instância das tabelas de negócio).
- Os PDFs gerados ficam em armazenamento de objetos; a tabela guarda
  apenas o caminho do arquivo e a data de expiração.
