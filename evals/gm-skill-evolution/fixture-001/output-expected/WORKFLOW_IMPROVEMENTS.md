---
project: MeuApp
module: workflow-improvements
audience: [tecnica]
status: draft
---

# WORKFLOW_IMPROVEMENTS — MeuApp

Registro append-only das melhorias aplicadas no workflow pelo
gm-skill-evolution (uma entrada IMP-XXX por mudança aplicada).

## IMP-042 — gm-criteria: exigir massa de dados explícita nos blocos "Dado"

- **Data:** 12/11/2026
- **Alvo:** `gm-criteria`
- **Problema:** critérios de aceite gerados com blocos "Dado" genéricos
  ("dado um pedido válido") sem massa de dados concreta — o teste vira
  tautologia e a implementação interpreta a RN de duas formas.
- **Causa raiz:** template do Gherkin não exigia exemplo concreto no
  bloco "Dado" e o Algoritmo não validava verificabilidade.
- **Correção:** passo novo no Algoritmo (validar que cada "Dado" cita
  valores literais ou tabela de massa de dados) + regra correspondente no
  template de critério.
- **Projetos que motivaram:** alfa, beta, gama
- **Arquivos:** skills/gm-criteria/SKILL.md
- **Status:** aplicada
