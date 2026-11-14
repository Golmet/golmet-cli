# gm-knowledge-graph — Grafo Navegável da Base de Conhecimento (T1-T5)

Gera um grafo tipado (nós categorizados por store/camada, arestas rotuladas) a partir das
fontes REAIS de metadados do workflow — não de links inventados — e o emite como HTML
self-contained (estilo graph view do Obsidian, mas com hierarquia e arestas verdadeiras),
fallback Mermaid em Markdown e graph.json (nós/arestas crus, consumível por outras tools).

Motivação: abrir os repos como vault no Obsidian ficou ruim — arquivos
soltos, sem links, sem hierarquia. Esta skill deriva as arestas das estruturas que JÁ
existem (registry, índices, TRACEABILITY, front matter) em vez de exigir links manuais.

## Diferenciação

| Componente | O que gera | Por que não substitui |
|---|---|---|
| gm-registry-render | Mermaid ESTÁTICO só da delegação (registry → views) | Não cobre T1-T4; não é interativo; só views do registry |
| Kit golmet `progress` | Relatório TEXTUAL do pipeline (cobertura, drift, pendências) | Sem visualização; sem nós de conhecimento |
| gm-knowledge-auditor | Métricas de CONSUMO de conhecimento (hit rate, tokens) | Mede uso, não estrutura |
| MCP de grafo de código | Grafo de CÓDIGO (tree-sitter/LSP) | Domínio do código-fonte, não da base de conhecimento |
| gm-knowledge-graph (esta) | Grafo navegável UNIFICADO dos stores T1-T5 | — |

## Fundação de conhecimento

- Base canônica consumida (T1): knowledge/knowledge-management/KNOWLEDGE.md (v1.5.0 — arquitetura
  dos stores T1-T5, padrões de memória/grafo de conhecimento) — base VÁLIDA (proxima_revisao
  2027-04-21), reuso direto, NÃO re-fundar.
- Decisão de fundação (builder, 05/11/2026): padrões de renderização de grafo visual
  (d3/vis-network/cytoscape) são DETALHE DE IMPLEMENTACÃO desta skill — o layout
  force-directed é embutido em vanilla JS (zero dependência, file:// offline) e a skill não
  faz julgamento de domínio sobre visualização. Sem fundação nova; se a skill passar a emitir
  julgamentos visuais (ex.: clustering semântico), aí sim fundar domínio via gm-domain-research.

## Lê (upstream)

Fontes read-only (parse tolerante — fonte ausente vira aviso, não aborta, exceto registry):

- **T5 (workflow)**: registry.yaml no config do engine (OBRIGATÓRIO — delegação, invokes, domains),
  AGENTS.md do config (índice lazy de rules/ + menções a knowledge/packs)
- **T1 (conhecimento)**: knowledge/INDEX.md do config (domínios, versão, data_pesquisa,
  proxima_revisao → flag stale)
- **T3 (rules)**: rules/*.md do config (exceto *.template.md) + índice lazy do AGENTS.md
- **T2 (acúmulo)**: packs/<pack-ativo>/accumulation/patterns/INDEX.md (tabela do
  Índice: padrão, tipo, categoria, projetos, maturidade, data_ultima_revisao, status)
- **T4 (módulos)**: em <projects-root> (ex.: diretorio_projetos do AGENTS.md), por produto:
  REQUIREMENTS.md (RNs/PENDs + promoção PEND→RN), DECISIONS.md (ADRs/DECs e refs),
  quality/TRACEABILITY.md (matriz RN x downstream com OK/FALHA),
  delivery/**/RN-NNN — <título> — Feature.md (front matter status + âncora; fallback
  "— Tasks.md" IMP-092), specs/*.md
- Referências textuais knowledge/<domínio>/KNOWLEDGE.md em agents/*.md e skills/*/SKILL.md
  → arestas fundado_em
- Fundação: knowledge/knowledge-management/KNOWLEDGE.md (T1)

## Gera (downstream)

Artefato de leitura do operador — NUNCA grava no diretório de config do engine fora da
skill própria (regra do bloco de acúmulo do AGENTS.md). Destino padrão: <cwd>/knowledge-graph/ (parametrizável via --out).

- knowledge-graph/graph.json — grafo cru: { meta, nodes[{id,type,label,store,attrs}], edges[{from,to,type,attrs}] }
- knowledge-graph/knowledge-graph.html — visualização self-contained (inline JS/CSS, canvas
  force-directed, filtro por store/tipo, busca, highlight de vizinhos, painel de atributos,
  auto-fit do viewport pós-layout (bbox + margem 40px) com botão "reset view", layout
  assíncrono em chunks com indicador de progresso no #stats (grafos >= 200 nós; < 200
  síncrono), pan suave até o nó selecionado e clique que seleciona nó (threshold de 4px
  distingue clique de arrasto; clique/pan/zoom funcionam também durante o layout assíncrono))
  — abre no browser via file://, sem rede
- knowledge-graph/knowledge-graph.md — resumo (contagem por tipo, bases T1 stale, avisos) + Mermaid top-N
- Atributos úteis nos nós: T1 versao/stale; T2 status/projetos; T4 feature status/ancora_legada

## Nunca edita

- Qualquer fonte lida (read-only nas fontes)
- O diretório de config do engine (a skill não escreve nada na config)
- Artefatos de módulo; evals/golden-human/ (golden set humano — intocado por agents)

## Tools disponíveis

- `shell` (executar o script determinístico scripts/build-knowledge-graph.mjs com Node)
- `file` (inspeção pontual das fontes e do output para o relatório)
- `glob` (localizar módulos/repo de docs)
- `ask` (confirmar modo global vs delta e destino quando ambíguo)

> **Escopo de tools (IMP-148):** o script exige Node >= 18 (node --version valida). Se Node
> estiver indisponível, NÃO improvise parse manual em escala — registre pendência IMP-080
> (instalar Node) e encerre sem output parcial. Sem rede: o HTML é self-contained por
> construção (canvas + JS inline); qualquer script externo no HTML é bug.

## Modo de erro

warn — fonte T1-T4 ausente ou unparseável: aviso, grafo parcial (o grafo mostra o que
existe). strict para UMA exceção: registry.yaml ausente/inválido ABORTA (sem ele não há
camada T5, que é a espinha dorsal do grafo). Output sempre integral: ou escreve os 3
artefatos, ou nada (escreve apenas após montar o grafo em memória).

## Formato de commit

    docs(knowledge): regenera grafo da base de conhecimento

## Algoritmo

### Passo 1 — Resolver parâmetros

- --config-root (default: diretório de config do engine), --packs-root (default config-root),
  --projects-root (ex.: o diretorio_projetos do AGENTS.md — sem ele, T4 é pulado com aviso),
  --out (default <cwd>/knowledge-graph/), --scope <modulo> (delta: T1-T5 integrais + APENAS
  o módulo nomeado; sem scope = global).
- Global vs delta: se o operador não especificou e há ambiguidade (muitos módulos), pergunte
  (1 pergunta agrupada — IMP-043). Delta para trabalho focado em um produto.

### Passo 2 — Rodar o extrator determinístico (R4)

Comando (uma linha):

    node <skill-dir>/scripts/build-knowledge-graph.mjs --config-root <config> --projects-root "<diretorio_projetos>" --out <destino> [--scope <modulo>]

O script parseia as fontes (lista fechada de parsers — mini-parser de registry.yaml, INDEX.md,
lazy index, patterns INDEX, REQUIREMENTS/DECISIONS/TRACEABILITY/front matter de Feature.md),
dedupe e ordena nós/arestas deterministicamente, aplica teto de nós (--max-nodes, default
3500 — poda primeiro RN/PEND do modo global e reporta em meta.podado) e escreve os 3 artefatos.

Regeneração sem re-parseio (grafo congelado — sem alterar dados): para reaplicar correção
de template a artefatos já gerados, use --from-json apontando para um graph.json existente:

    node <skill-dir>/scripts/build-knowledge-graph.mjs --from-json <caminho>/graph.json --out <destino>

### Passo 3 — Julgamento de ruído (o que NÃO é automático)

O script é determinístico; o julgamento é seu, lendo graph.json + avisos:

- Verifique nós órfãos suspeitos (ex.: rule fora do índice lazy é SINAL válido — reporte,
  não apague; pattern superado mantém nó com attr status: superado).
- Se um parser produziu lixo evidente (ex.: RNs de prose contadas como nós), ajuste o
  --scope/flags ou reporte o bug da skill para o gm-workflow-improver — NUNCA edite o
  output à mão (a idempotência morre).

### Passo 4 — Entregar

- Abra o HTML: start <destino>\knowledge-graph\knowledge-graph.html (Windows).
- Relatório curto: contagem de nós/arestas por store, bases T1 stale, avisos, modo usado.

## Template de saída

    knowledge-graph/
    ├── graph.json            # { meta: {modo, escopo, fontes, avisos, podado}, nodes, edges }
    ├── knowledge-graph.html  # self-contained — abrir no browser
    └── knowledge-graph.md    # resumo + Mermaid top-N + avisos

Tipos de nó: t1_dominio, t1_indice, t2_pack, t2_pattern, t3_rule, t5_config (AGENTS.md),
t5_primary, t5_subagent, t5_skill, t5_dominio, t4_modulo, t4_rn, t4_pend, t4_adr, t4_feature,
t4_spec.
Tipos de aresta: indice, indice_lazy, agrega, aplica_em, delega, invoca, invoca_direta,
owner, disponivel_para, pertence, orquestra, fundado_em, define, abre_questao, promove
(PEND→RN, IMP-094), ancora (RN→feature), cobre (RN→spec, attrs do TRACEABILITY),
decide (ADR→RN/PEND), artefato.

## Casos de Borda

- **registry.yaml ausente/YAML inválido**: ABORTA (exit 1) com o erro de parse — camada T5 é
  a espinha dorsal do grafo.
- **Módulo sem QUALITY/TRACEABILITY.md**: nós T4 de spec/RN sem arestas cobre — grafo
  parcial com aviso.
- **Feature com âncora PEND legada (pré-IMP-094)**: nó t4_feature com attr ancora_legada
  sinalizado; aresta promove NÃO é inventada (a promoção é papel do gm-requirements — IMP-094).
- **Grafo acima de 3500 nós**: poda determinística (RN/PEND do modo global primeiro) com
  meta.podado no graph.json — não falha.
- **Config do engine em máquina nova (sem knowledge/)**: grafo só com T3/T5 e avisos T1.
