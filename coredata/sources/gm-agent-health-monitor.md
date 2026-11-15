# gm-agent-health-monitor — Monitor de Saúde dos Agents

Verifica a integridade do ecossistema do pipeline: subagents registrados,
skills com erros, artefatos órfãos, staleness e propagation status.
Gera um relatório de saúde que pode ser usado para manutenção preventiva.

**Diferenciação:**

| Verificação | Escopo |
|-------|--------|
| `gm-validate` | Integridade da DAG de documentos (RNs → downstreams) |
| `gm-progress` | Progresso do pipeline (cobertura, drift, pendências) |
| `gm-agent-health-monitor` | Saúde dos componentes do workflow (agents, skills, artefatos) |

## Referências

- Subagents e skills: `<config>/agents/*.md`, `<config>/skills/*/SKILL.md`
- PROPAGATION_STATUS.md: `quality/PROPAGATION_STATUS.md`
- TRACEABILITY.md: `quality/TRACEABILITY.md`

## Resolução de caminhos

Os artefatos do módulo seguem a estrutura com subpastas:

- **Raiz:** PROJECT_CHARTER.md, CONSTITUTION.md, REQUIREMENTS.md, DECISIONS.md, CHANGELOG.md
- **`specs/`:** DATA_MODEL.md, API_CONTRACTS.md, ERROR_CATALOG.md, UI_SPEC.md, ARCHITECTURE.md, SECURITY.md, ACCEPTANCE_CRITERIA.md, USER_STORIES.md, US<NNN>.md, FIGMA_PROMPTS.md, COMPLIANCE.md
- **`quality/`:** TEST_STRATEGY.md, DEFINITION_OF_DONE.md, TRACEABILITY.md, VALIDATION_REPORT.md, PROPAGATION_STATUS.md, PROGRESS_REPORT.md
- **`delivery/`:** TASKS.md, TASKS_*.md, SCOPE_AGREEMENT.md, CLIENT_REQUIREMENTS.md, CLIENT_REPORT_*.md, HANDOVER_LOG.md
- **`support/`:** LEGACY_CONTEXT.md, KNOWN_ERRORS.md, MIGRATION.md, CHANGE_REQUESTS.md
- **`glossaries/`:** DEV_GLOSSARY.md, BUSINESS_GLOSSARY.md, USER_GLOSSARY.md, GLOSSARY.md
- **`docs/`:** howto/*.md e documentos auxiliares
- **`meetings/`, `contracts/`:** checkpoints (inalterados)

**Regra de fallback (transição):** ao **LER** um artefato, verifique primeiro
o caminho novo; se não existir, use o caminho legado na raiz do módulo. Ao
**ESCREVER/GERAR**, use sempre o caminho novo. Módulos ainda não migrados
continuam legíveis durante a transição.

## Lê (upstream)

- `<config>/agents/*.md` (subagents e primary agents registrados)
- `<config>/skills/*/SKILL.md` (skills registradas)
- `<config>/CHECKLISTS.md` (critérios da verificação F)
- `<config>/PERMISSIONS.md` (verificação H)
- `<config>/AGENTS.md` + `<config>/rules/*.md` (verificação G)
- `<modulo>/<projeto>/quality/PROPAGATION_STATUS.md` (se existir)
- `<modulo>/<projeto>/quality/TRACEABILITY.md` (se existir)
- `<modulo>/<projeto>/` — todos os artefatos (para verificação de staleness e integridade)
- Artefatos da DAG (specs/, quality/, delivery/, support/, glossaries/, docs/)

## Gera (downstream)

- `<modulo>/<projeto>/quality/AGENT_HEALTH_REPORT_<yyyy-MM-dd-hh:mm>.md`
  ou na raiz do workspace se nenhum módulo estiver em contexto

**Política de persistência (FULL):**

| Escopo do relatório | Destino | Commit? |
|---|---|---|
| Saúde de um **módulo** (gate formal) | `quality/` do módulo | ✅ commitável — evidência de auditoria do produto (padrão VALIDATION_REPORT) |
| Saúde da **config do workflow** ou **cross-module** | raiz do workspace / config | ❌ **efêmero — NUNCA commitar** |

Relatórios de raiz/workspace são snapshots regeneráveis que envelhecem a cada
evolução do pipeline (o conteúdo acionável é destilado no registro de
melhorias do workflow — fonte versionada; tendência = histórico de scans;
estado atual = scan determinístico da config). Ao gerar um relatório de
raiz, entregue-o na sessão e sinalize que é efêmero.

## Nunca edita

- Nenhum artefato, nenhum agente, nenhuma skill. **Read-only absoluto.**

## Capacidades

- `glob` (listar agents, skills, artefatos do módulo)
- `file` (ler metadados, PROPAGATION_STATUS, TRACEABILITY, artefatos)
- `grep` (buscar erros de parse, referências quebradas)
- `ask` (perguntar qual módulo analisar quando múltiplos — Passo 1)
- `shell` (SOMENTE leitura): scan determinístico da config (pré-scan Passo 0)

## Modo de erro

`warn` — se algum componente não puder ser avaliado (ex: artefato ausente),
inclua no relatório como "não avaliado" em vez de falhar. O relatório
deve ser gerado mesmo com dados parciais.

## Formato de commit

Somente para relatórios de **módulo** em `quality/` (política FULL —
relatório de raiz/workspace é efêmero, não gera commit):

```
docs(governance): add agent health report <timestamp>
```

## Algoritmo

### Passo 0 — Pré-scan determinístico da config

Rode o linter determinístico de componentes (absorvido pelo kit
`golmet`) e consuma `quality/health.json` (critérios: front matter
íntegro, campos proibidos, permissões, seções obrigatórias, referências
órfãs, coerência com o registry):

```
golmet scan config [--strict]
```

- `findings[]` com severidade `critical`/`warning` por componente — incorpore
  ao relatório como evidência; reavalie contexto antes de elevar (o script
  entrega candidatos, não vereditos: ex. path default documentado ou
  marcador legítimo em template não são violações).
- Regras semânticas que continuam suas: qualidade de `description`
  (triggers reais), drift de views do ARCHITECTURE.md vs registry,
  staleness de artefatos do módulo e PROPAGATION_STATUS.
- `--strict` (exit 1 com críticos) é o gate de publicação.

### Passo 1 — Descobrir módulo

Use `glob` para encontrar diretórios com `PROJECT_CHARTER.md`.

- Se nenhum → gere o relatório na raiz do workspace (sem contexto de módulo).
- Se 1 → use-o.
- Se múltiplos → `ask` para perguntar qual módulo analisar (ou "todos").

### Passo 2 — Verificação A: Subagents Registrados

1. Use `glob` em `<config>/agents/*.md` para listar todos os
   subagents registrados.
2. Para cada subagent, leia o front matter e verifique:
   - `name` está presente e válido
   - `description` está presente
   - `permission` está definido e coerente com a função
3. Verifique se há subagents referenciados nos primary agents que não
   existem como arquivo (referência quebrada).
4. Leia o `registry` e valide, para cada subagent, `owner`, `available_to`
   e a compatibilidade entre registro legado e `owner`.
5. Verifique se cada delegate de um primary está presente em `available_to`.
6. Verifique se as arestas primário→subagent do ARCHITECTURE.md estão
   coerentes com o registry (delegates ∪ available_to) —
   drift de views = ⚠️.

### Passo 3 — Verificação B: Skills com Erros

1. Use `glob` em `<config>/skills/*/SKILL.md` para listar skills.
2. Para cada skill, verifique:
   - `name` == nome da pasta (convenção)
   - `description` está presente e não está vazia
   - Seções obrigatórias presentes: `## Capacidades`/`## Tools`,
     `## Modo de erro`, `## Algoritmo`/`## Workflow`
3. Busque por padrões de erro comum (case-insensitive): `TODO`, `FIXME`, `PLACEHOLDER`,
   `<descrição>`, `[NÃO IMPLEMENTADO]` no corpo.

### Passo 4 — Verificação C: Artefatos Órfãos

1. Liste todos os artefatos no módulo (estrutura de subpastas).
2. Para cada artefato, verifique se é referenciado em pelo menos um
   outro artefato (usando `grep`).
3. Artefatos não referenciados por ninguém são "órfãos".

### Passo 5 — Verificação D: Staleness dos Artefatos

1. Para cada artefato, verifique a data da última modificação
   (`git log -1 --format=%ai -- <arquivo>` se em Git).
2. Classifique por idade:
   - < 7 dias → ✅ Fresco
   - 7-30 dias → ⚠️ Stale
   - > 30 dias → ❌ Muito stale

### Passo 6 — Verificação E: Propagation Status

1. Leia `quality/PROPAGATION_STATUS.md` (se existir).
2. Conte entradas pendentes (CRs aprovados sem reflexo).
3. Verifique se há CRs antigos (> 7 dias) ainda pendentes.

### Passo 7 — Verificação F: Conformidade com CHECKLISTS.md

Execute automaticamente os itens verificáveis das checklists de
config contra cada componente:

**Para cada skill:**
1. Front matter: `name` == pasta, lowercase-hyphen, ≤ 64; `description`
   com triggers; campo proibido `tools:` ausente
2. Seções obrigatórias presentes (Lê/Gera/Nunca edita/Capacidades/Modo de
   erro/Algoritmo ou Workflow/Casos de Borda)
3. Caminhos na seção Lê (subpasta primeiro, fallback raiz)

**Para cada subagent (mode: subagent):**
4. `name` casa com arquivo, ≤ 64 chars; `mode: subagent`; temp 0.1/0.2
5. Campo proibido `tools:` ausente; `permission:` presente
6. `edit`/`write` nunca `allow` amplo (deny/ask/restrito)
7. **Portabilidade:** nenhum path de máquina absoluto no front
   matter (padrões `C:/...`, `D:/...` com usuário/sistema → violação)
8. **Referências de skills:** para cada skill citada no arquivo (tabela
   de skills, seção Lê, Capacidades, Algoritmo/Workflow) — de subagents E
   primary agents:
   - Existe a pasta da skill? **Não** → a citação
     está qualificada como condicional ("quando existir no
     projeto", "(se existir)")? Sim → ⚠️ registrar como
     dependência externa; não → ❌ referência órfã
   - Existe mas NÃO está registrada no registry? → ❌ não registrada
   - Está no registry mas a pasta não existe (ou vice-versa)? → ❌ drift
8b. **Referências de tools/MCP — escopo de capacidades:** para cada
    componente (skills, subagents E primary agents), extraia as tools/MCP
    citados no corpo e verifique:
    - Tool citada não existe no catálogo da sessão corrente nem está
      configurada → ⚠️
      finding **não-bloqueante** na seção de referências pendentes.
    - Tool é **hard-dependency** do fluxo sem guard de disponibilidade
      (sem cláusula de fallback documentada) → ⚠️ finding não-bloqueante.
    - Tool citada com guard/fallback documentado ou marcada como opcional
      → ✅.
    Nota: check textual — compare contra as capacidades realmente
    disponíveis na sessão em que o relatório roda e registre esse contexto.

**Para cada primary (mode: primary):**
9. Sem campo `name:`; `temperature: 0.3`; permissão ampla; seções
   (Identidade, Precedência, Subagents, Roteamento, Comportamento)

**Integridade estrutural de front matter — transversal a todo agent e SKILL.md:**

10. **Delimitadores íntegros:** a 1ª linha do arquivo deve ser exatamente
    `---` e deve existir linha de fechamento exatamente `---` após o bloco
    YAML (antes de qualquer conteúdo). Malformado (` --`, `—`, trailing
    space, ausente) = ❌ FINDING **crítico** (não warning): o parser do
    engine descarta o front matter INTEIRO em silêncio — `permission:`,
    `mode` e `temperature` caem nos defaults do runtime, sem erro nenhum.
    Caso real documentado: ` --` no fim do front matter de um primary
    derrubou o `ask: allow` do agente.
10b. **BOM no início do arquivo:** o arquivo NÃO deve começar com
    BOM UTF-8 (`EF BB BF` / U+FEFF) — o primeiro byte legível deve ser o `---`
    do front matter. BOM presente = FINDING **crítico** (mesma classe:
    o parser pode não reconhecer o `---` com BOM como delimitador e
    descartar o front matter em silêncio). Caso real documentado: os 3
    primaries foram reescritos pelo runtime com BOM durante uma
    instalação de skill. Verificação barata: ler os 3 primeiros bytes.
11. **YAML parseável:** o bloco entre delimitadores deve ter estrutura
    `chave: valor` consistente — tabs, indentação quebrada ou chave solta
    sem valor onde mapping é esperado = ❌ FINDING.
12. **Permissões declaradas (agents):** o bloco `permission:` deve existir
    dentro dos delimitadores, com as chaves esperadas do componente. Lembrar:
    as checagens 1-9 acima leem o ARQUIVO e podem parecer conformes mesmo
    quando o runtime descarta o front matter — as checagens 10-12 são o que
    tornam o defeito DETECTÁVEL no texto.
13. **Confirmação em runtime (prova definitiva, quando disponível):**
    comando de debug do engine para o agent — se o campo `prompt` do JSON
    começa com o YAML cru (`---...`), o front matter NÃO foi parseado; se
    `mode` ou `temperature` no JSON divergirem do declarado no arquivo, idem.
    Registrar como ❌ confirmado em runtime (aplica-se só a agentes
    carregáveis).

**Itens não automatizáveis** (qualidade de conteúdo, smoke test) ficam
como "revisão humana" no relatório.

Saída: tabela por componente — ✅ conforme | ⚠️ item com exceção a
justificar | ❌ violação.

### Passo 7b — Verificação G: Índice lazy do AGENTS.md global

1. Leia o AGENTS.md global e extraia as entradas do índice
   "Índice de Referências — lazy loading" (padrão: `` `rules/<arquivo>.md` ``).
2. Para cada entrada, verifique se o arquivo existe em `rules/` →
   **referência quebrada = ❌**.
3. Para cada arquivo em `rules/`, verifique se está referenciado no
   índice → arquivo órfão = ⚠️ (nunca será carregado).
4. Verifique a seção "Hierarquia de Convenções" → ausente = ❌.
5. Tamanho do AGENTS.md global: > 60 linhas = ⚠️ (teto: 40;
   absorve variação de blocos locais).
6. Marcadores de blocos locais presentes (compat do setup) → ausentes = ⚠️.

Saída: tabela entrada ↔ arquivo ↔ status + contagem de linhas.

### Passo 7c — Verificação H: coerência front matter ↔ PERMISSIONS.md

1. Leia o `PERMISSIONS.md` (matriz agent × permissão com justificativa e status).
2. Para cada agent, extraia o bloco `permission:` do front matter e compare
   com a linha correspondente da matriz.
3. Divergência = finding:
   - severidade `error` se o front matter for MAIS restritivo que a matriz
     documentada (matriz desatualizada) OU se a matriz for MAIS restritiva
     que o front matter (permissão real escapando do menor privilégio documentado);
   - severity `warning` se for divergência apenas de nota/justificativa.
4. `PERMISSIONS.md` ausente = ⚠️ (matriz não mantida — criar via
   gm-workflow-improver).
5. Correção: atualizar a matriz OU o front matter via gm-workflow-improver —
   nunca "consertar" a matriz para refletir allow amplo sem justificativa.

### Passo 7d — Verificação I: Varredura estrutural (`gm_lint`)

Varredura estrutural do config/pack via tool `gm_lint` (ou CLI
`golmet lint --workspace .`): refs mortas (L1), órfãos (L3), manifest stale
(L4), nomes fora do padrão (L5), componentes fantasma (L6), estrutura
canônica (L7) e conformidade definitivo↔template (L8).

- **Quando:** rode após mudanças estruturais (renomear/mover/apagar
  arquivos, morte de modelo) e inclua o resultado no relatório de saúde.
- **Gatilhos de usuário:** "verifica se tem inconsistência", "está tudo em
  ordem estrutural?" — qualquer pedido nesse sentido roda o lint.
- **Correção:** o lint APONTA, o monitor corrige o que for trivial
  (paths mortos, renomeações kebab-case, entradas stale de manifest);
  decisões (apagar órfão com dado, ampliar allowlist) → operador.
- Findings entram na seção de saúde do relatório com severidade + remediação.

### Passo 8 — Montar e salvar relatório

Combine as 9 verificações (A-I) num relatório único com resumo executivo,
seções detalhadas e recomendações. Salve em
`<modulo>/<projeto>/quality/AGENT_HEALTH_REPORT_<timestamp>.md`.

## Template — AGENT_HEALTH_REPORT_<timestamp>.md

```yaml
---
project: <NOME_DO_MODULO>
module: agent-health-report
audience: [governanca, tecnica]
status: draft
generated_at: "YYYY-MM-DDTHH:mm:ssZ"
---
```

```markdown
# Agent Health Report — <nome-do-modulo>

> **Gerado em:** DD/MM/AAAA HH:MM
> **Resultado Geral:** ✅ Saudável | ⚠️ Atenção necessária | ❌ Crítico

---

## Resumo

| Verificação | Status | Detalhes |
|-------------|--------|----------|
| A. Subagents | ✅/⚠️/❌ | N/N/N |
| B. Skills | ✅/⚠️/❌ | N/N/N |
| C. Artefatos Órfãos | ✅/⚠️/❌ | N órfãos |
| D. Staleness | ✅/⚠️/❌ | N stale |
| E. Propagation | ✅/⚠️/❌ | N pendentes |
| F. Conformidade (checklists) | ✅/⚠️/❌ | N conformes / N violações |
| G. Índice lazy (AGENTS.md) | ✅/⚠️/❌ | N refs OK / N quebradas |

---

## A. Subagents Registrados

| Subagent | Name | Description | Permission | Status |
|----------|------|-------------|------------|--------|

**Referências quebradas:** <lista ou "nenhuma">

---

## B. Skills

| Skill | Front Matter | Seções | Placeholders | Status |
|-------|-------------|--------|--------------|--------|

**Skills com problemas:** <lista detalhada>

---

## C. Artefatos do Módulo

| Artefato | Existe | Referenciado | Última atualização | Staleness |
|----------|--------|-------------|--------------------|-----------|

**Artefatos órfãos:** <lista>

---

## D. Staleness

| Faixa | Quantidade | Artefatos |
|-------|-----------|-----------|
| ✅ < 7d | N | <lista> |
| ⚠️ 7-30d | N | <lista> |
| ❌ > 30d | N | <lista> |

---

## E. Propagation Status

<se PROPAGATION_STATUS.md existir, listar entradas pendentes;
senão, "Não avaliado — PROPAGATION_STATUS.md não encontrado">

---

## F. Conformidade com CHECKLISTS.md

| Componente | Tipo | Front Matter | Permissões | Portabilidade | Seções | Registro | Status |
|------------|------|--------------|------------|---------------|--------|----------|--------|

**Violações:** <lista com componente, item da checklist, o que corrigir>
**Refs de tools/MCP pendentes (não-bloqueante):** <lista ou "nenhuma">
**Não automatizáveis (revisão humana):** <lista>

---

## G. Índice lazy do AGENTS.md global

| Entrada no índice | Arquivo existe | Tamanho | Status |
|-------------------|----------------|---------|--------|

**Referências quebradas:** <lista ou "nenhuma">
**Arquivos rules/ órfãos:** <lista ou "nenhuma">
**AGENTS.md global:** <N> linhas (teto: 40 + blocos locais; ⚠️ se > 60)

---

## H. Coerência PERMISSIONS.md

| Agent | Front matter | Matriz | Status |
|-------|--------------|--------|--------|

**Divergências:** <lista ou "nenhuma">

## Recomendações

1. <ação sugerida 1>
2. <ação sugerida 2>
3. <ação sugerida 3>
```

## Casos de Borda

- **Nenhum módulo encontrado:** gere relatório genérico apenas com
  verificações A e B (agents e skills do workspace).
- **Módulo pré-estrutura de subpastas:** verifique artefatos na raiz
  e note no relatório que o módulo ainda não foi migrado.
- **Verificação D sem Git:** pule a verificação de staleness por data
  e marque como "Não avaliado — sem acesso a Git".
- **Dados parciais:** gere o relatório mesmo com verificações incompletas.
  Cada verificação não avaliada é marcada como "N/A".
- **Relatório anterior existe:** não sobrescreva — gere um novo com
  timestamp diferente (permite diff entre execuções).