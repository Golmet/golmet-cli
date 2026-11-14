---
project: MeuApp
module: decisoes
audience: [tecnica]
status: draft
---

# Architecture Decision Records — MeuApp

## DEC-001: Exportação síncrona para relatórios pequenos

**Status:** Aceito
**Data:** 20/08/2026
**Decisor:** Não informado
**Contexto:** Entrada legada (pré-migração para ADR-NNN) — IDs DEC-XXX são imutáveis e nunca são renumerados.
**Decisão:** Relatórios com até 500 linhas são exportados de forma síncrona; acima disso, a geração é enfileirada.
**Consequências:** Timeouts do gateway evitados; fila assíncrona documentada apenas na prosa do legado.
