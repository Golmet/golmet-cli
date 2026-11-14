# Perfil de Infra — MeuApp (fixture de eval)

> **Propósito:** insumo neutro para o eval de pipeline de deploy do produto MeuApp.

## Produto e organização

- **Produto:** MeuApp
- **Repositório alvo:** MeuApp-API (branch de deploy: `main`)
- **Organização GitHub:** `meuapp`
- **Registry de imagens:** `ghcr.io/meuapp`

## VPS de destino

| Campo | Valor |
|-------|-------|
| Host | `vps.meuapp.example` |
| Usuário SSH | `deploy` |
| Porta | 22 |
| Pasta de trabalho | `/opt/meuapp` |

## Domínio e validação

- **Domínio da API:** `api.meuapp.example` (via tunnel)
- **Validação ponta a ponta:** container do serviço `api` precisa aparecer `(healthy)` no `docker ps` após o deploy.

## Compose de produção

- Arquivo: `docker-compose.prod.yml`
- Serviço alvo: `api`
- Imagem esperada: `ghcr.io/meuapp/meuapp-api:latest`

## Secrets necessários no GitHub

| Secret | Onde obter |
|--------|-----------|
| `SSH_HOST` | host da VPS (`vps.meuapp.example`) |
| `SSH_USERNAME` | usuário de deploy (`deploy`) |
| `SSH_PRIVATE_KEY` | chave privada do usuário de deploy (com BEGIN/END completos) |

Nunca registrar o valor de secrets no chat nem no workflow — só o nome.

## Branch e gatilho

- Deploy automático em push na branch `main`.
- Sem deploy por tag neste produto.
