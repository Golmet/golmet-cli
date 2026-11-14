---
project: MEUAPP
module: requisitos
status: draft
---

# REQUIREMENTS — MeuApp (migração de dados)

## 6. Dados e Campos

### RN-001 — Migração completa da base de clientes

A carga inicial deve importar 100% dos clientes do sistema legado
(TB_CLIENTE_LEGADO) antes do go-live, preservando os IDs originais.

| Campo | Tipo | Obrigatório | Descrição |
|-------|------|-------------|-----------|
| cdCli | Número | Sim | ID legado preservado |
| nome | Texto (100) | Sim | Nome padronizado em maiúsculas |
| cpfCnpj | Texto (14) | Sim | Sem máscara de formatação |

### RN-002 — Migração somente de pedidos abertos

Apenas pedidos com `ST_PEDIDO = 'A'` (aberto) migram para o MeuApp;
o histórico fechado permanece consultável no legado.

| Campo | Tipo | Obrigatório | Descrição |
|-------|------|-------------|-----------|
| nuPedido | Número | Sim | ID legado preservado |
| valorTotal | Número (14,2) | Sim | Arredondado a 2 casas |

## 9. Persistência e Auditoria

- RN-001 → CLIENTE (INSERT em carga única)
- RN-002 → PEDIDO (INSERT com filtro de status)
