---
name: gm-pack-publish
description: >-
   Publica o pack da organização no repo remoto — valida o working tree do clone, roda os evals do pack, commita por path explícito, faz push origin main (nunca força), reaplica o overlay no engine (`golmet pack update <repo> --engine <id>` ou MCP `gm_pack_update` com confirm), move o pin do `golmet.yaml` para o SHA novo e reporta o delta. Par de publicação do `gm-pack-update` (que consome).
requires: [file, glob, grep, shell, ask]
degrades: [edit, write, webfetch]
---

# gm-pack-publish — Publicação do pack da organização

Fecha o ciclo do mantenedor: editou o conteúdo do pack → publica (commit +
push) → reaplica no engine → move o pin. Espelho do par: `gm-pack-update`
traz do remote para o engine; esta skill leva do working tree do clone para
o remote e o engine. Contrato do `pack.yaml` em FORMATO §3.1;
compatibilidade em `BREAKING-CHANGES.md`.

## Lê (upstream)

- Clone do pack — working tree, estado git, `pack.yaml`
- `pack.yaml` — version/requires-core/no-override/provides
- `golmet.yaml` — registro do pack (`packs: [{id, repo, pin}]`)
- Registry + `BREAKING-CHANGES.md` — versão do core e compatibilidade
- Manifest de conteúdo do engine — camada `pack:<id>` com origem e SHA-256

## Gera (downstream)

- Commit + push do repo do pack (origin main) — nunca força push
- Bump de versão no `pack.yaml` quando o conteúdo muda (SemVer)
- Reaplicação do overlay no engine — `golmet pack update <repo> --engine
  <id>` (ou MCP `gm_pack_update` com `confirm=true`)
- Pin movido no `golmet.yaml` para o SHA novo
- Relatório do delta (arquivos ±, versão e SHA novos, escritos/conflitos).
  **Nunca persiste nada fora disso.**

## Nunca edita

- Componentes canônicos do core (L1) — no-override: `provides` só instala
  paths novos; colisão com path canônico aborta a aplicação inteira
- `accumulation/` do pack — é conteúdo da organização: publique via commit
  normal do pack, mas NUNCA altere você mesmo o conteúdo acumulado
- Packs de terceiros — só o pack registrado em operação
- `golmet.yaml` fora do campo `pin` — repo/id são decisão do scaffold

## Tools

- `file`/`glob`/`grep`: **allow** — `pack.yaml`, `golmet.yaml`, manifest,
  compat
- `ask`: **allow** — confirmações agrupadas (mensagem de commit, push, bump
  de versão, movimento de pin)
- `shell`: **ask** — `git status/diff/add/commit/push` no clone do pack,
  `golmet pack update`, `golmet eval`
- `edit`/`write`: **apenas** `pack.yaml` (bump de versão) e `golmet.yaml`
  (campo `pin`)

## Modo de erro

- **Working tree limpo e nada a publicar** → informa *"nada a publicar"* e
  encerra sem tocar em nada.
- **Push rejeitado (non-FF)** → **ABORTA** e orienta rebase manual no clone;
  nunca força push.
- **`requires-core` incompatível** com o core local → **ABORTA** apontando
  `BREAKING-CHANGES.md` (o que mudou no major e como migrar).
- **Colisão de path canônico × `provides`** → **ABORTA** sem aplicar nada
  deste pack (no-override — transacional; o runtime não fica pela metade).
- **Eval de regressão do pack falhou** → NÃO publique; corrija primeiro
  (`golmet eval <fixture> --workspace <ws> --json`).

## Workflow

1. **Valide o id:** confira o `<id>` do pack no `golmet.yaml`; ausente →
   pergunte (pode ser pack novo — ofereça o `gm-pack-scaffold`).
2. **Inspecione o clone:** `git status` + `git diff` — resuma o delta
   (arquivos ±).
3. **Rode os evals do pack** (se houver fixtures): pontue com `golmet eval
   <fixture> --workspace <ws> --json` — todas com status PASS antes de
   publicar.
4. **Confirme (uma rodada única):** apresente o pacote completo — arquivos,
   mensagem de commit sugerida (Conventional Commits), bump de versão, push
   e movimento de pin — opções aprovar / ajustar / cancelar.
5. **Commite:** `git add` por path explícito (nunca `git add -A`/`git add .`)
   + commit; bump de `version` no `pack.yaml` quando conteúdo muda.
6. **Pusha:** `git push origin main` sem travar em prompt de credencial
   (falha rápida); credencial ausente → commit local mantido + pendência de
   push no relatório.
7. **Reaplique:** `golmet pack update <repo> --engine <id>` — valida
   `requires-core` × core local e no-override, ressincroniza o cache
   `~/.golmet/packs/` e aplica o overlay; arquivos locais divergentes são
   preservados e reportados. Alternativa agêntica: MCP `gm_pack_update` com
   `source`, `engine` e `confirm=true` (`dry_run=true` para ensaiar sem
   escrever).
8. **Mova o pin:** atualize o `pin` no `golmet.yaml` para o SHA novo.
9. **Reporte:** delta de arquivos (±), versão e SHA novos, resultado da
   aplicação (escritos/preservados/conflitos).

## Formato de commit

```
release(pack): v<nova-versão> do pack <id>
```

## Casos de borda

- **Primeiro publish pós-scaffold:** repo do pack sem remote `origin` →
  publica apenas local (commit ok, push pendente) e registra a pendência de
  `git remote add origin` no relatório.
- **Pack sem mudanças:** working tree limpo e HEAD == origin/main → *"nada a
  publicar"*; nada é commitado nem reaplicado.
- **`provides` colidindo com path do core:** aborta a aplicação
  (no-override) — commit/push já feitos permanecem; oriente renomear o path
  no `pack.yaml`.
- **Múltiplos packs:** rode por pack, um de cada vez — nunca em lote; falha
  de um não publica os demais por cima.
