---
project: MeuApp
audience: [suporte, usuario_final]
status: draft
---

# Known Errors

> Incidentes de suporte reais. Erros de API previstos em contrato vão em
> ERROR_CATALOG.md (ver CONSTITUTION.md §7).

---

## [KE-001] — PDF corrompido para relatórios acima de 10 mil linhas

**Data do incidente:** 03/09/2026
**Status:** resolvido

**Sintoma:**
Usuários reportavam que o PDF baixado não abria — o arquivo chegava
truncado para relatórios com mais de 10 mil linhas.

**Causa:**
O timeout do gateway cortava a geração antes do arquivo ser finalizado.

**Workaround:**
Filtrar o relatório por períodos menores até a correção definitiva.

**Resolução definitiva:**
Geração assíncrona com arquivo temporário assinado, implantada em
04/09/2026 no MeuApp.API v1.1.3.
