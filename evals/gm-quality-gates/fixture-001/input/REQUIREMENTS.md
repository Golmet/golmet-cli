---
project: MEUAPP
module: requisitos
status: draft
---

# REQUIREMENTS — MeuApp

## 6. Regras de Negócio

### RN-001 — Cadastro de Pedido

Sistema registra pedidos com cliente, itens e total calculado pelo
servidor (nunca aceito do cliente).

| Campo | Tipo | Obrigatório | Descrição |
|-------|------|-------------|-----------|
| clienteId | Número | Sim | ID do cliente |
| itens | Lista | Sim | Itens com quantidade e preço unitário |
| valorTotal | Número (12,2) | Sim | Soma calculada pelo servidor |

### RN-002 — Baixa de Estoque

Ao confirmar o pedido, o estoque é decrementado; insuficiência bloqueia
a confirmação com erro de negócio.

| Campo | Tipo | Obrigatório | Descrição |
|-------|------|-------------|-----------|
| itemId | Número | Sim | ID do item |
| quantidade | Número | Sim | Quantidade baixada |

## Rastreabilidade da task

- Task em escopo do gate: RN-001 (cadastro de pedido)
- RN-002: fora do escopo desta task
