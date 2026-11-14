# Modelo de Dados — MeuApp

## 1. Diagrama Entidade-Relacionamento

```mermaid
erDiagram
    CLIENTE ||--o{ PEDIDO : faz
```

## 2. Tabelas

### Tabela `Clientes`

PK `Id`, unique `Email` (DEC-001, RN-001).

### Tabela `Pedidos`

FK `ClienteId` → `Clientes.Id` (RN-002: valor mínimo na aplicação).
