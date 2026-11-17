---
project: AGROCRED-DOCS
module: rastreabilidade
audience: [tecnica]
status: draft
---

# Rastreabilidade - Portal do Cliente

> **Gerado por:** gm-traceability · **Data:** 17/11/2026
> **Fonte:** REQUIREMENTS.md (RN-001, RN-002) + downstreams de specs/

---

## 1. Matriz RN vs Downstreams

| RN | DATA_MODEL | API_CONTRACTS | UI_SPEC | ERROR_CATALOG | Status |
|---|---|---|---|---|---|
| RN-001 | ✓ | ✓ | — | — | completo |
| RN-002 | ✓ | ✓ | — | — | completo |

> UI_SPEC e ERROR_CATALOG não são downstreams esperados deste módulo
> (escopo de consulta backend-only).

---

## 2. Gaps Detectados

Nenhum gap detectado — todas as RNs têm os downstreams esperados cobertos.

---

## 3. CRs Aprovados vs Implementados

| CR | Status | Reflexo |
|---|---|---|
| CR-001 | implemented | ✓ — DATA_MODEL (EXTRATO_EXPORT) + API_CONTRACTS (GET /extrato/xlsx) |
