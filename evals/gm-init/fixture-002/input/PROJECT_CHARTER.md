---
status: confirmed
legado: true
sistema_existente: true
---

# PROJECT CHARTER - Portal do Cliente (AgroCred)

## Visão Geral

Modernizar o acesso do produtor rural ao status de crédito: consulta de
saldo, extrato de parcelas e download de boletos. Hoje tudo isso acontece
por telefone e e-mail, com atendimento manual. Sistema existente em
funcionamento — sem criação de módulos novos de crédito.

## Escopo (primeira entrega)

- Consulta de saldo e limite disponível pelo produtor
- Extrato de parcelas (pagas e a vencer)
- Download de boletos em PDF

## Fora de escopo (por enquanto)

- Simulação de novas operações de crédito
- Assinatura eletrônica de contratos
- App móvel nativo

## Stakeholders

- Produtor rural (consulta self-service)
- Analista de crédito (esclarece dúvidas)
- Patrocinador: gerência de atendimento

## Decisões de abertura

- Sistema existente em produção — modernização da fachada de consulta
  (legado: true, sistema_existente: true)
- Escopo simplificado: sem repos novos declarados nesta fase
