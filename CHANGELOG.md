# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

> **Note:** the repository has no tags or releases yet (npm is a `0.0.2` name
> placeholder; the kit's internal version is `0.3.0`, per `KitVersion` in
> `internal/core/indexstate/writer.go`). Comparison links will be added in the
> footer once the first tag is cut.

## [0.3.0] - Unreleased

### Added

#### CLI (`golmet`)

14 commands shipped on the `golmet` binary:

- `golmet validate` — end-to-end workspace validation against FORMATO v1 (H1–H10 / S1–S5) with parseable evidence, `--json` structured output, and `--tolerant`/`--strict` modes; non-zero exit blocks CI.
- `golmet init` — scaffolds a product workspace or organization container (`.golmet/` kit zone, `golmet.yaml`, engine adapters for `.opencode/`, `.mcp.json`, `.cursor/`).
- `golmet install` — onboarding wizard covering 7 agent engines and a catalog of 12 MCP servers, with already-registered badges and cascaded post-install verification.
- `golmet update` — refreshes the vendored BASE skill subset via `go:embed` with a SHA-256 manifest; `--full` materializes the FULL workflow into the engine config (embedded or external core).
- `golmet status` — workspace dashboard summarizing artifacts, states, and staleness.
- `golmet next` — deterministic recommendation of the next action, prioritized `p1..p9`.
- `golmet doctor` — environment health checks (9 checks) with remediation hints; runs a Jev smoke test by default (`--no-network` skips it).
- `golmet adopt` — FORMATO adoption handshake for existing workspaces: classified dry-run first, then deterministic apply, preserving legacy layouts.
- `golmet eval` — deterministic eval checker (15 checks) over a fixture suite, with JSON output for CI.
- `golmet tools` — installs supporting tooling (e.g. `tools install codegraph`) with supply-chain verification (SHA-256 checksums + attestation).
- `golmet mcp` — stdio MCP server exposing 10 tools (see below).
- `golmet index` — rebuilds `golmet.index.json` (the kit-written state index) after merges or corruption.
- `golmet pack` — manages L2 content packs: `pack add`, `pack update` (sync + overlay, engine required), and `pack list`.
- `golmet db` — database layer commands; `db drift` deterministically compares `DATA_MODEL.md` entities against the EF Core `ModelSnapshot` (migrations drift for the `gm-dba` layer).

#### MCP server

- `golmet mcp` serves 10 tools over stdio: `gm_validate`, `gm_doctor`, `gm_next`, `gm_status`, `gm_index`, `gm_eval`, `gm_jev_classify`, `gm_update`, `gm_pack_update`, and `gm_pack_list` — so agent engines drive the kit without shelling out.
- Jev client integration (TypeSafe System One classification) via the `/v1/systemone` route with typed `state`+`model` payloads and question specs.

#### Content

- BASE skill subset: 16 `gm-*` skills embedded in the binary with capability front matter (complete port of the collection, fully organization-neutral), vendored into the workspace `.golmet/` zone.
- Content layers L0–L3 (DR-021): L0 embedded BASE, L1 FULL (93 registry-rendered components, embedded or external), L2 organization/community packs, L3 local customizations — resolved with precedence `local > pack > full > base`.
- Pack authoring layer (L2) embedded in the registry — 4 new components (68 skills + 4 primary + 21 subagents total): `gm-pack-scaffold` (onboarding of a new organization's pack from the FORMATO §3.1 contract: `pack.yaml`, first commit, `gh repo create <org>/<nome> --private`, `golmet-pack` GitHub topic convention for public packs, registration and `golmet pack add`), `gm-pack-publish` (commit + push + version bump + overlay reapply + pin move, eval-gated), `gm-pack-update` (reinstall from pin, `requires-core` + no-override validation, `golmet pack list`/`gm_pack_list` inventory) and the `gm-pack-orchestrator` subagent (routes the full L2 cycle: create → publish → consume → discover). 3 new eval fixtures (self-score PASS) in `evals/gm-pack-{scaffold,publish,update}/fixture-001/`.
- Three-way overlay with provenance manifest: each installed file records `layer`, origin, and SHA-256; locally divergent files are preserved and reported, never silently overwritten (`--force` is the explicit exception and still never touches pack-managed files).
- External core (`--core`) accepted by the CLI and MCP tools — Git or local path — while the embedded FULL remains the offline default, atomically compatible with the kit's renderer.
- Pack contract (L2): Git repo with `pack.yaml` and `provides` restricted to `agents/`/`skills/`, pinned by resolved SHA in the content manifest, with `no-override` collision protection.
- FORMATO v1 specification (normative workspace contract): entities, ID grammar, evidence grammar, H1–H10/S1–S5 rules, YAML subset, default states, kit vocabulary, graph-first policy (S10/DR-020), content layers (§2.1) and pack contract (§3.1); legacy layout documented for `adopt`.
- Eval corpus: a gm-based fixture suite (22 fixtures) with audited organization neutrality (0 org hits), consumed by `golmet eval`.

### Changed

- Install wizard profiles renamed with **contract semantics**: `essencial` (the repository kit — 16 `gm-*` skills in the workspace `.golmet/` zone, versioned with the repo, any agentic engine) and `completo` (the full experience on your machine — 93 components in the engine's format, installed in your global config). **`completo` requires an engine with native skills** (criterion: `SkillsDir != ""` — today only opencode and claude-code): the wizards filter the engine list when completo is chosen, and `install`, `update --full` and `gm_update` fail closed with the compatible list instead of silently degrading; the workspace-zone pipeline remains internal (lib-level). The wizard asks the profile **before** the engine (plain and VT); `base`/`full` remain accepted as internal flag aliases and the report always cites the canonical name. Layer names L0–L3 (DR-021) are unchanged.
- The legacy human-knowledge layer was renamed from `reverse/guia/` to English `reverse/guide/` (OVERVIEW.md, IMPACT_MAP.md, GUIDES/, FORMS/) — naming settled after the initial FORMATO draft.
- FULL-layer installation evolved from a plain embedded materialization into the three-way overlay (DR-021) with conflict preservation and provenance tracking, superseding the first L1 drop.
- Jev client transport reworked to the `/v1/systemone` route with corrected `state`+`model` payloads, replacing the initial client wiring.
- Install wizard improved with already-registered badges for vscode/cursor/gemini and cascaded post-verify, refining the initial onboarding flow.

### Fixed

- `install`/`update` no longer treat EOL differences (CRLF/CR on disk vs the kit's canonical LF — leftover from PowerShell edits/migrations) as local customization: all installed-vs-target comparisons normalize EOL, and an EOL-only difference is **silently cured** by rematerializing the file in LF, counted as written — no scary "modificadas localmente"/conflict warnings, and content manifests converge. Real content differences keep the exact DR-008 preservation behavior. Re-runs with byte-identical content are now true no-ops (0 written).

### Security

- Path safety on both source and destination: rendered paths validated against traversal (allowlist `agents/`/`skills/`), symlink/junction escape rejected with real containment via `EvalSymlinks`.
- Credential redaction: Git URLs are redacted in manifests, payloads, and error messages.
- Fail-closed Git cache: a dirty sync cache aborts before any `pull --ff-only` runs.
- MCP mutation safety: state-changing tools (`gm_update`, `gm_pack_update`) require `confirm=true`, and `dry_run` writes nothing to runtime, manifest, or cache.
- Pack compatibility gate: `requires-core` is validated against the kit version before a pack overlay applies.

<!-- No tags or releases exist yet. Comparison links ([Unreleased]: ...compare/vX.Y.Z...HEAD)
     will be introduced in this footer when the first tag is cut. -->
