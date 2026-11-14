# PEDIDO — Fundar base de conhecimento (fixture de eval)

> **Contexto:** projeto fictício MeuApp. Pedido do operador para o
> gm-workflow-builder encadear a fundação antes de criar a camada que
> consome a base.

## Domínio e escopo

- **Domínio:** `event-sourcing`
- **Escopo:** base para projetar persistência com event sourcing em
  Postgres — agregados como stream de eventos, projeções, snapshots e
  outbox para integração.

## Observações

- `packs/org-pack/domains/INDEX.md` já existe com o domínio `code-quality` fundado —
  o merge é incremental: a entrada existente deve ser preservada e a
  nova entrada do `event-sourcing` adicionada.
- Usar fontes primárias atuais com URL + data de acesso (documentação
  oficial quando aplicável) — nunca confiar na memória.
- Nenhum componente pode ser criado antes da base existir
  (conhecimento antes de componente).
