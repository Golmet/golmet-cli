# Modelo de Dados - Consulta

## Tabela SALDO

| Coluna | Tipo | RN |
|---|---|---|
| COD_PRODUTOR | NUMBER(10) PK | RN-001 |
| SAFRA | VARCHAR2(9) | RN-001 |
| SALDO_DISPONIVEL | NUMBER(14,2) | RN-001 |

## Tabela BOLETO

| Coluna | Tipo | RN |
|---|---|---|
| COD_BOLETO | NUMBER(10) PK | RN-002 |
| COD_PARCELA | NUMBER(10) FK | RN-002 |
| URL_PDF | VARCHAR2(255) | RN-002 |
