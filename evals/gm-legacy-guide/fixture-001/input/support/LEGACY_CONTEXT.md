# LEGACY_CONTEXT — MeuApp

> Inventário do legado (saída do gm-reverse). Fonte dos fatos técnicos.

## Stack

- ASP.NET (Framework legado), controllers + services diretos
- Banco SQL Server, acesso por SQL inline nos services

## Armadilhas

- **Fuso horário:** datas gravadas na hora local do servidor, não em UTC
  (ver DEC-001 — decisão da org: novo código usa UTC)
- **SQL inline concatenado** nos services — risco de injection em trechos antigos
- **Sem testes automatizados** — toda evolutiva precisa de regressão manual

## Glossário de equivalência

| Legado | Destino |
|---|---|
| Pedido | Pedido (RN-001) |
| Estoque | Estoque |
