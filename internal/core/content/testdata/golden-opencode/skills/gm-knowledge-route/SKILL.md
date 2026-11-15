---
name: gm-knowledge-route
description: >-
   Roteador de conhecimento do workflow — decide o DESTINO de um artefato a salvar quando não há gatilho determinístico ("salva isso aqui", "onde gravo X?", "isso é norma ou aprendizado?"). A tabela knowledge-routing (T1-T5) do AGENTS.md é o contrato; 4 perguntas de encaixe NA ORDEM: componente de engine (não roteia — gm-skill-creator/gm-subagent-creator), norma de comportamento (packs/<id>/rules/<assunto>.md — uma norma por arquivo com gatilho próprio; orienta edit no working copy + golmet pack publish, nunca escreve), fundamento perene de domínio (delega gm-domain-research) ou aprendizado de projeto concluído (delega gm-pattern-extractor/ gm-metrics-collector). Desempate calibrado via gm_jev_classify; sem a tool, julgamento manual pela ordem com incerteza declarada. Saída auditável: destino + escritor canônico + formato + porquê (1 linha). A skill NÃO grava conteúdo — roteia e delega. Modo de erro continue. Fora do escopo: gatilhos determinísticos (gm-workflow-builder, gm-project-retrospective) — destino resolvido pelo processo. Use quando o operador ou um agent quiser salvar algo sem destino óbvio.
requires: [file, glob, grep, ask, skill]
degrades: [edit, write, webfetch]
---

# gm-knowledge-route — Roteador de Conhecimento (T1-T5)

Decide o **DESTINO** de um artefato/estado que alguém quer salvar quando não
há gatilho determinístico — o caso genérico ("salva isso aqui", "onde gravo
X?", "isso é norma ou aprendizado?"). A tabela **knowledge-routing** do
AGENTS.md global (bloco `<!-- knowledge-routing -->`) é o **contrato**: esta
skill é o julgamento estruturado que a aplica, com saída auditável. A skill
**roteia e delega — nunca grava conteúdo**.

## Fronteira — fora do escopo (destino já resolvido pelo processo)

- `gm-workflow-builder` fundando base de domínio (T1) — `gm-domain-research`
  é gatilho obrigatório do processo dele.
- `gm-project-retrospective` extraindo padrão/métrica (T2) — hand-off direto
  a `gm-pattern-extractor`/`gm-metrics-collector`.
- Componente novo de engine (skill/agent do workflow) — caminho próprio do
  `gm-workflow-builder` (pergunta 1 abaixo só CONFIRMA e devolve).

Detectou um desses gatilhos no pedido → devolva ao processo, não roteie.

## Lê (upstream)

- Bloco `<!-- knowledge-routing -->` do AGENTS.md global — tabela T1-T5 (o
  contrato) e o roteamento de conhecimento novo
- Bloco `<!-- gm-pack-accumulation -->` do AGENTS.md global — pack ativo
  (`packs/<pack-ativo>/`) para resolver os destinos relativos
- `packs/<pack-ativo>/domains/INDEX.md` e
  `packs/<pack-ativo>/accumulation/patterns/INDEX.md` — o destino já existe?
  (roteamento vira ATUALIZAÇÃO — merge incremental)
- O texto/estado a salvar + contexto do pedido (quem pediu, em que fase)

## Gera (downstream)

- Nada em disco — o veredito de roteamento (destino + escritor canônico +
  formato esperado + porquê) no formato de saída abaixo
- A delegação acontece no executante (chamador ou `gm-knowledge-router`),
  não nesta skill

## Nunca edita

- **QUALQUER conteúdo** — a skill julga; quem grava são os escritores
  canônicos (`gm-domain-research`, `gm-pattern-extractor`,
  `gm-metrics-collector`) ou o próprio operador
- `rules/` de pack — norma é decisão da ORG: orienta o edit no working copy
  + `golmet pack publish`, nunca escreve a rule
- Componentes do workflow (`registry.yaml`, `ARCHITECTURE.md`, `AGENTS.md`)
- Config do engine como destino de conhecimento — a config do workflow é
  genérica/portável, sem conteúdo de negócio (mesmo padrão das skills de
  acúmulo — IMP-061/IMP-111)

## Tools

- `file`/`glob`/`grep` — ler os blocos do AGENTS.md e os índices de destino
- `ask` — ambiguidade real de sobra (nem ordem nem desempate resolvem)
- `gm_jev_classify` (tool MCP do kit — server `golmet`) — desempate
  calibrado nos casos 2-4 sobrepostos

## Algoritmo — perguntas de encaixe NA ORDEM

A tabela knowledge-routing é o contrato; as perguntas abaixo a operam.
Passe na ordem — a PRIMEIRA que encaixa vence (a ordem é o desempate
default). As perguntas 1-4 roteiam para as stores do contrato; a pergunta 5
é a **válvula de escape** — o roteador nunca deixa um "salva isso aqui" sem
rota:

1. **É componente que o engine carrega (skill/agent)?** → `skills/`/
   `agents/` do pack (via `gm-skill-creator`/`gm-subagent-creator` +
   `golmet pack publish`). NÃO roteie aqui — é componente de workflow, não
   conhecimento de negócio.
2. **É NORMA que muda o comportamento (gatilho "AO fazer X, leia isto")?**
   → `packs/<id>/rules/<assunto>.md` — **uma norma por arquivo com gatilho
   próprio** (a linha do lazy index nasce com o arquivo). Oriente o edit no
   working copy do pack + `golmet pack publish` — norma é decisão da org; a
   skill NÃO escreve rules.
3. **É fundamento perene de domínio (vale sem a org)?** →
   `packs/<id>/domains/<dominio>/` — DELEGUE ao `gm-domain-research`
   (formato: `KNOWLEDGE.md` com fontes primárias URL + data de acesso).
4. **É aprendizado de projeto concluído?** →
   `packs/<id>/accumulation/{patterns|metrics|nuances}/` — DELEGUE ao
   `gm-pattern-extractor` (padrão) ou `gm-metrics-collector` (métrica);
   nuances da org → `accumulation/nuances/<dominio>-nuances.md`.
5. **VÁLVULA DE ESCAPE — não é nenhum destes?** (rascunho, relatório
   ad-hoc, dump de análise, artefato sem função definida) → roteia para
   `.golmet/` do container (zona de trabalho, temporária — FORMATO §2.3)
   e pergunte ao operador se merece virar estrutura canônica (**promover**,
   via o escritor canônico correspondente) ou permanecer temporário. O
   temporário nasce NA ZONA, nunca solto na raiz — e morre ignorado pelo
   Git (`.golmet/` é gitignored no container).

## Casos ambíguos (2-4 sobrepostos)

Desempate calibrado com `gm_jev_classify` — questões no formato TypeSafe:

```json
{"destino": {"type": "choice", "criteria": {
  "norma": "muda comportamento com gatilho AO fazer X, leia isto (T3 rules/)",
  "dominio": "fundamento perene da área, vale sem a org (T1 domains/)",
  "aprendizado": "resultado/lição de um projeto concluído (T2 accumulation/)"
}}}
```

Input: o texto do artefato + as perguntas 2-4. Sem a tool disponível:
julgamento manual pela ordem das perguntas e **declare a incerteza** no
output (nunca disfarce o desempate de certeza).

## Contrato de saída

```text
═══ gm-knowledge-route ═══
conteúdo:  <resumo em 1 linha>
destino:   <path T1-T5 resolvido (ou relativo, sem pack ativo) | .golmet/<nome> temporário (válvula de escape, pergunta 5)>
escritor:  <skill canônica | operador (rules/componente) | operador (promover depois, válvula)>
formato:   <KNOWLEDGE.md | PATTERN.md | METRICS.md | RULES.md | skill/agent | livre (temporário)>
porquê:    <1 linha de justificativa — auditável | "sem função definida — válvula de escape">
incerteza: <baixa | média — critério do desempate | declarada (fallback manual)>
ação:      <delegar a <skill> | orientar edit + golmet pack publish | devolver ao processo | gravar em .golmet/<nome> e perguntar ao operador: promover ou permanecer temporário>
```

## Modo de erro

`continue` — roteamento falho NÃO bloqueia o fluxo: reporte o veredito
parcial ou a falha e devolva o conteúdo ao operador (nunca descarte, nunca
grave "para não perder").

- **Pack ativo não configurado** (bloco de acúmulo ausente): pergunte qual
  é o pack instalado; sem resposta → veredito com destino RELATIVO
  (`packs/<id>/...`) + pendência de configurar o bloco.
- **Nem a ordem nem o Jev resolvem**: `ask` ao operador com as 2 hipóteses
  e o porquê de cada — nunca adivinhe destino.
- **Destino já existe no índice**: roteie como ATUALIZAÇÃO (os escritores
  canônicos fazem merge incremental — nunca delete).

## Casos de Borda

- **Parece norma mas é genérico demais para ter gatilho claro** ("sempre
  faça assim" sem "AO fazer X, leia isto") → provável fundamento de domínio
  (pergunta 3) — fundamento vira rule quando ganha gatilho acionável.
- **Métrica sem projeto concluído** (outcome de tarefa corrente) → não
  roteia para metrics: T2 metrics é de retrospectiva/projeto concluído;
  devolva ao ciclo do projeto.
- **Aprendizado com conteúdo teórico perene dentro** → o padrão fica no T2
  com a fundação citada; a parte teórica pode justificar fundar/atualizar o
  T1 via `gm-domain-research` (duas entradas, uma delegação cada).
- **Operador pede "salva no AGENTS.md"** → recuse por aqui: AGENTS.md é
  índice lazy/regras de interação, não store de conteúdo — rode o algoritmo.
