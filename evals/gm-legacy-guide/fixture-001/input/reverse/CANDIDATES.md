# Candidates — MeuApp (ledger append-only)

> Regras induzidas do reverse. approved → graduada RN · pending → aguarda
> confirmação humana · rejected → descartada pelo operador.

## Domínio Pedidos

### CAND-001 — Frete grátis acima de R$ 500

- **status:** approved → RN-001
- **evidência:** [legado/src/PedidoService.cs:6 @f3e4a5b]

### CAND-003 — Cancelamento devolve itens ao estoque

- **status:** pending
- **evidência:** [legado/src/PedidoService.cs:13 @f3e4a5b]

### CAND-004 — Reimpressão de cupom dispara bônus duplicado

- **status:** rejected (operador: comportamento acidental, não é regra)
- **evidência:** [legado/src/PedidoService.cs:18 @f3e4a5b]

## Domínio Estoque

### CAND-002 — Reserva de estoque por 30 minutos

- **status:** pending
- **evidência:** [legado/src/EstoqueService.cs:6 @f3e4a5b]
