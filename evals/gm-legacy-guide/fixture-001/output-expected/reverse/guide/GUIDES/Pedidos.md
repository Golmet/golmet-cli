# Guia — Domínio Pedidos

> Superfície de confirmação e evolutiva. Fontes: `reverse/CANDIDATES.md`
> (ledger governa), código legado e `DECISIONS.md`. Prosa narra; o ledger
> decide o estado de cada regra.

## Narrativa (fluxo ponta a ponta)

Um cliente emite um pedido; o sistema calcula o frete conforme o valor
total; o cancelamento marca o pedido como "Cancelado" e aciona o domínio
Estoque para devolver os itens. A reimpressão de cupom re-registra um
bônus no cliente — comportamento confirmado no código, rejeitado como
regra pelo operador (ver CAND-004).

## Arquivos do domínio

- `legado/src/PedidoService.cs` — frete, cancelamento, reimpressão

## Regras confirmadas ✅

### RN-001 — Frete grátis acima de R$ 500

Pedidos acima de R$ 500,00 têm frete grátis (dente: dinheiro).

- **Proveniência:** CAND-001 aprovada pelo cliente → RN-001
- **Evidência:** [legado/src/PedidoService.cs:6 @f3e4a5b]

## Regras pendentes ⏳

### CAND-003 — Cancelamento devolve itens ao estoque

O cancelamento do pedido devolve os itens ao estoque (integração com o
domínio Estoque).

- **Evidência:** [legado/src/PedidoService.cs:13 @f3e4a5b]
- **Decisão pendente:** confirme no `FORMS/Pedidos.md` — Confirma /
  Confirma com correção / Não é regra / Não sei (classificação é do
  agente no write-back; a resposta do cliente é livre)

## Regras rejeitadas ❌

### CAND-004 — Reimpressão de cupom dispara bônus duplicado

Rejeitada pelo operador: comportamento acidental do legado, não é regra
de negócio. Divergência registrada; se o cliente quiser mudança de
comportamento, registrar como CR.

- **Evidência:** [legado/src/PedidoService.cs:18 @f3e4a5b]

## Cenários de evolutiva

- **Alterar o valor do frete grátis:** tocar em
  `legado/src/PedidoService.cs` (RN-001) — regressão manual no cálculo
- **Mudar o fluxo de cancelamento:** CAND-003 ainda pendente — confirme
  antes de mudar (ver `FORMS/Pedidos.md`)
