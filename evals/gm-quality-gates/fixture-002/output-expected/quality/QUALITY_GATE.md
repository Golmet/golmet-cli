---
project: MEUCRED-DOCS
module: quality-gate
audience: [tecnica]
status: draft
---

# Quality Gate - Consulta de Saldo (MeuCred-API)

> **Escopo:** src/ConsultaSaldoService.cs (RN-001) e testes associados
> **Veredito:** **FAIL** — há findings bloqueantes

---

## 1. Resumo

| Métrica | Valor | Limite | Status |
|---|---|---|---|
| Complexidade ciclomática (máx. por método) | 1 | ≤ 5 | ✓ |
| Cobertura de testes | 62% | ≥ 85% | ✗ |
| Duplicação | 0% | ≤ 3% | ✓ |

---

## 2. Findings Bloqueantes

| # | Severidade | Finding | Local |
|---|---|---|---|
| 1 | **Crítico** | Injeção de SQL — a query concatena `safra` e `documentoProdutor` direto no texto do comando; parâmetros devem vir de `cmd.Parameters` (OWASP A03) | `ConsultaSaldoService.Consultar`, linha do `cmd.CommandText` |
| 2 | **Alto** | Cobertura abaixo do limite — RN-001 sem teste do caminho de saldo inexistente (0) | `tests/` |

Findings Crítico e Alto **bloqueiam** o merge pelo checklist A-E.

---

## 3. Checklist A-E

| Item | Status | Observação |
|---|---|---|
| A. RN coberta por teste | ✗ | Caminho de saldo inexistente sem teste (RN-001) |
| B. Segurança rasa OWASP | ✗ | Finding Crítico de injeção de SQL |
| C. KISS/YAGNI | ✓ | Método único e direto |
| D. Padrões da stack | ✓ | `IDbConnection` injetado por construtor |
| E. Sem dívida nova | ✓ | Sem TODO/FIXME |

---

## 4. Correção exigida antes do re-gate

1. Trocar a concatenação por parâmetros (`cmd.Parameters.AddWithValue` ou
   equivalente do provider) — resolve o finding Crítico.
2. Completar o teste de saldo inexistente (RN-001) para atingir ≥ 85%.
