# VALIDATION_REPORT — MeuApp

> **Gerado por:** gm-validate (pré-scan determinístico do kit: `golmet validate <workspace> --json`)
> **Data:** 2026-01-01
> **Escopo:** workspace inteiro contra o FORMAT v1 (regras H1–H10/S1–S5)
> **Modo:** strict (leitura pura — a skill reporta, NUNCA corrige a prosa)

## 1. Resumo

| Campo | Valor |
|---|---|
| Status | FAIL |
| Erros (H) | 4 |
| Avisos (S) | 0 |
| Arquivos varridos | 5 |

Resumo do pré-scan: `4 erro(s), 0 aviso(s) — 5 arquivo(s), RN 1, PEND 1, CR 0, ADR 0, CAND 0, Feature 3, Task 6, Spec 0`

## 2. Violações

| # | Regra | Entidade | ID | Local | ok | Mensagem |
|---|---|---|---|---|---|---|
| 1 | H3 | Feature | RN-001 | delivery/2026-11/RN-001 — Agendamento — Feature.md:1 | false | Feature ready exige anchor RN active — RN-001 está draft |
| 2 | H2 | Task | RN-001-1 | delivery/2026-11/RN-001 — Agendamento — Feature.md:13 | false | Task duplicada na Feature |
| 3 | H1 | Feature | PEND-001 | delivery/2026-11/RN-002 — Cancelamento — Feature.md:3 | false | Feature ancorada em PEND — PEND nunca ancora Feature |
| 4 | H5 | Feature | RN-003 — Relatórios — Feature.md | delivery/2026-11/RN-003 — Relatórios — Feature.md:1 | false | front matter ausente (status e anchor são obrigatórios) |

## 3. Correções sugeridas (fix do kit)

- H3 — mova RN-001 para active antes de ready
- H2 — mantenha uma seção por Task (RN-NNN-M)
- H1 — troque anchor para a RN que formalizou a pendência (ex.: anchor: RN-NNN)
- H5 — adicione '---/status: draft/anchor: RN-NNN/---' no topo do arquivo

## 4. Pendências de downstream

- RN-001 sem downstream previsto — feature `RN-001 — Agendamento` existe, mas a RN segue draft (pré-requisito H3 não cumprido).
- PEND-001 aberta com Feature ancorada — o gatilho de promoção (PEND nunca ancora feature) precisa rodar antes de qualquer advance.

## 5. Próximos passos

1. Corrigir as 4 violações acima (correção é do agente/dono — kit só valida).
2. Rodar `golmet validate` de novo — esperado: `ok — workspace conforme (FORMAT v1)`.
3. Re-gerar este relatório com contagens zeradas.
