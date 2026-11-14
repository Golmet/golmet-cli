# gm-pack-scaffold — Scaffold de pack de organização

Onboarding de uma nova organização no ecossistema de packs (camada L2): a
partir **apenas** do contrato de pack (FORMATO §3.1), cria o esqueleto
completo do pack — nada do template assume uma organização específica.
Passos de efeito externo (repo remoto, registro no `golmet.yaml`, instalação
no engine) só acontecem com confirmação explícita do operador.

## Lê (upstream)

- `FORMATO.md` §3.1 — contrato mínimo do pack: `id`, `name`, `version`,
  `requires-core`, `no-override`, `provides` (somente `agents/` e `skills/`)
- `golmet.yaml` — packs já registrados (`packs: [{id, repo, pin}]`)
- Registry local + `BREAKING-CHANGES.md` — versão do core para derivar a
  faixa `requires-core`
- Cache de packs (`~/.golmet/packs/`) — referência de estilo e unicidade de id

## Gera (downstream)

- Esqueleto do pack em `packs/<id>/` (NOVO):
  - `pack.yaml` (v0.1.0) — contrato FORMATO §3.1 completo, derivado das
    respostas
  - `README.md` do pack — propósito, instalação de exemplo, requisitos
  - `agents/` e/ou `skills/` — ao menos 1 componente de exemplo
  - `accumulation/` (opcional) — conteúdo da organização que NUNCA entra em
    `provides`
- Primeiro commit no repo do pack (`git init -b main`)
- Repo remoto — `gh repo create <org>/<nome> --private` + `git remote add
  origin` + push (só com confirmação explícita)
- GitHub topic `golmet-pack` no repo quando PÚBLICO — convenção de descoberta
  da comunidade (`gh repo edit --add-topic golmet-pack`); pack privado não
  recebe o topic
- Entrada em `golmet.yaml` — `packs: [{id, repo, pin}]` (sob confirmação)
- Instalação no engine — `golmet pack add <repo> --engine <id>` (pin pelo
  SHA resolvido)

## Nunca edita

- Paths já existentes do core (L1) — o no-override vale também na criação:
  `provides` que colide com componente canônico é renomeado antes de instalar
- Packs de outras organizações
- `accumulation/` com conteúdo real — o scaffold só cria esqueleto com README

## Tools

- `file`/`glob`/`grep`: **allow** — contrato, registry, `golmet.yaml`, packs
  existentes
- `ask`: **allow** — perguntas agrupadas (uma rodada única por bloco de
  decisão)
- `shell`: **ask** — `git init`/`commit`/`push`, `gh repo create`,
  `golmet pack add`
- `edit`/`write`: **apenas** `packs/<id>/**` (NOVO) e `golmet.yaml` (com
  confirmação)

## Modo de erro

- **Id de pack já existente** (entrada em `golmet.yaml` ou cache
  `~/.golmet/packs/`) → **ABORTA** com orientação: o pack existe, use
  `golmet pack update <repo> --engine <id>` para reaplicar.
- **`requires-core` incompatível** com o core local → aborta apontando
  `BREAKING-CHANGES.md` (o que mudou no major e como migrar).
- **`provides` fora de `agents/`/`skills/`** → aborta (o kit rejeita na
  instalação — corrija o path no `pack.yaml` antes de publicar).
- **Sem `gh` autenticado** → cria o repo local completo e registra pendência
  do remote (`gh repo create` + push) no relatório.
- **Operador recusa qualquer confirmação** → entrega o esqueleto local mesmo
  assim e lista as pendências (remote, registro, instalação).

## Workflow

1. **Coleta (UMA rodada de perguntas agrupadas):**
   - nome da organização
   - id do pack (`lowercase-hyphen`)
   - org GitHub (`<org>`; default: a do `gh api user`, se houver)
   - nome do repo (default `<org>-pack-<id>`; valide com o operador)
   - visibilidade (default `private`; pública → topic `golmet-pack`)
   - componentes iniciais (1+ skill e/ou agent de exemplo)
2. **Valida:** unicidade do id (cache + `golmet.yaml`), ausência de colisão
   dos `provides` propostos com paths canônicos (L1) e compat da faixa
   `requires-core` proposta contra o registry + `BREAKING-CHANGES.md`.
3. **Cria o esqueleto** em `packs/<id>/`:
   - `pack.yaml` v0.1.0 — `id`, `name`, `version: 0.1.0`, `requires-core`
     derivado da versão local do core, `no-override: true`, `provides` com os
     componentes de exemplo
   - `README.md` do pack — propósito, `golmet pack add` de exemplo, requisitos
   - `agents/`/`skills/` — componentes de exemplo no formato do engine alvo
   - `accumulation/{patterns,metrics}/README.md` (opcional — só se a
     organização quer acumular conhecimento no repo do pack)
4. **Primeiro commit:** `git init -b main` + commit inicial no repo do pack
   (anote o SHA para o pin).
5. **Pergunta — repo remoto:** confirmado → `gh repo create <org>/<nome>
   --private` (ou `--public`, com o operador ciente) + `git remote add
   origin` + `git push -u origin main`. **Público → adicione o topic
   `golmet-pack`** (`gh repo edit <org>/<nome> --add-topic golmet-pack`) — é
   o marcador que a comunidade usa para descobrir packs.
6. **Pergunta — registro:** confirmado → entrada em `golmet.yaml`
   (`packs: [{id, repo, pin: <sha>}]`).
7. **Pergunta — instalar agora:** confirmado → `golmet pack add <repo>
   --engine <id>`; reporte escritos/preservados/conflitos do overlay.
8. **Relatório final:** o que foi criado (arquivos, commit, remote, topic,
   registro, instalação) e o que ficou pendente (⚠️).

## Formato de commit

```
feat(pack): scaffold inicial do pack <id>
```

## Casos de borda

- **Id com caractere inválido** (espaço, maiúscula, `_`) → normalize para
  `lowercase-hyphen`; recuse se não houver forma inequívoca.
- **Organização sem org GitHub** → pergunte a org na coleta (não assuma).
- **Máquina sem `gh`** → esqueleto completo local + pendência explícita do
  remote no relatório.
- **Pack público com dado de organização** → recuse o push público; sugira
  voltar para `--private` (topic `golmet-pack` só sem dado sensível).
- **`provides` colide com path do core** → aborta (no-override vale também na
  criação): renomeie o path no `pack.yaml` antes de instalar.
