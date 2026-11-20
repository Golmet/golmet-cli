# fixture-001 — gm-knowledge-graph

Workspace sintético T1-T5 + **paridade Go↔Node** + determinismo + from-json +
strict (ver descrição completa em `criteria.yaml`; ADR-019).

## Baselines em `output-expected/`

| Arquivo | Origem | Papel |
|---|---|---|
| `graph.json` | **extrator Node (.mjs)** — congelado em 20/11/2026 | PARIDADE: o Go deve ser byte-igual (exceto `gerado_em`) |
| `knowledge-graph.html` | **extrator Node (.mjs)** | PARIDADE (inclui o embed JSON do grafo) |
| `knowledge-graph.md` | **motor Go** (`golmet graph`) | DETERMINISMO do md (linha de proveniência diverge do .mjs por design — ADR-019) |

## Protocolo de execução (sandbox isolado)

Onde `<golmet>` = o binário do kit (`golmet` no PATH, ou `go run ./cmd/golmet`
no repo; a tool MCP `gm_graph` é equivalente).

```bash
# 1. semear o sandbox com TODO o conteúdo de input/
cp -r input/* <sandbox>/

# 2. extração global (caminhos RELATIVOS ao sandbox — os baselines foram
#    congelados nesse formato)
cd <sandbox>
<golmet> graph --config-root config --packs-root config --pack demo \
  --projects-root projects --out knowledge-graph

# 3. segunda execução (determinismo)
<golmet> graph --config-root config --packs-root config --pack demo \
  --projects-root projects --out knowledge-graph-run2

# 4. regeneração do grafo congelado (sem re-parseio)
<golmet> graph --from-json knowledge-graph/graph.json --out knowledge-graph-fromjson

# 5. strict: config sem registry deve ABORTAR (exit 1) sem output
<golmet> graph --config-root config-vazia --packs-root config-vazia --pack demo \
  --projects-root projects --out kg-abort ; echo "exit=$?"

# 6. pontuar
golmet eval <fixtures-root>/gm-knowledge-graph/fixture-001 --workspace <sandbox>
```

## Congelamento dos baselines (output-expected/)

- **Paridade (graph.json/html)**: produzidos pelo extrator Node
  (`coredata/sources/gm-knowledge-graph/scripts/build-knowledge-graph.mjs` no
  commit `87b78d1` — o arquivo foi REMOVIDO na mesma release do motor Go,
  ADR-019; o baseline congelado é o registro do comportamento de referência).
  Para regenerar seria preciso recuperar o arquivo do git histórico.
- **md**: produzido pelo motor Go. Para regenerar após mudança INTENCIONAL de
  template: rodar o passo 2 num sandbox fresco, copiar o md para
  `output-expected/` e revisar o diff.
