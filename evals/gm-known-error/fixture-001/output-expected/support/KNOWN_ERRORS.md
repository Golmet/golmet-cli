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

---

## [KE-002] — Exportação travada em 100% sem entregar o arquivo

**Data do incidente:** 10/11/2026
**Status:** aberto

**Sintoma:**
Usuário aciona **Exportar PDF**, a barra de progresso chega a 100% e o
arquivo não aparece em Minhas Exportações. Sem mensagem de erro na tela —
distinto dos erros previstos em contrato (EXP-422 e EXP-401).

**Causa:**
Em investigação — suspeita de falha na notificação de conclusão do
serviço de relatórios do MeuApp.API.

**Workaround:**
Reduzir o período do filtro (exportar semana a semana) e tentar
novamente; se persistir, registrar o horário do incidente e acionar o
suporte de nível 2.

**Resolução definitiva:**
—
