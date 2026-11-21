<p align="center">English | <a href="CONTRIBUTING.pt-br.md">Português (Brasil)</a></p>

# Contributing to golmet-cli

Direct contributions are welcome — and the bar is mechanical, not social:
the CI gates in [`.github/workflows/ci.yml`](.github/workflows/ci.yml)
enforce everything below before a PR reaches `main`.

## Setup

- **Go** (the version pinned in [`go.mod`](go.mod)) and **Node ≥ 14** (the
  npm launcher tests run it).
- Fork/clone, then `go build ./...` — the tree builds clean out of the box.

## Before any PR (all mandatory)

```bash
go test ./... -timeout 10m        # green — new behavior needs table-driven tests
gofmt -l .                        # empty output (CI fails on any file listed)
go vet ./...
go run ./cmd/golmet lint --workspace .   # the kit lints itself: 0 errors, 0 warnings
```

Tests are table-driven; the launcher and npm packaging tests need `node`
on the PATH (they skip cleanly without it).

## The commit convention (Conventional Commits)

Every PR commit (merge commits excepted) and the PR title must match:

```
<type>(<optional scope>)!: <subject>
```

Types: `feat` `fix` `docs` `style` `refactor` `perf` `test` `build` `ci`
`chore` `revert`. The CI gate fails the PR on any non-matching commit and
asks for a reword (`git rebase -i origin/main`). The types drive the
automatic semver bump on merge — `feat` → minor, `fix` → patch, `!` or a
`BREAKING CHANGE:` footer → major, everything else no release. The full
rule lives in
[docs/reference/versioning.md](docs/reference/versioning.md).

**Commit language: English — always.** This is a public, international
repo: the entire commit history is readable by anyone, and commit subjects
ship verbatim into release notes and the CHANGELOG via git-cliff. Write
every part of the commit in English — subject, body and footers. Internal
context belongs to the team's communication channels, not to the public
history. Example: `fix(lint): resolve org components in the ghost check`
(not "resolvem componentes da org").

## The PR flow

1. Open the PR against `main` using
   [the template](.github/PULL_REQUEST_TEMPLATE.md). **The PR title follows
   the same convention** (on a squash merge the title becomes the commit
   subject).
2. CI runs three gates in parallel: the test matrix (3 OS),
   `conventional-commits` and `release-readiness` (version sync across the
   6 `package.json` files, npm packaging shape, cross-compile of the 5
   release targets, launcher contract).
3. A maintainer merges. **You never tag** — versions and releases are
   automatic ([docs/reference/versioning.md](docs/reference/versioning.md)).

## Neutrality (non-negotiable)

This is a public repo. Contributions must not introduce real client or
organization names, real people, real URLs/e-mails, credentials, internal
repo prefixes/codenames, or engagement-identifying dates and facts. Use
neutral placeholders instead — the same neutrality the kit already
practices in its own docs.

## DCO

By signing off your commits (`git commit -s`), you certify the
[Developer Certificate of Origin](https://developercertificate.org) — your
contribution is yours to contribute, MIT-licensed like
[LICENSE](LICENSE).

## Promoting a pattern

If your contribution encodes a reusable practice, consider promoting it to
[`docs/patterns/`](docs/patterns/INDEX.md): write it on
[`docs/patterns/TEMPLATE.md`](docs/patterns/TEMPLATE.md), run
[`docs/patterns/validate.sh`](docs/patterns/validate.sh), and open the PR
against the INDEX.
