# Cenario - produto trata dado pessoal (LGPD)

O charter do modulo acabou de nascer com compliance: true (o produto trata
dado pessoal de usuarios). O operador pede:

> "Preciso do checklist LGPD desse produto antes de seguir para as specs."

## O que o e2e valida (roteamento do orchestrator)

1. **Checklist LGPD de 10 eixos** -> gm-compliance-check (specs/COMPLIANCE.md)
2. **Gatilho no charter** -> nota: compliance: true no charter veio do gm-charter
3. **Eixo de review** -> gm-dev-reviewer recebe os itens aplicaveis de COMPLIANCE.md
4. **RNs legais induzidas** -> gm-reverse-orchestrator pode induzir RNs de conformidade
5. Roteia, nao executa
