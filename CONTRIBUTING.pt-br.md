<p align="center"><a href="README.md">English</a> | <a href="README.pt-br.md">Português (Brasil)</a> | <a href="CONTRIBUTING.md">versão base deste doc</a></p>

# Contribuindo para o golmet-cli

Contribuições diretas são bem-vindas — e a barra é mecânica, não social: os
gates de CI em [`.github/workflows/ci.yml`](.github/workflows/ci.yml)
reforçam tudo abaixo antes de um PR chegar à `main`.

## Setup

- **Go** (a versão cravada no [`go.mod`](go.mod)) e **Node ≥ 14** (os testes
  do launcher npm o executam).
- Fork/clone e `go build ./...` — a árvore compila limpa de primeira.

## Antes de qualquer PR (tudo obrigatório)

```bash
go test ./... -timeout 10m        # verde — comportamento novo pede testes table-driven
gofmt -l .                        # saída vazia (o CI falha com qualquer arquivo listado)
go vet ./...
go run ./cmd/golmet lint --workspace .   # o kit faz o lint de si mesmo: 0 erros, 0 avisos
```

Testes são table-driven; os testes do launcher e do empacotamento npm
precisam de `node` no PATH (sem ele, eles se skipam limpo). A mesma suíte
fecha self-score do corpus de evals (`evals/`) — regras de autoria de
fixture em
[docs/reference/evals.pt-br.md](docs/reference/evals.pt-br.md).

## A convenção de commit (Conventional Commits)

Todo commit do PR (merge commits excetuados) e o título do PR devem
casar:

```
<tipo>(<escopo opcional>)!: <subject>
```

Tipos: `feat` `fix` `docs` `style` `refactor` `perf` `test` `build` `ci`
`chore` `revert`. O gate de CI falha o PR em qualquer commit fora do padrão
e pede o reword (`git rebase -i origin/main`). Os tipos dirigem o bump
automático do semver no merge — `feat` → minor, `fix` → patch, `!` ou
footer `BREAKING CHANGE:` → major, os demais sem release. A regra completa
vive em
[docs/reference/versioning.md](docs/reference/versioning.pt-br.md).

## O fluxo de PR

1. Abra o PR contra a `main` usando
   [o template](.github/PULL_REQUEST_TEMPLATE.md). **O título do PR segue a
   mesma convenção** (num squash merge o título vira o subject do commit).
2. O CI roda três gates em paralelo: a matriz de testes (3 SO),
   `conventional-commits` e `release-readiness` (sincronia de versões dos 6
   `package.json`, shape do empacotamento npm, cross-compile dos 5 alvos da
   release, contrato do launcher).
3. Um mantenedor mergia. **Você nunca crava tag** — versões e releases são
   automáticas ([docs/reference/versioning.md](docs/reference/versioning.pt-br.md)).

## Neutralidade (inegociável)

Este é um repo público. Contribuições não podem introduzir nomes reais de
clientes ou organizações, pessoas reais, URLs/e-mails reais, credenciais,
prefixos/codinomes internos de repo, datas e fatos que identifiquem
engajamentos. Use placeholders neutros — a mesma neutralidade que o kit já
pratica nos próprios docs.

## DCO

Ao assinar seus commits (`git commit -s`), você certifica o
[Developer Certificate of Origin](https://developercertificate.org) — sua
contribuição é sua para contribuir, licenciada MIT como
[LICENSE](LICENSE).

## Promovendo um pattern

Se a sua contribuição codifica uma prática reutilizável, considere
promovê-la para [`docs/patterns/`](docs/patterns/INDEX.md): escreva no
[`docs/patterns/TEMPLATE.md`](docs/patterns/TEMPLATE.md), rode o
[`docs/patterns/validate.sh`](docs/patterns/validate.sh) e abra o PR contra
o INDEX.
