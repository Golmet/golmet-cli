# SECURITY AUDIT — MeuApp-API

- data: 2026-01-01-0000 · modo: baseline
- escopo: src/ + specs/ · exclusões: testes, binários, vendor
- resumo: 3 findings — Crítico: 2 · Alto: 0 · Médio: 1 · Baixo: 0

## Findings

- src/PedidoController.cs:21 — injeção de SQL por concatenação (AP-01 / P-02) — Crítico — `"SELECT * FROM pedidos WHERE id = " + id` com input de rota — trocar por query parametrizada (`WHERE id = @id`)
- src/PedidoController.cs:12 — segredo hardcoded na connection string (AP-03 / P-05) — Crítico — senha de banco em literal no código-fonte — migrar para variável de ambiente ou secret manager e rotacionar a credencial exposta
- src/TokenService.cs:10 — hash fraco (AP-04 / P-07) — Médio — MD5 no hash de token — substituir por SHA-256/HMAC com chave ou hash de senha dedicado

## Não verificado

- Lockfile e pipeline de CI — arquivos fora do escopo semeado no sandbox
- Headers de segurança/TLS — sem arquivo de configuração de servidor no sandbox

## Observações

- specs/SECURITY.md exige segredos fora do código e primitiva de hash dedicada — os findings de AP-03 e AP-04 conflitam com o requisito declarado do módulo
- Sem findings de authZ: nenhum endpoint com regra de acesso por objeto no escopo semeado (verificação P-03 cumpre-se por ausência de superfície, não por controle confirmado)
- Controle presente: controller recebe conexão via injeção (_db) — apenas a connection string duplicada escapa da DI
