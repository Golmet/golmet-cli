# Auditoria de Qualidade dos Agents RDD (via Jev)

- **Data:** 2026-09-27
- **Escopo:** todos os agents do workflow RDD em `~/.config/opencode/agents/rdd-*.md` — 20 componentes (3 primaries + 17 subagents). Os 34 agents `gsd-*` do mesmo diretório pertencem ao workflow GSD e ficaram fora do escopo "workflow RDD".
- **Modelo:** Jev `jev-1.13.0` (retornado pela API em todas as 20 chamadas)
- **Total avaliado:** 20/20 — nenhum componente com erro de avaliação
- **Score médio geral:** 86,5 (primaries: 90,1 · subagents: 85,9)
- **Distribuição por severidade:** ok = 11 · atenção = 9 · crítico = 0

## Tabela Completa (ordenada por score crescente)

Score = média normalizada de Clareza, Fidelidade, Estrutura e Acionabilidade (score 0–2 → ×50; noul → ×100). Escala 0–100.

| Agent | Tipo | Clareza | Fidelidade | Estrutura | Acionabilidade | Score | Severidade |
|---|---|---:|---:|---:|---:|---:|---|
| rdd-github-orchestrator | subagent | 99,0 | 41,0 | 51,0 | 80,0 | 67,8 | atenção |
| rdd-jev-classifier | subagent | 100,0 | 55,0 | 35,5 | 88,5 | 69,8 | atenção |
| rdd-support-orchestrator | subagent | 99,5 | 15,0 | 97,5 | 93,5 | 76,4 | atenção |
| rdd-quality-orchestrator | subagent | 98,5 | 15,0 | 99,0 | 96,0 | 77,1 | atenção |
| rdd-translation-orchestrator | subagent | 99,0 | 43,0 | 98,5 | 87,5 | 82,0 | atenção |
| rdd-progress | subagent | 99,5 | 58,0 | 98,5 | 90,0 | 86,5 | atenção |
| rdd-requirements-orchestrator | subagent | 99,5 | 56,0 | 99,0 | 95,5 | 87,5 | atenção |
| rdd-workflow-improver | subagent | 100,0 | 59,0 | 98,5 | 97,5 | 88,8 | ok |
| rdd-dev | primary | 99,5 | 70,0 | 93,5 | 94,0 | 89,3 | atenção |
| rdd-workflow-builder | subagent | 100,0 | 67,0 | 98,5 | 93,5 | 89,8 | ok |
| rdd-reverse-orchestrator | subagent | 99,5 | 69,0 | 96,5 | 94,5 | 89,9 | ok |
| rdd-next | subagent | 100,0 | 63,0 | 98,5 | 99,5 | 90,3 | ok |
| rdd-doc | primary | 100,0 | 76,0 | 88,5 | 97,0 | 90,4 | ok |
| rdd-req | primary | 100,0 | 76,0 | 89,0 | 97,0 | 90,5 | ok |
| rdd-cr-propagation-guard | subagent | 100,0 | 67,0 | 98,5 | 99,0 | 91,1 | atenção |
| rdd-intake-orchestrator | subagent | 100,0 | 71,0 | 99,0 | 98,0 | 92,0 | ok |
| rdd-dev-backend | subagent | 99,5 | 71,0 | 99,5 | 99,5 | 92,4 | ok |
| rdd-dev-frontend | subagent | 99,5 | 72,0 | 99,5 | 99,5 | 92,6 | ok |
| rdd-governance-orchestrator | subagent | 100,0 | 74,0 | 98,0 | 98,5 | 92,6 | ok |
| rdd-figma-orchestrator | subagent | 100,0 | 75,0 | 99,0 | 98,5 | 93,1 | ok |

## Críticos e Atenção

Nenhum componente com severidade `critico`. Abaixo, os 9 com score <90-relacionado ou severidade ≠ ok:

- **rdd-github-orchestrator (67,8, atenção)** — Pior score geral. Estrutura 51: é o único orquestrador sem as seções padrão (Princípios/Lê/Gera/Formato de saída/Casos de Borda); o H3 "Skills Transversais" aparece aninhado dentro do bloco `BEGIN_REGISTRY` (erro estrutural do render). Fidelidade 41: description promete "gerencia milestones" sem instrução correspondente no corpo, e a permissão `write: allow` sobre os SKILL.md das skills contrasta com o princípio "roteia, não executa".
- **rdd-jev-classifier (69,8, atenção)** — Estrutura 35,5: arquivo mínimo (56 linhas) sem nenhuma seção padrão RDD (sem Lê/Gera, Nunca edita, Casos de Borda, formato de saída estruturado). Fidelidade 55: o "formato das questions" documentado (`noul` com `instructions`) diverge do schema validado da API (`noul` com `criteria`) — risco de payload rejeitado.
- **rdd-support-orchestrator (76,4, atenção)** — Fidelidade 15 (a mais baixa do conjunto): description promete só "known-errors e migração de dados", mas o corpo roteia 3 skills, incluindo `rdd-incident-postmortem` (com tabela, pré-condições e casos de borda próprios) — habilidade existente e invisível na porta de entrada.
- **rdd-quality-orchestrator (77,1, atenção)** — Fidelidade 15: description lista 5 domínios, mas a tabela roteia 8 skills — `rdd-wiki-lint`, `rdd-perf-auditor` e `rdd-azure-tasks-ingestor` não anunciados. Corpo rico; vitrine defasada.
- **rdd-translation-orchestrator (82,0, atenção)** — Fidelidade 43: a rota "gerar user stories" aponta para `rdd-gerar-especificacao-us`, skill ausente da tabela "Skills disponíveis" (que lista 6; a description promete 7 domínios). Um primary que consultasse só a tabela não rotearia user stories.
- **rdd-progress (86,5, atenção)** — Fidelidade 58/acionabilidade 90: identidade diz "5 blocos" enquanto a description lista 6 itens (tasks separadas do gate); `edit: ask`/`write: ask` genéricos quando a única escrita legítima é `PROGRESS_REPORT.md` com `--save` (padrão de escopamento do `rdd-cr-propagation-guard` seria o ideal).
- **rdd-requirements-orchestrator (87,5, atenção)** — Fidelidade 56: description menciona "transcrição de reuniões" como skill centralizada, mas não há skill de transcrição na tabela (é apenas entrada alternativa de `rdd-requirements`) — promessa sem componente.
- **rdd-dev (89,3, atenção)** — Fidelidade 70: referência morta a `@rdd-azure-orchestrator` na seção Comportamento ("PRs seguem @rdd-azure-orchestrator") — o componente foi substituído pelo `rdd-github-orchestrator`. Mesmo defeito estrutural de registro do github-orchestrator: H3 "Skills Transversais" dentro do bloco `BEGIN_REGISTRY`.
- **rdd-cr-propagation-guard (91,1, atenção)** — Fidelidade 67: o Passo 9 mapeia `USER_STORIES.md → rdd-gerar-especificacao-us`, mas USER_STORIES não está entre os 6 artefatos escaneados ( Passo 5) — regra de re-execução para artefato que o detector nunca classifica como stale.

## Recomendações (top 5, priorizadas)

1. **Reestruturar `rdd-github-orchestrator` no padrão RDD de orquestradores** (Princípios, Lê/Gera, Formato de saída, Casos de Borda) e corrigir o registro do grafo — responsável: `@rdd-workflow-improver` (estrutura) + `rdd-registry-render` (H3 aninhado no registry).
2. **Sincronizar descriptions com o corpo real** em `rdd-support-orchestrator`, `rdd-quality-orchestrator` e `rdd-translation-orchestrator` (listar as skills efetivamente roteadas, incluir post-mortem/wiki-lint/perf-auditor/azure-tasks-ingestor; resolver a skill fantasma `rdd-gerar-especificacao-us` ou adicioná-la às tabelas) — responsável: `@rdd-workflow-improver`.
3. **Elevar `rdd-jev-classifier` ao padrão estrutural RDD** (Lê/Gera, Nunca edita, Casos de Borda, formato de saída) e corrigir o formato de questions para o schema validado (`noul` com `criteria`, não `instructions`) — responsável: `@rdd-workflow-improver`.
4. **Corrigir referência morta `@rdd-azure-orchestrator` → `@rdd-github-orchestrator` em `rdd-dev`** e normalizar o bloco registry do mesmo arquivo (H3 "Skills Transversais" fora do `BEGIN_REGISTRY`) — responsável: `@rdd-workflow-improver` + `rdd-registry-render`.
5. **Fechar o loop de escrita do `rdd-progress`**: escopar `edit`/`write` apenas a `PROGRESS_REPORT.md` (padrão usado pelo `rdd-cr-propagation-guard`) e alinhar "5 blocos" da identidade com os 6 itens da description — responsável: `@rdd-workflow-improver`.

Padrão transversal: **clareza de escopo (média 99,8) e acionabilidade (média 95,4) são excelentes em toda a frota**; os gaps concentram-se em **fidelidade description↔corpo (média 62,6)** — drift de manutenção nas descriptions após evolução das skills roteadas — e em **2 pontos de estrutura** (github-orchestrator e jev-classifier). Sugere-se incluir "description deve listar o inventário real de skills" no checklist do `rdd-skill-eval`/golden set para prevenir recorrência.

## Metodologia

- **Fonte:** glob `~/.config/opencode/agents/*.md` → 54 arquivos; filtrados os 21 do workflow RDD (prefixo `rdd-`); 1 arquivo (`rdd-azure-orchestrator`) não existe mais — total avaliado 20.
- **Leitura:** arquivo completo lido para cada componente; digest compacto (front matter verbatim + seções H1/H2 + linhas representativas de identidade, princípios, Lê/Gera, workflow e formato de saída) submetido ao Jev.
- **Questões Jev** (schemas validados contra jev-1.13.0; score→criteria lista, noul→criteria dict, choice→criteria dict):
  - `clareza_escopo` (score 0–2): a description comunica inequivocamente quando delegar?
  - `fidelidade_desc_corpo` (noul): o corpo cumpre o que a description promete? → p×100
  - `completude_estrutural` (score 0–2): seções padrão RDD presentes e preenchidas?
  - `acionabilidade` (score 0–2): gatilhos permitem decisão autônoma do primary?
  - `severidade_gap` (choice): ok / atenção / crítico
- **Score final do componente** = média das 4 dimensões normalizadas ×100. Severidade = argmax do choice.
- **Persistência incremental:** snapshot gravado no mesmo arquivo aos 16/20 componentes (proteção contra interrupção), sobrescrito pelo relatório final.
- **Limitações:**
  - O Jev avaliou digests, não os arquivos verbatim integrais — decisões de nuance muito longa no corpo podem ter sido simplificadas (nenhum caso exigiu reavaliação).
  - Julgamento de modelo único, sem inter-rater; noul de fidelidade carrega incerteza inerente (confidence típica 0,6–0,8).
  - Análise estática: nenhum agent foi executado nem testado com evals — a auditoria mede qualidade de especificação, não comportamento em runtime.
  - Scores individuais não devem ser comparados com precisão decimal — diferenças <2 pontos estão dentro do ruído do modelo.
  - Os 34 agents `gsd-*` (workflow GSD) não foram avaliados nesta auditoria.
