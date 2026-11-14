# gm-pack-update — Atualização de packs instalados

Mantém os packs instalados no engine em sincronia com os remotes. O **pin**
manda: reaplicar reinstala a partir do estado sincronizado no cache
`~/.golmet/packs/`; mover o pin (atualizar de fato a versão) é sempre
decisão explícita do operador. Contrato do `pack.yaml` em FORMATO §3.1;
compatibilidade em `BREAKING-CHANGES.md`.

## Lê (upstream)

- `golmet.yaml` — registro (`packs: [{id, repo, pin}]`)
- Cache local `~/.golmet/packs/` — packs já sincronizados (`pack.yaml`,
  paths de `provides`)
- Manifest de conteúdo do engine — camada `pack:<id>` com origem e SHA
- Registry + `BREAKING-CHANGES.md` — versão do core e compatibilidade
  (`requires-core`)

## Gera (downstream)

- Overlay reaplicado no engine — cópia dos paths de `provides` (nunca
  `accumulation/` — que sequer entra em `provides`)
- Pin atualizado no `golmet.yaml` somente sob pedido explícito
- Relatório do delta (versão antiga → nova, escritos/preservados/conflitos).
  **Nunca persiste nada fora disso.**

## Nunca edita

- Componentes canônicos do core (L1) — só instala paths de `provides` sem
  colisão; `no-override: true` impede sobrescrever até arquivo gerido por
  outro pack
- `golmet.yaml` fora do campo `pin`
- Repos remotos — nunca commita nem pusha (o publish é de outra skill)

## Tools

- `file`/`glob`/`grep`: **allow** — `golmet.yaml`, manifest, `pack.yaml` do
  cache
- `shell`: **ask** — `golmet pack list`/`pack update`, git read-only no cache
- `skill`: **allow** — `gm-pack-publish` quando o operador quiser AVANÇAR o
  pin
- `edit`/`write`: **apenas** `golmet.yaml` (campo `pin`)

## Modo de erro

- **Pack ausente/corrompido no cache** → ressincroniza a partir do repo
  registrado (`golmet pack add <repo> --engine <id>` — idempotente) e
  reaplica.
- **`requires-core` incompatível** com a versão local do core → **FALHA
  clara** apontando `BREAKING-CHANGES.md`; pula o pack e registra pendência.
- **Colisão de path** (um `provides` colide com componente canônico) →
  **aborta sem instalar nada** deste pack (transacional — o runtime não fica
  pela metade).

## Workflow

1. **Inventário:** `golmet pack list` (ou MCP `gm_pack_list`) + leia o
   `golmet.yaml` — monte a lista de packs alvo: todos os registrados, ou o
   `<id>` pedido pelo operador.
2. **Sincronize e reaplique:** para cada pack, `golmet pack update <repo>
   --engine <id>` ressincroniza o cache e aplica o overlay (valida
   `requires-core` e no-override ANTES de escrever — transacional por pack).
   Alternativa agêntica: MCP `gm_pack_update` com `source`, `engine` e
   `confirm=true` (`dry_run=true` para ensaiar sem escrever).
3. **Resolva a versão:** o **pin** do `golmet.yaml` é a referência de
   reprodutibilidade. Mover o pin (bump de versão no remote) **SÓ sob pedido
   explícito do operador** ("atualize para a última" move; "atualize os
   packs" reaplica o pin atual). Reaplicar por outro caminho é erro.
4. **Reporte:** por pack — versão sincronizada, arquivos
   escritos/preservados/conflitos, pin (movido ou mantido), pendências (⚠️).

## Casos de borda

- **Pin vs versão nova:** a entrada em `golmet.yaml` autoriza o destino; o
  pin garante reprodutibilidade. **Não move o pin sem pedido explícito.**
- **Múltiplos packs:** processados de forma independente — falha de um não
  bloqueia os demais; pendências vão para o relatório final.
- **Arquivo local divergente:** preservado e reportado como conflito pelo
  overlay 3-way — nunca sobrescrito silenciosamente (`--force` é exceção
  explícita e ainda não toca arquivos geridos por packs).
- **Cache apagado (limpeza de disco):** `golmet pack add <repo> --engine
  <id>` reidrata a partir do repo e reaplica no pin do `golmet.yaml` (o
  engine volta ao estado exato do pin).
