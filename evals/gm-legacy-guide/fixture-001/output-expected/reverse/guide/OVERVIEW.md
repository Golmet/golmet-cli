# Overview — MeuApp (camada humana do legado)

> Gerado pela gm-legacy-guide a partir de `reverse/CANDIDATES.md`,
> `support/LEGACY_CONTEXT.md` e `DECISIONS.md`. Todo fato técnico cita a
> fonte. Regras pendentes NÃO afirmam comportamento — só o que está
> confirmado alimenta a narrativa de conversação.

## O que o sistema é

O MeuApp é um sistema de pedidos legado (ASP.NET clássico + SQL Server,
acesso por SQL inline nos services — ver `support/LEGACY_CONTEXT.md`).
O código organizado em controllers e services é a fonte dos domínios abaixo.

## Domínios (derivados do código — um domínio só existe se existe código)

| Domínio | Código | CANDs |
|---|---|---|
| Pedidos | `legado/src/PedidoService.cs` | CAND-001 (approved → RN-001) · CAND-003 (pending) · CAND-004 (rejected) |
| Estoque | `legado/src/EstoqueService.cs` | CAND-002 (pending) |

## Como os domínios se conversam (só regras confirmadas)

- **Pedidos → Estoque:** o cancelamento do pedido devolve os itens ao
  estoque — comportamento citado no código
  ([legado/src/PedidoService.cs:13 @f3e4a5b]) e sustentado pela CAND-003,
  ainda pendente de confirmação do cliente (não afirmar como regra até a
  confirmação).

## Por onde começar

1. Leia `GUIDES/Pedidos.md` e `GUIDES/Estoque.md` antes de tocar no código
2. Consulte `IMPACT_MAP.md` para as armadilhas (fuso horário, SQL inline)
3. Responda `FORMS/Pedidos.md` e `FORMS/Estoque.md` para confirmar as
   regras pendentes — o write-back gradua CAND → RN ou registra PEND/CR
