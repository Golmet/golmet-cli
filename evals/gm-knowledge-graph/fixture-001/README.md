# fixture-001 — gm-knowledge-graph

Workspace sintético T1-T5 + determinismo + from-json + strict (ver descrição
completa em `criteria.yaml`).

## Protocolo de execução (sandbox isolado)

Onde `<script>` = `scripts/build-knowledge-graph.mjs` da skill
(`coredata/sources/gm-knowledge-graph/scripts/` no repo do kit;
`skills/gm-knowledge-graph/scripts/` no config materializado).

```bash
# 1. semear o sandbox com TODO o conteúdo de input/
cp -r input/* <sandbox>/

# 2. extração global (caminhos RELATIVOS ao sandbox — os baselines de
#    output-expected/ foram congelados nesse formato)
cd <sandbox>
node <script> --config-root config --packs-root config --pack demo \
  --projects-root projects --out knowledge-graph

# 3. regeneração do grafo congelado (sem re-parseio)
node <script> --from-json knowledge-graph/graph.json --out knowledge-graph-fromjson

# 4. strict: config sem registry deve ABORTAR (exit 1) sem output
node <script> --config-root config-vazia --packs-root config-vazia --pack demo \
  --projects-root projects --out kg-abort ; echo "exit=$?"

# 5. pontuar
golmet eval <fixtures-root>/gm-knowledge-graph/fixture-001 --workspace <sandbox>
```

## Congelamento do baseline (output-expected/)

Os arquivos em `output-expected/` são a saída do passo 2 congelada. Para
regenerar de propósito (após mudança INTENCIONAL de template/formato): rodar o
passo 2 num sandbox fresco, copiar os 3 artefatos para `output-expected/` e
revisar o diff — mudanças não intencionais são regressão.
