<p align="center">English | <a href="SECURITY.pt-br.md">Português (Brasil)</a></p>

# Security Policy

GOLMET (`golmet-cli`) materializes workflows and specs from external sources — Git-based cores and packs — into engine configurations (opencode, claude-code, and others). Because it installs content written by third parties into configs consumed by AI agents, we treat its validation boundaries (pack contracts, path containment, credential redaction, MCP mutation gating) as security-sensitive surface.

## Supported Versions

| Version | Supported |
| ------- | --------- |
| 0.3.x   | ✅ Yes    |
| < 0.3   | ❌ No     |

Only the latest 0.3.x release receives security fixes. If you are on an older release, upgrade first — we will not backport patches to unsupported versions.

## Reporting a Vulnerability

**Primary and official channel: GitHub Private Vulnerability Reporting** on this repository (Security tab → "Report a vulnerability"). This is the only channel we currently commit to monitoring. Please do **not** open a public issue for anything you believe is exploitable.

> Maintainer note: if/when a dedicated e-mail address is available for security reports, it will be listed here. Until then, GitHub Private Vulnerability Reporting is the official channel — do not send reports to other addresses.

Please avoid public disclosure until a fix is released and we have coordinated a disclosure window with you.

## What to Include

To triage quickly, please include:

- **GOLMET version** (`golmet version` or the release tag) and commit SHA if built from source.
- **OS and affected engine** (e.g., Windows 11 + opencode, Linux + claude-code), including whether the flow went through the CLI or the MCP server tools.
- **Reproduction steps** — exact commands and any `golmet.yaml` / `pack.yaml` used.
- **A malicious pack or core PoC** when applicable — a minimal Git repository whose content triggers the issue (e.g., a `pack.yaml` whose `provides` entries attempt traversal, a symlink/junction in a pack, content crafted for prompt injection that gets materialized). Attach or link the repo; do not send real credentials.
- **Expected vs. actual behavior**, including any structured error output.

## Scope

### In scope

- **Code execution or config corruption via materialized content** — malicious packs/cores that abuse the installation pipeline to inject instructions (prompt injection surfacing in materialized skill/agent files) or overwrite files they should not.
- **Path escape during materialization** — `pack.yaml` `provides` entries bypassing the `agents/`/`skills/` allowlist, path traversal, or symlink/junction escape from the source cache or the destination root.
- **Redaction failures** — Git credentials, tokens, or secrets from source URLs leaking into manifests, payloads, reports, or error messages.
- **Pack contract validation bypasses** — including `requires-core` (and `no-override`) validation that can be evaded, letting an incompatible or hostile pack install.
- **The MCP server (`stdio` transport)** — mutation gating (`confirm=true` enforcement), `dry_run` side effects, and tool argument handling.

### Out of scope

- **Vulnerabilities in the host engines** (opencode, claude-code, editors, `git` itself). Report those to the respective projects; we are happy to help coordinate if the issue requires changes on our side.
- **Operator misconfiguration that disables validations** — e.g., explicitly using `--force` to override managed files, removing checks from configs, or pointing sources at untrusted repos with validations stripped. The tool's guarantees assume its shipped validation paths are in use.
- **Brute force of Git credentials** — credential strength, rate limiting, or cracking of Git hosting providers is out of scope.
- Social engineering of users into installing packs they do not trust (see the trust note below).

## Trust Model Note

GOLMET validates the *structural* safety of packs and cores (allowlisted paths, containment, contract compliance) but does **not** vet the *semantic* content of third-party packs. A structurally valid pack can still contain prompt-injection text or bad instructions that an AI agent will read. Installing a pack means trusting its author at the content level. Treat pack sources like you treat dependencies in any package manager: pin them (GOLMET records the resolved SHA in the manifest), review what you install, and report content-level attacks as vulnerabilities when they exploit a validation gap rather than mere author choice.

## Safe Harbor

We consider good-faith security research to be authorized and welcome, provided you:

- Only test against repositories and configs you own or have explicit permission to modify;
- Do not exfiltrate data, degrade service, or target infrastructure you do not control (including Git hosting providers' infrastructure);
- Use benign PoCs that demonstrate the issue without causing harm;
- Report privately through the official channel and give us reasonable time to fix before any public disclosure;
- Do not demand payment or make threats tied to disclosure.

We will not pursue legal action for research that respects these boundaries, even if it causes temporary, reversible disruption in your own test environments. If you are unsure whether an action falls within this policy, ask us first via the reporting channel.

## Response Expectations (Honest SLA)

GOLMET is a young project maintained with limited resources, so we commit to process rather than timelines:

- **Triage acknowledgement: within 7 days** of your report, you will get a response confirming receipt and an initial assessment of severity and validity.
- **No fixed fix timeline.** We will work with you on a coordinated disclosure plan, but we will not promise a specific patch date we cannot keep. Complex fixes (e.g., containment redesign on Windows junction handling) may take longer than simple ones.
- We will keep you informed of progress at each meaningful step and credit you in the release notes (unless you prefer to remain anonymous).
- Reports that are out of scope will be answered with an explanation and, where possible, a pointer to the right project or configuration change.