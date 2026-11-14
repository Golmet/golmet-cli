# Checklist de deploy em VPS antes de atualizar containers

**Categoria:** deployment
**Tipo:** processo
**Projetos que usaram:** MeuApp (2026-11)
**Stacks aplicáveis:** all
**Maturidade:** novo
**Última revisão:** 2026-11-12
**Status:** ativo
**Superado por:** —
**Evidência:** DEC-004

## Cenário

Antes de rodar o compose up em produção na VPS, conferir os pré-requisitos
para evitar atualizar às cegas.

## Descrição

Passo a passo executado antes de cada atualização de containers: conferir
espaço em disco, imagem publicada no registry, estado atual dos containers
e caminho de rollback documentado.

## Implementação

1. Conferir espaço em disco e memória livre na VPS
2. Confirmar que a imagem nova foi publicada no registry
3. Registrar o estado atual dos containers (ps + versão da imagem)
4. Executar o pull e o up do serviço
5. Validar o healthcheck e guardar o comando de rollback

## Quando NÃO usar

Ambientes de desenvolvimento descartáveis — o checklist viraria cerimônia.

## Variações

Com pipeline de deploy automatizado, os itens 1-3 viram checks do workflow.

## Referências

- Retrospectiva: MeuApp/RETROSPECTIVE.md
- Evidência de aplicação: DEC-004
