---
name: calc-imposto
description: Calcula impostos do pedido conforme a tabela vigente por estado.
permission:
  edit:
    "*": ask
  write:
    quality/*: allow
---

# calc-imposto — Cálculo de Impostos

## Lê (upstream)

- Payload do pedido (itens, estado de destino)
- specs/DATA_MODEL.md (contrato de PEDIDO)

## Gera (downstream)

- Campos de imposto no orçamento do pedido

## Nunca edita

- specs/DATA_MODEL.md

## Capacidades

- `file` (ler payload e contrato)
- `shell` (consulta à tabela de alíquotas vigente)

## Modo de erro

`warn` — estado sem alíquota registrada: retorne 0 com flag
`imposto_estimado: true` e registre no log do pedido.

## Algoritmo

1. Determine o estado de destino
2. Aplique a alíquota por categoria do item
3. Acumule por pedido e retorne o total

## Casos de Borda

- Pedido interestadual: aplica alíquota de origem + diferencial
- Item isento: registra isenção sem alíquota
