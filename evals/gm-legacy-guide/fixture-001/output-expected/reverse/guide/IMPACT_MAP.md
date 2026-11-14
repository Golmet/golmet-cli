# Impact Map — MeuApp (o mapa de minas)

> Leia ANTES de qualquer evolutiva no legado. Fontes: `support/LEGACY_CONTEXT.md`,
> `DECISIONS.md` e `reverse/CANDIDATES.md`. Classificação: 🔴 dente duro
> (dinheiro/penalidade) · 🟡 atenção · 🟢 informativo.

## Armadilhas técnicas globais

| Armadilha | Fonte | Mitigação |
|---|---|---|
| Datas gravadas na hora local do servidor, não UTC | `support/LEGACY_CONTEXT.md` + DEC-001 | Novo código usa UTC (DEC-001, accepted); converter na borda ao migrar |
| SQL inline concatenado nos services | `support/LEGACY_CONTEXT.md` | Risco de injection em trechos antigos — parametrizar ao tocar |
| Sem testes automatizados | `support/LEGACY_CONTEXT.md` | Toda evolutiva exige regressão manual guiada |

## Regras confirmadas com dente

| Regra | Classificação | Evidência |
|---|---|---|
| RN-001 — Pedido acima de R$ 500 tem frete grátis | 🔴 (dinheiro) | [legado/src/PedidoService.cs:6 @f3e4a5b] |

> Regras pendentes (CAND-002, CAND-003) não entram nesta tabela — veja os
> guias do domínio e o formulário de confirmação. CAND-004 foi rejeitada
> pelo operador (comportamento acidental, não é regra).
