## GOLMET Kit v0 — F1 + camadas de conteúdo (L1/L2)

### O que este PR entrega

**Kit (Go, 14 comandos funcionais, 321 testes):**

- golmet validate — FORMAT v1 (H1–H10/S1–S5) com evidência grammar, modo tolerant/strict, --json
- golmet init — scaffold de workspace/container (.golmet/ + adapters)
- golmet install — wizard de onboarding (7 engines, catálogo 12 MCPs, badges de já-registrados, pós-verify em cascata)
- golmet update — BASE skills via go:embed, manifest sha256; --full materializa o workflow FULL no engine (embarcado ou core externo)
- golmet status / golmet next — dashboard + recomendação p1..p9
- golmet doctor — 9 checks com remediação, smoke Jev por default (--no-network pula)
- golmet adopt — handshake de adoção do FORMAT (dry-run classificado, apply determinístico, keep-legacy-layouts)
- golmet eval — checker determinístico de evals (15 checks, suite, json)
- golmet tools — instalação de codegraph com supply-chain (SHA-256 + attestation)
- golmet mcp — servidor MCP stdio (10 tools)
- golmet index — recomposição do índice de estados
- golmet pack — add/update/list de packs L2 (pack.yaml, overlay sem destruir customizações)
- golmet db — drift DATA_MODEL ↔ migrations/snapshot (camada do gm-dba)

**Subset BASE gm-\***: 16 skills com front matter de capacidades (port integral do acervo, neutralidade total)

**Camadas de conteúdo (DR-021)**: L0 BASE embarcado · L1 FULL (89 componentes renderizados do registry) com core externo opcional (`--core`, Git ou local) · L2 packs (provides em agents//skills/, `no-override`, `requires-core` validado contra a versão do kit) · L3 local. Precedência `local > pack > full > base` — customização local e arquivo gerido por pack nunca são sobrescritos sem `--force`; overlay 3-way com proveniência (layer/origem/SHA-256) no manifest de conteúdo.

**Segurança de conteúdo/fontes**: paths renderizados validados contra traversal (allowlist `agents//skills/`), destino e fonte rejeitam symlink/junction escape (containment real via EvalSymlinks), cache Git sujo falha fechado antes do `pull --ff-only`, credenciais de URL redigidas em manifest/payload/erros, mutações MCP exigem `confirm=true` e `dry_run` não escreve runtime, manifest nem cache.

**MCP (10 tools)**: gm_validate, gm_doctor, gm_next, gm_status, gm_index, gm_eval, gm_jev_classify, gm_update (FULL embarcado/core externo, dry-run), gm_pack_update, gm_pack_list

**FORMAT v1**: spec normativa (entidades, H1–H10/S1–S5, evidência, subset YAML §6.1, estados-default §7) + S10 (política graph-first) + §2.1 camadas de conteúdo e §3.1 contrato de pack

**Infra**: registry de evals com corpus gm (22 fixtures), neutralidade auditada (0 hits de org), cross-compile pronto

### Commits (19)

BLOCK 1: validate end-to-end. BLOCK 2: init/status/doctor/index. BASE gm-_. FORMAT v1. kit vocabulário. BLOCK 3: update/mcp/tools/next. adopt. eval + corpus gm-_. dívidas técnicas. install wizard. stack playbooks. fix Jev transport. FORMAT 2. gm_status/gm_index/gm_eval via MCP. FORMAT S10 graph-first. content L1 + FULL embarcado com overlay 3-way (install/update --full). tools MCP gm_update/gm_pack_update/gm_pack_list com dry-run e segurança de camadas. db drift (gm-dba determinístico).

### Testes

- **321 testes Go** (unit + integr + e2e do binário) — `go test ./...` verde, `go vet` limpo, `gofmt` limpo
- **E2e real**: 41 fixtures/slices tocadas. 37 provadas. 0 FAIL de substância
- Cross-projeto validado
- Security baseline: 5 findings documentados
- Cobertura nova deste delta: dry-run sem efeitos (runtime/manifest/cache), confirmação ausente, precedência pack/local sob `--force`, no-override do próprio pack, traversal/symlink/junction na origem e no destino, cache Git sujo, redaction de credenciais, requires-core incompatível
