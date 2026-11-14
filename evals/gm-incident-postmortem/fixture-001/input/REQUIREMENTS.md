---
project: MEUAPP
module: requisitos
status: draft
---

# REQUIREMENTS — MeuApp

## 6. Regras de Negócio

### RN-001 — Finalização de Pedido

O cliente finaliza o pedido com itens, valor total calculado pelo
servidor e data de emissão. Pedido finalizado dispara baixa de estoque.

| Campo | Tipo | Obrigatório | Descrição |
|-------|------|-------------|-----------|
| clienteId | Número | Sim | ID do cliente |
| itens | Lista | Sim | Itens com quantidade |
| valorTotal | Número (12,2) | Sim | Total calculado pelo servidor |

### RN-002 — Baixa de Estoque

Ao confirmar o pedido, o estoque de cada item é decrementado.
Estoque insuficiente bloqueia a confirmação com erro de negócio.

| Campo | Tipo | Obrigatório | Descrição |
|-------|------|-------------|-----------|
| itemId | Número | Sim | ID do item |
| quantidade | Número | Sim | Quantidade baixada |
