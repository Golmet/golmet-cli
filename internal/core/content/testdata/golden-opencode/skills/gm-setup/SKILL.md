---
name: gm-setup
description: >-
   Wizard de primeira configuração do workflow numa máquina nova — idempotente e executado por demanda. Pergunta o diretório padrão de projetos, coleta os defaults do cliente (org Azure, projeto, repos, usuário) e grava `rules/client-profile.md` (IMP-103), coleta a identidade do operador (nome, e-mail, papel, contas git/GitHub) e grava `rules/operator-profile.md` (IMP-136), cria/repara o AGENTS.md local A PARTIR do `AGENTS.template.md` versionado (neutro — IMP-105), repõe o índice lazy de `rules/` (IMP-048), aplica os overrides de permissão dos subagents de dev (gm-dev-backend, gm-dev-frontend) no opencode.jsonc, instala os packs declarados no perfil (`packs:` — clone em `packs/<id>/`, instala `provides`, grava `sources.yaml`), delega para gm-azure-setup, gm-gh-setup, gm-aws-setup e gm-jev-setup (opcionais) e, quando presente na instalação (skill de pack opcional — IMP-105), gm-pia-connect-setup; valida o ambiente conforme o **perfil de adoção** — **core-only** (só o workflow: sem tracker/infra, valida só gh/git) ou **completo** (tracker Azure + infra + packs + validações de stack). Use quando o usuário pedir "configurar o workflow", "setup do workflow", "primeira configuração", "onboarding do workflow" ou ao migrar de máquina.
requires: [file, edit, glob, grep, shell, ask]
---

# gm-setup — Wizard de Primeira Configuração do workflow

Onboarding de máquina nova: configura os artefatos **locais** (que não
versionados) com os paths desta máquina e valida o ambiente. Idempotente
— pode ser reexecutado; passos já completos são pulados.

## Contexto do problema que resolve

Os componentes do kit (`agents/gm-*.md`, `skills/gm-*/SKILL.md`) são
CACHE reconstrutível (materializados pelo `golmet install --profile
completo`) — o que varia por máquina vive em dois artefatos locais:

| Artefato | O que contém | Versionado? |
|----------|-------------|-------------|
| `AGENTS.md` (`~/.config/opencode/`) | blocos `<!-- azure-devops -->`, `<!-- org-pack -->` (quando o cliente tiver pack — IMP-105), `<!-- projeto-padrao -->`, `<!-- gm-pack-accumulation -->` + Índice de Referências lazy (`rules/`, IMP-048) | ❌ gitignored — nasce do `AGENTS.template.md` versionado (IMP-105) |
| `opencode.jsonc` (`~/.config/opencode/`) | `permission.external_directory` + `agent.gm-dev-*.permission` (overrides de path) | ❌ gitignored |
| `rules/client-profile.md` (`rules/`) | defaults do cliente (org Azure, projeto, repos, usuário, diretório de projetos — IMP-103) | ❌ gitignore explícito (dado de cliente nunca versionado) |
| `rules/operator-profile.md` (`rules/`) | identidade do operador desta máquina (nome, e-mail, papel, contas git/GitHub — IMP-136) | ❌ gitignore explícito (dado pessoal nunca versionado) |
| `.secrets/` (`~/.config/opencode/.secrets/`) | credenciais (APM Keys, PATs, tokens) referenciadas via `{file:...}` | ❌ gitignored |

Este wizard preenche/repõe esses artefatos e valida o ambiente. O template
versionado do AGENTS.md é `AGENTS.template.md` (neutro, IMP-105) — o local é
gerado dele + client-profile; nunca invente conteúdo fora dos blocos marcadores.

## Lê (upstream)

- `~/.config/opencode/AGENTS.md` — blocos existentes (azure, projeto-padrao, repositorios) e índice lazy
- `~/.config/opencode/rules/*.md` — arquivos de referência lazy existentes (IMP-048)
- `rules/client-profile.md` / `rules/client-profile.template.md` — perfil do cliente (IMP-103)
- `rules/operator-profile.md` / `rules/operator-profile.template.md` — perfil do operador (IMP-136)
- `~/.config/opencode/opencode.jsonc` — permission e agent overrides existentes
- `~/.config/opencode/skills/gm-azure-setup/SKILL.md` — prerequisites de Azure
- `<raiz-de-projetos>/REPOSITORIES.md` (ou o path do bloco `<!-- repositorios -->`) — inventário de repos para validação de clones — OPCIONAL: só no perfil **completo**; core-only não lê nem valida

**Pessoal-por-design (obrigatório ao final do setup):** registre em `<config-root>/.golmet/personal.json` (`{"version": 1, "paths": [...], "updated_at": RFC3339}`) todos os arquivos de estado pessoal criados/editados nesta execução — `AGENTS.md`, `rules/client-profile.md`, `rules/operator-profile.md`, `opencode.jsonc`, `cli.json`, `service.json` — para que o `golmet doctor --engine <id>` os classifique como **pessoal-por-design (isento)** em vez de "não-versionado em risco" (IMP-103/136: dado de cliente/operador nunca versionado). Path exato de arquivo ou terminado em `/` para subárvore.

- Catálogo de tools da sessão — presença da tool `jev_classify` (runtime Jev provido pelo harness da sessão; Passo 5d delega ao `gm-jev-setup`)
## Gera (downstream)

- Edição em `~/.config/opencode/AGENTS.md` — bloco `<!-- projeto-padrao -->` e Índice de Referências lazy (merge, idempotente)
- Substituição no AGENTS.md gerado: o placeholder `rules/<pack-da-organizacao>.md` (índice lazy e bloco `<!-- org-pack -->`) é trocado pelo valor "Regra da organização" do perfil (ex.: `packs/<id>/rules/<assunto>.md` — leitura IN-PLACE do working copy do state pack, **uma norma por arquivo com gatilho próprio**; o kit NÃO copia a rule para o config); sem pack, a linha do índice é removida (IMP-115)
- Edição em `~/.config/opencode/opencode.jsonc` — `permission.external_directory` + `agent.gm-dev-backend/frontend.permission` com os paths da máquina
- Relatório textual de validação de ambiente (não persiste arquivo)
- Criação/atualização de `~/.config/opencode/rules/client-profile.md` — defaults do cliente (gitignored; IMP-103)
- Criação/atualização de `~/.config/opencode/rules/operator-profile.md` — identidade do operador (gitignored; IMP-136)

## Nunca edita

- Agentes e skills gerados pelo kit (`agents/gm-*.md`, `skills/gm-*/SKILL.md`) - são CACHE reconstrutível; reparo via `golmet install --profile completo` (o kit nunca edita à mão o que o kit materializa)
- Blocos `provider`, `model`, `mcp` do opencode.jsonc — só `permission` e `agent`
- Artefatos de módulo workflow (REQUIREMENTS, TASKS, etc.)
- Remote — este wizard nunca commita nem push

## Tools

- `read` (AGENTS.md, opencode.jsonc, REPOSITORIES.md)
- `glob` (detectar módulos/repos no filesystem)
- `question` (perguntar paths e opções — 1 pergunta por bloco)
- `edit`/`write` (editar os artefatos locais — AGENTS.md, opencode.jsonc e perfis de `rules/`)
  gm-pia-connect-setup somente se existir na instalação — skill de pack opcional, IMP-105)
- `bash` (validação de ambiente conforme o perfil de adoção: core-only valida só gh/git; completo valida Test-Path, java -version, mvn -version, node -v)
- `skill` (delegar gm-azure-setup / gm-gh-setup / gm-aws-setup / gm-jev-setup; gm-pia-connect-setup somente se existir na instalação — skill de pack opcional, IMP-105)

## Modo de erro

`warn` — validações de ambiente que falharem não bloqueiam a conclusão:
 entram no relatório como ⚠️ com orientação de correção. Só é `strict`
 na escrita do opencode.jsonc (JSONC inválido quebra o OpenCode — validade
 o resultado com parse antes de encerrar).

## Formato de commit

Nenhum — este wizard edita apenas artefatos **locais** (gitignored) e
nunca commita.

## Algoritmo

### Fluxo síncrono (obrigatório)

O wizard executa os passos em **ordem sequencial**, esperando a conclusão
de cada um antes de prosseguir. O usuário pode **pular** qualquer passo
opcional, mas nunca executar dois passos em paralelo.

```
Passo 0 → Passo 1 → Passo 2 → Passo 2b (perfil do cliente) → Passo 2c (perfil do operador) → Passo 3 → Passo 4 (Azure) → Passo 5 (PIA) → Passo 5b (gh) → Passo 5c (AWS) → Passo 5d (Jev) → Passo 6 → Passo 6b (Packs) → Passo 6c (hardening) → Passo 7
    ↓          ↓           ↓           ↓                ↓              ↓
 [define]   [pular]     [pular]     [pular]          [pular]        [pular]
```

### Passo 0 — Perfil de adoção (primeira pergunta — IMP-043, agrupada com o Passo 1)

Uma única `question`: **"Qual o perfil de adoção?"** com duas opções:

- **Core-only** (sugerido como default quando o opencode.jsonc não tem o
  MCP `azure-devops` habilitado e o usuário não mencionou tracker): só o
  workflow. Sem tracker Azure, sem validações de stack de dev (mvn/ng/JDK
  viram informativas no relatório), sem inventário de repos/REPOSITORIES.md.
  Perfil gravado com campos de tracker/infra como
  `<não configurado — core-only>`. **Passos pulados**: 4 (Azure), 5 (PIA) e
  validações de stack do Passo 7 — marcados "não aplicável" no relatório.
  **Passos mantidos**: 1, 2, 2b (apenas usuário + diretório + org_github),
  2c (perfil do operador), 3, 5b (gh, opcional), 5c (AWS, opcional), 5d (Jev, opcional), 6, 6b (Packs), 6c (hardening), 7.
- **Completo** — fluxo integral (tracker Azure, infra, packs, validações).

O modo é registrado no relatório final e no perfil (campo `modo` no
client-profile) — reexecutar o wizard permite trocar.

### Passo 1 — Detectar estado atual

Leia AGENTS.md e opencode.jsonc e monte o diagnóstico:

| Item | Fonte | Estado esperado |
|------|-------|----------------|
| Bloco `<!-- azure-devops -->` | AGENTS.md | org, project, team, iteration preenchidos |
| Bloco `<!-- projeto-padrao -->` | AGENTS.md | diretório de projetos informado |
| Bloco `<!-- repositorios -->` | AGENTS.md | caminho do REPOSITORIES.md |
| Índice de Referências (lazy) | AGENTS.md | seção presente, 1 linha por `rules/*.md` (IMP-048) |
| Arquivos `rules/*.md` | `~/.config/opencode/rules/` | cada arquivo referenciado no índice existe |
| `permission.external_directory` | opencode.jsonc | inclui o diretório de projetos |
| `agent.gm-dev-backend` / `agent.gm-dev-frontend` | opencode.jsonc | overrides de edit/write com o path |

Exiba o diagnóstico antes de alterar qualquer coisa.

### Passo 2 — Perguntar o diretório de projetos

Via `question`, uma única pergunta: "Qual o diretório padrão onde ficam
os seus projetos (ex: `<raiz-de-projetos>`)?"

- Se `<!-- projeto-padrao -->` já existe, ofereça o valor atual como default
- Valide: se o diretório não existe no filesystem, confirme (pode ser
  intencional — máquina nova) e marque como ⚠️ no relatório

### Passo 2b — Perfil do cliente (rules/client-profile.md — IMP-103)

Coleta os defaults do cliente desta máquina e grava em
`~/.config/opencode/rules/client-profile.md`, a partir do template
versionado `rules/client-profile.template.md`. O perfil é a FONTE DE
VERDADE do cliente para o workflow — skills de tracker (Azure DevOps,
GitHub) leem o perfil em vez de hardcode.

1. Via `question` (pergunta consolidada), colete:
   - organização Azure DevOps e projeto padrão — em **core-only**, grave
     `<não configurado — core-only>` e pule estes campos
   - usuário (nome + e-mail)
   - repos padrão do setor (ex.: prefixos `PROD.*`) — em **core-only**, pule
   - diretório de projetos (se ainda não coletado no Passo 2)
   - organização GitHub (org_github) — usada por packs e gh-setup
2. Grave `rules/client-profile.md` preenchendo o template. O arquivo é
   gitignored — NUNCA versionar dado de cliente; se o `.gitignore` não
   contiver `rules/client-profile.md`, adicione a linha.
3. O wizard do Azure (Passo 4) e qualquer tracker passam a validar contra
   o perfil — nunca contra valores hardcoded.
4. Reexecutável: se o perfil já existe, mostre os valores e pergunte se
   deseja atualizar (stale = atualizar).

### Passo 2c — Perfil do operador (rules/operator-profile.md — IMP-136)

Coleta a identidade do operador DESTA máquina e grava em
`~/.config/opencode/rules/operator-profile.md`, a partir do template
versionado `rules/operator-profile.template.md`. O perfil é a FONTE DE
VERDADE do operador: gm-clarify (triagem de PENDs — campo Responsável),
gm-decision-log (campo Decisor) e o registro de IMPs (campo Autor) leem o
perfil em vez de perguntar a cada sessão.

1. Via `question` (pergunta consolidada), colete:
   - nome e e-mail do operador
   - papel (ex.: arquiteto, dev, PO)
   - identidade git de commit (user.name / user.email — sugira os valores
     atuais de `git config` como default)
   - conta GitHub (username — sugira o login ativo de `gh auth status` como
     default)
2. Grave `rules/operator-profile.md` preenchendo o template. O arquivo é
   gitignored — NUNCA versionar dado pessoal; se o `.gitignore` não
   contiver `rules/operator-profile.md`, adicione a linha.
3. Reexecutável (idempotente): se o perfil já existe, mostre os valores e
   pergunte se deseja atualizar (stale = atualizar). 1 operador por
   máquina — trocar de operador = editar o arquivo.

### Passo 3 — Aplicar config local

1. **AGENTS.md (a partir do template — IMP-105):**
   - Se `AGENTS.md` NÃO existir (máquina nova ou pós-`self-update` que
     removeu o arquivo do versionamento): copie `AGENTS.template.md`
     (versionado, neutro) para `AGENTS.md` e preencha os placeholders
     `<preenchido pelo gm-setup>` com os valores do
     `rules/client-profile.md` (tracker, diretório de projetos,
     pack do acúmulo).
   - Bloco `<!-- org-pack -->`: preencha SOMENTE se o perfil do cliente
     tiver pack de organização (ex.: Acme → `rules/<pack-da-organizacao>.md`);
     cliente sem pack → mantenha o placeholder do template.
   - Se `AGENTS.md` já existe: apenas sincronize os blocos marcadores
     (`<!-- projeto-padrao -->`, `<!-- gm-pack-accumulation -->`, azure/org-pack)
     com o perfil (se já existe com valor igual, pule — idempotente; com
     valor diferente, confirme a troca antes).
2. **AGENTS.md — índice lazy (IMP-048):** se a seção "Índice de
   Referências — lazy loading" estiver ausente (máquina antiga), repõe o
   padrão: instrução CRITICAL (ler só sob demanda) + 1 linha por
   `rules/*.md`. Se algum `rules/*.md` referenciado não existir,
   oriente a ressincronização do estado versionado (`golmet pack update`
   no pack que fornece o rules/, ou nova instalação do kit) e registre ⚠️ —
   nunca crie conteúdo de rules/ por conta própria.
3. **opencode.jsonc**:
   - `permission.external_directory["<dir>/**"] = "allow"`
   - `agent.gm-dev-backend.permission = { edit: { "<dir>/**": "allow", "*": "ask" }, write: { ... } }`
   - `agent.gm-dev-frontend.permission = { ... idem ... }`
   - **Preserve** todas as chaves existentes (provider, model, mcp, hooks).
     Edite por inserção cirúrgica — nunca reescreva o arquivo inteiro.
   - Após editar, valide que o JSONC continua parseável.

### Passo 4 — Azure (síncrono, opcional)

Se o bloco `<!-- azure-devops -->` estiver ausente ou incompleto →
pergunte: "Deseja configurar o Azure DevOps agora?"

- **Sim** → carregue `gm-azure-setup` e **aguarde** a conclusão
  completa antes de prosseguir para o Passo 5.
- **Pular** → registre como pendência no relatório e siga para o Passo 5.

### Passo 5 — PIA (síncrono, opcional)

Se o provider `pia-broker` não existir no opencode.jsonc → pergunte:
"Deseja configurar o PIA agora?"

- **Sim** → verifique se a skill `gm-pia-connect-setup` existe nesta
  instalação (é skill de pack opcional — IMP-105, pode não estar presente).
  Existe → carregue-a e **aguarde** a conclusão completa antes de prosseguir
  para o Passo 5b. Não existe → registre ⚠️ pendência no relatório (padrão
  do `@gm-onboarding-orchestrator` — IMP-080) e siga para o Passo 5b.
- **Pular** → registre como pendência opcional no relatório e siga para
  o Passo 5b.

### Passo 5b — GitHub CLI / gh (síncrono, opcional)

Se `gh auth status` falhar (ou `gh --version` ausente) → pergunte:
"Deseja configurar a GitHub CLI (gh) agora?"

- **Sim** → carregue `gm-gh-setup` e **aguarde** a conclusão completa
  antes de prosseguir para o Passo 6.
- **Pular** → registre como pendência no relatório e siga para o Passo 6.

### Passo 5c — AWS (síncrono, opcional)

Se a AWS CLI estiver ausente (`aws --version` falhar — fallback de path
padrão `C:\Program Files\Amazon\AWSCLIV2\aws.exe`) ou nenhum perfil
estiver configurado (`aws configure list-profiles` sem perfis) → pergunte:
"Deseja configurar o acesso AWS agora?"

- **Sim** → carregue `gm-aws-setup` e **aguarde** a conclusão completa
  antes de prosseguir para o Passo 6.
- **Pular** → registre como pendência opcional no relatório e siga para
  o Passo 6.

### Passo 5d — Jev (síncrono, opcional)

Se a tool `jev_classify` estiver AUSENTE na sessão (o executor confirma no
catálogo de tools da sessão — o runtime é provido pelo harness, não há
binário nem config em arquivo) → pergunte: "Deseja configurar/verificar o
Jev agora?"

- **Sim** → carregue `gm-jev-setup` e **aguarde** a conclusão completa
  antes de prosseguir para o Passo 6.
- **Pular** → registre como pendência opcional no relatório e siga para o
  Passo 6.

Presente → passe direto sem perguntar (idempotente, padrão dos irmãos).

### Passo 6 — Validar ambiente

Use `bash` (read-only) para verificar e reporte em tabela:

| Verificação | Comando | O que confirma |
|-------------|---------|----------------|
| REPOSITORIES.md acessível | `Test-Path` no path do bloco | inventário disponível |
| Repos clonados | `Test-Path` em cada repo do REPOSITORIES.md | desenvolvimento possível |
| Git | `git --version` | rastreabilidade |
| JDK | `java -version` | repos Java/Quarkus |
| Maven | `mvn -version` (senão `mvnw` no repo) | build backend |
| Node | `node -v` | repos frontend |
| Angular CLI | `npx ng version` (opcional) | MFE Angular |
| GitHub CLI | `gh --version` + `gh auth status` | pré-condição de `gm-gh-issue-creator`/`gm-gh-pr-creator` (issues/PRs) |
| Jev runtime | tool `jev_classify` disponível na sessão (executor confirma; mecanismo: plugin `plugins/jev-tool.js` + env `TYPESAFE_API_KEY` — detalhes no `gm-jev-setup`) | camada semântica do workflow funcional (`gm-jev-classifier`, calibração em `gm-decision-audit`/`gm-quality-gates`) |

Repos ausentes: **não clone automaticamente** — liste os comandos
`git clone` sugeridos no relatório para o usuário decidir.

Jev ausente: **não bloqueia o setup** — registre como pendência no relatório
("runtime Jev indisponível — skills de calibração semântica degradam para
opinião do modelo; remediação via gm-jev-setup — runtime provisto pelo
ambiente de sessão, verificar com o provedor do ambiente/harness da
organização") e siga; as skills afetadas já possuem fallback sem Jev.

### Passo 6b — Packs (opcional)

Se `rules/client-profile.md` declarar `packs:` (lista `<id>@<faixa>`),
instale cada pack. Sem `packs:` no perfil → pule silenciosamente. Falha de
um pack **não bloqueia os demais** (registre pendência — padrão do Passo
PIA). Contrato completo em `PACK_SCHEMA.md`; compatibilidade em
`BREAKING-CHANGES.md`.

Para cada pack:

1. **Clone:** `git clone <repo-do-pack> packs/<id>/` no runtime
   (`~/.config/opencode/`; `packs/` é gitignored pelo core). Já existe →
   `git -C packs/<id> fetch` e prossiga.
2. **`requires-core`:** leia o `pack.yaml` do clone e valide a faixa contra
   a versão local do core (campo `version` do `registry.yaml`). Mismatch →
   leia `BREAKING-CHANGES.md`, registre ⚠️ com a orientação de migração e
   pule o pack.
3. **no-override:** nenhum path de `provides` pode colidir com path do core
   — colisão → NÃO instale nada deste pack e registre a pendência.
4. **Instalação:** copie os paths de `provides` do clone para o runtime
   (por cópia; `accumulation/` NUNCA é instalado) e gere/atualize a seção no
   `.gitignore` local para os paths instalados.
5. **Lockfile:** grave/atualize o registro do pack no `golmet.yaml` com pin de commit exato
   (pack → version, commit, paths[]).

### Passo 6c — Hardening (opcional): pre-push hook de neutralidade

Instala o hook de neutralidade no core (bloqueia push de dado de cliente):

1. Verifique hooks existentes ANTES: `git config core.hooksPath` e
   `.git/hooks/` — se já houver hooks configurados, AVISE que a instalação
   **substituirá** os hooks existentes e só prossiga com confirmação.
2. Confirme que `evals/neutrality/git-hooks` existe nesta instalação.
   Não existe → pule com ⚠️ (padrão pack opcional — IMP-110).
3. Instale: `git config core.hooksPath evals/neutrality/git-hooks`.

### Passo 7 — Relatório final

```text
═══ gm-setup — <RESULTADO> ═══

✅ Config local aplicada:
   AGENTS.md: projeto-padrao = <dir> | índice lazy rules/ = <N> refs OK
   opencode.jsonc: external_directory + overrides gm-dev-backend/frontend
   Perfis: rules/client-profile.md (cliente — IMP-103) | rules/operator-profile.md (operador — IMP-136)

✅ Azure: completo | ⚠️ Configure via gm-azure-setup
✅ PIA: configurado | ℹ️ Pendência opcional: gm-pia-connect-setup
✅ GitHub (gh): autenticado | ⚠️ Pendência: gm-gh-setup
✅ AWS: configurado | ℹ️ Pendência opcional: gm-aws-setup
✅ Jev: runtime presente (smoke test ok) | ℹ️ Pendência opcional: gm-jev-setup
✅ Packs: <N> instalado(s), pin em `golmet.yaml` | ℹ️ Sem `packs:` no perfil
✅ Hardening: hooks de neutralidade ativos | ⚠️ Pendência opcional: pre-push hook (IMP-110)

🔍 Ambiente:
| Verificação | Estado | Observação |

⚠️ Reinicie o OpenCode para a config nova valer (permission/agent não são hot-reloaded).
Próximo passo sugerido: @gm-next no módulo em que vai trabalhar.
```

## Casos de Borda

- **Re-execução (idempotente):** itens já configurados com valor igual
  são pulados; valores diferentes pedem confirmação antes de trocar.
- **JSONC com comentários:** preserve os comentários existentes; valide
  parse após a edição (JSONC tolera `//` e blocos `/* */`).
- **Diretório de projetos diferente do exemplo:** nenhum path é fixo —
  o wizard sempre usa o valor informado; o `<!-- repositorios -->` pode
  apontar para outro clone do core (em qualquer localização).
- **Bloco azure completo mas stale (org mudou):** o wizard não adivinha —
  peça confirmação e então delegue o re-setup.
- **`gh` ausente ou desautenticada:** não bloqueia o setup — registre
  pendência `gm-gh-setup` no relatório; sem ela, `gm-gh-issue-creator`/
  `gm-gh-pr-creator` abortam na pré-condição.
- **AWS CLI ausente ou sem perfil configurado:** não bloqueia o setup —
  registre pendência `gm-aws-setup` no relatório; sem ela, usos AWS
  (infra/auditoria) orientam o wizard dedicado.
- **Tool `jev_classify` ausente na sessão:** não bloqueia o setup — registre pendência `gm-jev-setup` no relatório; sem Jev, `gm-decision-audit`/`gm-quality-gates`/`gm-prompt-gate` degradam para fallback (opinião do modelo); a remediação é do provedor do ambiente/harness, não local.
- **Sem repos clonados:** normal em máquina nova; reporte os comandos
  `git clone` do REPOSITORIES.md e marque ⚠️.
- **opencode.jsonc malformado (parse quebra após edição):** reverta a
  última edição e informe — nunca deixe a config num estado inválido.
- **Índice lazy ausente ou `rules/*.md` faltando (máquina antiga):**
  repõe o índice (Passo 3.2); arquivo `rules/` ausente é
  resolvido ressincronizando o estado versionado (`golmet pack update` ou
  nova instalação do kit) - o wizard não inventa conteúdo de regras.
- **Perfil do cliente ausente ou stale (IMP-103):** reexecute o Passo 2b antes do Passo 4 (Azure) — nunca rode tracker com defaults hardcoded.
- **Perfil do operador ausente (IMP-136):** skills consumidoras (gm-clarify, gm-decision-log, registro de IMPs) caem no fallback manual — pedem os dados ao operador na hora; reexecute o Passo 2c para eliminar o fallback.
- **Blocos `azure-devops` no AGENTS.md (legado):** continuam suportados por compatibilidade, mas a fonte de verdade do wizard passa a ser `rules/client-profile.md` (IMP-103); sincronize o bloco a partir do perfil quando divergirem.

## Atualização do kit (self-update)

O binário `golmet` tem self-update próprio e a tool MCP `gm_upgrade` o
expõe aos engines. **Gatilhos explícitos:** o operador pedir "atualizar o
Golmet CLI", "tem nova atualização do Golmet CLI?", "upgrade do kit" ou
relatar comportamento que pareça corrigido numa versão nova.

Fluxo orientado (a tool cobre o determinístico; esta skill orienta o
julgamento sobre quando checar, quando pular versão e como reagir a
erro de checksum):

1. **Cheque read-only primeiro:** `golmet upgrade --check` (ou MCP
   `gm_upgrade` com `check=true`) e **reporte** versão local vs latest ao
   operador (`{current, latest, has_update}`) — nunca baixe nada nesta
   etapa. Versão local `dev` (build local) não é semver: a tool reporta
   update disponível — confirme com o operador antes de aplicar.
2. **Só aplique com confirmação explícita do operador** — o MCP exige
   `confirm=true` e o CLI aplica direto: lembre que a substituição troca
   o binário em execução (o anterior fica em `<bin>.old`).
3. **Após aplicar, re-materialize os geridos:** `golmet install --engine
   <id> --profile completo --yes` (ou MCP `gm_update` com `confirm=true`)
   para o conteúdo (agents/skills/registry) seguir o binário novo — o
   config é CACHE do kit, o update de binário não re-materializa sozinho.
4. **Informe o rollback:** o binário anterior fica em `<bin>.old` — o
   operador restaura renomeando de volta. Erro de checksum SHA-256
   (ausente/divergente) ABORTA sem tocar no binário — reporte o erro e
   sugira repetir mais tarde ou instalar manualmente da release; nunca
   force a aplicação sem checksum válido.
