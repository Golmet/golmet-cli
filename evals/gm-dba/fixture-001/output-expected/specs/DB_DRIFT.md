# DB DRIFT — DATA_MODEL ↔ Migrations

> Gerado por `golmet db drift` (determinístico). Interpretação: gm-dba.
> Schema vivo: não conectado nesta rodada.

**Fontes:** DATA_MODEL.md (4 tabelas) · Migrations (2, `Migrations`) · ModelSnapshot (4 entidades)

## 1. Documentado sem backing de schema (1)

| Tabela documentada | Ação sugerida |
|---|---|
| `CuponsDesconto` | Verificar delivery: spec à frente (aguardando implementação) ou atualizar o modelo |

## 2. Schema sem documentação (1)

| Entidade/Tabela | Ação sugerida |
|---|---|
| `ConfiguracaoEmail` | Documentar no DATA_MODEL (ou confirmar como infra) |

Infra de framework (aceitável, citada à parte): IdentityUser

## 3. Naming entidade ↔ tabela (3 pares)

| Entidade (snapshot) | Tabela (migração/DB) |
|---|---|
| `Cliente` | `Clientes` |
| `ItemPedido` | `ItensPedidos` |
| `Pedido` | `Pedidos` |

## Resumo

matched: 3 · documented-without-schema: 1 · schema-without-docs: 1 · naming-pairs: 3
