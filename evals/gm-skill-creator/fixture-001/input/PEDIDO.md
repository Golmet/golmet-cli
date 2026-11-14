# PEDIDO — Criar skill gm-foo (fixture de eval)

> **Contexto:** projeto fictício MeuApp. Requisitos já coletados via
> questionário (3 blocos) — a skill não precisa perguntar novamente.

## Bloco 1 — Identificação

- **Nome:** `gm-foo`
- **Propósito:** auditar a cobertura de testes por RN — cruza as RNs do
  REQUIREMENTS.md com os relatórios de teste do módulo e aponta quais RNs
  estão sem teste ou com teste parcial.

## Bloco 2 — Domínio e fluxo

- **Domínio/fase:** Qualidade.
- **Upstreams:** `REQUIREMENTS.md` (RNs), relatórios de teste do módulo
  (`quality/test-results.md` quando existir), `delivery/` (tasks e status).
- **Downstreams:** `quality/COVERAGE_AUDIT.md` — matriz RN × teste com
  status (coberta / parcial / descoberta).
- **Orchestrator:** `@gm-quality-orchestrator` registra a skill na tabela
  "Skills disponíveis".

## Bloco 3 — Comportamento

- **Modo de erro:** `warn` — módulo sem relatório de teste gera relatório
  parcial com a lacuna declarada, sem bloquear o pipeline.
- **Formato de commit:** `docs(quality): audit coverage por RN`.

## Referência de estilo

Seguir o padrão das skills existentes do config do engine (front matter
do kit com `name`, `description`, `requires`, `degrades`; seções Lê/Gera/
Nunca edita/Tools/Modo de erro/Formato de commit/Algoritmo/Template/
Casos de Borda). PT-BR.
