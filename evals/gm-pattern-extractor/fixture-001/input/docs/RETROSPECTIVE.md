# RETROSPECTIVE — MeuApp (fixture de eval)

> **Propósito:** insumo neutro para o eval da pattern library.

## Resultado do projeto

Entregas concluídas dentro do prazo; deploy inicial na VPS sem incidentes
após a adoção do checklist.

## Padrões extraídos

### Envelope único de erro HTTP nas APIs

- **Tipo:** codigo
- **Categoria:** erros
- **Gatilho:** toda resposta de erro da API precisa chegar ao frontend com
  a mesma forma, sem cada endpoint inventar seu formato
- **Projetos:** MeuApp, AppIrmao
- **Stacks:** dotnet
- **Evidência:** MeuApp-API PR #42

### Checklist de deploy em VPS antes de atualizar containers

- **Tipo:** processo
- **Categoria:** deployment
- **Gatilho:** antes de rodar o compose up em produção, conferir disco,
  healthcheck e rollback — evitar atualizar às cegas
- **Projetos:** MeuApp
- **Stacks:** all
- **Evidência:** DEC-004
