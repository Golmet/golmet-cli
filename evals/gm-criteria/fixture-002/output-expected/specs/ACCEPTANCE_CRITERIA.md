---
project: FINANCIA-DOCS
module: aceitacao
audience: [tecnica]
status: draft
---

# Critérios de Aceitação - Simulação de Financiamento

> **Derivado de:** REQUIREMENTS.md (RN-001, RN-002)
> **Formato:** Gherkin (Dado-Quando-Então), derivado de EARS

---

## RN-001

**Tipo EARS:** Evento - "Quando o produtor solicitar..., então o sistema deve
exibir... e registrar..."

### Cenário: Simulação acima do limite disponível

**Dado** que o produtor está autenticado
**Quando** o produtor solicita uma simulação com valor acima do limite disponível
**Então** o sistema exibe a mensagem de limite insuficiente e registra a tentativa

---

## Pendências de clarificação

> RN-002 não é expressável em EARS (critério subjetivo de interface, sem
> comportamento observável). Pergunta aberta ao operador — nenhum cenário
> Gherkin foi inventado para ela.

- **RN-002** ("A interface da simulação deve ser bonita e intuitiva"):
  quais critérios objetivos e mensuráveis definem "bonita e intuitiva"?
  Candidatos a discutir: tempo até a primeira simulação concluída, taxa de
  erro de preenchimento, avaliação com usuário real. Definição pendente —
  vira PEND no REQUIREMENTS até o operador responder.
