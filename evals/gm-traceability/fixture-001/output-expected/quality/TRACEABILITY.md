---
project: MeuApp
module: rastreabilidade
audience: [tecnica]
status: draft
---

# Traceability Matrix — MeuApp

> **Última atualização:** 12/11/2026

---

## 1. Matriz RN vs Downstreams

| RN | Descrição | DATA_MODEL | API_CONTRACTS | ERROR_CATALOG | UI_SPEC | Status |
|---|---|---|---|---|---|---|
| RN-001 | Exportação de relatórios em PDF | ✅ | ✅ | ✅ | ✅ | completo |
| RN-002 | Filtro por período nas exportações | ✅ | ✅ | ✅ | ✅ | completo |
| RN-003 | Agendamento de envio automático de relatórios | ❌ | ✅ | ❌ | ❌ | gap |

**Legenda:**
- ✅ = coberto, ❌ = gap, — = não se aplica

---

## 2. Gaps Detectados

| RN | Downstream faltante | Impacto |
|---|---|---|
| RN-003 | DATA_MODEL | Modelo não cobre o agendamento — sem estrutura para frequência, horário e destinatários |
| RN-003 | ERROR_CATALOG | Erros do agendamento sem contrato de erro definido |
| RN-003 | UI_SPEC | Tela de configuração do agendamento não especificada |

---

## 3. CRs Aprovados vs Implementados

| CR | RN afetada | Aprovado em | Refletido em downstreams |
|---|---|---|---|
| CR-001 | RN-001 | 29/09/2026 | ✅ API_CONTRACTS, ✅ UI_SPEC |
