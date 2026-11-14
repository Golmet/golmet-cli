---
project: MEUAPP
module: requisitos
status: draft
---

# REQUIREMENTS — MeuApp

## 6. Dados e Campos

### RN-001 — Cadastro de Pedido

| Campo | Tipo | Obrigatório | Descrição |
|-------|------|-------------|-----------|
| clienteId | Número | Sim | ID do cliente |
| itens | Lista | Sim | Itens do pedido com quantidade |
| valorTotal | Número (12,2) | Sim | Total do pedido |
| dataPedido | Data | Sim | Data de emissão |

### RN-002 — Baixa de Estoque

| Campo | Tipo | Obrigatório | Descrição |
|-------|------|-------------|-----------|
| itemId | Número | Sim | ID do item |
| quantidade | Número | Sim | Quantidade baixada |

### RN-003 — Cadastro de Cliente

| Campo | Tipo | Obrigatório | Descrição |
|-------|------|-------------|-----------|
| nome | Texto (100) | Sim | Nome do cliente |
| cpfCnpj | Texto (14) | Sim | CPF ou CNPJ |
| limiteCredito | Número (12,2) | Não | Limite de crédito |

### RN-004 — Relatório de Vendas

| Campo | Tipo | Obrigatório | Descrição |
|-------|------|-------------|-----------|
| periodo | Data | Sim | Período de apuração |
| agrupamento | Choice | Sim | Dia, semana ou mês |

## Rastreabilidade

- RN-001 → PEDIDO (INSERT, SELECT)
- RN-002 → ESTOQUE (UPDATE)
- RN-003 → CLIENTE (INSERT, SELECT, UPDATE)
- RN-004 → consulta agregada sobre PEDIDO
