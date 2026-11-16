# Cenário — task de feature completa no gm-dev

O operador abre sessão no primary `gm-dev` (módulo Contatos já associado,
branch-base `main` confirmada no Gate de branch) e solicita:

> "Implemente a task RN-010 do módulo: endpoint POST /assinaturas
> (contrato em API_CONTRACTS.md), teste de contrato e documentação
> na spec. A task já está no tracker."

A task é uma **feature completa** (endpoint + teste + docs = 3+ arquivos),
código ainda não iniciado, sem review prévio.

## O que o e2e valida (roteamento do primary)

1. O primary **não implementa inline** (feature 3+ arquivos → delegação)
2. O **ciclo de fechamento** é roteado ao `gm-dev-orchestrator`
3. O executor recebe a **delegação no formato da skill** (task literal,
   módulo, RNs afetadas, relatório compacto exigido)
4. **Gate de branch** antes de implementar (branch-base confirmada)
5. **QA independente** após o executor (gm-dev-qa, nunca o próprio executor)
6. **Revisão formal** (gm-dev-reviewer) antes de qualquer PR
7. PR só via `gm-gh-pr-creator` de task **aprovada pelo reviewer**
