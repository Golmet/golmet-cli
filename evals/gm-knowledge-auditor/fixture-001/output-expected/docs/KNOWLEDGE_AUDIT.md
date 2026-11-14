# Knowledge Audit — Hit Rate + Tokens de Conhecimento (MODO DEGRADADO)

> **Gerado em:** DD/MM/AAAA
> **Janela:** últimos 7 dias
> **Fonte:** banco do runtime (read-only)
> **Modo:** DEGRADADO — denominadores são PROXY (heurística de globs)

## Erro de coleta (declarado)

`DB_MISSING` — banco do runtime não presente no sandbox desta fixture.
Cenário de fallback padrão (IMP-080): contadores zerados explícitos e
métricas `N/A (sem dados)` — nunca zero fingido. O merge incremental no
METRICS.md **não é executado** (relatório sem dados nunca faz merge).

## Declaração PROXY (obrigatória)

1. Denominadores são **PROXY** — heurística de globs sobre tool calls de
   leitura, não medida calibrada de necessidade real de conhecimento.
2. Acionamento do bloco `knowledge-routing` em prosa não é observável no
   banco — o proxy não captura esse roteamento.
3. Tokens de conhecimento são estimados por bytes/4 sobre o tamanho atual
   do arquivo em disco — não no momento da leitura; leitura parcial conta
   como arquivo inteiro; primeira leitura por sessão+arquivo.
4. Calibração Jev (classificadores de relevância) é variante FUTURA da
   Fase 5, condicionada ao baseline de 2 ciclos (D17).

## Contadores brutos

| Contador | Valor |
|---|---|
| sessoes_observaveis | 0 |
| sessoes_que_precisavam | 0 |
| sessoes_com_hit | 0 |
| hits | 0 |
| sessions_hit_outside | 0 |
| unclassified_paths | 0 |
| blocks_skipped_status | 0 |
| erro_coleta | DB_MISSING |

## Hit Rate Global

| Métrica | Valor |
|---|---|
| Hit rate global | N/A (sem dados) |
| Tokens de conhecimento por tarefa (média) | N/A (sem dados) |
| Tokens de conhecimento (total da janela) | N/A (sem dados) |

## Hit Rate por Store

| Store | Local | Hits | Sessões com hit |
|---|---|---|---|
| T1 | `knowledge/**` | N/A (sem dados) | N/A (sem dados) |
| T2 | `patterns/INDEX.md` + `**/PATTERN.md` | N/A (sem dados) | N/A (sem dados) |
| T3 | `rules/**.md` | N/A (sem dados) | N/A (sem dados) |

## Hit Rate por Modelo (P-11)

N/A (sem dados) — nenhum modelo observado na janela.

## Janela por Dia

N/A (sem dados) — nenhum dia observável na janela.

## Metodologia e Limitações

- Fonte: banco SQLite do runtime, leitura exclusivamente read-only
  (node:sqlite, Node >= 22.5) — precedência gm-perf-auditor.
- Nesta fixture o banco não existe no sandbox: o erro é declarado acima,
  os contadores são zerados explícitos e nenhuma métrica é inventada.
- Interpretação contra P-09/P-11/AP-11/AP-13 exige 2 ciclos medidos (D17):
  decisão de infraestrutura NÃO é tomada com este relatório.
- Stores existentes no sandbox (T1 `knowledge/api-design/`, T2
  `accumulation/patterns/`, T3 `rules/stack-playbook.md`) foram mapeados
  para os globs do proxy — sem dados de leitura, nenhum hit é afirmado.
