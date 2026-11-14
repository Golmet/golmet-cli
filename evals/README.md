# Evals de paridade — corpus do kit

> **Protocolo de paridade (§7.4 do plano de migração):** as fixtures rodam
> contra as DUAS rodas — o engine legado executa a skill antiga no workspace A,
> o engine gm executa a `gm-*` no workspace B — e o **`golmet eval` pontua os
> dois com os mesmos criteria**. Divergência de resultado entre as rodas
> **bloqueia release** até reconciliar (ou declarar divergência intencional
> como decisão registrada).

## Estrutura

```
evals/gm-<skill>/fixture-N/
├── input/              # seeds: o engine copia para o workspace e roda a skill
├── criteria.yaml       # checks que o kit pontua (dialeto YAML completo)
└── output-expected/    # (opcional) baseline congelado p/ reference_match
```

## Como rodar

```bash
# fixture única (o CI semeia o workspace com input/ e roda a skill antes)
golmet eval evals/gm-datamodel/fixture-001 --workspace <workspace-pós-skill> --json

# suite completa (convenção espelhada fixtures ↔ workspaces)
golmet eval --all evals/ <workspaces-root> --json
```

- Exit **0** tudo verde · **1** algum check error-severity quebrou (warnings
  não falham) · **2** uso inválido.
- `severity: warning` no criteria = ⚠️ informativo; `expect: fail` = eval
  negativo (o check DEVE falhar — IMP-100).
- `reference_match` compara com o baseline em `output-expected/` normalizando
  datas/horas, espaços à direita e blank lines (mesma tolerância do runner
  de referência).

## Paridade (duas rodas)

1. **Workspace A (legado):** semeia `input/`, roda a skill predecessor correspondente.
2. **Workspace B (gm):** semeia o MESMO `input/`, roda a `gm-*` correspondente.
3. **`golmet eval`** pontua A e B com o MESMO `criteria.yaml`.
4. Score por roda = passed / (passed + failed); SKIPPED não entra.
5. A != B em qualquer check error → **release bloqueada** (§7.4).

## Mapa nominal

O mapa fonte→destino das skills do workflow predecessor para o namespace `gm-*`
fica no registro privado da janela de migração (fora do escopo público do kit).
As fixtures públicas referenciam apenas o destino `gm-*`.

## Status do corpus

- **Portadas: 22 fixtures** (21 do corpus de referência + 1 newborn gm-adopt),
  cobrindo as 16 skills gm com ≥1 fixture cada.
- **Neutralização aplicada (input↔criteria em par):** produtos reais →
  `MeuApp`/`MeuApp.API`; trackers reais → `tracker #401`; nomes de skill do
  predecessor → `gm-*`; `IMP-<n>` → `IMP-REF` (identificador real removido,
  frase preservada); paths da org fora. Zero resíduos verificados por scan.
- **Follow-up (v0.5)** — não portado neste corte:
  - fixtures extras das skills multi-fixture: gm-apicontracts fixture-002,
    gm-criteria fixture-002, gm-datamodel fixture-002, gm-uispec fixture-002;
  - corpus das demais skills sem mapa nominal (orchestration/infra —
    sem equivalente gm-* no subset BASE; os packs ganharam camada de
    autoria embarcada no lote packs de 2026-11-13);
  - integração do `golmet eval --all` no gate de release do pack (§7.4).
- `mermaid_valid` é HEURÍSTICO por contrato (bloco fechado, keyword de
  diagrama válida, chaves balanceadas exceto linhas `--`) — não é parser
  completo da linguagem mermaid.
