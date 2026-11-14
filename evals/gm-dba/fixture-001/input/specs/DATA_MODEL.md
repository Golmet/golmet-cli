---
projeto: MeuApp
artefato: DATA_MODEL
status: confirmed
---

# Modelo de Dados — MeuApp

> Fonte: REQUIREMENTS.md (RN-001, RN-002). Revisão técnica: gm-dba.

## 1. Diagrama Entidade-Relacionamento

```mermaid
erDiagram
    CLIENTE ||--o{ PEDIDO : faz
    PEDIDO ||--|{ ITEM_PEDIDO : contem
```

## 2. Tabelas

### Tabela `Clientes`

Cadastro de clientes do MeuApp. PK `Id`. Unique `Email`.

### Tabela `Pedidos`

Pedidos emitidos pelos clientes. FK `ClienteId` → `Clientes.Id` (RN-001).

### Tabela `ItensPedidos`

Itens de cada pedido. FK `PedidoId` → `Pedidos.Id` (RN-002).

### Tabela `CuponsDesconto`

Cupons promocionais — spec à frente: entrega futura, ainda sem migration.
