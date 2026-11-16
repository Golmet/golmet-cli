---
project: GOLMET
module: docs/architecture
audience: [tecnica]
status: draft
---

# GOLMET — Arquitetura de Plataforma v2 (Formato, Kit e Runtime)

> **Fonte:** sessões de estratégia, desenho e batismo 05–08/11/2026 (Operador).
> Consolidada após 12 turnos de arquitetura + 8 rodadas de naming (~52 nomes verificados com veredicto).
> **Status:** design e batismo aprovados pelo Operador — implementação em curso no repo do kit. Cada seção gera tasks óbvias no core.
> **Nome oficial (DR-013):** GOLMET — ver §17 (Marca e Persona) e §14.
> **Ativos ao vivo:** npm `golmet-cli` · GitHub `github.com/Golmet` · domínio `golmet.io`

---

## 1. Contexto e Motivação

O workflow anterior era um workflow maduro **internamente** (produtos documentados em container próprio), mas tinha barreiras estruturais para adoção externa:

1. **Modelo de distribuição corporativo para um produto dev**: exigia clone do repo público → instância privada → repo de pack → wizard de setup → runtime global obrigatório.
2. **Acoplamento a um único engine**, num mercado convergindo para padrões abertos (AGENTS.md, SKILL.md, MCP).

A análise concluiu: o método já era top; o que faltava era **prova, distribuição e porta de entrada**. Fork de runtime foi rejeitado; port por engine foi rejeitado; a resposta é a arquitetura abaixo.

## 2. Tese Central

> **O workflow é uma máquina de estados documental do produto.** O formato é o contrato; o kit é o motor determinístico; o runtime é a camada agêntica que executa por capacidades.

Três camadas divididas **por natureza** (cada uma com cadência de mudança e canal de distribuição próprios), não por tier de cliente:

```
┌──────────────────────────────────────────────────────────┐
│ 1. FORMAT — spec versionada                             │
│    layout do workspace · schemas dos artefatos ·         │
│    transições de estado · gramática de evidência         │
├──────────────────────────────────────────────────────────┤
│ 2. KIT — CLI determinístico (SEM LLM)                    │
│    init · validate · status · doctor · update ·          │
│    pack · contribute · mcp                               │
├──────────────────────────────────────────────────────────┤
│ 3. RUNTIME — camada agêntica, por capacidades            │
│    perfil BASE (qualquer engine) · perfil FULL           │
│    (runtime de referência: OpenCode)                     │
└──────────────────────────────────────────────────────────┘
```

**Princípios:**

- **Julgamento no engine** — elicitar, extrair regra, redigir spec, generalizar contribuição: sempre camada agêntica.
- **Determinismo no kit** — IDs, transições legais, invariâncias, staleness, scaffold, neutralidade mecânica: sempre CLI/MCP, centralizadamente atualizável, nunca copiado.
- **Verdade no formato** — os artefatos sobrevivem a engines, kits e releases; qualquer consumidor futuro ingere o FORMAT.

**Regra de ouro da degradação:** degrada ergonomia, nunca invariância do formato.

## 3. FORMAT v1

### 3.1 Entidades e máquina de estados

| Entidade | Estados v1 | Transições legais | Pré-condição (kit valida) |
|---|---|---|---|
| **Charter** | `draft → confirmed` | confirmar | §1 com repos registrados + HEADs |
| **RN** | `draft · active · superseded · dropped` | draft→active; active→superseded(→RN-XXX); active→dropped | evidência parseável ≥1; `dropped` nunca reverte |
| **PEND** | `open · deferred · resolved · dropped` | open→resolved{via RN/CR/ADR}; open→deferred; →dropped(ADR cita) | resolução cita o destino |
| **CR** | `pending · approved · rejected · implemented` | pending→approved/rejected; approved→implemented | approved exige decisão registrada |
| **Feature** | `draft · ready · in_progress · done · cancelled` | sequencial; →cancelled | ready exige âncora RN `active`; done exige tasks `done` |
| **Task** | `todo · in_progress · done · blocked` | livre entre não-finais | — |
| **Spec** | `current · stale` | ↔ (kit escreve) | `stale` obrigatoriamente com `reason: CR-XXX` |
| **ADR** | `proposed · accepted · superseded` | proposed→accepted→superseded(→ADR) | conteúdo imutável (append-only) |
| **Candidate** (reverse) | `pending · approved · rejected · promoted` | → com {dono, data} | já formalizado hoje — formato codifica |

### 3.2 Decisões estruturais

- **D1 — Verdade declarada vs. inferida.** O formato governa apenas o **estado normativo** (o que foi decidido). Estado de **implementação** é inferido por análise de código — nunca afirmado pelo formato. O kit só detecta **evidência stale** (hash ≠ HEAD atual), nunca "não implementada".
- **D2 — Estado por entidade vive em índice gerado pelo kit.** Prose continua sendo a verdade humana; o Índice de Estados é cache com **único escritor = kit** (mesmo padrão do registry-render: canonical = prose, cache = índice). Sem YAML por seção, sem HTML comments.
- **D3 — Enforcement em dois níveis.** **Hard** (bloqueia `done`/publish): PEND nunca ancora Feature (IMP-094); IDs únicos e imutáveis; âncora de Feature sempre RN `active`; RN `active` com evidência parseável. **Soft** (aparece no doctor, não bloqueia): CR `implemented` com specs ainda `stale` — a prática atual tolera propagação deliberadamente pendente.

### 3.3 Gramática de evidência

Forma canônica parseável: `[caminho:linha @hash]` — hash resolve contra o registry de repos do charter §1. A evidência deixa de ser convenção variável e vira **chave de junção** entre docs e qualquer consumidor externo. O `doctor` compara hashes vs HEAD e emite worklist de refresh.

### 3.4 Guarda de escopo v1

NÃO entra: lifecycle do produto · severidade como estado (fica label) · assignees/prazos · estados de compliance · validação de qualidade de conteúdo (território de clarify/prompt). Critério de entrada: só formaliza o que **três consumidores** precisam — kit (validar), engines (executar transições), ingestores futuros.

### 3.5 Versionamento

`format: 1` declarado no `golmet.yaml` da raiz do workspace. Kit versão X suporta formatos 1..X. Migração = `golmet update`. A spec do formato vive no repo público — é o contrato que terceiros implementam.

## 4. KIT v0

### 4.1 Comandos

| Comando | O que faz | Absorve (hoje é skill) |
|---|---|---|
| `golmet init` | scaffolding do workspace (ou `--org` para container; `--into` para adicionar workspace a container existente; `adopt` envolve estrutura existente) | — |
| `golmet validate` | conformidade ao FORMAT: front matters, gramática de ID, unicidade/sequência, transições legais, gramática de evidência, invariâncias hard. Exit code ≠ 0 → CI-block | skills da geração anterior de validação + mecânica de wiki-lint e decision-audit |
| `golmet status` | dashboard: estados por entidade, PENDs, CRs, propagação devedora, versão de formato | skills da geração anterior de progresso e tasks pendentes |
| `golmet doctor` | freshness de evidência vs HEAD; capacidade do engine vs `requires` das skills; versões. Detecta — nunca reescreve verdade | novo |
| `golmet update` | atualiza subset embarcado com manifest SHA-256 preservando arquivos modificados; migra formato | novo |
| `golmet pack sync / link` | sincroniza pack no pin / modo dev (link para clone local) | skill da geração anterior de pack, adaptada |
| `golmet index` | ingestão em grafo de bases de código — codebase-memory **nativo** (DR-016): download verificado, indexação incremental, cobertura reportada no doctor | skill de setup do codegraph da geração anterior (absorvida) |
| `golmet mcp` | expõe validate/status/doctor/next + tools do grafo (DR-016) + `gm_jev_classify` (DR-015) como tools MCP | novo |
| `golmet contribute` | pipeline de contribuição upstream: neutralidade mecânica + payload de PR (branch, DCO) via gh | novo (ver §10) |

**v0.5:** `next` (hints determinísticos de prioridade) · `eval` (DR-017) · `validate --json` · GitHub Action de 1 linha.
**v1+:** `export` (JSON normalizado) · `shard-check` (contadores IMP-143).

Stack: **Go nativo — zero dependências de runtime** (DR-018, substitui Node/TS); distribuído via npm (instalador fino + binários por plataforma, modelo esbuild); funciona offline; CI-first.

### 4.2 Regra de quem escreve o quê

```
HUMANO   confirma gates (via runtime/question ou PR review)
AGENTE   escreve verdade de domínio (prose: RNs, specs, charter) — julgamento
KIT      escreve só o que RECOMPUTA: scaffolding, índices, flags stale, relatórios
         — nunca prose de domínio, nunca re-pin de hash sem agente
```

### 4.3 Relação agêntico ↔ determinístico

> O kit é para os documentos o que o typechecker/linter é para o código: **o agente escreve, o kit checa, o agente corrige, o kit recompõe os caches.**

Quatro modos de integração:

1. **MCP server (primário)** — `golmet mcp` expõe tools com nomes únicos (`gm_validate`, `gm_status`…). A superfície de tools do workflow passa a ser **definida pelo kit**, independente de primitivos nativos de engine. Config MCP vive no próprio workspace (`.opencode/`, `.mcp.json`, `.cursor/`) — escrita pelo `init`, invisível ao usuário.
2. **CLI via shell (fallback universal)** — skills mandam rodar `golmet validate`.
3. **Skills como protocolo** — cada skill declara seus checkpoints de kit (ver §5.3).
4. **CI como árbitro independente** — GitHub Action com `validate --strict` no push. Hierarquia: agente se-auto-checa → CI força → humano revisa.

Erros do kit são estruturados (`--json`: arquivo, entidade, regra, correção sugerida) — consumíveis pelo agente no loop.

## 5. RUNTIME

### 5.0 Distribuição (M31 — a via npm)

O binário chega ao público por 3 vias, na ordem de preferência:

- **npm (a via principal)**: `npm install -g golmet-cli` — o `bin/golmet.js`
  (launcher, Node ≥14, sem dependências) detecta `process.platform/arch`,
  resolve o binário nativo do sub-pacote `@golmet/cli-<os>-<arch>`
  (optionalDependencies do principal) e o despacha com stdio inherit — o
  exit code propaga. Pacote ausente → erro claro com os 2 caminhos de
  recuperação. O release workflow **monta os 5 pacotes de plataforma**
  (`npm/<os>-<arch>/` com o binário cross-compilado, versão fixada pela
  tag) e publica via `npm publish` — **os 5 de plataforma ANTES do
  principal** (o npm resolve os opcionais na instalação do principal);
  publish tolerante a falha (o release nunca quebra por npm — decisão M4).
- **Go**: `go install …/cmd/golmet@latest` (a via do desenvolvedor Go).
- **GitHub Releases**: os binários cross-compilados + checksums SHA-256
  (a via manual).

O `golmet upgrade` (self-update) serve as vias Go/binary; sob gestão npm
(o path do binário contém `node_modules`) ele NÃO substitui o arquivo —
orienta `npm update -g golmet-cli`.

### 5.1 Modelo de capacidades

Skills não conhecem engines — conhecem **capacidades**. Engine novo = 1 linha em manifest, nunca um port.

| ID | Capacidade | Disponibilidade (2026) |
|---|---|---|
| `file` | ler/editar arquivos | todos |
| `shell` | executar comandos | todos |
| `mcp` | cliente MCP | OpenCode, Claude Code, Cursor, Copilot, Codex, Gemini CLI |
| `ask` | pergunta estruturada c/ opções | OpenCode (`question`), Claude Code (`AskUserQuestion`), Cursor (plan mode) |
| `subagent` | spawn em contexto isolado + retomada | OpenCode, Claude Code |
| `parallel` | N subagentes simultâneos | OpenCode, Claude Code |
| `workspace-ops` | worktrees, session_move/rename | só OpenCode |
| `db-introspect` | estado interno do runtime | só OpenCode |

### 5.2 Perfis

```
BASE = file + shell (+mcp recomendado, +ask com degradação)
FULL = BASE + ask nativo + subagent + parallel + workspace-ops
```

- **BASE** roda os 2 fluxos de entrada (greenfield + reverse single-agent) em qualquer engine dos padrões.
- **FULL** desbloqueia: reverse sharding multi-repo (IMP-143), orchestrators, dev executors, devops.
- Roteamento no BASE: **um único agente de entrada** — sem Quadri/Tab; auto-rotea pelas descrições das skills + `golmet next`. No FULL: Quadri como hoje.

### 5.3 Contrato de skill (front matter estendido)

```yaml
name: gm-requirements
requires: [file, ask, kit]
degrades:
  ask: chat-stop            # pergunta no chat e PARA (IMP-085) — nunca adivinhar
kit:
  checkpoints: [post-write: validate]
```

Escadas de degradação explícitas por capacidade (`ask`: nativo → chat+stop → checklist async; `subagent`: spawn → inline sequencial → desabilitado com recomendação de FULL; `parallel`/`background`: sequencial/síncrono). O `doctor` compara `requires` instaladas vs capacidades do engine detectado.

**Capacidades providas pelo kit** (ambiente — não engine): `kit.jev` (classificação calibrada, DR-015) e `kit.graph` (grafo de código, DR-016). Skills as declaram em `requires`; o `doctor` checa presença (key/conectividade/binário indexado) e emite AVISO com remediação quando ausentes — o julgamento degrada para heurística declarada, nunca adivinha. MCPs de serviço NÃO são capacidades do kit (DR-016): ficam sob gestão do runtime.

## 6. Distribuição e Instalação

- **Canal:** pacote npm (`npx golmet-cli golmet` até o appeal do unscoped `golmet`). **Fonte de desenvolvimento = repo público `Golmet/golmet-cli` desde a F1 (DR-019):** kit (Go) + FORMAT + `gm-*` BASE, abertos. Cores privados de adotantes mantêm workflows FULL próprios até a migração — realidade de mantenedor, não requisito da arquitetura (DR-006).
- **`init` instala por perfil, em dois lugares:**
  - subset **BASE** → dentro do **workspace** (portátil — é o que torna o workspace multi-engine);
  - perfil **FULL** → estrutura do runtime de referência (OpenCode: agents/skills).
- **Skills BASE embarcadas são VENDORED** (commitadas no container/workspace, com VERSION + manifest) — precisam ser idênticas para todo mundo; updates via `golmet update` viram commit de bump, propagando por PR como qualquer código.
- **Pré-requisitos do usuário:** Node + qualquer agente compatível. **Zero** config global, wizard, instância privada ou pack para começar.

## 7. Container de Organização

```
MinhaOrg.docs/                  # container (docs-repo da organização — versionado)
├── golmet.yaml                    # container: true · pins de pack · format
├── CONSTITUTION.md             # princípios da org
├── INDEX.md                    # TOC dos workspaces — GERADO pelo kit
├── BarbeariaPro/ …             # workspaces (mesmo formato do modo solo)
├── packs/                      # GITIGNORED — conteúdo fetched por pin (ver §8)
└── local.yaml                  # GITIGNORED — config de cliente/org (nunca versionada)
```

- `golmet init --org` cria; `golmet init <Produto> --into <container>` adiciona; `golmet adopt` envolve estrutura existente (caso de adoção de container já em produção) sem tocar nos produtos.
- **Identidade do operador:** `git config` (default) + opcional `~/.golmet/profile.yaml` (local, gitignored) para extras. **Nada pessoal no repo de docs.**
- **Multi-dev:** onboarding do dev #2 = `git clone` + `golmet doctor` (checklist autodiagnóstico) + `pack sync` + abrir o agente. Sem wizard. Aprovações organizacionais = PR reviews + registros de decisão `{dono, data}` (dono = identidade git). Conflito de Índice de Estados pós-merge = mecânico (`validate --fix` recompõe); conflito de prosa = humano.

## 8. PACK (overlay da organização)

- **Versiona apenas DELTAS** — nunca cópia integral da base: overrides de skills (por nome, precedência pack > base), skills/subagents criados pela org, regras da org, acumulação (patterns/metrics/reverses).
- **Lockfile no container:** `golmet.yaml` carrega `packs: [{id, repo, pin}]` — versionado. **Conteúdo** em `packs/` gitignored, fetched por `golmet pack sync` (acesso ao repo privado do pack).
- **Repo do pack = única fonte da verdade** do conteúdo org (história própria, PRs próprios, tags por release).
- **Por que não vendorar:** auditabilidade vem do pin (bump é commit revisável); ciclos de vida separados (pack ≠ docs ≠ base); fronteira de vazamento aplicável por permissão.
- **Modo dev:** `golmet pack link <clone-local>` — o champion edita o clone e vale imediatamente no container dele (`npm link` pattern); publicação = push + tag + bump de pin.
- **Exceção opt-in:** `golmet export --vendor` para handover/arquivo pontual.

## 9. Jornadas de Uso

**Solo, produto novo:** `npx golmet init Produto` → kit pergunta engine e repos → abre o agente na pasta → conversa (charter → RNs → tasks) → markdown versionado no git do usuário. MCP configurado automaticamente pelo init.

**Solo, repo legado:** `init --repo <path>` → charter §1 preenchido com HEAD congelado → agente extrai RNs com evidência `[path:linha @hash]` → lotes de CANDIDATES aprovados pelo humano → REQUIREMENTS.md. Repo grande demais? Kit avisa: perfil FULL recomendado.

**Dev #2 da empresa:** acesso aos 2 repos → clone → `doctor` → `pack sync` → abrir agente. Checklist sem manual.

**Champion de melhorias:** link do pack em modo dev → improver/builder na sessão agêntica → evals → push/tag no pack → bump de pin para a org (ver §10).

## 10. Contribuição e Evolução do Workflow

Builder (`gm-workflow-builder`) e improver (`gm-workflow-improver`) são **skills agênticas da camada FULL** — chamadas diretamente na sessão de IA pelo improvement champion da org (sem mecanismo passivo de captura de feedback). O agente edita **fonte versionável**, nunca a instalação viva:

```
melhoria org-specific → PACK (release própria → pin bump → fast lane: mesmo dia)
melhoria genérica     → FORK do repo público → PR (evals × engine + neutrality gate)
                        → merge → release npm → golmet update → todos
                        → org remove o override do pack (pack encolhe = sinal saudável)
```

**Três velocidades de propagação:** meu container (pack link — imediato) → minha org (pack release — mesmo dia) → o mundo (PR upstream — ritmo do review).

**`golmet contribute`:** pipeline determinístico de contribuição — roda o neutrality gate (scan mecânico de nomes de org/cliente/domínios/IPs; reutiliza o script existente nos evals), monta branch + commit DCO + abre PR via gh. A **generalização** (tirar o específico da org) é julgamento — fica no improver/builder, camada agêntica.

**Gate de promoção em todos os saltos:** `gm-skill-eval` antes de release de pack, antes de PR upstream e antes de release npm — com matriz eval × engine (skills agora rodam em múltiplos engines).

## 11. Destino do Inventário Atual (25 subagents + 78 skills)

| Categoria | Qtd aprox. | Destino |
|---|---|---|
| Subset BASE (charter, clarify, requirements, glossary, task-spec, reverse single-agent, candidates…) | ~12–15 | embarcadas no workspace |
| Absorvidas pelo kit (validate, progress, tasks-pending, wiki-lint-mecânica, decision-audit-mecânica, propagation-guard, next) | ~7–8 | skill encolhe para wrapper fino ou deixa de existir |
| FULL-only (orchestrators, dev-backend/frontend/qa/reviewer, devops, reverse sharded) | ~20 | distribuição FULL — opt-in |
| Engine-only (perf-auditor, knowledge-auditor) | 2–3 | marcadas `db-introspect`, degradam com aviso (IMP-080) |

v0 do tier community é **docs-focused de propósito**: skills de implementação (`gm-dev-*`) ficam FULL — implementar código não é a porta de entrada.

## 12. O que deixa de existir

- **CORE privado como conceito de adotante/org**: ninguém clona engine. npm (base+full) + pack (deltas org) + fork público (contribuição) cobrem tudo. O repo interno de desenvolvimento sobrevive **apenas como realidade de mantenedor** (fonte das releases npm + snapshot público) — e até isso é escolha de governança, não requisito da arquitetura.
- **Wizard de setup obrigatório** (perfis de cliente/operador como barreira de entrada) — identidade via git config; config de cliente é `local.yaml` gitignored opcional.
- **Instâncias globais como pré-requisito** — o estado do workflow mora no repo do usuário.
- **Port por engine / fork de runtime** — rejeitados como estratégia (ver §14).

Mapa de substantivos final do usuário/org: **kit · workspace · container · pack · engine** (cinco, nenhum herdado do modelo antigo).

## 13. Fases de Implementação

| Fase | Conteúdo | Observação |
|---|---|---|
| **F0 — Cirurgia de camadas** (prep, no core atual) | auditar as 78 skills marcando deps de tool (grep `question`/`task`/opencode); classificar inventário conforme §11 | barato; serve mesmo sem portar |
| **F1 — FORMAT + KIT v0** | spec do FORMAT v1; kit com `init/validate/status/doctor/update`; extrair subset BASE (~12 skills); front matter de capacidades nas skills | semanas, não meses; subtrai manutenção de ~6-8 skills |
| **F2 — Prova de adoção externa** | publicar npm; demo-assassina (legado → specs rastreáveis); case dogfooding; docs EN + landing | **KPI: adoção externa real dos artefatos** — não estrelas |
| **F3 — FULL via npm + contribuição** | `install --profile full` no runtime de referência; `golmet contribute`; evals × engine no CI upstream | |
| **F4 — Container/pack na org** | `--org`, `adopt`, pack sync/link/pin no container; multi-dev | |

Ordem deliberada: **prova antes de expansão** — nenhuma fase além de F2 depende de validação externa para ser decidida.

## 14. Registro de Decisões (DR-001..022)

| ID | Decisão | Síntese |
|---|---|---|
| DR-001 | Distribuição por padrões, não ports | AGENTS.md/SKILL.md/MCP como substrato; engine novo = 1 linha de manifest |
| DR-002 | Sem fork de runtime | distro/plugins primeiro; fork só com gatilhos objetivos (PMF + financiamento + primitivo impossível via plugin/MCP) |
| DR-003 | Unidade = workspace de produto | nunca pasta de config por repo nem instância global; repos de código referenciados no charter, nunca invadidos |
| DR-004 | KIT determinístico sem LLM | kit escreve só o que recomputa; agente escreve prosa; humano confirma gates |
| DR-005 | FORMAT v1 (D1–D3) | verdade declarada ≠ inferida; índice gerado pelo kit; enforcement hard/soft |
| DR-006 | npm como canal; core privado sai do camado de adoção | core privado = só realidade de mantenedor (opcional até para eles) |
| DR-007 | Pack = deltas com pin no container | conteúdo gitignored; repo do pack é a SóT; nunca vendorado (exceto export opt-in) |
| DR-008 | Skills BASE vendored no workspace | idênticas para todos; update = bump commitado via PR |
| DR-009 | Builder/improver agênticos, sem /feedback passivo | melhorias nascem em sessão de IA; editam fonte versionável (pack/fork), nunca instalação viva |
| DR-010 | `golmet contribute` | neutralidade mecânica (gate existente) + mecânica de PR; generalização é julgamento agêntico |
| DR-011 | Identidade via git config | nada pessoal no repo de docs; config de cliente = `local.yaml` gitignored |
| DR-012 | Produtos irmãos da organização fora do escopo desta arquitetura | consumidores futuros potenciais do FORMAT; nenhuma dependência no desenho |
| DR-013 | **Batismo: o workflow passa a se chamar GOLMET** | **GOLEM + EMET** (אמת, "verdade" — a palavra inscrita que anima o golem): o nome performa o mito do produto — o T da verdade que não pode faltar. Escolhido após 8 rodadas de naming com ~52 nomes verificados (npm/GitHub/marcas); reservas de canais públicos executadas na data — ver §17 |
| DR-014 | Prefixo padrão de skills/agents = `gm-*` | iniciais de GolMet (Gol+Met); curto, sem colisão de namespace; skills BASE nascem `gm-*`, legadas migram na F3 |
| DR-015 | **Transporte do Jev migra para o kit** | `golmet mcp` expõe `gm_jev_classify` (POST HTTPS mecânico — mesma entrada, mesma chamada); key via env `TYPESAFE_API_KEY`; o transporte triplo atual (plugin MCP do runtime + task para subagent + curl na skill) colapsa em 1 caminho e o plugin `jev-tool.js` deixa de ser necessário; skills mantêm o julgamento (o que classificar, com quais critérios) e encolhem o transporte; degradação heurística declarada no front matter (padrão do prompt-gate vira lei) |
| DR-016 | **MCPs opcionais ficam na RUNTIME layer; codegraph é a exceção nativa** | MCPs de serviço (playwright, postgres, sqlite, figma, hostinger) continuam gerenciados pelo runtime (skills de setup como hoje); **codebase-memory (grafo de código) entra no kit**: download com verificação supply-chain em código de produto (SHA-256/attestation testável), `golmet index` torna a ingestão em grafo nativa do bootstrap (`init --repo` já indexa), tools do grafo expostas via `golmet mcp` — engine-agnostic; skill de setup do codegraph absorvida |
| DR-017 | **`golmet eval` (v0.5)** | runner determinístico de evals (fixtures + criteria + relatório) vira comando do kit; CI do core e o gate de promoção de packs (§10) o invocam — skill-eval encolhe para wrapper |
| DR-018 | **Stack do kit: Go nativo, zero-dep** | binário único por plataforma (decisão do operador 08/11 — substitui a linha Node/TS do §4.1); npm = **instalador fino** (modelo esbuild: `optionalDependencies` por plataforma, Node só no bootstrap do wrapper; `npx golmet-cli golmet` até o appeal do unscoped); SDK MCP **oficial go-sdk**; parser de front matter = **subset YAML próprio** normatizado no FORMAT §6.1; arquitetura **core-lib** (`internal/core` puro, adapters injetados — CLI, MCP server, GitHub Action e testes consomem a mesma lógica); motor de scan **portado do scanner da geração anterior** (testes como âncora de paridade); bin `golmet` (`gm` descartado — colide com GraphicsMagick). Rust permanece para onde performance real importa (grafo, produtos irmãos) |
| DR-019 | **Casa canônica do kit: repo público desde a F1** | kit (Go) + **FORMAT.md (spec canônica)** + skills BASE `gm-*` (tier community) desenvolvem-se **abertos** em `Golmet/golmet-cli`; cores privados de adotantes mantêm workflows FULL próprios até a migração. Histórico público desde o primeiro commit (credibilidade F2, contribuição F3 trivial). FORMAT.md migra para o repo do kit no início da implementação |
| DR-020 | **Política graph-first para o grafo de código** | obriga a **disciplina**, não a infra: com `kit.graph` presente e índice fresco, navegação estruturada (`get_architecture`/`search_graph`/`trace_path`) é a primeira instância — grep/glob ficam para literais e para o que o grafo não resolve (ex.: atributos ASP.NET `[HttpGet]`). Índice stale responde com confiança errada (pior que grep) → re-indexar antes de usar. Sem grafo → degradação `grep-fallback` declarada no relatório (`navegação: graph \| grep-fallback`) — observável, nunca silenciosa, nunca bloqueante. Motivação: elevar a qualidade do workflow permanentemente sem fragilidade de pré-condição dura (decisão do operador 11/11). Aplicada em: gm-dev-backend/frontend (de "complemento opcional" → graph-first), gm-reverse/gm-reverse-orchestrator (mineração de fatia começa pelo estrutural) e playbooks de stack (rust/dotnet/java/angular) |
| DR-021 | **FULL embarcado + camadas de conteúdo com overlay seguro** | o CLI embarca o FULL (89 componentes) como L1 default offline; `golmet install --full` e `update --full` renderizam o registry em Go e materializam agents/skills no engine. Camadas: L0 BASE embedded → L1 FULL → L2 packs (organização/comunidade) → L3 local; precedência `local > pack > full > base`. Manifest registra origem/layer/SHA por arquivo. Update usa detecção 3-way: customização local divergente é preservada e reportada como conflito; só `--force` substitui. O core externo é a variante opt-in detalhada na DR-022. Decisão do operador 12/11: conteúdo FULL junto ao CLI, sem acoplar o usuário a clone/rede no bootstrap; packs continuam a unidade futura de compartilhamento comunitário. |
| DR-022 | **Core externo e packs usam o mesmo transporte Git/cache** | `golmet update --full --core <url-ou-diretório> --engine <id>` sincroniza o core externo em `~/.golmet/sources/core`, fixa o SHA e reutiliza o mesmo parser/render/overlay do FULL embarcado. `golmet pack add <url-ou-diretório> --engine <id>` sincroniza um pack L2 em `~/.golmet/packs`, valida `pack.yaml` (`id`, `version`, `provides`, `no-override`) e aplica apenas `agents/`/`skills/`; conflitos locais são preservados. Isso deixa a comunidade compartilhar packs sem acoplar o binário e reserva um índice de descoberta para evolução posterior. |

> **Guarda de entrada no kit (DR-015..017):** só entra o que passa nos 3 filtros — (1) determinístico/mecânico, (2) consumido por 2+ skills, (3) degrada com graça offline. Julgamento fica agêntico; serviços com ciclo de vida próprio são **gerenciados, nunca embutidos** (exceção única deliberada: o grafo, DR-016).

## 15. Fora de Escopo (explícito)

- **Produtos irmãos da organização**: não integram esta arquitetura; permanecem potenciais consumidores do FORMAT no futuro.
- **TOP 1 por adoção geral**: meta substituída — liderança no nicho brownfield/documentação com prova.
- **Fork de runtime / CLI gerador estilo Spec Kit**: rejeitados para o horizonte atual (ver DR-002).

## 16. Riscos Abertos

| Risco | Mitigação |
|---|---|
| Manutenção de CLI é um produto a mais | v0 fino (parser + 5 comandos); substitui trabalho mecânico de ~6-8 skills |
| Regex de evidência em paths Windows/esquisitos | testes dedicados no kit; gramática com casos de borda documentados |
| Engines divergirem dos padrões (guerra de lock-in) | camada metodológica independe; OpenCode segue como runtime de referência FULL |
| Regex/front matter editados à mão ilegalmente | `validate` pega; CI faz colar |
| Skills customizadas no pack divergirem de upstream sem genericização | improver decide override vs fork; evals × engine no gate |
| Duas rodas vivas na convivência divergirem (legado vs `gm-*`) | **evals de paridade** nas fixtures durante toda a janela — divergência bloqueia release (DR-017; plano de migração da org) |
| Jurídico/marca derrubar o nome GOLMET | reservas de canais públicos executadas na data do batismo; alternativas de nome avaliadas na mesma sessão (processo no §17) |

## 17. Marca, Nome e Persona (Batismo — 08/11/2026)

### 17.1 O nome

**GOLMET = GOLEM + EMET** (אמת — "verdade" em hebraico). Na lenda do Golem de Praga, o rabino anima o guardião de barro inscrevendo *emet* na sua testa — e quando uma letra se apaga, *emet* vira *met* (morto) e o golem desaba. **Drift é a letra que faltou.** O nome performa o mito do produto: o workspace guarda a palavra da verdade que mantém o sistema acordado.

**Disponibilidade verificada na data do batismo:** npm `golmet` livre · GitHub org `golmet` livre · `golmet.dev` não registrado · zero empresas/produtos de software com o nome · única colisão conhecida: uma linha de maçanetas de porta de garagem no Reino Unido (irrelevante — categorias e mercados distintos). Canais públicos reservados: npm `golmet-cli` (o comando `golmet`), org GitHub `github.com/Golmet`, domínio `golmet.io`.

### 17.2 Persona e mascote — **Arki**, o golem smol

> **Nome do mascote formalizado (08/11/2026):** **ARKI** — de *arkhē* (ἀρχή, o princípio primeiro): o pequeno guardião do princípio primeiro do produto. A marca chama Golmet; o personagem que a guarda chama Arki (padrão Linux/Tux · Rust/Ferris · GitHub/Mona).

**Arki** é o guardião de obsidiana com a placa dourada no peito — a chartula com as RNs gravadas — e olhos que acendem em laranja-âmbar quando detecta divergência. É o **golmet smol**: o guardião mais smol que já bloqueou o seu push.

- **Referência visual:** golem de blocos de obsidiana com veios de brasa, olhos âmbar sorridentes e **placa dourada redonda incrustada na testa** (o *shem* — brilho suave, sem insígnia). Peito: rocha pura, sem emblema. **Variações da placa aprovadas:** monograma **GM** entalhado (community/playful) e **אמת** entalhada (renders solenes — a palavra do mito).
- **Regra de ouro do mascote:** Arki **nunca cria conteúdo** — só diagnostica, cita evidência e bloqueia (mesmo contrato do kit).

**Prompt do Arki (placa GM — atualizado, gerar 4 variações):**

```
A cute tiny stone golem mascot for a developer tool brand called "Golmet". A small
friendly golem made of dark obsidian rock blocks, glowing ember-orange eyes, stubby
rounded arms, and a golden engraved tablet on its chest bearing the letters "GM"
carved in a geometric rune-like style. Chibi proportions, standing pose, soft studio
lighting, modern 3D tech-mascot style, dark neutral background, protective and
approachable expression. High detail.
```

### 17.3 Paleta e tipografia

| Uso | Valor |
|---|---|
| Fundo/primário | Obsidiana `#1A1D21` |
| Acento da arkhē (placa, losango, selo) | Ouro `#E8B44A` |
| Sinal de divergência (olhos do Arki, flags stale) | Laranja-âmbar `#FF6B35` |
| Tipografia | **Archivo** (geometria dura; pun embutido — *archivo* = arquivo/acervo) |
| Banner wide (README top, OG image) | ✅ gerado 08/11/2026 (wordmark dourado + Arki acenando, pedra obsidiana com fissuras de brasa; polimento F2: escurecer fissuras ~20%, crop 2:1, variação com tagline) |

### 17.4 Voz da persona

O Guardião Estoico: frases curtas, zero hype, sempre cita evidência. Erros do kit chegam como laudo do Arki — `{arquivo, entidade, regra, correção sugerida}` — e o mascote aparece nas mensagens de bloqueio: *"Arki bloqueou seu push: 2 RNs sem evidência parseável."*

### 17.5 Namespace pós-batismo

| Hoje | Pós-batismo |
|---|---|
| CLI da geração anterior | `golmet` (`npx golmet init`) |
| MCP tools da geração anterior | `golmet_*` |
| yaml da geração anterior | `golmet.yaml` |
| `~/.` da geração anterior | `~/.golmet/` |
| Skills BASE embarcadas | `gm-*` — prefixo padrão (DR-014: iniciais de **Gol**met + **M**et) |
| Skills FULL legadas | prefixo legado até a migração F3 (runtime de referência) |
| Repos | fonte privada de desenvolvimento; espelho público (renome de repo é decisão da F3) |

### 17.6 Distribuição npm (reserva de nomes — 08/11/2026)

O nome unscoped `golmet` está bloqueado pelo filtro anti-typosquatting do npm (distância ≤ 2 do popular `helmet` — bloqueio permanente enquanto o helmet existir). Estratégia em camadas, todas verificadas livres em 08/11/2026:

| Camada | Pacote npm | Papel | Status |
|---|---|---|---|
| Identidade | pacote com escopo do mantenedor | o nome com escopo, propriedade do mantenedor/org | ✅ publicado `0.0.1` (08/11) |
| CLI instalável | `golmet-cli` (unscoped) | `npm i -g golmet-cli` entrega o comando `golmet` | ✅ publicado `0.0.1` (08/11) |
| Futuro | `golmet` unscoped | appeal no suporte do npm (projeto real ≠ helmet); sem prazo | ⏳ appeal pendente |

> A marca é GOLMET; o nome de distribuição npm é encanamento. Projetos reais convivem com nomes de distribuição diferentes da marca (`tailwindcss` ≠ "Tailwind Labs") — e o appeal do unscoped é grátis para protocolar.

> IDs de artefatos (RN-XXX, PEND-XXX, CR-XXX, ADR-NNN) **não mudam** — IDs são imutáveis por contrato (§3.2) e o batismo é de marca, não de gramática. O Brasil entra como **dialeto interno** do workflow — homenagem sem encargo de marca.
>
> **Plano executável de migração:** mantido no repositório privado da organização adotante (matriz completa por superfície, fases e checklists).
