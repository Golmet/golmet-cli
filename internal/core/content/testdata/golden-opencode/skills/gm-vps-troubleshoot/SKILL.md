---
name: gm-vps-troubleshoot
description: >-
   Diagnóstico guiado por sintoma para VPS já configurada: container unhealthy, GHCR denied, SSH action falhando, services duplicados, tunnel compartilhado, porta ocupada, banco vazio. Modo guiado com triagem por tabela de sintomas (runbook).
requires: [file, glob, grep, ask, shell, ssh]
degrades: [edit, write]
---

# gm-vps-troubleshoot — Diagnóstico de VPS por Sintoma

Diagnóstico guiado para VPS da organização **já configurada** (setup inicial é
papel da `gm-vps-setup`). Recebe o sintoma, tria na tabela de sintomas,
conduz o diagnóstico em ordem de verificação e propõe a correção. Dois
modos de execução (seção abaixo): **guiado** (default — comandos são
sugeridos com explicação e o operador executa na VPS) e **SSH** (opcional —
a skill executa o diagnóstico direto na VPS via capacidade `ssh`, sempre
com confirmação).

## Diferenciação

| Skill | Escopo |
|-------|--------|
| `gm-vps-troubleshoot` (esta) | Diagnóstico **por sintoma** em VPS existente |
| `gm-vps-setup` | Setup **inicial** de VPS nova (Docker, GHCR, SSH, tunnel, backup) |
| `gm-known-error` | Registrar incidente resolvido em KNOWN_ERRORS.md (pós-resolução) |

## Lê (upstream)

- `runbook.md` (nesta pasta) — sintomas, comandos de diagnóstico, causas e
  correções validados em produção
- Repo local do projeto quando disponível (`docker-compose.prod.yml`,
  `scripts/deploy.sh`) — para comparar porta do healthcheck, service names
  e variáveis

## Gera (downstream)

- Diagnóstico guiado no chat: sintoma → causa provável → comandos → correção
- Relatório final com causa raiz, correção aplicada e verificação

## Nunca edita

- Qualquer arquivo na VPS no modo guiado (default) — diagnóstico apenas
  sugerido; no modo SSH, executar EXCLUSIVAMENTE comandos de diagnóstico
  aprovados pelo operador (correções destrutivas SEMPRE exigem confirmação
  explícita, em ambos os modos)
- Artefatos de módulo e config do engine
- Segredos: nunca pedir PAT/chave privada/token no chat

## Capacidades

- `file` / `glob` / `grep` (runbook.md, scripts e compose do repo local)
- `ask` (coletar sintoma e outputs; confirmar correção destrutiva — sem
  ask nativo, pergunte no chat e PARE)
- `shell` (apenas leitura local — nunca SSH direto)
- `ssh` (APENAS no modo SSH opcional — sessão/comandos na VPS
  com confirmação; ver seção "Modos de execução")

## Modos de execução

| Modo | Como ativa | Comportamento |
|------|-----------|---------------|
| **Guiado** (default) | — | Comandos de diagnóstico sugeridos com explicação; operador executa na VPS e cola o output |
| **SSH** (opcional) | Operador pede "diagnosticar via SSH" | Skill usa a capacidade `ssh` (`ssh_session_start` para sessão persistente, comandos de diagnóstico na sessão, `ssh_health_check`/`ssh_service_status` quando aplicável) |

**Regras do modo SSH (obrigatórias):**
- Comandos de DIAGNÓSTICO (read-only: `docker ps`, `docker logs`,
  `journalctl`, `curl`, `ss`, `nc`) podem rodar em sequência após o
  operador autorizar o início do diagnóstico na VPS
- **Correção**: cada correção é
  proposta e confirmada individualmente — destrutivas (`kill -9`,
  `systemctl stop/disable`, `rm` de unit, `docker compose down`) exigem
  confirmação explícita SEMPRE
- Confirmar o server alvo via `ssh_list_servers` antes de conectar — nunca
  assumir por omissão
- Saída de cada comando validada contra o critério do passo antes de
  avançar na ordem de verificação
- Server não configurado na capacidade `ssh` → orientar cadastro e cair para
  o modo guiado

## Modo de erro

`warn` — sintoma não catalogado na tabela não aborta: caia para o
diagnóstico genérico (seção 8 do runbook: logs + conectividade) e, se
resolvido, **proponha registrar o novo padrão** no `runbook.md` desta skill
(com confirmação do operador). Correções destrutivas (remover service
systemd, kill -9, `docker compose down`) exigem confirmação explícita
antes de sugerir execução.

## Formato de commit

```
docs: atualiza runbook da gm-vps-troubleshoot
```

## Algoritmo

### Passo 1 — Coletar o sintoma

Pergunte (um `ask` agrupado): sintoma literal, servidor, o que
foi feito antes, output de erro completo. Compare com a tabela de triagem.

### Passo 2 — Triar na tabela de sintomas

| # | Sintoma | Causa provável | Seção do runbook |
|---|---------|----------------|------------------|
| 1 | Container `(unhealthy)` / deploy.sh expira: `💔 Container is not healthy` | Porta do healthcheck divergente da porta real da app (ex.: compose aponta 8080 e a app sobe na **8082**), ou rota `/health` inexistente, ou app não terminou de subir | S1 |
| 2 | `Head "https://ghcr.io/v2/...": denied` / `unauthorized` no pull | Sem login GHCR ou visibilidade do pacote | S2 |
| 3 | `fatal: could not read Username for 'https://github.com'` | Remote em HTTPS no `deploy.sh` dentro de Action | S3 |
| 4 | `ssh: handshake failed: ... no supported methods remain` | authorized_keys/permissões/secret da ssh-action | S4 |
| 5 | `ssh.ParsePrivateKey: ssh: no key found` | Secret incompleto ou nome errado | S5 |
| 6 | Service systemd em `activating (auto-restart)` / dois services cloudflare | Service duplicado pós re-execução do script de autostart | S6 |
| 7 | Requisições caindo no servidor errado / indisponibilidade intermitente | Tunnel compartilhado entre servidores | S7 |
| 8 | `docker compose up` falha / comportamento estranho na porta | Processo ocupando a porta (ex: cloudflared órfão vs Postgres) | S8 |
| 9 | API no ar mas dados ausentes — `Did not find any relations` | Banco vazio, migrations não aplicadas | S9 |
| 10 | Backup/cron não roda | crond ausente ou rclone.conf no root | S10 |
| 11 | Sintoma não listado | — | S8 (genérico) |

### Passo 3 — Conduzir o diagnóstico da seção (ordem de verificação)

Siga a ordem da seção correspondente no `runbook.md` — cada comando tem
critério de interpretação (ex: `curl -i` SEMPRE com `-i` para ver o status
HTTP: `404` é rota ausente, `connection refused` é porta errada — problemas
diferentes). Colete o output de cada passo antes de avançar. Diagnóstico
básico de container: `docker ps` (estado/health) e `docker logs --tail 50`
(saída da app).

### Passo 4 — Propor correção (com confirmação)

Apresente a correção com explicação da causa raiz. Correções destrutivas
(param em S6/S8) → confirme antes. Nunca sugira duas correções ao mesmo
tempo — uma por vez, revalidando após cada.

### Passo 5 — Verificação e relatório

Confirme a correção (ex: container `(healthy)`, `curl -I
https://<hostname>.<dns_domain do perfil>/health` OK de fora) e feche:

```text
═══ gm-vps-troubleshoot — <RESULTADO> ═══

Sintoma: <sintoma>
Causa raiz: <causa>
Correção: <o que foi feito>
Verificação: <como confirmou>
Lição: <novo padrão? proponha registro no runbook.md>
```

Se a lição for nova, ofereça: registrar no `runbook.md` desta skill ou
formalizar incidente via `gm-known-error` / `gm-incident-postmortem`.

## Casos de Borda

- **Sintoma múltiplo (2+ da tabela)**: trate na ordem — S8 (genérico:
  logs/conectividade) primeiro para isolar camada (app vs tunnel vs DNS).
- **Tunnel compartilhado confirmado (S7)**: não "conserte" o balanceamento
  — crie tunnel exclusivo (Fase 7 da `gm-vps-setup`) e migre os hostnames.
- **Service duplicado (S6)**: rodar um comando POR VEZ — colar explicação
  junto com o comando quebra o shell. Sempre `daemon-reload` +
  `reset-failed` no fim, e `ps aux | grep cloudflared` para órfãos.
- **Deploy que falhou mas funcionava antes**: comece por S4/S3 (algo mudou
  na chave/secret) antes de S1.
- **Banco vazio (S9)**: Compose garante o Postgres no ar, não o schema —
  verificar migrations no startup (validação do schema no startup da app
  ou equivalente) ou rodar manualmente.
- **Novo sintoma resolvido**: proponha registro no `runbook.md` (com
  confirmação) — o runbook é base de conhecimento viva.
