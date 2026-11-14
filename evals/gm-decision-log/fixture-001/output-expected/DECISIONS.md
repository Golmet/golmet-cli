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

## ADR-001: Agendamento via BackgroundService no MeuApp.API

**Status:** Aceito
**Data:** 10/11/2026
**Decisor:** Bruno (líder técnico)
**Contexto:** A RN-003 (agendamento de envio automático) exige um executor recorrente. Reunião de 10/11/2026 (meetings/2026-11-10-01.md) comparou três alternativas: BackgroundService no MeuApp.API, serviço externo de agendamento e fila com worker dedicado.
**Decisão:** Usar BackgroundService hospedado no próprio MeuApp.API, com a tabela AGENDAMENTO como fonte de verdade dos horários e destinatários.
**Consequências:** Sem infraestrutura extra para operar; escala limitada a uma instância do MeuApp.API (dívida técnica registrada — migrar para worker dedicado se o volume crescer); DATA_MODEL e API_CONTRACTS devem refletir a tabela AGENDAMENTO.
