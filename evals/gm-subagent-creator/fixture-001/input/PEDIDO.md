# PEDIDO — Criar subagent gm-foosub (fixture de eval)

> **Contexto:** projeto fictício MeuApp. Requisitos já coletados via
> questionário (3 blocos) — a skill não precisa perguntar novamente.

## Bloco 1 — Identificação

- **Nome:** `gm-foosub`
- **Propósito:** diagnosticar a cobertura de testes do módulo e recomendar
  lacunas — leitura e recomendação, sem editar nada.

## Bloco 2 — Comportamento

- **O que faz:** lê `REQUIREMENTS.md`, `quality/` e `delivery/`; cruza as
  RNs com as evidências de teste; devolve diagnóstico estruturado com as
  lacunas priorizadas.
- **Escopo de permissão:** **Read-only** — diagnóstico/recomendação:
  read/glob/grep allow; edit/write/shell deny.
- **Skills que o subagent invoca:** nenhuma — função executável
  diretamente (lê artefatos e diagnostica).

## Bloco 3 — Registro

- Registrar no `registry.yaml`, no primary `gm-req` e no `ARCHITECTURE.md`
  (padrão: sim).
