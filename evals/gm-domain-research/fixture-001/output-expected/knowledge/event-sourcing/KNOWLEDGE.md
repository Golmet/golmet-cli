---
dominio: event-sourcing
versao: 1.0.0
data_pesquisa: 2026-11-12
proxima_revisao: 2027-05-12
escopo: persistência com event sourcing em Postgres (agregados como stream de eventos, projeções, snapshots, outbox)
fontes:
  - titulo: Event Sourcing — Martin Fowler (EAADev)
    url: https://martinfowler.com/eaaDev/EventSourcing.html
    acessado_em: 2026-11-12
    versao: n/a
  - titulo: PostgreSQL Documentation
    url: https://www.postgresql.org/docs/current/
    acessado_em: 2026-11-12
    versao: current
  - titulo: Event Sourcing pattern — Azure Architecture Center
    url: https://learn.microsoft.com/azure/architecture/patterns/event-sourcing
    acessado_em: 2026-11-12
    versao: n/a
---
# KNOWLEDGE — Event Sourcing (persistência com event sourcing em Postgres)

Base canônica do domínio para consumo por camadas/componentes.
Fonte canônica de critérios — componentes que consomem este domínio listam
este arquivo no "Lê (upstream)" deles. Atualizar via gm-domain-research
quando `proxima_revisao` vencer.

## Princípios

### P-01 — O estado é derivado da sequência de eventos
O registro de eventos (append-only) é a fonte da verdade; o estado atual
do agregado é reconstruído reproduzindo a sequência de eventos. Gravar
estado mutável ao lado do stream destrói a propriedade de auditoria.
- **Heurística de violação**: tabela com UPDATE/DELETE no agregado
  coexistindo com o stream sem justificativa de projeção descartável.
- Fonte: (Fowler, Event Sourcing — "The fundamental idea")

### P-02 — Eventos descrevem fatos do negócio, não mudanças de tabela
O nome e o payload do evento refletem a linguagem do domínio
(pedido.confirmado) e são imutáveis; renomear coluna não é motivo para
reescrever eventos históricos (use upcasting de versão).
- **Heurística de violação**: eventos nomeados como update_tabela_x com
  payload espelhando colunas físicas.
- Fonte: (Azure Architecture Center — Event Sourcing pattern, Issues and considerations)

### P-03 — Projeções são descartáveis e reconstruíveis
Projeções (read models) em Postgres são materializações para consulta;
qualquer projeção pode ser refeita do zero replays do stream — o
checkpoint da projeção acompanha o offset do stream.
- **Heurística de violação**: projeção sem mecanismo de rebuild e sem
  checkpoint consistente com o offset lido.
- Fonte: (PostgreSQL Documentation — logical replication concepts aplicados a checkpoints)

## Checklist aplicável

- [ ] O stream é append-only (sem UPDATE/DELETE nos eventos) (ref: P-01)
- [ ] Cada evento tem tipo, versão e payload no vocabulário do domínio (ref: P-02)
- [ ] Snapshot strategy definida para agregados longos (a cada N eventos) (ref: P-01)
- [ ] Toda projeção tem rotina de rebuild + checkpoint por offset (ref: P-03)
- [ ] Outbox para integração usa a mesma transação do append do evento (ref: P-03)

## Anti-patterns

### AP-01 — Stream como tabela mutável
Eventos gravados em tabela com UPDATE para "corrigir" eventos antigos —
a auditoria é perdida e o rebuild de projeção diverge do histórico real.
Correção de evento é evento novo de compensação.
- Viola: P-01 · Severidade: Crítico

## Classificação/Severidade

| Nível | Critério | Ação na triagem |
|-------|----------|-----------------|
| Crítico | mutabilidade no stream ou perda de auditoria | bloqueia aprovação |
| Alto | projeção sem rebuild/checkpoint | corrige antes de aprovar |
| Médio | payload de evento sem versão explícita | backlog priorizado |
| Baixo | nomenclatura de evento inconsistente | backlog comum |

## Referências

1. Event Sourcing — Martin Fowler — https://martinfowler.com/eaaDev/EventSourcing.html (acessado em 2026-11-12)
2. PostgreSQL Documentation — https://www.postgresql.org/docs/current/ (acessado em 2026-11-12)
3. Event Sourcing pattern — Azure Architecture Center — https://learn.microsoft.com/azure/architecture/patterns/event-sourcing (acessado em 2026-11-12)
