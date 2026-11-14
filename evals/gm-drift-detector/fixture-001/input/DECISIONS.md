---
project: MEUAPP
module: decisoes
status: draft
---

# DECISIONS — MeuApp

## DEC-001 — PostgreSQL como banco único

Adotado PostgreSQL 16 para o MeuApp-API. Motivo: operação simples e
suporte a constraints declarativas. Alternativas descartadas: MySQL
(padronização já existente em PostgreSQL na stack da casa).

## DEC-002 — Auditoria com colunas CTR_*

Toda tabela transacional carrega `CTR_DTH_INC` no mínimo. Motivo:
rastreabilidade de criação sem dependência de triggers. Atualização
com `CTR_DTH_ATU` fica para tabelas editáveis.
