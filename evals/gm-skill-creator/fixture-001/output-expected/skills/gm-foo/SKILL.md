---
name: gm-foo
description: >-
  Audita a cobertura de testes por RN — cruza as RNs do REQUIREMENTS.md
  com os relatórios de teste do módulo e gera quality/COVERAGE_AUDIT.md
  com o status de cada RN (coberta / parcial / descoberta). Use quando o
  usuário pedir "auditar cobertura", "cobertura por RN" ou "RNs sem teste".
requires: [file, glob, grep, ask]
degrades:
  graph: grep
---
# gm-foo — Auditoria de Cobertura de Testes por RN

Audita a cobertura de testes do módulo a partir das RNs do
REQUIREMENTS.md e dos relatórios de teste disponíveis. Responde a
pedidos como "quais RNs estão sem teste?" e "cobertura por RN".

## Diferenciação

| Skill | Faz |
|-------|-----|
| `gm-foo` | Matriz RN × teste com status de cobertura (qualidade) |
| `gm-quality-gates` | Gate universal de qualidade da stack (build, testes verdes, security) |
| `gm-traceability` | Rastreabilidade de RNs para documentos downstream (specs), não para testes |

## Lê (upstream)

- `REQUIREMENTS.md` — RNs do módulo (IDs no padrão RN-NNN)
- `quality/test-results.md` — relatório de testes do módulo (quando existir)
- `delivery/` — tasks e status de implementação por RN

## Gera (downstream)

- `quality/COVERAGE_AUDIT.md` — matriz RN × teste com status
  (coberta / parcial / descoberta) e resumo quantitativo

## Nunca edita

- REQUIREMENTS.md e demais artefatos de módulo (apenas leitura)
- Código de aplicação e testes
- Config do engine (agents, skills, registry.yaml)

## Tools

- `glob` (localizar REQUIREMENTS.md, relatórios de teste e tasks)
- `grep` (extrair IDs de RN e referências a testes)
- `file` (ler artefatos e escrever o relatório)
- `ask` (confirmar escopo quando o módulo for ambíguo)

> **Escopo de tools:** cite apenas capacidades que a skill verifica
> disponíveis na sessão; capacidade indisponível ganha caminho alternativo
> (fallback) no Algoritmo — nunca há passos que só funcionam com a
> capacidade habilitada.

## Modo de erro

`warn` — módulo sem relatório de teste (`quality/test-results.md`
ausente) gera o relatório parcial com a lacuna declarada na matriz
(todas as RNs marcadas como descobertas), sem bloquear o pipeline.

## Formato de commit

```
docs(quality): audit coverage por RN
```

## Algoritmo (ou Workflow)

### Passo 1 — Delimitar o módulo

Confirme com o operador (via `ask`, se ambíguo) qual módulo auditar:
localize o `REQUIREMENTS.md` do módulo com `glob` e extraia a lista de
RNs (IDs no padrão RN-NNN). Critério de saída: lista de RNs sem
duplicatas. Módulo sem REQUIREMENTS.md → encerre com `needs-requirements.`

### Passo 2 — Coletar evidências de teste

Localize `quality/test-results.md` e as tasks em `delivery/`. Para cada
RN, procure referências de teste (grep por `RN-\d+` nos relatórios e
specs de teste). Critério de saída: para cada RN, 0+ evidências citadas.

### Passo 3 — Gerar a matriz e o resumo

Escreva `quality/COVERAGE_AUDIT.md` com a matriz RN × teste (status
coberta / parcial / descoberta — parcial = RN com teste que não cobre
todos os campos/cenários do §6) e o resumo quantitativo (total por
status). Cada linha cita a evidência (caminho do teste). RN sem
evidência entra como descoberta — nunca omitida.

## Template de saída (se aplicável)

```markdown
---
project: <projeto>
module: coverage-audit
audience: [tecnica]
status: draft
---
# Auditoria de Cobertura — <módulo>

| RN | Status | Evidência |
|----|--------|-----------|
| RN-001 | coberta | tests/pedido.spec.ts |
| RN-002 | descoberta | — |

Resumo: N cobertas · N parciais · N descobertas
```

## Casos de Borda

- **Módulo sem relatório de teste**: gere a matriz com todas as RNs
  descobertas e declare a lacuna no topo do relatório (modo warn).
- **RN sem downstream de teste citável**: marque como descoberta —
  nunca infira cobertura sem evidência.
- **Nome já existe**: informe e pergunte — sobrescrever, renomear, cancelar.
- **Skill de projeto específico**: registre no mapa do projeto, não na
  seção do workflow do config.
