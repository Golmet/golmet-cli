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

14 commands, one binary (`golmet`):

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
| `eval`     | Deterministic eval checker: scores `criteria.yaml` fixtures against the workspace                                                                                              |
| `tools`    | Installs kit-native tools (code graph) with supply-chain verification (SHA-256 + attestation)                                                                                  |
| `mcp`      | Serves the kit as an MCP server over stdio — validation, update and packs                                                                                                      |
| `index`    | Rebuilds `golmet.index.json` from the prose (the prose is canonical; the index is a cache)                                                                                     |
| `pack`     | Manages L2 packs (`add` / `update` / `list`) without destroying local customizations                                                                                           |
| `db`       | Database services (the `gm-dba` layer): DATA_MODEL ↔ migrations/snapshot drift                                                                                                 |

The normative contract behind all of this is [FORMATO.md](FORMATO.md) v1.

## MCP server

`golmet mcp` exposes the kit to any agentic engine over stdio, with 10 tools:

| Tool              | Purpose                                                               |
| ----------------- | --------------------------------------------------------------------- |
| `gm_validate`     | Validates the workspace against FORMATO v1                            |
| `gm_doctor`       | Soft checks (S1–S5) and environment health                            |
| `gm_next`         | Next-action recommendation (p1..p9)                                   |
| `gm_status`       | Read-only entity-state dashboard                                      |
| `gm_index`        | Rebuilds the state index from prose                                   |
| `gm_eval`         | Scores eval fixtures against `criteria.yaml`                          |
| `gm_jev_classify` | Calibrated classification (TypeSafe System One)                       |
| `gm_update`       | Content update (embedded FULL or external `--core`); supports dry-run |
| `gm_pack_update`  | Re-syncs and applies a pack overlay                                   |
| `gm_pack_list`    | Lists installed packs                                                 |

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
| **L2 packs** | Organization/community overlays (`pack.yaml`, `provides`, `no-override`, `requires-core`)                         |
| **L3 local** | Operator creations and customizations                                                                             |

**Precedence: `local > pack > full > base`.** The content manifest records
`layer`, origin and a SHA-256 per file. Updates apply a 3-way overlay: a local
file that diverges from its last recorded hash is preserved and reported as a
conflict — never silently overwritten. `--force` is the explicit exception, and
even it never touches files managed by packs.

## Security posture

- Rendered paths validated against traversal (allowlist: `agents/`, `skills/`).
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
