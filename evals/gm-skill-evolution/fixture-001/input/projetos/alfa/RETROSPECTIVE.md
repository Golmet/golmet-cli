# RETROSPECTIVE — Projeto Alfa (fixture de eval)

> Projeto fictício do MeuApp — portal de vendas.

## Resultado

Módulo de pedidos entregue; cobertura de testes dentro da meta.

## Lições aprendidas

- Critérios de aceite gerados sem massa de dados explícita geraram
  retrabalho na implementação: o dev precisou inventar valores e a
  validação divergiu da intenção da RN.

## Ações para o workflow

- `gm-criteria`: exigir que cada bloco "Dado" do Gherkin cite massa de
  dados concreta (valores literais ou tabela de exemplo) — o critério
  genérico sem exemplo não é verificável.
