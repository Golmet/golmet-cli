# DB PERFORMANCE — Queries, Índices e N+1

> Serviço performance do gm-dba — análise estática de padrões no código
> quente (sem conexão nesta rodada: EXPLAIN indisponível).
> Schema vivo: não conectado — achados NÃO verificados contra schema vivo.

**Fontes:** código quente (`repo/MeuApp.Api` — 3 arquivos) · DATA_MODEL.md (4 tabelas) · Migrations (1, `Migrations`)

## 1. Resumo de achados (3)

| ID | Severidade | Padrão | Local |
|---|---|---|---|
| PRF-001 | Crítica | N+1 (query em loop) | `RelatorioService.ObterResumoPedidosAsync` |
| PRF-002 | Alta | Filtro sem índice em coluna de alta cardinalidade | `ClienteService.BuscarPorCpfCnpjAsync` |
| PRF-003 | Média | SELECT * | `PedidoRepository.ListarPorClienteAsync` |

## 2. N+1 — PRF-001 (Crítica)

`RelatorioService.ObterResumoPedidosAsync`: `FindAsync` de cliente DENTRO
do `foreach` dos pedidos — 1 query por pedido (N+1). Com 500 pedidos no
período, são 501 round-trips ao banco.

Correção sugerida: uma única query com JOIN/`Include` carregando
`Pedido` + `Cliente` juntos e montagem do resumo em memória. A correção
roda pelo fluxo do gm-dev — este parecer descreve, nunca aplica.

## 3. Filtro sem índice em coluna de alta cardinalidade — PRF-002 (Alta)

`ClienteService.BuscarPorCpfCnpjAsync` (RN-001): filtro por
`Clientes.CpfCnpj` — coluna de alta cardinalidade (quase única por
cliente) e SEM índice na migration inicial (`IX_Clientes_Email` e
`IX_Produtos_Sku` são os únicos não-clusterizados existentes). Cada busca
varre a tabela inteira (scan) e degrada linearmente com o volume da base.

## 4. SELECT * — PRF-003 (Média)

`PedidoRepository.ListarPorClienteAsync` (RN-002): SQL literal com
`SELECT * FROM Pedidos` — trafega todas as colunas (incluindo as pesadas)
e quebra em runtime quando o schema evolui. Substituir pela projeção das
colunas usadas (ou query LINQ equivalente com `AsNoTracking`).

## 5. Índices sugeridos (2)

```sql
CREATE INDEX IX_Clientes_CpfCnpj ON Clientes (CpfCnpj);
CREATE INDEX IX_Pedidos_ClienteId_CriadoEm ON Pedidos (ClienteId, CriadoEm DESC);
```

O segundo índice atende a RN-002 (listagem por cliente, mais recentes
primeiro) e a query do PRF-003 — filtro e ordenação cobertos por um
único índice.

> `CREATE INDEX` pronto para o playbook da stack avaliar (EF hand-written —
> migration nova roda pelo fluxo do gm-dev). DDL exige gate explícito do
> gm-dba — NADA aplicado nesta análise (read-only por default; o parecer
> descreve, nunca aplica).

## Resumo

crítica: 1 · alta: 1 · média: 1 · n+1: 1 · índices sugeridos: 2
