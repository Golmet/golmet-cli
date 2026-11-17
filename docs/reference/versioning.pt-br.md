<p align="center"><a href="../../README.md">English</a> | <a href="../../README.pt-br.md">Português (Brasil)</a> | <a href="versioning.md">versão base deste doc</a></p>

# Versionamento (Conventional Commits → semver)

A versão do kit é decidida **pelo merge, não por uma pessoa**: todo merge na
`main` roda o workflow de versão, que lê os subjects dos commits desde a
última tag `v*`, classifica por Conventional Commits e calcula o próximo
semver. **A tag é a fonte única da verdade** — a versão publicada no npm é
sempre igual à tag; os `package.json` versionados no repo são patchados no
momento do publish (a cópia do repo pode ficar um degrau atrás; o registry,
nunca).

## O formato do commit

```
<tipo>(<escopo opcional>)!: <subject>

corpo opcional
BREAKING CHANGE: <footer opcional>
```

Tipos válidos: `feat` `fix` `docs` `style` `refactor` `perf` `test` `build`
`ci` `chore` `revert`. Todo commit de PR (menos merge commits) e o título do
PR são validados no CI — commit fora do padrão não entra na `main`, porque
seria invisível para o cálculo de versão.

## A regra do bump

| Commits desde a última tag `v*` | Bump | Exemplo |
|---|---|---|
| `feat:` | **minor** | `feat(lint): checagem L14 de manifest binário` |
| `fix:` | **patch** | `fix(launcher): exit code do fallback de resolução` |
| qualquer tipo com `!`, ou footer `BREAKING CHANGE:` | **major** | `feat!!: renomeia flags do init` |
| só `docs:` `chore:` `refactor:` `test:` `ci:` `style:` `build:` | **nenhum** | sem tag, sem release |

A precedência é a do semver: major > minor > patch. Um `feat!` entre dez
`fix` sobe o major.

## A pipeline

1. **PR** (`.github/workflows/ci.yml`) — três gates em paralelo: a matriz
   de testes (gofmt, vet, test, build, lint do kit ×3 SO),
   `conventional-commits` (subjects + título do PR) e `release-readiness`
   (sincronia de versões dos 6 `package.json`, shape por
   `npm pack --dry-run` ×6, cross-compile dos 5 alvos da release e o
   contrato do launcher — resolução, despacho e fallback).
2. **Merge na `main`** (`.github/workflows/version.yml`) — os commits desde
   a última tag são classificados; merge sem commit releasável termina aqui
   em silêncio. Com bump: a tag anotada `vX.Y.Z` nasce no HEAD, é empurrada
   e o workflow release é despachado explicitamente (push de tag feito com
   o `GITHUB_TOKEN` não dispara workflows — o `workflow_dispatch` é a
   exceção documentada).
3. **Tag** (`.github/workflows/release.yml`) — cross-compile das 5
   plataformas com a versão cravada pela tag, checksums SHA-256, GitHub
   Release idempotente (create cai para upload `--clobber` no re-run),
   notas como digest dos Conventional Commits e, no npm: pré-flight do
   token (`npm whoami`, fail fast), os 5 pacotes de plataforma primeiro, o
   principal por último, já-publicados são skip. **Qualquer publish falho
   fecha o job vermelho** — a GitHub Release e os assets permanecem no ar,
   e o re-run completa só o que falta.

## Release manual

Tag empurrada manualmente (`git tag v1.2.3 && git push origin v1.2.3`)
lança pelo mesmo caminho. O workflow release também aceita dispatch manual
(Actions → release → tag), útil para retomar um publish interrompido — os
passos de npm são idempotentes.
