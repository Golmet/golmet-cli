# pack-meuapp

Pack da organização (camada L2) com componentes do produto MeuApp para
compartilhar entre workspaces de qualquer engine com skills nativas.

## Contrato

`pack.yaml` conforme FORMAT §3.1 — `provides` somente em `agents/` e
`skills/`; `no-override: true` nunca sobrescreve componente canônico (L1)
nem arquivo gerido por outro pack; `requires-core: ">=0.3"`.

## Instalação

```bash
golmet pack add minha-org/pack-meuapp --engine opencode
```

Reinstalação e inventário: `golmet pack update` e `golmet pack list`
(MCP: `gm_pack_update` / `gm_pack_list`).

## Descoberta

Sendo privado, o repo não recebe topic. Se tornar-se público, aplique o
topic GitHub `golmet-pack`
(`gh repo edit minha-org/pack-meuapp --add-topic golmet-pack`) — é o
marcador de descoberta que a comunidade usa para encontrar packs. Não aplique
o topic em repo com dado sensível de organização.

## Accumulation

Conteúdo da organização em `accumulation/` NUNCA entra em `provides` — é
publicado via commit normal do pack, nunca instalado no engine.
