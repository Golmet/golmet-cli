# AGENTS — Config OpenCode (template versionado — IMP-105)

<!--
TEMPLATE VERSIONADO E NEUTRO (IMP-105). Este arquivo NÃO roda direto: o
`gm-setup` copia este template para `AGENTS.md` (gitignored, local) e
preenche os placeholders `<preenchido pelo gm-setup>` a partir do
`profile/client-profile.md` (IMP-103) e do pack da organização. Nada de
organização/registry/DNS/canal/OS/tracker no template — a base publicada
fica 100% neutra. O kit materializa este arquivo em <config>/AGENTS.template.md
como GERIDO (layer core no manifest de conteúdo) — sincronizado pelo
`golmet install --full`.

Blocos marcadores reconhecidos pelo gm-setup (substituição idempotente):
  <!-- azure-devops --> ... <!-- /azure-devops -->   tracker do cliente
  <!-- org-pack --> ... <!-- /org-pack -->           identidade da org (pack lazy)
  <!-- projeto-padrao --> ... <!-- /projeto-padrao --> diretório de projetos
  <!-- gm-pack-accumulation --> ... <!-- /gm-pack-accumulation -->  destino do acúmulo (pack ativo)
  <!-- knowledge-routing --> ... <!-- /knowledge-routing -->  roteamento T1-T5 (conteúdo padrão)

-->

<!-- azure-devops -->
<preenchido pelo gm-setup a partir do client-profile>
Tracker do cliente: org/projeto/repos-padrão/usuário (IMP-103). Ler
`profile/client-profile.md` ANTES de qualquer operação de tracker; nunca usar
repos de outros setores/produtos como padrão. O perfil é gitignored —
template em `profile/client-profile.template.md`.
<!-- /azure-devops -->

<!-- org-pack -->

<preenchido pelo gm-setup quando o cliente tiver pack de organização>
Organização GitHub: **<org>** — sempre em `gh`/API. Repos:
`<Projeto>-<Camada>` MAIÚSCULAS, privados, branch `main`. Normas da org
vivem no pack lazy, em `packs/<pack>/rules/<assunto>.md` — UMA norma por
arquivo, com gatilho no índice lazy abaixo (IMP-105): identidade/repos
(`org.md`), pipeline de deploy (`deploy.md`), runbook de VPS (`vps.md`),
observabilidade (`observability.md`). Ler a norma ANTES de criar repos,
pipelines ou operar a infra da organização. Escrever/atualizar normas é da
skill `gm-rule-keeper` (T3); sem pack registrado, este bloco fica vazio e o
perfil do cliente responde pelos defaults.

<!-- /org-pack -->

### Primary Agents Golmet (selecionáveis via Tab)

- **`gm-req`:** descoberta de negócio, clarificação, `REQUIREMENTS.md`, escopo, Change Requests (`CR-XXX`).
  - **Triagem de PENDs (IMP-031):** nunca classificar PEND como "dev direto" sem aprovação explícita do usuário. Inferência por omissão é proibida.
- **`gm-doc`:** engenharia reversa multi-repo, `specs/`, fechamento da DAG.
- **`gm-dev`:** implementação de código a partir de tasks aprovadas.
- **`gm-devops`:** infraestrutura e deploy — setup de VPS (Docker, GHCR, SSH, Cloudflare tunnels, systemd, backup) e troubleshooting guiado (skills `gm-vps-setup` / `gm-vps-troubleshoot`).

### Hierarquia de Convenções

AGENTS.md Global → AGENTS.md do Projeto → Primary Agent → Subagent → Skill.
Perguntas "o que fazer agora" / "próxima skill" → skill `gm-next` (não analise o estado você mesmo).

### Tools MCP do kit (servidor `golmet`)

O kit serve 13 tools MCP — **prefira-as à exploração de filesystem para estado do kit**:

- **Evals**: `gm_eval` (roda fixture e grava relatório) · `gm_eval_reports` (**resultado do eval mais recente** — use antes de procurar EVAL_REPORT em pastas)
- **Estado do workspace**: `gm_status` (dashboard RN/PEND/CR) · `gm_next` (próxima ação) · `gm_index` (recompõe índice) · `gm_validate` (FORMAT v1)
- **Diagnóstico**: `gm_doctor` (saúde + estado não-reconstrutível do config, `--engine`) · `gm_lint` (varredura estrutural determinística: refs mortas, órfãos, manifest stale, nomes, componentes fantasma, estrutura canônica, diagramas Mermaid) · `gm_jev_classify`
- **Instalação**: `gm_update` (BASE/FULL no engine, `confirm`) · `gm_upgrade` (**self-update do binário do kit** — check read-only com `check=true`, aplica com `confirm=true`; rollback: `<bin>.old`) · `gm_pack_update` / `gm_pack_list` (packs)

## OUTPUT STYLE: concise

- Bullet points over paragraphs
- Skip filler words and hedging ("I think", "probably", "it seems")
- 1-sentence explanations max, then code/action
- No repeating what the user said

## Regras de Interação e Delegação (IMP-043)

- Perguntas ao usuário: agrupe TODAS as decisões pendentes numa única chamada de `question` (nunca 1 por vez quando houver N pendências). Pergunta de decisão é bem-vinda — com contexto e opções concretas.
- Encerramento de turno (anti-idle, IMP-085): se restar APENAS decisão do operador, emita UMA `question` agrupada e ENCERRE o turno — a resposta do operador é o próximo evento. Nunca emita tool calls de preenchimento/idle (ex.: bash trivial em loop) enquanto aguarda. Responder no chat e parar é ação válida e completa.
- Pedágio ≠ pergunta: o que se minimiza é a espera mecânica de permissão de tool; decisão de negócio/arquitetura nunca é suprimida.
- Delegação de alta fidelidade (R1): todo prompt de `task` carrega insumos (paths), decisões já tomadas, formato de retorno esperado e restrições. Nunca delegue esperando que o subagente adivinhe. Cole só o essencial; o resto o subagente lê.
- Retomada (R2): continuação do mesmo trabalho = resume via `task_id` (memória do subagente retomada); sessão nova só para trabalho novo.
- Paralelismo (R3): tasks independentes em paralelo (uma message, N calls) — nunca escrita concorrente no mesmo arquivo.
- Determinístico (R4): script apenas para lista fechada (evals, registry render, SQL de stats) — julgamento permanece agêntico.

<!-- projeto-padrao -->
<preenchido pelo gm-setup a partir do client-profile>
diretorio_projetos: <raiz-de-projetos>
<!-- /projeto-padrao -->

<!-- gm-pack-accumulation -->
<preenchido pelo gm-setup a partir do client-profile>
pack_ativo: <pack>
accumulation: packs/<pack>/accumulation/
Destino de artefatos de negócio acumulados pelo workflow (patterns/,
metrics/, nuances/) — IMP-061/IMP-111. Skills de negócio NUNCA gravam em
~/.config/opencode/ nem no core.
<!-- /gm-pack-accumulation -->

<!-- knowledge-routing -->

Gatilho (P-01): aplicar este bloco apenas AO decidir onde gravar/consumir conhecimento novo OU ao fundar base de domínio (gm-domain-research). Fora disso, ignore.

### Roteamento de conhecimento novo (decidir na ordem)

| #   | Pergunta                                                 | Destino (tipo)                                                         | Escritor canônico                                                     |
| --- | -------------------------------------------------------- | ---------------------------------------------------------------------- | --------------------------------------------------------------------- |
| 1   | Padrão técnico cross-domínio que fundamenta componentes? | T1 `packs/<pack>/domains/<domínio>/`                                    | gm-domain-research                                                    |
| 2   | Lição/métrica/descoberta de projeto concluído da org?    | T2 `packs/<pack>/accumulation/`                                         | gm-pattern-extractor · gm-metrics-collector · gm-nuance-collector · gm-reverse-orchestrator |
| 3   | Config/contrato com gatilho de leitura claro?            | T3 `packs/<pack>/rules/*.md` + índice lazy deste AGENTS.md              | gm-rule-keeper (pack) · gm-setup (pessoal) · packs                    |
| 4   | Verdade sobre um produto específico?                     | T4 `<REPO_DOCS>/<Produto>/` (specs/, support/, glossaries/, quality/)   | skills Golmet do módulo                                               |
| 5   | Sobre o próprio workflow?                                | T5 registry.yaml · ARCHITECTURE.md · WORKFLOW_IMPROVEMENTS.md · evals/  | builder · improver · eval                                             |

- Nuance da org sobre domínio genérico → T2 `packs/<pack>/accumulation/nuances/<dominio>-nuances.md` (working copy do state pack; sobe via publicação manual).
- Caso ambíguo ou destino não óbvio ("salva isso aqui") → `gm-knowledge-route` (julgamento de encaixe; delega ao `@gm-knowledge-router` quando orquestrado).

### ROTA, NÃO REPLIQUE (P-04/AP-01)

Este bloco é roteiro curto — cada tipo tem SEU índice: `packs/<pack>/domains/INDEX.md` (T1) · `packs/<pack>/accumulation/patterns/INDEX.md` (T2) · índice lazy de profile/ (T3, abaixo) · `INDEX.md` do módulo (T4 — IMP-011) · registry.yaml (T5). NUNCA criar meta-índice central nem copiar conteúdo entre stores.

<!-- /knowledge-routing -->

## Índice de Referências — lazy loading

CRITICAL: os arquivos abaixo NÃO são carregados automaticamente. Leia com o Read tool APENAS quando a tarefa atual exigir — nunca preventivamente. Quando lidos, trate como instruções obrigatórias que sobrescrevem defaults.

- `profile/context7.md` — ANTES de responder sobre library/framework/SDK/CLI/cloud (MCP context7; não confie na memória). Infraestrutura do server: skill `gm-mcp-context7-setup`.
- `packs/<pack>/rules/org.md` — norma da organização (identidade/repos/clientes — IMP-105) — AO criar repos/projetos na org ou editar o repo de docs do produto.
- `packs/<pack>/rules/deploy.md` — pipeline de deploy validado (deploy.yml/secrets/deploy.sh) — AO criar/editar pipeline de deploy ou diagnosticar falha de deploy.
- `packs/<pack>/rules/vps.md` — runbook de VPS (Docker/GHCR/tunnel/backup) — AO configurar/operar VPS.
- `packs/<pack>/rules/observability.md` — observabilidade mínima (health/watchdog/alertas) — AO configurar saúde de serviços.
- `profile/stack-templates.yaml` — Registry de boilerplates Golmet por stack (.NET, React, Angular, Rust, Java). A org vem de `boilerplate_org` no client-profile. Carregado pelo gm-stack-boilerplate antes de scaffold de novo projeto; editar para adicionar/atualizar templates e recomendar o PR no .config do workflow.
- `profile/client-profile.md` — ANTES de qualquer operação Azure DevOps/tracker (org, projeto, repos e usuário do cliente; IMP-103). Gravado pelo gm-setup; nunca versionado — template em `profile/client-profile.template.md`.
- `profile/operator-profile.md` — ANTES de registrar responsabilidade/autoria (triagem de PENDs no gm-clarify, campo Decisor no gm-decision-log, campo Autor de IMP — IMP-136). Gravado pelo gm-setup; nunca versionado — template em `profile/operator-profile.template.md`.
- `packs/<pack>/accumulation/patterns/INDEX.md` (bloco `gm-pack-accumulation` acima) — AO levantar/desenhar/implementar feature de domínio já tratado em outro produto ou cenário recorrente (pattern library: process/architecture/code/ui/test/data).
- bloco `<!-- knowledge-routing -->` (neste arquivo) + `packs/<pack>/domains/INDEX.md` — AO decidir onde gravar/consumir conhecimento novo ou ao fundar base de domínio (gm-domain-research).
- skill `gm-knowledge-auditor` — AO pedir auditoria de consumo de conhecimento, hit rate ou custo de tokens de conhecimento.
