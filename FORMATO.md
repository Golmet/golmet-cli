# FORMATO v1 — Especificação Normativa do Workspace GOLMET

> **Versão:** 1 (draft F1) · **Consumidores:** kit GOLMET (valida) · engines agênticos (executam transições) · ingestores externos futuros.
> Este documento é **o contrato** que terceiros implementam. Vive no repo público via snapshot.

**Convenção de requisitos** (RFC 2119 adaptado):
- **DEVE / DEVE-NÃO** — violação = erro **hard** (`golmet validate` exit ≠ 0, bloqueia CI).
- **AVISO** — violação reportada pelo `golmet doctor`, **não bloqueia**.
- **PODE** — opcional, sem verificação.

---

## 1. Escopo (D1 — verdade declarada vs. inferida)

O FORMATO governa apenas o **estado normativo** do produto: o que foi decidido e formalizado em documento. Estado de **implementação** é inferido por análise de código — nunca afirmado pelo formato. O kit detecta **evidência stale** (hash ≠ HEAD atual); nunca declara "não implementada".

## 2. Layout do workspace

```
<Produto>/                          # workspace (raiz versionada pelo usuário)
├── golmet.yaml                     # manifesto — DEVE existir
├── golmet.index.json               # Índice de Estados — ESCRITO SÓ PELO KIT (§8)
├── .golmet/                        # ZONA DO KIT — vendored (DR-008: update reescreve preservando modificados)
│   ├── skills/gm-*/SKILL.md        #   subset BASE embarcado
│   └── manifest.json               #   VERSION + SHA-256 por skill
├── PROJECT_CHARTER.md              # entidade Charter
├── REQUIREMENTS.md                 # entidades RN + PEND
├── CHANGE_REQUESTS.md              # entidade CR
├── DECISIONS.md                    # entidade ADR
├── analysis/<RN-NNN — <título>>/   # entidade Spec (funcional + técnica)
│   ├── RN-NNN — Analise-Funcional.md
│   └── RN-NNN — Analise-Tecnica.md
├── delivery/<grupo>/RN-NNN — <título> — Feature.md   # Feature + Tasks
├── reverse/CANDIDATES.md           # entidade Candidate (opcional)
│   └── guide/                        # camada humana do legado (gm-legacy-guide): OVERVIEW.md,
│                                   #   IMPACT_MAP.md, GUIDES/<domínio>.md, FORMS/<domínio>.md
│                                   #   (+ LEGACY_CONTEXT.md) — júri no topo, leitura em guia/
└── specs/                          # downstream técnico (DATA_MODEL, API_CONTRACTS, UI_SPEC…)
```

### 2.1 Camadas de conteúdo do GOLMET (DR-021)

O conteúdo instalado pelo kit segue quatro camadas, da mais fraca para a mais
forte: **L0 embedded** (BASE mínima no binário), **L1 FULL** (93 componentes
canônicos embarcados ou, futuramente, um core externo), **L2 packs** (camadas
de organização/comunidade) e **L3 local** (criações e customizações do
operador). A precedência é `local > pack > full > base`.

- `golmet install --full` materializa L1 no config do engine; o kit é o
  escritor único dos artefatos renderizados.
- O manifest registra `layer`, origem e SHA-256 por arquivo; manifests
  anteriores sem esses campos continuam válidos.
- `golmet update --full --engine <id>` aplica overlay com detecção 3-way:
  arquivo local divergente do último SHA é preservado e reportado como
  conflito; nunca é sobrescrito silenciosamente. `--force` é a exceção
  explícita para customizações locais, sem sobrescrever arquivos geridos por
  packs.
- O core externo (`--core`) é aceito pelo CLI e pelas tools MCP; o FULL
  embarcado continua sendo o default offline e atomicamente compatível com o
  renderizador do kit.

> **NOTA (perfis do CLI):** os perfis do `golmet install` são `essencial`
> (L0+BASE na zona `.golmet/` do workspace — kit do repositório, versionado
> com o repo, qualquer engine) e `completo` (L1 FULL no engine com skills
> nativas — hoje só opencode e claude-code, critério derive via `SkillsDir`;
> engines sem skills nativas falham fail-closed nos pontos de entrada). Os
> nomes de **CAMADA** L0–L3 continuam os do DR-021 — perfil é contrato de
> experiência, camada é procedência de conteúdo; `base`/`full` seguem
> aceitos como aliases internos das flags.

- **Grupo de delivery** = subpasta livre (`2026-10`, `core`, …) — o kit não interpreta o nome.
- **Zona do kit (`.golmet/`)**: skills BASE vendored + manifest (DR-008) — os adapters do engine escritos pelo `init` (`.opencode/`, `.mcp.json`, `.cursor/`) apontam para cá; é o que torna o workspace multi-engine sem config global.
- **Variante legada tolerada no `golmet adopt`:** `delivery/<grupo>/<RN-NNN — <título>>/RN-NNN — <título> — Feature.md` (Feature dentro de pasta própria). O `adopt` não move arquivos; o `validate` aceita ambos e recomenda a normalização.
- **Container de organização** (opcional): raiz com `golmet.yaml` (`container: true`), `CONSTITUTION.md`, `INDEX.md` (gerado pelo kit) e workspaces como subpastas.

## 3. `golmet.yaml`

```yaml
format: 1                # DEVE — versão do FORMATO suportada
container: false         # true na raiz de container de organização
products: [MeuApp]        # só quando container: true
packs: []                # [{id, repo, pin}] — overlays da organização (§Pack)
```

### 3.1 Contrato de pack (L2) — arquétipos overlay e state

Um pack compartilhável é um repositório Git com `pack.yaml` na raiz. O campo
`kind` escolhe o arquétipo (default: `overlay` — omitir o campo mantém o
comportamento original):

```yaml
id: community-accessibility
version: 1.0.0
name: Helpers de acessibilidade
kind: overlay            # opcional — overlay é o default
requires-core: ">=0.1"
no-override: true
provides:
  - skills/gm-a11y/SKILL.md
  - agents/gm-a11y-reviewer.md
```

| Semântica | `overlay` (padrão) | `state` |
|---|---|---|
| Conteúdo | arquivos declarados em `provides` | o próprio working copy Git (tudo exceto `pack.yaml`) |
| Cópia local | cache somente-leitura em `~/.golmet/packs/<slug>` | working copy bidirecional em `<config>/packs/<id>` (clone completo) |
| `provides` | obrigatório, restrito a `agents/` e `skills/` | **proibido** (erro — em state o conteúdo é o clone) |
| Materialização | overlay 3-way de `provides` no config do engine | overlay 3-way de `skills/` e `agents/` do working copy (posse do pack substitui o render do core para o mesmo caminho) — nada mais é copiado para o config |
| `knowledge/`, `accumulation/` e `rules/` | - | **não materializadas - lidas IN-PLACE do working copy**; os leitores (gm-domain-research, pattern library, lazy index) apontam para `packs/<id>/...` (ex.: norma da org em `packs/<id>/rules/<assunto>.md` — **uma norma por arquivo, cada uma com gatilho de leitura próprio** no lazy index) |
| Sync (`pack update`) | `pull --ff-only` no cache (cache sujo = erro) | commit do WIP local (`"sync: acumulação local <UTC>"` — o dado nunca se perde) + `fetch` + `rebase origin/<branch>`; conflito = FALHA com remediação (`git rebase --continue/--abort` e rode de novo) |
| Push | — | manual por design: `golmet pack publish <fonte>` (commit do WIP `"publish: <mensagem>"` + `push origin <branch>`) |
| Working copy pré-existente | n/a | **adotado** (`pack add`/`pack update` com `.git` e `origin` igual à fonte — nunca reclone nem sobrescreve; origem divergente = erro claro) |
| Doctor | cache/manifest do config | arquivos do working copy são seguros (versionados no repo do pack); só mudanças não-commitadas contam como risco |

`provides` só pode apontar para `agents/` ou `skills/` dentro do pack; path
	traversal é rejeitado. `golmet pack add <source> --engine <id>` sincroniza o
	repo, valida o contrato e aplica o overlay; `golmet pack update <source>
--engine <id>` repete a sincronização (o `--engine` é obrigatório nos dois).
Arquivos locais divergentes são
preservados e reportados; `no-override: true` impede colisões mesmo quando o
arquivo anterior foi instalado por outra camada. O pack é fixado pelo SHA
resolvido no manifest de conteúdo.

`requires-core` aceita apenas o formato `>=X.Y` (ex.: `">=0.3"`) — ranges
(ex.: `">=X <Y"`) e outros operadores são rejeitados com erro citando o
formato aceito, nos dois arquétipos. O pack central da organização (remoto)
distribui o arquétipo `state` para todos os colaboradores; forks ramificam a
partir dele.

## 4. Gramática de IDs

| Entidade | Forma | Exemplo | Sequência |
|---|---|---|---|
| Charter | (sem ID — arquivo único) | — | — |
| RN | `RN-NNN` | `RN-109` | por workspace |
| PEND | `PEND-NNN` | `PEND-007` | por workspace |
| CR | `CR-NNN` | `CR-013` | por workspace |
| ADR | `ADR-NNN` | `ADR-021` | por workspace |
| IMP | `IMP-NNN` | `IMP-143` | **global do workflow** (nunca por workspace) |
| Candidate | `CAND-NNN` | `CAND-044` | por workspace |
| Task | `RN-NNN-M` | `RN-109-3` | M sequencial dentro da Feature |

- `NNN` = 1–4 dígitos, zero-padding recomendado (3). IDs são **únicos e imutáveis** (H2) — renomear ID quebra o `validate` (hash de seção no índice) e reescreve história.
- Cabeçalho de RN/Feature usa em-dash: `## RN-109 — <título>`.

## 5. Gramática de evidência

Forma canônica **parseável**:

```
[caminho/relativo.ext:NNN @hash]
```

- Regex de reconhecimento: `\[[^\[\]:\n]+:\d{1,6}\s+@[0-9a-f]{7,40}\]`
- `caminho` é relativo à raiz do repo registrado no **charter §1**.
- `hash` (SHA-1, mínimo 7 chars) resolve **por pertencimento** contra o registry de repos do charter §1: o kit consulta cada repo (`git cat-file`) e o hash DEVE existir em exatamente 1 repo — ambiguidade = erro (H7).
- Evidência sem hash (`[arquivo:linha]`) é aceita como **legado** no `adopt`, mas é **AVISO** permanente até refresh.
- O `doctor` compara cada hash vs HEAD do repo e emite **worklist de refresh** (S2). O kit NUNCA re-pin de hash sem o agente (§Quem escreve).

## 6. Estado: diretivas e front matter

Dois mecanismos, escolhidos por granularidade (D2 — sem YAML por seção, sem HTML comments):

| Nível | Mecanismo | Entidades |
|---|---|---|
| **Arquivo inteiro** | YAML front matter `status:` | Charter, Feature |
| **Seção dentro de arquivo** | Diretiva em blockquote `> estado: <valor>` | RN, PEND, CR, ADR, CAND, Task, Spec |

**Gramática da diretiva:**

```
> estado: active                          # estado simples
> estado: superseded → RN-142             # transição com destino (RN/ADR/Spec)
> estado: resolved → CR-013               # resolução de PEND cita o destino
> estado: stale · reason: CR-021          # Spec stale — reason OBRIGATÓRIA (H9)
```

- Diretiva ausente = **estado default** da entidade (§7). O `golmet adopt` escreve as diretivas iniciais.
- Estado **nunca** é mantido pelo agente direto no índice — o agente edita a diretiva na prosa; o kit recompõe o índice.

### 6.1 Subset YAML aceito (normativo — DR-018)

O kit implementa **parser próprio** (zero deps). O FORMATO aceita APENAS:

- **Chaves:** escalares simples (`status`, `anchor`, `name`) — sem aninhamento além de 1 nível
- **Valores:** escalares (`draft`, `true`, `42`, `RN-109`) ou **array inline de escalares em chaves de topo** (`[file, ask, kit.jev]`)
- **Mapas de 1 nível** (`kit:`, `degrades:`): valores **sempre escalares** — sem arrays, sem multiline e sem dois-pontos internos; notação de capacidade com dot (`kit.jev`) é válida como **item de array** em `requires`, nunca como chave
- **Strings:** bare ou entre aspas simples/duplas — **sem multiline** (`|`, `>`) nesta versão
- **Sem** anchors/aliases (`&`, `*`) e sem tags (`!!`) — rejeitados com erro estruturado (§11)
- **Comentários** `#` tolerados em qualquer linha e ignorados
- Front matter delimitado por `---` na primeira linha e na linha de fechamento

Qualquer constructo fora do subset = erro de parse com correção sugerida — o `doctor` reporta (S-level) e o `validate` bloqueia front matters obrigatórios ilegíveis (H5).

## 7. Entidades e máquina de estados

**Estados default (ausência de diretiva/front matter — §6):** `RN=draft · PEND=open · CR=pending · ADR=proposed · CAND=pending · Task=todo · Spec=current` · Charter/Feature não têm default — front matter obrigatório (H5).

### 7.1 Charter — `PROJECT_CHARTER.md`

| | |
|---|---|
| Front matter | `status: draft \| confirmed` (+ `compliance`, `legado`, `sistema_existente` opcionais) |
| Estados | `draft → confirmed` |
| Pré-condição de `confirmed` | §1 registra cada repo do produto: path + branch + **HEAD congelado** |

### 7.2 RN — `REQUIREMENTS.md`, seções `## RN-NNN — <título>`

| | |
|---|---|
| Estados | `draft · active · superseded · dropped` |
| Transições legais | `draft→active` · `active→superseded` (→RN-XXX) · `active→dropped` |
| Pré-condições | `draft→active`: **≥1 evidência parseável** na seção (H4) · `superseded`: cita a RN sucessora · `dropped`: motivo no corpo |
| Invariante | `dropped` **nunca reverte** |

### 7.3 PEND — `REQUIREMENTS.md`, seções `## PEND-NNN — <título>`

| | |
|---|---|
| Estados | `open · deferred · resolved · dropped` |
| Transições legais | `open→resolved` (cita destino RN/CR/ADR) · `open→deferred` · `→dropped` (ADR cita o fechamento) |
| Invariante **hard** | PEND **NUNCA** ancora Feature (H1) — PEND não é requisito; só RN ancora |

### 7.4 CR — `CHANGE_REQUESTS.md`, seções `## CR-NNN — <título>`

| | |
|---|---|
| Estados | `pending · approved · rejected · implemented` |
| Transições legais | `pending→approved` · `pending→rejected` · `approved→implemented` |
| Pré-condição de `approved` | decisão registrada com `{dono, data}` |
| AVISO | `implemented` com Spec ainda `stale` = propagação devedora (S1) |

### 7.5 Feature — `delivery/<grupo>/RN-NNN — <título> — Feature.md`

| | |
|---|---|
| Front matter obrigatório | `status: draft \| ready \| in_progress \| done \| cancelled` · `anchor: RN-NNN` |
| Transições | sequenciais entre vizinhos; `→cancelled` de qualquer não-final |
| Pré-condições | `ready` exige anchor RN **`active`** (H3) · `done` exige todas as Tasks `done` |
| Corpo | `### RN-NNN-M. [repo] <título>` — uma seção por Task; tabela de campos livre |

### 7.6 Task — seção `### RN-NNN-M` dentro da Feature

| | |
|---|---|
| Estados | `todo · in_progress · done · blocked` |
| Transições | **livre** entre estados não-finais |
| Estado | diretiva `> estado:` por seção — BASE funciona **sem tracker externo**; Tracking ID (issue/PR) é metadata opcional para usuários FULL |

### 7.7 Spec — `analysis/RN-NNN — <título>/*.md`

| | |
|---|---|
| Estados | `current · stale` (↔) |
| Escritor | **o kit escreve `stale`** quando CR `implemented` invalida (única diretiva com escritor não-agêntico) |
| Invariante **hard** | `stale` DEVE carregar `reason: CR-XXX` (H9) |

### 7.8 ADR — `DECISIONS.md`, seções `## ADR-NNN — <título>`

| | |
|---|---|
| Estados | `proposed · accepted · superseded` (→ADR-XXX) |
| Invariante | conteúdo de ADR aceito é **append-only** — modificação detectada por hash de seção no índice (H8) |

### 7.9 Candidate — `reverse/CANDIDATES.md`, entradas `### CAND-NNN — <título>`

| | |
|---|---|
| Estados | `pending · approved · rejected · promoted` |
| Pré-condição de transição | registro `{dono, data}` |
| `promoted` | cita a RN que formalizou o candidato |

## 8. Índice de Estados — `golmet.index.json` (D2)

Cache **commitado**, com **único escritor = kit** (padrão registry-render: canônico = prosa, cache = índice):

```jsonc
{
  "format": 1,
  "generated_at": "2026-11-08T12:00:00Z",
  "workspace": "MeuApp",
  "entities": {
    "rn":     { "RN-109": { "estado": "active", "secao_hash": "…", "evidencia": [{"path": "…", "line": 42, "hash": "a1b2c3d", "head_ok": true}] } },
    "feature":{ "RN-109": { "status": "in_progress", "anchor": "RN-109", "tasks": {"RN-109-1": "done"} } }
  },
  "counts": { "rn": 12, "pend_open": 3, "cr_pending": 1, "features_done": 7 }
}
```

- `golmet validate` recomputa e compara; divergência prosa↔índice sem transição registrada = erro (H6).
- Conflito pós-merge = mecânico: `golmet validate --fix` recompõe (conflito de prosa é humano).
- NUNCA editado à mão; NUNCA fonte de verdade de conteúdo.

## 9. Enforcement (D3 — dois níveis)

**Hard — `validate` exit ≠ 0 (bloqueia done/CI):**

| Regra | Enunciado |
|---|---|
| H1 | PEND nunca ancora Feature (`anchor: PEND-XXX` = erro) |
| H2 | IDs únicos e imutáveis (duplicado ou renome de seção com ID = erro) |
| H3 | Feature `ready`/`done` exige anchor RN `active` |
| H4 | RN `active` exige ≥1 evidência parseável |
| H5 | Front matter obrigatório presente e parseável (Charter, Feature, `golmet.yaml`) |
| H6 | Estados no vocabulário da entidade; só transições legais (índice é a história) |
| H7 | Toda evidência DEVE parsear; hash resolve em exatamente 1 repo do charter §1 |
| H8 | ADR aceito é imutável (hash de seção ≠ índice = erro) |
| H9 | Spec `stale` exige `reason: CR-XXX` |
| H10 | Referências cruzadas resolvem — todo `RN/PEND/CR/ADR/CAND/IMP` citado existe |

**Soft — `doctor` reporta (AVISO, não bloqueia):**

| Regra | Enunciado |
|---|---|
| S1 | CR `implemented` com Spec `stale` (propagação devedora — tolerada por prática) |
| S2 | Hash de evidência ≠ HEAD do repo (worklist de refresh) |
| S3 | `requires` de skill não satisfeita pelo engine detectado |
| S4 | Evidência legado sem hash |
| S5 | Capacidade de kit (`kit.jev`/`kit.graph`) declarada e ausente no ambiente — `doctor` emite AVISO com remediação (DR-015/016) |

### 9.1 `golmet lint` — varredura estrutural determinística (read-only)

O lint carrega o estado todo e APONTA inconsistências; a correção é do
agent/operador. Executor: os checks L1-L8 rodam via **CLI `golmet lint`**
(ou **MCP `gm_lint`** — read-only, sem confirm). Skills de manutenção que o
invocam: gm-agent-health-monitor (saúde estrutural), gm-setup (validação
pós-install), gm-workflow-improver (gate pós-correção),
gm-pack-orchestrator (pós publish/update). Exit 0 limpo, 2 com findings.
Sem check de vocabulário legado (L2 fora do escopo por decisão do operador).

| Check | Severidade | Enunciado |
|---|---|---|
| L1 | error | Referência morta: path citado em `` `backticks` ``/link de `*.md` (kit, config, rules) não existe em raiz conhecida |
| L3 | warning | Órfão: arquivo do config sem proveniência no manifest, sem designação pessoal, fora de state pack, fora da allowlist e não citado |
| L4 | error | Entrada stale no manifest de conteúdo (arquivo não existe no disco) |
| L5 | warning | Nome estrutural fora de kebab-case ASCII (dirs/temas canônicos; nomes com extensão isentos) |
| L6 | error/warning | Componente fantasma: `gm-*` citado que não existe no registry — **error** em `coredata/sources` (contrato vivo), **warning** em docs/AGENTS.md (histórico) |
| L7 | error | Estrutura fora do padrão canônico (declarado abaixo) |
| L8 | warning | Conformidade estrutural definitivo ↔ template: o definitivo (AGENTS.md, perfis) perdeu o MÍNIMO estrutural declarado (blocos marcadores/seções/campos). Conteúdo extra = OK (evolução); definitivo ausente = skip; template depositado no config = L7 warning |

**Template é etapa de construção — vive no kit, não se instala; o config
carrega só os definitivos.** O render emite os templates (byte-equality do
golden/docs), mas o materialize NÃO os deposita (`nonDepositTemplates`);
quem precisa do esqueleto usa `golmet template <agents|client-profile|
operator-profile>` (`--out` grava em arquivo). Política anti-ruído do L8:
o mínimo estrutural é DECLARADO no lint (l8*) — se o template ganhar seção
nova numa release, definitivos antigos NÃO flagam (evolução retroativa não
obriga); só a PERDA/ausência do mínimo declarado flaga.

**Estrutura canônica declarada (fonte única em código — `internal/core/lint`):**

- **State pack** (`packs/<id>/`): `pack.yaml` + qualquer subconjunto de
  {`skills/`, `agents/`, `rules/`, `domains/`, `accumulation/`}; dentro de
  `accumulation/` só {`patterns/`, `metrics/`, `nuances/`};
  `accumulation/patterns/<tipo>/<slug>/PATTERN.md` com tipo em
  {process, architecture, code, ui, test, data}.
- **Config engine** (1º nível): dirs {agents, skills, rules, packs,
  plugins, knowledge (legado), .golmet} + files {AGENTS.md, opencode.jsonc}
  — dir desconhecido = warning; template depositado no config
  (AGENTS.template.md) = L7 warning (template vive no kit — M12).
- **Kit repo**: `coredata/{sources/, rules/, templates/, registry.yaml}` com sources
  FLAT (`*.md` + `*.agent.md`); rules canônicas do kit = `rules/*.md` +
  `rules/*.yaml` (context7, templates de perfil client/operator — IMP-103/136,
  registry de boilerplates stack-templates.yaml); `skills/` raiz = exatamente 16
  skills (`skills/<gm-x>/SKILL.md`); golden `testdata/golden-opencode/` espelha o
  render ({agents, skills, rules} + AGENTS.template.md).

**Heurísticas anti-falso-positivo (declaradas):** URLs/âncoras/`<placeholders>`/
wildcards/braces não são candidatos; paths SEM `/` (nome solto) não são
verificados quando não resolvem (família aberta de marcadores de repo de
produto: `angular.json`, `Cargo.toml`, `LEGACY_CONTEXT.md`); paths de store
de workspace de produto (`delivery/`, `specs/`, `accumulation/`, `docs/`,
`evals/`, `scripts/`, `config/`, `~/`, `$ENV`, `.opencode/`, `guide/`,
`howto/`, `migration/`, `quality/`, `ci/`, `.vscode/` e os nomes em
`moduleStoreNames`) são runtime por design — o L1 não opina; blocos de
código marcados `​```exemplo`/`​```example` não são verificados; `*.md` de
packs não são varridos pelo L1 v0.1 (docs de negócio citam paths de
produto por design). Allowlists declarativas e versionadas no lint:
`orphanAllowlist` (plugins/, quality/health, .github/, settings raiz,
AGENTS.md, .gitignore), `ghostAllowlist` (nomes legados/exemplos com
motivo), `refAllowlist` (refs mortas com pendência declarada — ex.:
`AGENTS.template.md`, wave do template). `packs/` do config é excluído do
L3 (proveniência própria do pack). Match `gm-x` seguido de `*`/`{`/`-`/`/`
é prefixo de família em prosa — não é fantasma.

## 10. Contrato de skill (front matter de capacidades)

```yaml
---
name: gm-requirements          # DEVE ser único; prefixo gm- no BASE
description: Uma frase — é o que o roteamento BASE lê.   # escalar — sem multiline
requires: [file, ask, kit.jev] # arrays inline em chaves de TOPO
degrades:                      # mapa 1 nível — valores SEMPRE escalares
  ask: chat-stop               # pergunta no chat e PARA — nunca adivinhar
kit:                           # mapa 1 nível — valores SEMPRE escalares
  jev: heuristic               # degradação de capacidade de kit (dot-key é item de array, não chave)
  checkpoints: post-write validate   # "<momento> <o-quê>" — sem dois-pontos interno
---
```

**Vocabulário de capacidades:** `file · shell · mcp · ask · kit · subagent · parallel · workspace-ops · db-introspect`. — `kit` = presença do binário golmet (checkpoints §10, `golmet update`); provida pelo próprio kit quando instalado no workspace (`.golmet/manifest.json`).

**Capacidades de ambiente providas pelo kit:** `kit.jev` (classificação calibrada — DR-015) · `kit.graph` (grafo de código — DR-016, política **graph-first** — DR-020). Declaradas em `requires`, checadas pelo `doctor` (S5), degradação heurística sempre declarada. **Graph-first (DR-020):** com `kit.graph` presente e índice fresco, navegação estruturada é a PRIMEIRA instância — grep/glob ficam para literais e para o que o grafo não resolve; a disciplina é obrigatória, a infra nunca é pré-condição: índice stale exige re-indexação antes do uso (grafo velho responde com confiança errada — pior que grep); sem grafo, degradação `grep-fallback` declarada no relatório (observável, nunca bloqueante). MCPs de serviço NÃO são capacidades do kit — ficam sob gestão do runtime (DR-016).

**Escadas de degradação:** `ask`: nativo → chat+stop → checklist async · `subagent`: spawn → inline sequencial → desabilitado com recomendação FULL · `parallel`/`background`: sequencial/síncrono.

**Perfis:** `BASE = file + shell (+mcp recomendado, +ask com degradação)` · `FULL = BASE + ask nativo + subagent + parallel + workspace-ops`. O `doctor` compara `requires` vs capacidades do engine detectado (S3).

## 10.1 Duplo registro skill + subagent (mesmo id)

Skill e subagent com o mesmo id **coexistem por necessidade, não por simetria** — a **skill é o contrato** (o procedimento: gatilho, Lê/Gera, formato, modo de erro, portátil a qualquer engine); o **subagent é o executor especializado** (sessão filha isolada, temperatura e permissões próprias, contexto do primary preservado).

**Critério para criar o subagent do par** — existe quando pelo menos um vale:

- **Trabalho denso** que gastaria o contexto do primary (ex.: `gm-pattern-extractor` processando retrospectiva longa);
- **Delegação orquestrada** — um orquestrador abre o executor isolado (ex.: `gm-project-retrospective` → `@gm-pattern-extractor`);
- **Disponibilidade cross-engine** para o mesmo id em qualquer engine (ex.: pares de navegação `gm-next`, `gm-progress`).

**Skill-only** quando o trabalho é **leve, imediato e conversado** (o contexto da sessão dá o valor) — ex.: `gm-nuance-collector` (captura na hora), `gm-rule-keeper` (norma decidida com o operador, com confirmações no fluxo). O par pode ser completado depois, quando um dos critérios passar a valer.

## 11. Erros estruturados

`golmet validate --json` emite array consumível pelo agente no loop:

```jsonc
[{ "file": "REQUIREMENTS.md", "entity": "RN", "id": "RN-042",
   "rule": "H4", "message": "RN active sem evidência parseável",
   "fix": "adicione [caminho:linha @hash] ao corpo da seção" }]
```

## 12. Versionamento

`format: 1` no `golmet.yaml`. Kit versão X suporta formatos `1..X`. Mudança de formato = nova versão + `golmet update` migra (idempotente, preserva prosa). A spec do formato versiona por semântica: campo novo opcional = minor; quebra de parse = major.

## 13. Guarda de escopo — o que NÃO entra

Lifecycle de produto comercial · severidade como estado (fica label) · assignees/prazos · estados de compliance · validação de qualidade de conteúdo (território do clarify). Critério de entrada: só formaliza o que os **três consumidores** precisam — kit (validar), engines (executar transições), ingestores (exportar).

## 14. Exemplo mínimo de workspace conforme

```markdown
<!-- golmet.yaml -->
format: 1

<!-- PROJECT_CHARTER.md -->
---
status: confirmed
---
# Charter — MeuApp
## 1. Repos
| Repo | Branch | HEAD |
|---|---|---|
| ../MeuApp-API | main | a1b2c3d |

<!-- REQUIREMENTS.md (trecho) -->
## RN-001 — Agendamento com confirmação
> estado: active
O sistema DEVE confirmar agendamentos em até 24h `[src/Agenda.cs:88 @a1b2c3d]`.

## PEND-001 — Definir política de no-show
> estado: open
Cliente não definiu a política de cobrança.

<!-- delivery/2026-11/RN-001 — Agendamento — Feature.md -->
---
status: ready
anchor: RN-001
---
### RN-001-1. [BPro-API] Endpoint de confirmação
> estado: done
```
