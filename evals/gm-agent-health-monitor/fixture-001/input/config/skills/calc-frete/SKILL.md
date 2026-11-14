---
name: calc-frete
description: Calcula o frete de um pedido por faixa de peso e região.
permission:
  edit:
    "*": ask
  write:
    quality/*: allow
---

# calc-frete — Cálculo de Frete

## Lê (upstream)

- Payload do pedido (peso, CEP de destino)

## Gera (downstream)

- Valor do frete no orçamento do pedido

## Algoritmo

1. Classifique a região pelo prefixo do CEP
2. Aplique a faixa de peso da tabela vigente
3. Retorne o valor com acréscimo de prazo

<!-- TODO: preencher regras de frete internacional -->
