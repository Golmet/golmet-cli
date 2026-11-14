---
project: MEUAPP
module: api-contracts
status: draft
---

# API Contracts — MeuApp-API

> **Base:** /api/v1 · autenticação via token do operador

## Contratos por RN

### RN-001 — POST /pedidos

Request: `clienteId`, `itens[]`, `dataPedido`.
Response 201: `pedidoId`, `valorTotal` (calculado pelo servidor).

### RN-002 — POST /pedidos/{id}/confirmar

Confirma o pedido e dispara a baixa de estoque.
Erros: `ESTOQUE_INSUFICIENTE` (422).

### RN-003 — POST /clientes · PUT /clientes/{id}

Cadastro e atualização de cliente.
Erros: `CPF_CNPJ_DUPLICADO` (422).

## Convenções

- Erros de negócio usam 422 com código estável; recurso ausente, 404.
- Validação de payload no handler de aplicação, não no controller.
