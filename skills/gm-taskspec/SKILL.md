---
name: gm-taskspec
description: Gera as duas camadas da feature no workspace — analysis/RN-NNN — <título>/ (analise funcional + técnica pura) e delivery/<grupo>/RN-NNN — <título> — Feature.md com tasks RN-NNN-M — para revisão antes da execução.
requires: [file, ask, kit]
degrades:
  ask: chat-stop
kit:
  checkpoints: post-write validate
---
# gm-taskspec — Gera a análise e o arquivo de feature para revisão antes da execução

Gera, numa passada, AS DUAS CAMADAS da feature no diretório do workspace
(conforme FORMATO §2):

1. **Análise** — pasta `analysis/RN-NNN — <título da feature>/` contendo SEMPRE exatamente
   2 arquivos: `RN-NNN — Analise-Funcional.md` (o que a feature precisa fazer — linguagem de
   negócio, sem tecnicidade) e `RN-NNN — Analise-Tecnica.md` (spec técnica PURA da feature —
   contexto, mapa RN→repositórios/endpoints/telas, decisões, contratos e tabelas, riscos —
   SEM issue principal, SEM tasks);
2. **Delivery** — arquivo DIRETO `delivery/<grupo>/RN-NNN — <título da feature> —
   Feature.md` (SEM subpasta por feature) no canônico: Issue principal, Resumo das
   tasks, Ordem de Execução em waves e tasks `RN-NNN-M`, com a linha de proveniência
   OBRIGATÓRIA `> Análise: ../../analysis/RN-NNN — <título>/` linkando a pasta da análise.

**Regra de ouro:** dependência UNIDIRECIONAL tasks→análise — o Feature.md linka a
análise e carrega o campo **RNs**; a análise NUNCA cita tasks, grupos de delivery ou
pastas de entrega. Reorganizar, mover ou apagar tasks nunca invalida a documentação.

O grupo de pasta sob `delivery/` (`<grupo>`) é subpasta livre — o kit não interpreta o
nome (FORMATO §2: `2026-10`, `core`, …); default de escrita quando não decidido:
`YYYY-MM` (mês corrente). O usuário revisa e edita os arquivos; se o perfil tiver
tracking externo (recurso FULL), o fluxo de issues do tracker cria as tarefas lá —
no perfil BASE, o Feature.md é o próprio backlog e as tasks são executadas a partir dele.

## Resolução de caminhos

Os artefatos do workspace seguem o layout do FORMATO §2:

- **Raiz:** PROJECT_CHARTER.md, REQUIREMENTS.md, DECISIONS.md, CHANGE_REQUESTS.md
- **`analysis/`:** `RN-NNN — <título da feature>/` — PASTA por feature com exatamente 2
  arquivos: `RN-NNN — Analise-Funcional.md` + `RN-NNN — Analise-Tecnica.md`
  (RN-âncora + título sanitizado no nome)
- **`delivery/<grupo>/`:** arquivo DIRETO `RN-NNN — <título da feature> — Feature.md`
  por feature, SEM subpasta. Grupo é subpasta livre (default de escrita: `YYYY-MM`)
- **`specs/`:** DATA_MODEL.md, API_CONTRACTS.md, ERROR_CATALOG.md, UI_SPEC.md, ARCHITECTURE.md, SECURITY.md, ACCEPTANCE_CRITERIA.md, COMPLIANCE.md
- **`quality/`:** TEST_STRATEGY.md, DEFINITION_OF_DONE.md, TRACEABILITY.md
- **`reverse/`:** LEGACY_CONTEXT.md (contexto de legado)
- **`glossaries/`, `meetings/`:** glossários e checkpoints

**Regra de fallback (transição):** ao **LER** um artefato, verifique primeiro
o caminho novo (ex: `specs/DATA_MODEL.md`); se não existir, use o caminho
legado na raiz do workspace (ex: `DATA_MODEL.md`). Ao **ESCREVER/GERAR**, use
sempre o caminho novo. Workspaces ainda não adotados continuam legíveis
durante a transição.

**Duas camadas por feature:** a análise vive em `analysis/RN-NNN — <título da
feature>/` e o delivery em `delivery/<grupo>/RN-NNN — <título da feature> —
Feature.md` — ambos nomeados pela **RN-âncora + espaço + em-dash (`—`) + espaço + título da
feature**. **Grafia fixa:** os nomes usam EXATAMENTE `Analise-Funcional`
e `Analise-Tecnica` (sem acento no "Analise") — NÃO mudar para "Análise". O em-dash `—`
(U+2014) é o separador válido. A RN-âncora sempre existe em REQUIREMENTS.md (nunca inventar)
e é SEMPRE uma RN (âncora = só `RN-XXX`; PEND nunca ancora feature — FORMATO §7.3, H1).
**Sanitização do título:** substitua caracteres inválidos para nome de
pasta/arquivo (`/` `\` `:` `*` `?` `"` `<` `>` `|`) por espaço; colapse espaços duplicados;
remova espaços/pontos finais. NÃO troque o em-dash por hífen.

**Variante legada tolerada (FORMATO §2):** `delivery/<grupo>/<RN-NNN — <título>>/RN-NNN —
<título> — Feature.md` (Feature dentro de pasta própria) é legada de LEITURA — legível e
tolerada pelo `validate`, mas NUNCA gerada por esta skill.

Ao **ESCREVER/GERAR**, crie as duas camadas numa passada (batch multi-feature → um par
análise + Feature.md por feature/RN-âncora). Ao **LER**, o título é variável — resolva SEMPRE
por glob, NUNCA construindo o nome exato, nesta ordem:

1. **Feature:** glob recursivo `delivery/**/RN-NNN — *` (quando a RN é conhecida) — o
   candidato deve ser único (mais de 1 → ambiguidade: pergunte; 0 → próximo). Se for o
   ARQUIVO `RN-NNN — <título> — Feature.md`, leia direto; se for a variante legada
   (Feature dentro de pasta própria), leia o arquivo dentro da pasta.
2. **Análise:** glob na pasta `analysis/` com prefixo `RN-NNN*` — leia
   `RN-NNN — Analise-Tecnica.md` e/ou `RN-NNN — Analise-Funcional.md` conforme o pedido.
3. **Sem RN específica:** glob recursivo `delivery/**/RN-*` (todas/mais recente conforme
   o pedido), preferindo arquivos `— Feature.md`.

## Lê (upstream)

- `<Produto>/PROJECT_CHARTER.md` — visão geral, repos do produto (§1.1)
- `<Produto>/REQUIREMENTS.md` — RNs confirmadas e descrições
- `specs/API_CONTRACTS.md` — endpoints, request/response, validações e erros
- `specs/DATA_MODEL.md` — tabelas reais, colunas, PKs/FKs
- `specs/UI_SPEC.md` — telas, componentes, fluxos e validações de frontend
- `reverse/LEGACY_CONTEXT.md` (se existir) — queries e tabelas do legado
- `CONSTITUTION.md` do container de organização (se presente — convenções de nomenclatura e auditoria)
- Padrões acumulados de organizações (overlay de pack, se configurado no `golmet.yaml` — condicional): para cenários recorrentes no lote, consultar padrões aplicáveis (tipos `codigo`/`ui`/`dados`/`teste`) e citá-los na `Analise-Tecnica` das features afetadas

## Gera (downstream)

- `<Produto>/analysis/RN-NNN — <título da feature>/` — PASTA por feature/RN-âncora com exatamente 2 arquivos: `RN-NNN — Analise-Funcional.md` (análise funcional em linguagem de negócio) e `RN-NNN — Analise-Tecnica.md` (spec técnica PURA — contexto, mapa RN→repos, decisões, contratos e tabelas, riscos — SEM tasks); batch multi-feature → N pastas
- `<Produto>/delivery/<grupo>/RN-NNN — <título da feature> — Feature.md` — ARQUIVO DIRETO por feature/RN-âncora (sem subpasta): canônico (issue principal + tasks `RN-NNN-M` para revisão do operador) + linha OBRIGATÓRIA `> Análise:` linkando a análise

## Nunca edita

- Os documentos-fonte acima (somente leitura)
- Análises e features de outras features ou de grupos de delivery anteriores (`analysis/RN-MMM — <título>/` e `delivery/**/RN-MMM — <título> — Feature.md` — resolva por glob prefixo `RN-MMM*`) — apenas os arquivos das features alvo, na geração corrente, são criados
- `golmet.index.json` (único escritor é o kit)
- Tracker externo — a criação de issues é do fluxo de tracking do perfil (FULL), não desta skill

## Capacidades

- `file` — ler os documentos de entrada do workspace; materializar as duas camadas (`analysis/RN-NNN — <título>/` + `delivery/<grupo>/RN-NNN — <título> — Feature.md`)
- `ask` — apenas o mínimo necessário (RN-âncora da feature, título se não derivável, grupo do delivery quando não derivável, dados de tracking se o perfil usa tracker — sem ask nativo: pergunte no chat e PARE — nunca adivinhe)
- `kit` — checkpoint `post-write: validate` (o Feature.md gerado tem front matter obrigatório — FORMATO §7.5, H5)

## Fluxo

```
PROJECT_CHARTER.md ───────┐
REQUIREMENTS.md ──────────┤
specs/API_CONTRACTS.md ───┤
specs/DATA_MODEL.md ──────┼──► gm-taskspec ──► analysis/RN-NNN — <título>/ (Funcional + Técnica PURA)
specs/UI_SPEC.md ─────────┤                └───► delivery/<grupo>/RN-NNN — <título> — Feature.md (link > Análise:)
reverse/LEGACY_CONTEXT.md ┘
                           │ (revisão manual; issues no tracker só no perfil FULL)
```

## Workflow

### Fase 1: Ler documentos de entrada

1. **PROJECT_CHARTER.md**
   - Nome do produto (front matter/nome do diretório)
   - Visão geral do projeto
   - Repositórios envolvidos (§1.1 — repos com branch e HEAD)

2. **REQUIREMENTS.md**
   - RNs confirmadas: seções `## RN-NNN — <título>` com tag `[confirmado]`
   - Texto da descrição de cada RN
   - Seções: Dados e Campos, Regras de Negócio, Restrições Técnicas

3. **API_CONTRACTS.md**
   - Endpoints: método, path, request/response, path params, query params
   - Validações e erros (ERROR_CATALOG.md ou seção de erros)
   - Identificar quais RNs cada endpoint implementa

4. **DATA_MODEL.md**
   - Tabelas envolvidas: nome real da tabela, colunas, PKs, FKs
   - Relacionamentos entre tabelas
   - Mapeamento de colunas para uso no template "Tabelas de referência" e "Query de referência"

5. **UI_SPEC.md**
   - Telas, componentes, fluxos de navegação
   - Validações de frontend
   - Identificar quais RNs cada tela implementa

6. **LEGACY_CONTEXT.md** (se existir)
   - SQL queries de referência do sistema legado
   - Tabelas e colunas existentes
   - Mecanismos legados a evitar ou reaproveitar

7. **CONSTITUTION.md do container** (se presente)
   - Convenções de organização de delivery (estilo de grupo, template custom, tracker)
   - Colunas de auditoria padrão (quando definidas pela organização)
   - Convenção de nomenclatura de repositórios em tasks (prefixos `[Database]` / `[DevOps]` e regras de título)

### Fase 2: Mapear RNs para repositórios e telas/endpoints

**REGRAS DE MAPEAMENTO (NUNCA perguntar ao usuário):**

| Documento de origem | Repositório destino | Tipo de task |
|---|---|---|
| API_CONTRACTS.md | Backend (ex: `<nome>-API`) | Backend |
| UI_SPEC.md | Frontend (ex: `<nome>-FE`) | Frontend |
| LEGACY_CONTEXT.md (legado:true) | Legado (ex: `<nome>-Web`) | Legado |

**Como derivar o repositório:**
- Backend: extrair da §1.1 do PROJECT_CHARTER.md (repos do produto)
- Frontend: extrair da §1.1 ou da estrutura do produto
- Legado: se `PROJECT_CHARTER.md` contém `legado: true`, usar o repo do sistema legado

**⚠️ Exceção — sistema alvo diferente (CR com abordagem validada):**
Se um `CHANGE_REQUESTS.md` com status `approved` ou `implemented` já tiver validado a abordagem indicando repositório/sistema **diferente** do backend padrão do workspace, o repositório backend DEVE ser o indicado na validação de abordagem. O campo `Repositório alvo` no CR tem precedência sobre a inferência automática.

**Como agrupar tasks:**
- **1 task por endpoint backend** — agrupa todas as RNs que aquele endpoint implementa
- **1 task por tela/componente frontend** — agrupa todas as RNs que aquela tela implementa
- **1 task para alteração em legado** — se alguma RN exige alteração no sistema legado (ex: redirect, compatibilidade)

**Regra de escopo único (uma task = um repositório/escopo):**
- Cada task DEVE conter **apenas um repositório/escopo** no título (ex: `<nome>-API` ou `<nome>-FE`).
- **NUNCA** agrupar, numa mesma task, trabalho que envolva mais de um repo, módulo, frontend+backend simultaneamente, ou banco + código aplicacional como entregáveis misturados.
- Se uma RN toca backend e frontend: gere **duas tasks separadas** (uma `[<backend>]`, outra `[<frontend>]`).
- Se uma RN exige alteração de banco (changeset) e alteração de código: gere **task de banco separada** (`[Database]`) e task de código separada (`[<repo>]`).
- Se uma RN exige configuração DevOps (pipeline, deploy, secrets) e alteração de código: gere **task de DevOps separada** (`[DevOps]`) e task de código separada (`[<repo>]`).
- O objetivo é manter tasks pequenas, independentes e revisáveis — cada uma com um único propósito e um único local de entrega.

**Feature e RN-âncora:**
- Uma **feature** é um bloco de tasks entregue junto — a unidade de entrega. Pode realizar 1..N RNs registradas.
- Cada feature tem uma **RN-âncora**: a primeira RN do escopo (ou a que o usuário indicar). O nome da análise e do arquivo de feature e os IDs das tasks usam APENAS a RN-âncora (o nome inclui também o título da feature).
- As demais RNs realizadas continuam registradas em REQUIREMENTS.md e aparecem no campo **RNs** de cada task. **Proibido** criar `RN-064.md`, `RN-065.md`... para dividir a feature.
- O RN é gerado pelo sistema (REQUIREMENTS.md, sequencial, nunca se repete) — o arquivo reusa a RN-âncora existente; nunca inventa.
- **Âncora é SEMPRE uma RN:** PEND NUNCA ancora feature (H1) — âncora `PEND-XXX` ABORTA a geração (Modo de erro). A PEND de origem de uma RN promovida aparece apenas como nota de proveniência em REQUIREMENTS.md (`> Promovida de PEND-NNN em <data>`).

**Validação de IDs:** ao mapear as RNs de cada task, confira se **todos** os IDs referenciados existem em REQUIREMENTS.md e usam o prefixo `RN-XXX`. A âncora e o campo **RNs** das tasks aceitam SÓ RNs: PEND não ancora feature e não entra no campo RNs. Tasks NUNCA referenciam IDs inventados ou de prefixo paralelo (ex: `COM-01`). Se faltar registro, aborte e instrua o usuário a registrar as regras primeiro (via gm-requirements, numeração sequencial). Âncora `PEND-XXX` → aborte com orientação de promoção.

**O mapeamento alimenta as DUAS camadas:** a seção `## Mapa RN → repositórios/endpoints/telas` da spec técnica pura (`analysis/.../RN-NNN — Analise-Tecnica.md`) e o campo **RNs** + os títulos das tasks no Feature.md — cada camada com o seu papel; a spec pura NUNCA lista tasks.

### Fase 3: Perguntar ao usuário (mín necessário)

Perguntar APENAS:
- RN-âncora da feature: se o contexto não indicar, pergunte qual — ou apresente as RNs confirmadas de REQUIREMENTS.md sem análise/feature na geração corrente (comparar com glob `analysis/RN-*` e glob recursivo `delivery/**/RN-*`). Nunca derive por omissão. **Âncora `PEND-XXX` → ABORTE** com a orientação: "promova a PEND a RN primeiro (nova RN sequencial em REQUIREMENTS.md citando a origem — via gm-requirements), depois gere com a RN como âncora". Se a feature realiza várias RNs (ex: RN-063..073), confirme a âncora (a primeira do escopo ou a que o usuário indicar) — e NÃO crie uma análise por RN. Capture também o **título da feature** (obrigatório para o nome): use o título presente no contexto (REQUIREMENTS.md, discussão da feature); só se NÃO houver título derivável, pergunte APENAS o título da feature.
- Grupo de entrega/marcador (ex: `Sprint 33`, `v2.1`) — se o perfil usa tracker (FULL)
- Repositório/área para as issues (ex: `org/repo`) — se o perfil usa tracker (FULL)
- Se as issues já têm tracking externo (ex: sistema de demanda do cliente). Se sim, qual. — só no perfil FULL; no BASE, Tracking ID fica `—`
- **Grupo do delivery** — SOMENTE o que não for derivável:
  - Estilo mensal (default) → mês corrente `YYYY-MM` (derivado, sem pergunta)
  - Estilo por data de início de ciclo → a data de início (grupo = `YYYY-MM-DD`)
  - Estilo por release/épico → o nome do grupo (ex.: `v2.1`, `Pagamentos`)
  - Estilo por horizonte → qual horizonte (`now`, `next` ou `later`)
  - Estilo custom com template → o valor de CADA placeholder (ex.: `<cliente>`, `<trimestre>`)
  - Se houver `CONSTITUTION.md` de container definindo o estilo, siga-o; ausente → pergunte o estilo desejado e OFEREÇA registrá-lo na CONSTITUTION do container (só persiste com confirmação explícita do operador); se o operador recusar, use o estilo APENAS para a geração corrente. Default de escrita: `YYYY-MM`.

NÃO perguntar:
- Qual repo para cada RN (já derivado dos documentos)
- Se é front ou back (já derivado)
- Se uma RN é front ou back (derivar de API_CONTRACTS.md vs UI_SPEC.md)
- O grupo do delivery quando derivável das convenções
- O nome dos arquivos: é `analysis/RN-NNN — <título>/` + `delivery/<grupo>/RN-NNN — <título> — Feature.md`, derivado do grupo + RN-âncora + título da feature (sanitizado) — sem pergunta extra, desde que o título esteja no contexto. Os arquivos internos da análise têm nomes FIXOS: `RN-NNN — Analise-Funcional.md` e `RN-NNN — Analise-Tecnica.md`. Só pergunte o título se não for derivável (Fase 3).

### Fase 4: Gerar as duas camadas — analysis/ + delivery/<grupo>/RN-NNN — <título> — Feature.md

Criar, numa passada, para cada feature/RN-âncora do batch:

1. **Análise** — criar `<Produto>/analysis/RN-NNN — <título sanitizado da feature>/`
   (nome usa o título sanitizado e deve coincidir com o H1 gerado) com SEMPRE os
   2 arquivos: (a) **`RN-NNN — Analise-Tecnica.md`** — **spec técnica PURA** no formato da
   seção "Formato da spec técnica pura" (proveniência, H1, contexto técnico, mapa
   RN→repositórios/endpoints/telas, decisões técnicas, contratos e tabelas envolvidos —
   referenciando `specs/` —, riscos/pré-requisitos); (b) **`RN-NNN — Analise-Funcional.md`** —
   análise funcional no template da seção "Formato da análise funcional", derivada da
   RN-âncora + RNs do escopo em REQUIREMENTS.md. **Proibido** no funcional: endpoints/métodos
   HTTP, nomes de tabelas/colunas, frameworks, nomes de serviço/componente técnico, queries,
   códigos de erro técnicos ou menção a repos. **Proibido** na spec pura (Regra de ouro):
   `## Issue principal`, `## Resumo das tasks`, `## Ordem de Execução`, tasks
   `### RN-NNN-M` ou qualquer menção a sprint/estilo de delivery/pasta de delivery — tasks
   vivem EXCLUSIVAMENTE no Feature.md.
2. **Delivery** — criar o ARQUIVO DIRETO
   `<Produto>/delivery/<grupo>/RN-NNN — <título sanitizado da feature> —
   Feature.md` (SEM subpasta por feature) com o canônico COMPLETO (seção "Estrutura
   canônica do arquivo de feature": front matter, H1, proveniência + linha OBRIGATÓRIA
   `> Análise: ../../analysis/RN-NNN — <título>/`, Issue principal, Resumo das tasks, Ordem
   de Execução, tasks `RN-NNN-M`, Dependências opcional).

O H1 da spec pura e do Feature.md é `# RN-NNN — <título da feature>` (coincide com o nome da
pasta da análise e com o nome do arquivo de feature sem o sufixo ` — Feature.md`).

### Fase 4.5 — Classificar tasks e aplicar prefixos `[Database]` / `[DevOps]`

> **Fonte oficial da convenção:** `CONSTITUTION.md` do container de organização (se presente).
> **Destino:** a classificação alimenta EXCLUSIVAMENTE o arquivo de feature (`RN-NNN — <título> — Feature.md`) — títulos no `## Resumo das tasks`, na `## Ordem de Execução` e nas tasks individuais. A spec técnica pura NUNCA cita tasks.

Após derivar o título preliminar `[<repo>] <descrição>`, classifique cada task conforme as heurísticas abaixo.

**Prefixo `[Database]`** (quando o trabalho predominante for de banco de dados):
- Criação/alteração de tabelas, colunas, índices, constraints, sequences, triggers, views, procedures, functions.
- Criação/execução de scripts DDL, DML, changesets de migração.
- Modelagem derivada de `DATA_MODEL.md`, `LEGACY_CONTEXT.md` ou queries de referência.
- Migração de dados, carga, correção/ajuste de dados.
- Configuração de schema, partitionamento, comentários em colunas/tabelas.

Exemplo de reescrita:
- `[MeuApp-API] Criar changeset de migração para coluna IND_FLAG_EXEMPLO`
  → `[Database] Criar changeset de migração para coluna IND_FLAG_EXEMPLO`

**Prefixo `[DevOps]`** (quando o trabalho predominante for infraestrutura/CI/CD):
- Criação/configuração de pipelines (GitHub Actions, Azure Pipelines, Jenkins ou equivalente).
- Criação/configuração de ambientes, deploy, release, build, publicação de artefatos.
- Dockerfiles, Docker Compose, manifests Kubernetes, Helm charts.
- Config Server, Service Discovery, API Gateway, secrets, variáveis de ambiente.
- Terraform, Ansible, scripts de provisionamento, nova infraestrutura, certificados.
- Setup inicial de repositório quando a entrega principal for pipeline/dependências/ambiente/conexão.

Exemplo de reescrita:
- `[MeuApp-API] Configurar pipeline de build e deploy`
  → `[DevOps] Configurar pipeline de build e deploy`

**Regras gerais de classificação:**
- Aplique **no máximo um** prefixo por task (`[Database]` **OU** `[DevOps]`). Se ambos se aplicarem, use `[Database]` (o entregável principal assume o prefixo).
- **Nunca duplique**: se o título já começar com `[Database]` ou `[DevOps]`, mantenha como está.
- Não altere títulos que não se enquadrarem em nenhum dos dois prefixos.
- A classificação é sobre o **trabalho da task**, não sobre o repositório. Uma task em uma API pode ser `[Database]` se o trabalho for changeset/DML.
- Combine com a regra de escopo único: se uma task envolver banco + código aplicacional, quebre em duas — `[Database] ...` e `[<repo>] ...`.

### Fase 4.6 — Incluir colunas de auditoria em tasks `[Database]`

> **Fonte:** convenções de auditoria do container (se definidas).
> **Destino:** os DDLs verificados/garantidos aqui pertencem às tasks `[Database]` do Feature.md.

Para toda task classificada como `[Database]` cujo DDL inclua `CREATE TABLE` (tabela nova), **verificar e garantir** a presença das colunas de auditoria definidas nas convenções do container no DDL.

**Regras:**
- Colunas de inclusão são `NOT NULL`; colunas de atualização são nullable
- Se as convenções não definirem colunas de auditoria, esta fase é ignorada

## Formato da spec técnica pura (`analysis/RN-NNN — Analise-Tecnica.md`)

Camada da análise que descreve a feature do ponto de vista técnico SEM especificar tasks — a
execução é detalhada no Feature.md (dependência unidirecional tasks→análise):

```markdown
# RN-NNN — <título da feature>

> Gerado por gm-taskspec em <data>. Edite para revisar.

## Contexto técnico

<Parágrafo: onde a feature se encaixa no sistema, qual problema técnico resolve,
restrições de arquitetura e dependências de contexto.>

## Mapa RN → repositórios/endpoints/telas

| RN | Repositório | Endpoints / Telas |
|---|---|---|
| RN-063 | MeuApp-API | GET /pedidos/tipos, POST /pedidos |
| RN-064 | MeuApp-FE | Tela de pedidos (listagem + formulário) |

## Decisões técnicas

| # | Decisão | Justificativa |
|---|---|---|
| 1 | <decisão> | <motivo, ADR relacionado, alternativa descartada> |

## Contratos e tabelas envolvidos

- **Contratos de API:** `specs/API_CONTRACTS.md` — <endpoints afetados>
- **Modelo de dados:** `specs/DATA_MODEL.md` — <tabelas/colunas envolvidas>
- **UI:** `specs/UI_SPEC.md` — <telas/componentes afetados>
- **Legado (se houver):** `reverse/LEGACY_CONTEXT.md` — <queries/tabelas legadas>

## Riscos e pré-requisitos

- <risco, pré-requisito de ambiente/infra, dependência de outra feature>
```

**Regras da spec pura:**
- Fontes: as mesmas da Fase 1 (REQUIREMENTS.md, specs/, LEGACY_CONTEXT.md, convenções do container).
- **PROIBIDO** conter (Regra de ouro): seções `## Issue principal`, `## Resumo das
  tasks`, `## Ordem de Execução`; tasks `### RN-NNN-M` ou IDs `RN-NNN-M`; menção a sprint,
  estilo de delivery ou pasta/arquivo de delivery. A rastreabilidade RN→task vive
  EXCLUSIVAMENTE no campo **RNs** do Feature.md.
- O mapa RN→repos lista RNs (a âncora e as do escopo), NUNCA tasks.
- Seções técnicas extras são permitidas APENAS se descreverem a feature (nunca o plano de
  execução); em dúvida, detalhe de execução pertence ao Feature.md.

## Formato do arquivo de feature (`delivery/<grupo>/RN-NNN — <título> — Feature.md`)

### Estrutura canônica do arquivo de feature

Todo arquivo de feature `RN-NNN — <título> — Feature.md` — gerado por `gm-taskspec` OU por qualquer
outro executor — DEVE ter EXATAMENTE esta sequência de
elementos, com nomes de seção FIXOS. A estrutura é idêntica entre todos os arquivos; só
campos de estado legítimos divergem (`status` do front matter, linha `Status` da Issue
principal, datas, Tracking ID):

1. **Front matter YAML** — chaves fixas (FORMATO §7.5, obrigatório): `status` (`draft` = gerado para revisão) e `anchor: RN-NNN` (a RN-âncora). Transições legais: `draft → ready → in_progress → done` (`→cancelled` de qualquer não-final); `ready` exige anchor RN `active` (H3)
2. **H1** — `# RN-NNN — <título da feature>` (coincide com o nome do arquivo sem o sufixo ` — Feature.md`)
3. **Bloco de proveniência** (linhas `>`) — 1ª linha no padrão fixo `> Gerado por <executor> em <DD/MM/AAAA>.` (executor REAL: `gm-taskspec` na geração; o executor migrador real na migração); **2ª linha OBRIGATÓRIA** `> Análise: ../../analysis/RN-NNN — <título>/` — link relativo para a pasta da análise (ajuste a quantidade de `../` se o grupo do delivery tiver profundidade maior que 1 — ex.: grupo custom com múltiplos níveis); linhas `>` adicionais opcionais para anotações (`**Migrado em ...**`, `**Fonte:** ...`, `**Rastreamento:** ...`, `**✅ Implementado em ...**`)
4. **`## Issue principal`** — tabela Campo/Valor com as linhas fixas `Título`, `Tracking ID`, `Produto`, `Grupo`, `Repositórios` + linha opcional `Status` (somente quando o arquivo refletir implementação concluída; proibida em arquivo `draft`). Tracking ID é metadata opcional — `—` no perfil BASE; só preenchido no fluxo FULL com tracker
5. **`## Resumo das tasks`** — tabela de 4 colunas (Regra das 4 colunas)
6. **`## Ordem de Execução`** — formato canônico em waves (seção "Ordem de Execução" abaixo)
7. **Tasks** — `### RN-NNN-M. <título>` com diretiva `> estado: todo` (FORMATO §7.6 — estados `todo · in_progress · done · blocked`), separadas por `---`, cada uma no "Template de cada task"
8. **`## Dependências entre Tasks`** (OPCIONAL) — somente para features com grafo de dependências não-trivial; diagrama ASCII ou lista. Se ausente, as waves do item 6 são a representação canônica de ordem/dependência

**Proibido:** seções fora desta lista, nomes alternativos para as seções de nome fixo
("Ordem de Execução do Batch", "Resumo de Dependências entre Tasks", etc.) ou reordenação.
Anotações curtas entre parênteses junto aos IDs são permitidas na `## Ordem de Execução`;
rationale longo vai nas **Notas técnicas** da task individual. O Feature.md NUNCA duplica o
conteúdo da spec pura — ele linka a análise via `> Análise:` (Regra de ouro).

### Cabeçalho

```markdown
---
status: draft
anchor: RN-010
---
# RN-010 — <título da feature>

> Gerado por gm-taskspec em <data>. Edite para revisar.
> Análise: ../../analysis/RN-010 — <título>/
> Após revisão, mude `status` para `ready` (exige anchor RN `active`) e execute as tasks a partir daqui — ou use o fluxo de issues do tracker (perfil FULL).

## Issue principal

| Campo | Valor |
|---|---|
| Título | `RN-010 — <título>` |
| Tracking ID | — (metadata opcional — preenchida no fluxo FULL com tracker) |
| Produto | <nome do produto> |
| Grupo | <grupo do delivery> |
| Repositórios | <org/repo> (um ou mais, separados por vírgula) |
```

`<executor>` é o executor REAL do arquivo — a proveniência é verídica; o PADRÃO da linha
é fixo. A linha "Após revisão..." existe apenas em arquivo `status: draft`. A linha
`> Análise:` é OBRIGATÓRIA em todo arquivo de feature.

### Template de cada task

Cada task DEVE seguir este template completo. **Blocos condicionais:** os blocos
de `**Path params:**` em diante (Path params, Query params, Contrato, Tabelas de referência,
Query de referência (legado), Validações / Erros, Critério de Aceitação, Dependências,
Notas técnicas) são incluídos APENAS quando aplicáveis à task — SEMPRE nesta ordem canônica;
bloco não aplicável é omitido inteiro (sem placeholder). `**Descrição:**` e `**Dependências:**`
existem em TODA task (`**Dependências:**` usa `— (primeira task)` quando não há dependências).
`**Critério de Aceitação:**` e `**Notas técnicas:**` entram quando houver comportamento
verificável ou decisão técnica de implementação a registrar. `**Critério de Aceitação:**`,
`**Dependências:**` e `**Notas técnicas:**` fazem PARTE do canon — ao migrar tasks legadas,
nunca descarte esses blocos (são documentação de implementação valiosa).

```markdown
### RN-NNN-M. <título da task>
> estado: todo

| Campo | Valor |
|---|---|
| **Título** | `[Database] RN-NNN-M — <descrição>` ou `[DevOps] RN-NNN-M — <descrição>` quando classificável; caso contrário `[<repo>] RN-NNN-M — <descrição>`. Cada task tem **exatamente um** repo/escopo. **Nunca** concatenar dois prefixos. |
| **RNs** | RN-063, RN-064 — TODAS as RNs realizadas pela task (inclusive as não-âncoras) — SÓ RNs, nunca PEND |

**Descrição:**
<Parágrafo explicando o propósito, o que entrega, e por quê>

**Path params:**
| Parâmetro | Tipo | Descrição |
|---|---|---|
| <nome> | <tipo> (path) | <descrição> |

**Query params:**
| Parâmetro | Tipo | Obrigatório | Descrição |
|---|---|---|---|
| <nome> | <tipo> | sim/não | <descrição> |

**Contrato:**
```
<Método> <path>
Request: <JSON ou "vazio">
Response <status>: <JSON ou "vazio">
```

**Tabelas de referência:**
| Tabela | Coluna alvo | Uso |
|---|---|---|
| <tabela> | <coluna> | <propósito> |

**Query de referência (legado):**
```sql
<SQL existente do sistema legado, se aplicável>
```

**Validações / Erros:**
- <código>: <descrição> → <status code>
- <código>: <descrição> → <status code>

**Critério de Aceitação:**
- Dado <contexto>, Quando <ação>, Então <resultado>
- Dado <contexto>, Quando <ação>, Então <resultado>

**Dependências:** <tasks necessárias antes desta>

**Notas técnicas:**
- <observação relevante>
- <observação relevante>
```

### Tabela de tarefas (resumo)

Incluir também a seção de nome fixo `## Resumo das tasks` (entre a `## Issue principal` e a `## Ordem de Execução`) com a tabela de resumo para visão geral:

```markdown
| Task | Tipo | Repo | Descrição |
|---|---|---|---|
| RN-010-1 | Backend | MeuApp-API | Criar endpoint GET /pedidos/tipos |
| RN-010-2 | Backend | MeuApp-API | Criar endpoint POST /pedidos |
```

**Regra das 4 colunas (exclusividade obrigatória):** a tabela de resumo contém
**SOMENTE E APENAS** as colunas `Task | Tipo | Repo | Descrição` — **proibido**
adicionar qualquer coluna extra (RNs/PENDs, prioridade, status, data, ou qualquer
outra). A rastreabilidade RN→task vive no campo **RNs** de cada task individual;
status e prioridade, quando existirem, pertencem à task individual ou ao cabeçalho
do arquivo — nunca ao resumo. Ao migrar/reescrever tasks de formato legado,
**converta** a tabela para as 4 colunas canônicas — nunca copie a tabela legada como está.

### Ordem de Execução (seção canônica)

Todo arquivo traz, após o `## Resumo das tasks`, a seção de nome fixo `## Ordem de Execução`
— proibido nome alternativo (ex: "Ordem de Execução do Batch", "Sequenciamento", "Plano de
execução"). Formato canônico em **waves**:

```markdown
## Ordem de Execução

**Wave 1 — <fase>:**
RN-NNN-1 → RN-NNN-2 → RN-NNN-3

**Wave 2 — <fase> (paralelizável):**
RN-NNN-4, RN-NNN-5

**Wave N — Qualidade:**
RN-NNN-M (testes de regressão)
```

- `→` = execução sequencial (dependência); `,` = paralelizável dentro da wave.
- Anotações curtas entre parênteses junto ao ID são permitidas (ex: `RN-NNN-14 (crítica — depende do CR-001)`); rationale longo vai nas **Notas técnicas** da task.
- A última wave é sempre a de qualidade/testes de regressão.
- Ao migrar lista legada numerada ("1. ... 2. ..."), **CONVERTA** para waves — não copie o formato legado.

### Ordem das tasks

Dentro da seção canônica `## Ordem de Execução` (formato em waves — seção anterior), as tasks DEVEM ser ordenadas por dependência:
1. Primeiro endpoints de consulta (GET sem dependências)
2. Depois endpoints de criação (POST)
3. Depois endpoints de listagem/detalhe (GET com dependência)
4. Depois endpoints de ação (encerramento, etc.)
5. Depois tasks transversais (perfil, padronização, compatibilidade)
6. Por último tasks de legado (redirect, compatibilidade reversa)

### Regras de título

| Elemento | Formato |
|---|---|
| **PBI** | `[MinhaOrg] <demanda externa> - Título descritivo` (só no perfil FULL com tracker; BASE: o PBI é o próprio arquivo de feature) |
| **Task** | `[<repo>] RN-NNN-M — <descrição>` — ex: `[MeuApp-API] RN-010-1 — Criar endpoint de listagem de tipos` |
| **Task de banco** | `[Database] <descrição>` — ex: `[Database] Criar changeset de migração para coluna IND_FLAG_EXEMPLO` |
| **Task de DevOps** | `[DevOps] <descrição>` — ex: `[DevOps] Configurar pipeline de build e deploy` |
| **ID de task** | RN-NNN-M — M sequencial a partir de 1, dentro do arquivo |
| **Repo** | Nome do repositório de código (apenas um por task) |
| **RNs** | Campo da task com TODAS as RNs realizadas (âncora + não-âncoras) — único local de rastreabilidade RN→task; o Resumo tem SOMENTE as 4 colunas canônicas (Regra das 4 colunas — nenhuma coluna extra) |

### Regras de conteúdo técnico

Cada task DEVE conter:

1. **Path params e Query params** — listagem exaustiva com tipos
2. **Contrato request/response** — JSON completo com tipos dos campos
3. **Tabelas de referência** — nome real da tabela (ex: `TBL_PEDIDO`, não `Pedido`), colunas envolvidas (ex: `NRO_ID_CLIENTE`, não `clienteId`), propósito. Extrair de DATA_MODEL.md.
4. **Query de referência (se legado existe)** — SQL do sistema legado que o novo código precisa ser compatível (ex: WHERE clauses, joins)
5. **Validações e erros** — mapear cada validação para um código de erro e status HTTP
6. **Critério de aceitação** — no formato Dado/Quando/Então
7. **Dependências** — lista de tasks que precisam ter sido implementadas antes
8. **Notas técnicas** — observações de implementação, decisões arquiteturais, mecanismos a evitar/reusar

Itens 1–5 são aplicáveis conforme o tipo de task (endpoint backend: obrigatórios;
frontend/banco/DevOps: substituídos pelo artefato equivalente — modelo de dados/tela,
DDL, pipeline — dentro dos blocos canônicos). A ordem dos blocos é a canônica do
"Template de cada task".

## Formato da análise funcional (`analysis/RN-NNN — Analise-Funcional.md`)

Arquivo gerado na pasta da análise (`analysis/RN-NNN — <título>/`), irmão da spec técnica
pura, em linguagem 100% de negócio:

```markdown
# RN-NNN — <título da feature> — Análise Funcional

> Gerado por gm-taskspec em <data> a partir de REQUIREMENTS.md (RN-âncora + RNs do escopo).

## O que esta entrega faz

<Parágrafo curto, em linguagem de negócio, descrevendo o que o sistema passa a fazer e para quem.>

## Comportamento esperado

- <comportamento do ponto de vista do usuário, um por linha>
- <...>

## Regras de negócio

| Regra | O que garante |
|---|---|
| 1 | <descrição funcional da regra, sem termos técnicos> |
| 2 | <...> |

## O que fica de fora

<Opcional — limites de escopo em linguagem de negócio.>
```

Regras: fonte exclusiva REQUIREMENTS.md (todas as RNs do escopo, inclusive não-âncoras);
a coluna "Regra" usa numeração sequencial simples 1..N (1 linha por regra do escopo — por
RN ou agrupamento funcional) — numeração MERAMENTE COSMÉTICA e NÃO-rastreável: PROIBIDO
usar IDs `RN-XXX` no Analise-Funcional (a rastreabilidade formal por RN vive exclusivamente
no campo **RNs** do Feature.md e no REQUIREMENTS.md); proibido jargão técnico;
verificação rápida — se um leitor de negócio não entender uma frase, reescreva a frase
(nunca adicione jargão).

## Modo de erro

- **Demanda externa (tracker FULL) = "#pendente"**: abortar, pedir para atualizar PROJECT_CHARTER.md com o número real da demanda. No BASE (sem tracker), este erro não se aplica — Tracking ID fica `—`.
- **Nenhuma RN confirmada**: abortar, não há tasks a gerar
- **RN referenciada inexistente ou prefixo inválido**: se alguma task for referenciar um ID que não existe em REQUIREMENTS.md, ou usar prefixo diferente de `RN-XXX` (ex: `COM-01`, `REQ-01`), abortar a geração e instruir o registro das regras em REQUIREMENTS.md (numeração sequencial, via gm-requirements) antes de gerar o Feature.md. Tasks só referenciam IDs existentes. RN-âncora inexistente em REQUIREMENTS.md → abortar. Âncora e campo **RNs** aceitam SÓ `RN-XXX` — `PEND-XXX` não ancora feature nem entra no campo RNs (anotação de bloqueio tipo `RN-NNN-M (bloqueada por PEND-NNN)` na Ordem de Execução continua permitida).
- **Âncora PEND**: PEND NUNCA ancora feature (H1). Pedido com âncora `PEND-XXX` (ou âncora que corresponde a PEND de REQUIREMENTS.md) → ABORTAR a geração com a orientação: "promova a PEND a RN primeiro (nova RN sequencial em REQUIREMENTS.md citando a origem — via gm-requirements), depois gere com a RN como âncora". Nenhum arquivo é criado. Features `PEND-NNN — *` legadas (pré-adoção, ex: projeto legado) são apenas legíveis.
- **Sem DATA_MODEL.md, API_CONTRACTS.md, ARCHITECTURE.md ou UI_SPEC.md**: não abortar — gerar as duas camadas mesmo assim, mas marcar as tasks afetadas como **"sujeitas a revisão"** até o documento técnico correspondente ser fechado. Incluir aviso no bloco de proveniência do Feature.md listando quais documentos estão faltando e quais tasks são afetadas.
- **Análise ou Feature.md da RN-âncora já existe** (glob prefixo `RN-NNN*` em `analysis/` e glob recursivo `delivery/**/RN-NNN*`): perguntar sobrescrever ou merge. Se existir apenas variante legada (Feature dentro de pasta própria), perguntar se converte para o formato vigente (em lote).

## Checkpoint de kit (`post-write: validate`)

Após gerar as duas camadas, rode `golmet validate` no workspace e corrija o que apontar:
- O front matter do Feature.md é OBRIGATÓRIO e parseável (H5 — FORMATO §7.5): `status: draft` + `anchor: RN-NNN`
- Tasks com diretiva `> estado:` no vocabulário legal (`todo · in_progress · done · blocked` — FORMATO §7.6)
- IDs únicos (H2) e referências cruzadas resolvem (H10)
Sem kit disponível, anote a pendência "rode golmet validate" para o próximo passo.

## Formato de commit

```
docs(tasks): gerar analysis/RN-NNN — <título>/ + delivery/<grupo>/RN-NNN — <título> — Feature.md da feature RN-NNN do produto <nome> para revisão
```

## Casos de Borda

- **Grupo de delivery/tracking não informados** — pergunte o mínimo necessário (Fase 3); nunca invente dados de tracker.
- **Estilo de ciclo `sprint`** — grupo = `YYYY-MM-DD` da data de INÍCIO do ciclo (pergunte se não inferível), SEM prefixo de pasta extra.
- **Estilo `custom`** — resolva CADA placeholder do template por pergunta (Fase 3); profundidade de pastas maior que 1 exige ajustar a quantidade de `../` da linha `> Análise:`.
- **RN toca backend e frontend** — gere duas tasks separadas (`[<backend>]` e `[<frontend>]`).
- **RN exige banco + código** — task `[Database]` separada da task `[<repo>]`.
- **RN exige DevOps + código** — task `[DevOps]` separada da task `[<repo>]`.
- **CR com abordagem validada indicando outro repositório** — o `Repositório alvo` do CR tem precedência sobre a inferência automática.
- **Análise ou Feature.md da RN-âncora já existe (glob prefixo `RN-NNN*` em `analysis/` + recursivo `delivery/**`)** — pergunte se sobrescreve ou faz merge.
- **Workspace com features legadas (variante com pasta própria; arquivos flat)** — gere as duas camadas novas normalmente; não mova nem converta legados nesta skill.
- **Batch multi-feature** — um par análise + Feature.md por feature (RN-âncora); NUNCA divida uma feature em `RN-064`, `RN-065`... — as RNs não-âncora vão apenas no campo **RNs** de cada task. Referências cruzadas a outras RNs na descrição são permitidas.
- **Executor diferente (ex: migração de legado por outra skill)** — os arquivos resultantes seguem os formatos canônicos (spec pura + Feature.md) com estrutura IDÊNTICA à da geração pela skill; só campos de estado legítimos divergem.
- **Feature PEND legada** — features `delivery/**/PEND-NNN — *` existentes (ex: projeto legado) são LEGÍVEIS (globs de leitura mantêm o padrão como legado); re-geração exige promover a PEND a RN primeiro (gm-requirements) e gerar com a RN como âncora — nunca regenere com âncora PEND.

---

compatibility: multi-engine

> **Degradação de checkpoints:** o checkpoint `post-write: validate` depende do kit golmet disponível. Sem o kit, a skill continua funcionando — o checkpoint vira "rode `golmet validate` manualmente" após a escrita (o front matter do Feature.md segue obrigatório pelo FORMATO §7.5).
