---
project: AGROCRED-DOCS
ok: false
violations: 1
status: FAIL
---

# Relatório de Validação - Portal do Cliente

> **Executado em:** 17/11/2026 · **Pré-scan:** `golmet validate . --json` — 1 erro(s), 0 aviso(s)

## 1. Resumo

| Indicador | Valor |
|---|---|
| Violações hard | 1 |
| Violações soft | 0 |
| Status | FAIL |

## 2. Violações

| # | Regra | Severidade | Detalhe |
|---|---|---|---|
| 1 | CR aprovado sem reflexo downstream | H (hard) | CR-001 aprovado sem reflexo downstream — "exibir a data da última atualização" não tem campo correspondente em specs/DATA_MODEL.md (nenhum campo de atualização na tabela SALDO) |

## 3. Coerência do Workspace

- O CR-001 segue aprovado em REQUIREMENTS.md (intacto — a skill não edita prosa).
- A violação persiste até o downstream refletir o CR (DATA_MODEL documenting
  DTH_ULTIMA_ATUALIZACAO ou equivalente).

## 4. Próximo Passo

Propagar o CR-001 para o DATA_MODEL (coluna de atualização do saldo) e
re-rodar `golmet validate` — o status vira PASS sem violações.
