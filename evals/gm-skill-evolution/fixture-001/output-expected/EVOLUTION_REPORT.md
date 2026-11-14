---
project: MeuApp
module: skill-evolution
audience: [tecnica]
status: draft
---

# Relatório de Evolução — gm-skill-evolution

═══ Evolução de Skills ═══

Padrão detectado: critérios de aceite com blocos "Dado" genéricos, sem
massa de dados concreta — critério não verificável e testes tautológicos.
Ocorrências: 3 projetos (alfa, beta, gama)
Skill afetada: gm-criteria
Mudança proposta: no Algoritmo, validar que todo bloco "Dado" cita massa
de dados explícita (valores literais ou tabela de exemplo); no template
de critério, incluir exemplo com tabela de massa. Prioridade alta (3+
ocorrências → aplicar imediatamente).
Impacto: baixo — mudança localizada no template e na validação, sem
quebrar fixtures existentes (eval de regressão executado após aplicar).

Aplicar esta mudança? [sim/não/adiar] → **sim** (aprovada pelo operador)

## Resultado

- Skill evoluída: gm-criteria (Passo de validação + regra de template)
- Eval de regressão: PASS — nenhuma fixture quebrada
- IMP registrado: IMP-042 (WORKFLOW_IMPROVEMENTS.md)

## Resumo

projetos analisados: 3
padrões recorrentes: 1
skills evoluídas: 1
IMPs registrados: 1
avaliadas (3+ ocorrências): 1
