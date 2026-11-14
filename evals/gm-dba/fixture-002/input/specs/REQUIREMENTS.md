---
projeto: MeuApp
artefato: REQUIREMENTS
status: confirmed
---

# REQUIREMENTS — MeuApp (fixture de eval)

> Escopo mínimo para exercitar o serviço performance do gm-dba.

## RN-001 — Cadastro e busca de clientes

O MeuApp cadastra clientes com nome, e-mail único e CPF/CNPJ do titular.
A busca por CPF/CNPJ deve localizar o cliente exato (a entrada é única
por titular).

## RN-002 — Pedidos por cliente

Cada pedido pertence a um cliente. A tela do cliente lista os pedidos
daquele cliente, os mais recentes primeiro.

## RN-003 — Itens do pedido

Cada pedido tem 1..N itens; cada item referencia um produto do catálogo,
com quantidade e preço unitário.

## RN-004 — Catálogo de produtos

Produtos com SKU único, nome e preço unitário.
