<p align="center">English | <a href="overview.pt-br.md">Português (Brasil)</a></p>

# What GOLMET delivers

GOLMET is the stone guardian of your product's documented truth: a CLI plus
an agent toolkit that turns requirements into governed, validated, versioned
knowledge — for a solo developer or for a whole company. This page lists
everything it delivers, layer by layer, in one place. Each item links the
deep-dive page.

## 1. A complete agent toolset — installed in one command

`golmet install` materializes the embedded toolset into your engine
(OpenCode, Claude Code and more): **4 primary agents** (Requirements,
Documentation, Dev, DevOps), **25 subagents**, **95 skills** — 124
components total. Interactive wizard for a machine, explicit flags for
scripts, and a final health report. See
[init.md](init.md).

## 2. Content layers — where every file comes from

Every file on your machine has a declared owner and precedence:
**L0 embedded** (inside the binary) → **L1 FULL** (materialized, kit is the
sole writer) → **L2 packs** (the organization's layer) → **L3 local**
(your creations). Precedence: local > pack > full > base. See
[layers.md](layers.md).

## 3. `profile/` — your personal layer

Your client and operator identity plus the kit's canonical behavior rules
(`context7`, stack templates). The templates live in the kit and are
**never installed**; the config carries only the definitives
(personal-by-design — never versioned, never overwritten). See
[profile.md](profile.md).

## 4. Packs — the organization's knowledge, distributed

A **state pack** is a Git working copy cloned into your engine config —
bidirectional: the team inherits it via `golmet pack add`/`update`, and you
publish accumulated maturity via `golmet pack publish`. Inside, three
shelves with canonical writers: **`rules/`** (norms with reading triggers),
**`domains/`** (domain foundations from primary sources) and
**`accumulation/`** (patterns, metrics, nuances). Tooling inside the pack
(skills/agents) is materialized into the engine; data is read in-place.
See [knowledge.md](knowledge.md).

## 5. Organization — enterprise governance

A **container** workspace holds multiple product workspaces, with its own
golmet.yaml, a global catalog of organizations (`~/.golmet/organizations.json`)
and two versioning models (org-repo | products-repo). The company governs
norms once; every product inherits. See
[organization.md](organization.md).

## 6. Deterministic self-governance

Three validators with declared contracts: **`golmet validate`** (H1-H10 —
the workspace against the FORMAT), **`golmet doctor`** (S1-S5 + engine-config
state: managed / personal / pack) and **`golmet lint`** (L1-L12 — dead
references, orphans, stale manifest, canonical names, ghost components,
canonical structure, definitive↔template conformance, registry schema, pack
state, Mermaid diagrams, encoding). Runs in CI, CLI and MCP. The lint loads
the whole state; the agent fixes what it points. See
[self-governing.md](self-governing.md).

## 7. MCP server — 14 tools in the agent session

The kit serves itself via MCP: `gm_validate`, `gm_doctor`, `gm_next`,
`gm_status`, `gm_index`, `gm_eval`, `gm_eval_reports`, `gm_jev_classify`,
`gm_lint`, `gm_upgrade`, `gm_update`, `gm_pack_update`, `gm_pack_list`,
`gm_pack_publish`. The agent session validates, routes and fixes without
leaving the chat. Reference: [../reference/tools.pt-br.md](../reference/tools.md)
(or `tools.pt-br.md`).

## 8. Self-update with verified releases

`golmet upgrade` fetches the latest GitHub release, verifies **SHA-256**
before touching the binary, swaps Windows-safe (`.old` rollback) and
re-materializes managed content. Cross-platform binaries + npm launcher.
See [self-governing.md](self-governing.md).

## 9. Quality gates from day one

The kit tests itself: **evals** (61+ fixtures scoring skills deterministically),
a **neutrality gate** (no organization/client identity in shipped content)
and **golden byte-equality** (source ↔ materialized). Every release runs the
lint against the kit itself in CI.

## 10. Clean exit

`golmet uninstall` removes the whole structure **layer by layer** with
real counts — everything selected by default except the packs (your local
working copy is deselected; the remote repo is never touched). See
[init.md](init.md).

## Docs in two languages

Every public doc ships in **English** + **Português (Brasil)** with an
alternation badge on top — including this page.
