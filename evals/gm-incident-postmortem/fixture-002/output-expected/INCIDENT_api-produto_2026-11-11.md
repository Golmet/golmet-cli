# INCIDENT â€” API do produto: falha total em agendamentos pÃ³s-deploy v1.4.2

> **Severidade:** Alta Â· **DuraÃ§Ã£o:** 2h05min (14:20â€“16:25 UTC-3, 11/11/2026)
> **ServiÃ§o:** API do produto â€” POST /agendamentos Â· **Impacto:** 100% das tentativas de agendamento com 500
> **Status:** RESOLVIDO (hotfix v1.4.3)

## Timeline

| Hora (UTC-3) | Evento |
|---|---|
| 14:20 | Deploy v1.4.2 em produÃ§Ã£o (incluindo migration de schema) |
| 14:22 | Primeiras falhas 500 em POST /agendamentos â€” alerta PagerDuty dispara |
| 14:35 | Causa identificada: coluna `janela_fim` (NOT NULL, sem default) adicionada pela migration, mas o INSERT do endpoint nÃ£o incluÃ­a a coluna |
| 15:10 | Rollback da migration iniciado (mitigaÃ§Ã£o parcial) |
| 16:25 | Hotfix v1.4.3 deployado com INSERT corrigido â€” serviÃ§o normalizado |

## Causa raiz

**Migration e cÃ³digo dessincronizados no mesmo release:** a migration
adicionou a coluna `janela_fim` (NOT NULL, sem default), mas o INSERT do
endpoint nÃ£o foi atualizado no mesmo deploy â€” o schema exigia o campo que
o cÃ³digo nÃ£o enviava.

## AÃ§Ãµes corretivas

- Hotfix v1.4.3 com INSERT incluindo `janela_fim`
- **Regra nova do time:** migration e cÃ³digo deployam no mesmo release

## Follow-ups

- [ ] Gate de CI: migration sem cÃ³digo correspondente bloqueia o release
- [ ] Teste de contrato do INSERT vs schema pÃ³s-migration

