<p align="center">
  <a href="https://github.com/Golmet">
    <img src="banner-golmet.png" alt="GOLMET — the stone guardian of your product's documented truth" width="640">
  </a>
</p>

<h1 align="center">golmet-cli</h1>

<p align="center">
  <strong>The stone guardian of your product's documented truth.</strong><br>
  The installable CLI for <a href="https://github.com/Golmet">GOLMET</a>,
  the spec-driven workflow platform where requirements, specs and evidence stay
  <strong>congruent with the code</strong>.
</p>

<div align="center">

[![CI](https://github.com/Golmet/golmet-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/Golmet/golmet-cli/actions/workflows/ci.yml)
[![npm version](https://img.shields.io/npm/v/golmet-cli.svg)](https://www.npmjs.com/package/golmet-cli)
[![license](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

</div>

## Why

- **Declare.** Charter, requirements and specs: the a-priori truth of your product.
- **Trace.** Every rule carries its evidence: `[path:line @hash]`.
- **Validate.** Machine-checked congruence between what was written and what exists — drift blocked before it merges.

## Quickstart

```bash
# install (Go 1.22+)
go install github.com/Golmet/golmet-cli/cmd/golmet@latest

# onboard a workspace: scaffold → install BASE skills → validate → health
golmet init
golmet update
golmet validate
golmet doctor
```

`golmet install` is the one-command alternative: it asks for the **profile**
first — `essencial` (repo-versioned kit, 16 skills in the workspace
`.golmet/`, works on **any** agentic engine) or `completo` (full experience
on your machine, 93 components in your engine's format - **requires an
engine with native skills: opencode or claude-code**; the wizard filters
the list for you) — then detects your engine, installs the `gm-*` skills,
registers the MCP server and runs the doctor in a single pass.

> **Note:** the npm package `golmet-cli` is currently a **placeholder**. The real
> CLI ships there at the first tagged release — until then, use `go install`.

## Commands

17 commands, one binary (`golmet`):

| Command    | What it does                                                                                                                                                                   |
| ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `validate` | Validates the workspace against FORMATO v1 — hard rules H1–H10 (exit ≠ 0 blocks CI), warnings S1–S5; `--json` emits structured errors, `--tolerant` degrades H4/H7 to warnings |
| `init`     | Scaffolds a workspace or container (`golmet.yaml`, charter, `.golmet/` zone, engine adapters)                                                                                  |
| `install`  | One-command onboarding: engine + `gm-*` skills + MCP catalog + doctor, with post-verify cascade                                                                                |
| `update`   | Installs the BASE skill subset into `.golmet/` via `go:embed` (SHA-256 manifest); `--full` materializes the FULL workflow into your engine                                     |
| `status`   | Read-only dashboard of entity states (RN, PEND, CR, ADR, Feature, Task)                                                                                                        |
| `next`     | Recommends the next action on the workspace (priorities p1..p9)                                                                                                                |
| `doctor`   | Soft checks (S1–S5) plus environment health, with remediation hints                                                                                                            |
| `adopt`    | Adopts FORMATO v1 in an existing workspace (classified dry-run → deterministic apply)                                                                                          |
| `eval`     | Deterministic eval checker: scores `criteria.yaml` fixtures against the workspace; writes a markdown report per fixture to `~/.golmet/eval-reports` (50-report retention)      |
| `tools`    | Installs kit-native tools (code graph) with supply-chain verification (SHA-256 + attestation)                                                                                  |
| `mcp`      | Serves the kit as an MCP server over stdio — validation, update and packs                                                                                                      |
| `index`    | Rebuilds `golmet.index.json` from the prose (the prose is canonical; the index is a cache)                                                                                     |
| `pack`     | Manages L2 packs (`add` / `update` / `publish` / `list`) without destroying local customizations — two archetypes: `overlay` (read-only cache) and `state` (bidirectional Git working copy)          |
| `db`       | Database services (the `gm-dba` layer): DATA_MODEL ↔ migrations/snapshot drift                                                                                                 |
| `template` | Emits a kit template (AGENTS.md skeleton, client/operator profile) straight from the embed — templates are construction artifacts: they live in the kit, never installed into the config |
| `upgrade`  | Self-update of the binary from the latest GitHub release (SHA-256 verified; `--check` read-only; rollback via `<bin>.old`)                                                      |
| `lint`     | Deterministic structural sweep (dead refs, orphans, stale manifest entries, naming, ghost `gm-*` components, canonical structure, definitivo↔template conformance, stack-templates schema) — findings with severity + remediation; exit 0 clean, 2 with findings |

The normative contract behind all of this is [FORMATO.md](FORMATO.md) v1.

## Updating

Self-update the binary from the latest GitHub release (tag `v*`; assets are
cross-compiled per platform and SHA-256-verified against the release's
`checksums.txt` — a missing or mismatched checksum aborts without touching
the binary):

```bash
golmet upgrade --check   # read-only: reports local vs latest (exit 0 either way)
golmet upgrade           # downloads, verifies and replaces the binary
```

Rollback: the previous binary is kept at `<bin>.old` — restore it by
renaming back (each upgrade replaces the `.old`, so copy it somewhere safe
if you need to keep it). After upgrading the binary, re-materialize the
managed content so the engine follows the new kit:
`golmet install --engine <id> --profile completo --yes` (or MCP `gm_update`
with `confirm=true`). HTTP(S)_PROXY is respected; `GITHUB_TOKEN`/`GH_TOKEN`
is used when present (API rate limit) — never required. Agentic engines can
drive the same flow via the MCP `gm_upgrade` tool (`check=true` /
`confirm=true`).

## MCP server

`golmet mcp` exposes the kit to any agentic engine over stdio, with 13 tools:

| Tool              | Purpose                                                               |
| ----------------- | --------------------------------------------------------------------- |
| `gm_validate`     | Validates the workspace against FORMATO v1                            |
| `gm_doctor`       | Soft checks (S1-S5) and environment health                            |
| `gm_next`         | Next-action recommendation (p1..p9)                                   |
| `gm_status`       | Read-only entity-state dashboard                                      |
| `gm_index`        | Rebuilds the state index from prose                                   |
| `gm_eval`         | Scores eval fixtures against `criteria.yaml`; returns `report_path`   |
| `gm_eval_reports` | Lists eval reports stored in `~/.golmet/eval-reports` (newest first)  |
| `gm_jev_classify` | Calibrated classification (TypeSafe System One)                       |
| `gm_update`       | Content update (embedded FULL or external `--core`); supports dry-run |
| `gm_pack_update`  | Re-syncs and applies a pack (overlay or state - state preserves local accumulation with auto-commit + rebase) |
| `gm_pack_list`    | Lists installed packs (with `kind`: overlay or state)                |
| `gm_upgrade`      | Self-update of the binary (`check=true` read-only; applying requires `confirm=true`) |
| `gm_lint`         | Deterministic structural sweep (dead refs, orphans, stale manifest, naming, ghost components, canonical structure, stack-templates schema) - read-only |

Example registration in `opencode.json`:

```json
{
  "mcp": {
    "golmet": {
      "type": "local",
      "command": ["golmet", "mcp"],
      "enabled": true
    }
  }
}
```

## Content layers

Installed content follows four layers, weakest to strongest
([DR-021](FORMATO.md)):

| Layer        | What it is                                                                                                        |
| ------------ | ----------------------------------------------------------------------------------------------------------------- |
| **L0 BASE**  | Minimal skill subset embedded in the binary — offline by default                                                  |
| **L1 FULL**  | The 93 canonical components rendered from the registry, embedded (or an external core via `--core`, Git or local) |
| **L2 packs** | Organization/community overlays (`pack.yaml`, `provides`, `no-override`, `requires-core`) — archetype `overlay` (read-only cache) or `state` (bidirectional working copy at `<config>/packs/<id>`: skills/agents materialize by overlay while `knowledge/`, `accumulation/` and `rules/` live in the clone; `pack publish` pushes the operator's accumulation) |
| **L3 local** | Operator creations and customizations                                                                             |

**Precedence: `local > pack > full > base`.** The content manifest records
`layer`, origin and a SHA-256 per file. Updates apply a 3-way overlay: a local
file that diverges from its last recorded hash is preserved and reported as a
conflict — never silently overwritten. `--force` is the explicit exception, and
even it never touches files managed by packs.

## Security posture

- Rendered paths validated against traversal (allowlist: `agents/`, `skills/`, `profile/`).
- Symlink/junction escape rejected at both source and destination (real containment via `EvalSymlinks`).
- A dirty Git cache fails closed before any `pull --ff-only`.
- Credentials in URLs are redacted from manifests, payloads and errors.
- `requires-core` versions validated against the kit before an overlay applies.
- MCP mutations require explicit `confirm=true`; `dry_run` writes nothing — runtime, manifest or cache.

## Development

The CLI is written in **Go 1.22+** (module `github.com/Golmet/golmet-cli`),
with [spf13/cobra](https://github.com/spf13/cobra) as the only external
dependency. **321 Go tests** (unit + integration + binary e2e) back the current
state.

```bash
# build
go build ./...

# local binary
go build -o golmet ./cmd/golmet

# tests (table-driven; fixtures in internal/testdata/)
go test ./...

# static analysis and formatting (required before committing)
go vet ./...
gofmt -l .
```

```bash
# validate a workspace against FORMATO v1
golmet validate <workspace>            # exit 0 ok · 1 hard violations · 2 usage/config error
golmet validate <workspace> --json     # array of structured errors (§11)
golmet validate <workspace> --tolerant # H4/H7 degrade to warnings with a worklist
golmet validate <workspace> --strict   # default: blocks H4/H7
```

## Roadmap

See [ROADMAP.md](ROADMAP.md). Milestone **F1** (this release) ships the full
CLI, the MCP server, the FORMATO v1 contract and the L0–L3 content layers;
F2 targets the documentation site and a dogfooding case study; F3 adds the
contribute flow and workspace migration tooling.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Table-driven tests and `gofmt` are
mandatory for every change.

## The GOLMET family

| Name       | Layer                                 |
| ---------- | ------------------------------------- |
| **GOLMET** | the platform                          |
| **Arki**   | the guardian (official mascot, above) |
| **gm-\***  | the namespace of skills and agents    |

## License

[MIT](LICENSE)
