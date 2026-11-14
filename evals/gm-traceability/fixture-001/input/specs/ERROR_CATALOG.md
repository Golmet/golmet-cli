---
project: MeuApp
module: catalogo-erros
audience: [tecnica]
status: draft
---

# Error Catalog — MeuApp.API (módulo Relatórios)

> Erros previstos em contrato. Incidentes reais de suporte vão para
> support/KNOWN_ERRORS.md (ver CONSTITUTION.md §7).

## Exportações

| Código | HTTP | Situação | RN |
|---|---|---|---|
| EXP-422 | 422 | Período inválido — data final anterior à data inicial | RN-002 |
| EXP-401 | 401 | Sessão expirada durante a exportação | RN-001 |
| EXP-404 | 404 | Relatório inexistente ou sem permissão | RN-001 |
