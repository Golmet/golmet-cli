# Inventário do repo — MeuApp-API (fixture de eval)

> **Propósito:** estado do repositório local antes da geração do workflow.

## Estrutura confirmada

- `Dockerfile` — presente (build multi-stage da API)
- `docker-compose.prod.yml` — presente, serviço `api` com healthcheck
- `.github/workflows/` — pasta vazia (nenhum workflow de deploy)

## Pendências do repo

- `scripts/deploy.sh` — **ausente**; precisa ser criado no repo antes do
  workflow (padrão da organização: `git pull` + `compose pull` + `compose up`).
  O workflow assume o script presente e executável na VPS.

## Contexto de build

- A API é empacotada em imagem única; o deploy atualiza o serviço `api`
  do compose de produção a partir da imagem `latest` do registry.
