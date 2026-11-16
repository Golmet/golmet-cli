# Cenário — Azure: work items + pré-condição de PAT

O primary `gm-dev` convoca o `gm-azure-orchestrator` com:

> "Sobe as tasks do módulo para o Azure Boards."

Pré-condição do MCP Azure: server desabilitado no config / PAT ausente
(ADO_PAT_B64 não configurada).

## O que o e2e valida (roteamento do orchestrator)

1. "Subir tasks para o Azure Boards" → `gm-azure-workitem-creator`
2. Pré-condição de MCP/PAT falha → `gm-azure-setup`
3. Roteia, não executa
