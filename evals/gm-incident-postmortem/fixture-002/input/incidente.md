# Input — Incidente de produção (para o post-mortem)

## Sumário

**Data:** 11/11/2026 · **Duração:** 2h05min (14:20–16:25) · **Severidade:** Alta
**Serviço afetado:** API do produto — agendamentos (POST /agendamentos)
**Impacto:** 100% das tentativas de agendamento falhavam com 500

## Timeline (UTC-3)

- 14:20 — deploy v1.4.2 em produção (migração de schema incluída)
- 14:22 — primeira falha 500 em POST /agendamentos (alerta PagerDuty)
- 14:35 — causa identificada: coluna `janela_fim` adicionada na migration
  mas o `INSERT` do endpoint não incluía a coluna NOT NULL sem default
- 15:10 — rollback da migration iniciado
- 16:25 — deploy do hotfix v1.4.3 (INSERT corrigido) — serviço normalizado

## Causa raiz

Migration adicionou coluna NOT NULL sem default, mas o código do INSERT
não foi atualizado no mesmo deploy (migration e código dessincronizados
no mesmo release).

## Ações corretivas

- Hotfix v1.4.3 com INSERT incluindo a coluna
- Migration e código passam a deployar no mesmo release (regra nova)
