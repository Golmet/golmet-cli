# fixture-001 - gm-knowledge-graph

Workspace sintético T1-T5 + **paridade Go-Node** + determinismo + from-json +
strict (ver descrição completa em `criteria.yaml`; ADR-019).

## Baselines em `output-expected/`

| Diretório | Origem | Papel |
|---|---|---|
| `knowledge-graph/` | **motor Go** (`golmet graph`, re-baseline 21/11/2026) | OUTPUT CANÔNICO do protocolo (passo 2): estado final do workspace |
| `knowledge-graph-run2/` | **motor Go** (2ª execução, passo 3) | DETERMINISMO: igual ao run1 (criteria tolera `gerado_em`) |
| `knowledge-graph-fromjson/` | **motor Go** (`--from-json`, passo 4) | RE-RENDER do grafo congelado, sem re-parseio |

O `corpus_selfscore_test.go` semeia o workspace com `input/` e ESPELHA
`output-expected/` como estado final — por isso os baselines incluem os 3
diretórios de saída do protocolo completo.

## Protocolo de execução (sandbox isolado)

Onde `<golmet>` = o binário do kit (`golmet` no PATH, ou `go run ./cmd/golmet`
no repo; a tool MCP `gm_graph` é equivalente).

```bash
# 1. semear o sandbox com TODO o conteúdo de input/
cp -r input/* <sandbox>/

# 2. extração global (caminhos RELATIVOS ao sandbox - os baselines foram
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

## Re-baseline dos baselines (`output-expected/`)

- **21/11/2026 (motor Go, commits 9b1bdaf..6bcef47)**: re-baseline completo -
  mudanças INTENCIONAIS do template (setas direcionais escaláveis, fluxo
  animado no highlight, painel de Vizinhos agrupado) + convenção `pertence_a`
  (componente -> container; BREAKING no graph.json). Os baselines passaram a
  espelhar o ESTADO FINAL do workspace do protocolo (knowledge-graph/ +
  knowledge-graph-run2/ + knowledge-graph-fromjson/) - é o que o self-score
  (`corpus_selfscore_test.go`) semeia e valida.
- **Histórico**: o golden de paridade Go-Node (extrator .mjs congelado em
  20/11/2026, gerado no commit `87b78d1` - o .mjs foi removido na release do
  motor Go, ADR-019) está no git histórico deste diretório.
- **Para re-baseline após mudança INTENCIONAL**: rodar o protocolo (passos
  2-4) num sandbox fresco com o binário novo, copiar os 3 diretórios de saída
  para `output-expected/` e revisar o diff.
