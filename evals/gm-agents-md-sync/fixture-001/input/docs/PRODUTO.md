# Produto MeuApp — contexto da sincronização (fixture de eval)

> **Propósito:** insumo neutro para o eval de sincronização de AGENTS.md
> entre os repositórios do produto MeuApp.

## Repositórios do produto

| Sandbox | Repositório | Camada |
|---------|-------------|--------|
| `repo-a/` | MeuApp-API | backend |
| `repo-b/` | MeuApp-FE | frontend |

## Base comum aprovada na auditoria (Passo 4)

O relatório de auditoria classificou como `COMMON_CANDIDATE` e o operador
aprovou a promoção das seguintes regras ao bloco `PRODUCT_COMMON`:

1. Commits seguem Conventional Commits (feat:, fix:, docs:, chore:, ci:)
2. Documentação do produto escrita em PT-BR
3. Cobertura mínima de testes: 80% (padrão mais rigoroso entre os repos)

Os três repos divergiam apenas na redação — nenhum conflito semântico foi
detectado, portanto nada permanece como pendência.

## Preservação obrigatória

Instruções específicas de cada repo permanecem no bloco `REPO_SPECIFIC` e
nunca são promovidas ao bloco comum:

- MeuApp-API: porta 8080, migrations na pasta `migrations/`
- MeuApp-FE: build via `npm run build`, lint antes de PR

A sincronização não pode mover paths, comandos ou portas para o bloco
comum por engano.
