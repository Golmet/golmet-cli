#!/usr/bin/env bash
# deploy.sh — atualiza o MeuApp-API na VPS (padrao da organizacao).
# Executado pelo workflow GitHub Actions via ssh-action.
# Contrato: git pull + compose pull + compose up do servico api.
set -euo pipefail

cd /opt/meuapp

echo "[deploy] atualizando codigo na branch main..."
git pull --ff-only origin main

echo "[deploy] baixando imagem ghcr.io/meuapp/meuapp-api:latest..."
docker compose -f docker-compose.prod.yml pull api

echo "[deploy] recriando o servico api..."
docker compose -f docker-compose.prod.yml up -d --remove-orphans api

echo "[deploy] status dos containers:"
docker compose -f docker-compose.prod.yml ps
