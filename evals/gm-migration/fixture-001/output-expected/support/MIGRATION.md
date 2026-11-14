---
project: MEUAPP
module: migracao
audience: [tecnica]
status: draft
---

# Migration Plan — MeuApp (legado → destino)

> **Última atualização:** 01/01/2026
> **Status:** planejamento

---

## 1. Escopo da Migração

| Tabela legada | Tabela nova | Volume estimado | Estratégia |
|---|---|---|---|
| TB_CLIENTE_LEGADO | CLIENTE | 12.000 linhas | Big bang (carga única) |
| TB_PEDIDO_LEGADO | PEDIDO | 3.500 linhas (ST_PEDIDO = 'A') | Big bang com filtro de status (RN-002) |

Fora do escopo: pedidos fechados e cancelados (permanecem no legado,
consultáveis por relatório histórico).

## 2. Mapeamento de Dados

### 2.1 TB_CLIENTE_LEGADO → CLIENTE (RN-001)

| Campo legado | Campo novo | Transformação | Observação |
|---|---|---|---|
| CD_CLI | COD_CLIENTE | Cópia direta | PK legada preservada (RN-001) |
| NM_CLI | NOME | TRIM + UPPER | padronização de cadastro |
| CPF_CGC | CPF_CNPJ | Remover máscara (., -, /, espaço) | exigido pela UK deduplicante |
| DT_CAD | DTH_CADASTRO | Cópia direta | sem conversão de fuso |
| VL_LIMITE | VAL_LIMITE_CREDITO | COALESCE(VL_LIMITE, 0) | nulo legado vira zero |
| ST_ATIVO | FLAG_ATIVO | DECODE('S' → 1, 'N' → 0) | domínio S/N → 1/0 |

### 2.2 TB_PEDIDO_LEGADO → PEDIDO (RN-002)

| Campo legado | Campo novo | Transformação | Observação |
|---|---|---|---|
| NU_PEDIDO | COD_PEDIDO | Cópia direta | PK legada preservada |
| CD_CLI | COD_CLIENTE | Cópia direta | FK → CLIENTE |
| DT_PEDIDO | DTH_PEDIDO | Cópia direta | |
| VL_TOTAL | VAL_TOTAL | ROUND(VL_TOTAL, 2) | |
| ST_PEDIDO | STATUS | Filtro 'A' → 'ABERTO' | F/C ficam no legado |

Pré-requisito: corrigir os 12 pedidos órfãos no legado antes da carga
(falham na FK de PEDIDO → CLIENTE).

## 3. Scripts

### 3.1 Migração (UP)

```sql
-- RN-001: carga única de clientes (IDs preservados)
INSERT INTO CLIENTE (COD_CLIENTE, NOME, CPF_CNPJ, DTH_CADASTRO,
                     VAL_LIMITE_CREDITO, FLAG_ATIVO)
SELECT CD_CLI,
       UPPER(TRIM(NM_CLI)),
       REPLACE(REPLACE(REPLACE(REPLACE(CPF_CGC, '.', ''), '-', ''), '/', ''), ' ', ''),
       DT_CAD,
       COALESCE(VL_LIMITE, 0),
       CASE ST_ATIVO WHEN 'S' THEN 1 ELSE 0 END
  FROM TB_CLIENTE_LEGADO;

-- RN-002: somente pedidos abertos
INSERT INTO PEDIDO (COD_PEDIDO, COD_CLIENTE, DTH_PEDIDO, VAL_TOTAL, STATUS)
SELECT NU_PEDIDO, CD_CLI, DT_PEDIDO, ROUND(VL_TOTAL, 2), 'ABERTO'
  FROM TB_PEDIDO_LEGADO
 WHERE ST_PEDIDO = 'A';
```

### 3.2 Rollback (DOWN)

```sql
DELETE FROM PEDIDO
 WHERE COD_PEDIDO IN (SELECT NU_PEDIDO FROM TB_PEDIDO_LEGADO);

DELETE FROM CLIENTE
 WHERE COD_CLIENTE IN (SELECT CD_CLI FROM TB_CLIENTE_LEGADO);
```

## 4. Cronograma

| Etapa | Data | Responsável |
|---|---|---|
| Correção dos pedidos órfãos no legado | 05/01/2026 | DBA do legado |
| Carga de CLIENTE (RN-001) | 10/01/2026 | DBA do destino |
| Carga de PEDIDO (RN-002) | 10/01/2026 | DBA do destino |
| Validação pós-carga e aceite | 12/01/2026 | Analista de negócio |

## 5. Validação Pós-Migração

| Verificação | Query de validação | Resultado esperado |
|---|---|---|
| Total de clientes (RN-001) | `SELECT COUNT(*) FROM CLIENTE` | igual a `SELECT COUNT(*) FROM TB_CLIENTE_LEGADO` |
| Clientes com CPF inválido | `SELECT COUNT(*) FROM CLIENTE WHERE LENGTH(CPF_CNPJ) NOT IN (11, 14)` | 0 |
| Pedidos migrados (RN-002) | `SELECT COUNT(*) FROM PEDIDO` | igual a pedidos com ST_PEDIDO = 'A' no legado |
| Pedidos sem cliente | `SELECT COUNT(*) FROM PEDIDO p LEFT JOIN CLIENTE c ON p.COD_CLIENTE = c.COD_CLIENTE WHERE c.COD_CLIENTE IS NULL` | 0 |

## 6. Riscos

| Risco | Probabilidade | Impacto | Mitigação |
|---|---|---|---|
| CPF_CGC com máscara residual | Média | Alto — viola UK e aborta a carga | dedup de pré-carga com mesmo REPLACE do script UP |
| Pedido novo no legado durante a janela | Média | Médio — destino inconsistente | congelar digitação no legado durante a carga |
| VL_LIMITE nulo ignorado | Baixa | Baixo — crédito zerado | coberto por COALESCE; conferir amostra de 50 linhas |
