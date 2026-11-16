# Cenário — Tradução Técnica (fan-out a partir de REQUIREMENTS)

O primary `gm-req` fechou REQUIREMENTS.md (RNs confirmadas, sem PENDs
bloqueantes) e convoca o `gm-translation-orchestrator`:

> "RNs confirmadas e fechadas. Gera a camada técnica completa."

## O que o e2e valida (roteamento do orchestrator)

1. **Fan-out paralelo** a partir de REQUIREMENTS.md
2. Pré-condição: REQUIREMENTS completo **sem PENDs bloqueantes**
3. Skills de tradução roteadas: data-model, api-contracts, ui-spec,
   architecture, glossary, acceptance-criteria, task-spec
