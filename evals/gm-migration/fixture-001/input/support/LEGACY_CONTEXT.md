---
project: MEUAPP
module: contexto-legado
status: draft
---

# LEGACY_CONTEXT — Sistema Legado MeuApp-Legado

> Inventário reverse: banco Oracle 11g, acesso via SQL direto no ERP.

## Tabelas

### TB_CLIENTE_LEGADO (~12.000 linhas)

| Coluna | Tipo | Observação |
|--------|------|------------|
| CD_CLI | NUMBER(10) | PK |
| NM_CLI | VARCHAR(100) | com espaços extras |
| CPF_CGC | VARCHAR(18) | com máscara (., -, /) |
| DT_CAD | DATE | |
| VL_LIMITE | NUMBER(12,2) | nulo em ~8% das linhas |
| ST_ATIVO | CHAR(1) | domínio S/N |

### TB_PEDIDO_LEGADO (~250.000 linhas; ~3.500 com ST_PEDIDO = 'A')

| Coluna | Tipo | Observação |
|--------|------|------------|
| NU_PEDIDO | NUMBER(12) | PK |
| CD_CLI | NUMBER(10) | FK → TB_CLIENTE_LEGADO |
| DT_PEDIDO | DATE | |
| VL_TOTAL | NUMBER(14,2) | |
| ST_PEDIDO | CHAR(1) | A=aberto, F=fechado, C=cancelado |

## Armadilhas

- CPF_CGC com máscara inconsistente — normalizar antes de dedup
- VL_LIMITE nulo — tratar como zero na carga
- Pedidos órfãos (CD_CLI sem cliente) — 12 ocorrências detectadas
