# Guia — Domínio Estoque

> Superfície de confirmação e evolutiva. Fontes: `reverse/CANDIDATES.md`,
> código legado e `support/LEGACY_CONTEXT.md`.

## Narrativa (fluxo ponta a ponta)

O estoque reserva itens no checkout do pedido. O código do domínio é
enxuto: uma única operação de reserva com expiração. Não há regra
confirmada neste domínio — a única regra induzida (CAND-002) aguarda
confirmação do cliente.

## Arquivos do domínio

- `legado/src/EstoqueService.cs` — reserva de itens

## Regras confirmadas ✅

Nenhuma regra confirmada ainda neste domínio.

## Regras pendentes ⏳

### CAND-002 — Reserva de estoque por 30 minutos

A reserva de um item no checkout expira em 30 minutos.

- **Evidência:** [legado/src/EstoqueService.cs:6 @f3e4a5b]
- **Decisão pendente:** confirme no `FORMS/Estoque.md` — Confirma /
  Confirma com correção / Não é regra / Não sei (classificação é do
  agente no write-back; a resposta do cliente é livre)

## Cenários de evolutiva

- **Alterar o tempo de reserva:** tocar em `legado/src/EstoqueService.cs`
  — confirme a regra antes (CAND-002 pendente)
- **Integração com Pedidos:** o cancelamento do pedido devolve itens ao
  estoque (ver `GUIDES/Pedidos.md`, CAND-003 pendente)
