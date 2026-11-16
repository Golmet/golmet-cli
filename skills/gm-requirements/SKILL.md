---
name: gm-requirements
description: Gera ou atualiza REQUIREMENTS.md a partir da transcrição de reunião de negócios — merge incremental, promove PEND a RN com proveniência e registra decisões em DECISIONS.md.
requires: [file, ask, kit.jev]
degrades:
  ask: chat-stop
kit:
  checkpoints: post-write validate
  jev: heuristic
---
# gm-requirements — Gerenciamento de REQUIREMENTS.md

Gera ou atualiza `REQUIREMENTS.md` do workspace a partir de transcrição
de reunião de negócios. Faz merge incremental, nunca reescreve do zero.

## Referências

- Entidades RN/PEND (estados, diretivas `> estado:`): FORMAT §7.2/§7.3
- ADR (estados, append-only): FORMAT §7.8
- IDs RN-NNN, PEND-NNN, ADR-NNN: FORMAT §4
- Template de 11 seções: embutido abaixo por praticidade

## Resolução de caminhos

Os artefatos do workspace seguem o layout do FORMAT §2:

- **Raiz do workspace:** PROJECT_CHARTER.md, REQUIREMENTS.md, DECISIONS.md, CHANGE_REQUESTS.md
- **`specs/`:** DATA_MODEL.md, API_CONTRACTS.md, ERROR_CATALOG.md, UI_SPEC.md, ARCHITECTURE.md, SECURITY.md, ACCEPTANCE_CRITERIA.md, COMPLIANCE.md
- **`quality/`:** TEST_STRATEGY.md, DEFINITION_OF_DONE.md, TRACEABILITY.md
- **`delivery/`:** Features (`<grupo>/RN-NNN — <título> — Feature.md`), SCOPE_AGREEMENT.md, CLIENT_REQUIREMENTS.md
- **`analysis/`:** specs por RN
- **`meetings/`:** checkpoints

**Regra de fallback (transição):** ao **LER** um artefato, verifique primeiro
o caminho novo (ex: `specs/DATA_MODEL.md`); se não existir, use o caminho
legado na raiz do workspace (ex: `DATA_MODEL.md`). Ao **ESCREVER/GERAR**, use
sempre o caminho novo. Workspaces ainda não adotados continuam legíveis
durante a transição.

## Lê (upstream)

- `<Produto>/PROJECT_CHARTER.md` (escopo, stakeholders, demanda externa se houver)
- `<Produto>/REQUIREMENTS.md` (se existir)
- `<Produto>/DECISIONS.md` (se existir)
- `<Produto>/meetings/` (checkpoints do gm-clarify e meetings anteriores)
- `<Produto>/LINGUAGEM_UBIQUA.md` (opcional — se existir, use os termos canônicos; termo divergente na transcrição vira nota de rodapé)

## Gera (downstream)

- `<Produto>/REQUIREMENTS.md`
- `<Produto>/DECISIONS.md` (se houver novas decisões)
- `<Produto>/CHANGELOG.md` (efeito colateral automático)
- `<Produto>/meetings/YYYY-MM-DD-NN.md` (resumo da reunião + perguntas/respostas, NN = próximo sequencial na pasta)

## Nunca edita

- `CHANGE_REQUESTS.md` (mudança em RN existente só por CR)
- `PROJECT_CHARTER.md`
- `golmet.index.json` (único escritor é o kit)

## Capacidades

- `file` — ler arquivos existentes e escrever/atualizar REQUIREMENTS.md, DECISIONS.md, CHANGELOG.md
- `ask` — para ambiguidades não cobertas pelo gm-clarify (sem ask nativo: pergunte no chat e PARE — nunca adivinhe)
- `kit.jev` — classificação calibrada da confiança de RNs induzidas (FORMAT §10); sem kit.jev: julgamento heurístico conservador (degradação declarada)

## Modo de erro

`strict` — se REQUIREMENTS.md não for gerado/atualizado, o fluxo para.

## Algoritmo

1. Leia o checkpoint mais recente em `meetings/` (saída do `gm-clarify`) e o REQUIREMENTS.md existente, se houver
2. Extraia da transcrição: RNs candidatas, PENDs, decisões (com decisor), campos, atores e escopo
3. Se o arquivo existe, faça merge incremental (novas RNs/PENDs sequenciais a partir da última; PENDs resolvidas movem para a seção de resolvidas; PENDs promovidas a RN seguem a seção "Promoção de PEND a RN"); se não, gere do template
4. Registre novas decisões em DECISIONS.md e atualize o CHANGELOG.md como efeito colateral
5. Escreva o checkpoint da reunião em `meetings/YYYY-MM-DD-NN.md` (NN = próximo sequencial)
6. Se houver ambiguidade não coberta pelo clarify, use `ask` antes de registrar
7. **Portão de cobertura downstream:** ao fechar um lote de RNs novas (≥ 3 RNs), verifique antes de encerrar se cada RN nova tem downstream previsto — rode a verificação de rastreabilidade do perfil (skill de traceability, se disponível) ou verifique manualmente os gaps na matriz — e reporte explicitamente: `RN sem downstream em specs/ (DATA_MODEL/API_CONTRACTS/UI_SPEC) = pendência do lote`. Um lote NÃO está encerrado enquanto houver RN nova sem destino downstream declarado (ou justificativa explícita de por que não aplica).
8. **Confiança de RNs induzidas (legado):** quando a fonte é relatório de engenharia reversa (`gm-reverse` — workspace com `sistema_existente: true`), cada RN induzida carrega no corpo a tag de confiança da análise reversa: `**Confiança:** Alta` | `Média` | `Baixa` (escala: Alta→🟢 CONFIRMADO, Média→🟡 INFERIDO, Baixa→🔴 GAP; na dúvida, registre o nível mais baixo). RN com confiança **Baixa** que não passar por validação humana segue a regra de promoção (vira PEND — nunca ancora feature).

### Segunda camada de classificação calibrada (`kit.jev`, opcional) — passo 8

- Antes de gravar a tag de confiança de cada RN induzida de legado, submeta ao classificador calibrado do kit (`kit.jev` — FORMAT §10): texto = corpo da RN + evidência citada (`[arquivo:linha @hash]`); critérios = `{confirmada: "Evidência citada confirma diretamente a afirmação", inferida: "Afirmação é dedução de padrão, não leitura direta", lacuna: "Evidência insuficiente para julgar"}`.
- **Regra conservadora:** se o classificador retorna `confirmada` com probabilidade < 0.7 (ou retorna `inferida`/`lacuna`), **rebaixe** um nível (Alta→Média→Baixa) e anote no corpo da RN: `Calibração: inferida (p=0.55)` — a calibração **só rebaixa, nunca eleva** o grau.
- Agrupe as chamadas em lote (uma sessão para todas as RNs do lote), nunca 1 por RN.
- **Degradação (`kit.jev: heuristic`):** sem o classificador disponível, o julgamento é heurístico do agente — aplique a mesma regra conservadora (na dúvida, rebaixe), anote `Calibração: heurística` no corpo e siga; a calibração é **opcional** — nunca gate bloqueante; se falhar, grave a tag original e siga sem ela (avise no checkpoint da reunião).

### Contradição semântica no merge (calibração, opcional) — passo 3

- RN induzida nova de re-análise reversa que toque no mesmo domínio de RN confirmada existente → submeta o PAR (texto antigo, texto novo) ao classificador com critérios `{consistente, contraditoria, incomparavel}`.
- `contraditoria` com p >= 0.7 → **NÃO sobrescrever silenciosamente**: registre como candidato a Change Request (CR) em `CHANGE_REQUESTS.md` e sinalize no CHANGELOG.md, seguindo o fluxo de CR existente.
- Sem calibração disponível: siga o merge normal (regra geral 2) e registre o aviso; nunca bloqueie.

## Formato de commit

```
docs: add/update requirements from meeting YYYY-MM-DD
```

## Regras Gerais

0. **Propague a demanda externa do charter** — se o perfil usa tracker externo (recurso FULL, ex.: issue registrada pelo cliente), leia o campo no charter e inclua `> **Demanda:** #[numero]` no REQUIREMENTS.md. Sem tracker (perfil BASE), omita.

1. **Não invente** — definições, prazos, regras ou valores que não estiverem na transcrição. Se algo não estiver claro, use `ask` (sem ask nativo: pergunte no chat e PARE).

2. **Se o arquivo já existe**, leia-o primeiro e faça merge incremental:
   - Nunca reescreva do zero
   - Novas RNs seguem numeração sequencial a partir da última existente
   - Novas PENDs seguem numeração sequencial a partir da última existente
   - PENDs resolvidas na reunião atual → diretiva `> estado: resolved → CR-NNN` (ou destino RN/ADR) com marcação `[confirmado] (data da reunião)`
   - Contradições com o existente → use `ask` em vez de corrigir

3. **Se o arquivo não existe**, gere do zero seguindo o template.

4. **Toda RN nova deve ser registrada no CHANGELOG.md** como efeito colateral.

5. **Tabelas markdown: o número de segmentos `|---|` no separador deve ser idêntico ao número de colunas do cabeçalho.** Conte manualmente. Exemplo: 4 colunas → `|---|---|---|` (4 segmentos). 3 colunas → `|---|---|` (3 segmentos). Um segmento a mais ou a menos quebra a renderização.

6. **Decisões de design/escopo** mencionadas na reunião devem ser registradas em DECISIONS.md no formato MADR:
   - Extraia da transcrição **quem decidiu** (autor/decisor da decisão). Pode ser uma pessoa, um papel ('Lead Técnico', 'Cliente') ou múltiplos nomes separados por vírgula.
   - Se o decisor não estiver explícito na transcrição, use `ask` para perguntar ao usuário.
   - Persista o decisor no campo `**Decisor:**` de cada novo ADR (ver template abaixo).
   - Novos ADRs seguem numeração sequencial a partir do último existente; nunca renumerar ADRs antigos.
   - ADRs existentes sem o campo `Decisor:` devem ser preservados inalterados (merge incremental; não adicione `Decisor:` retroativamente).

7. **LINGUAGEM_UBIQUA.md** — se existir no workspace, use exatamente os termos definidos nele. Termo divergente na transcrição: use o termo canônico e registre a variante em nota de rodapé.

8. **Gatilho de divisão do documento** — se o REQUIREMENTS.md resultante ultrapassar ~600-700 linhas, ou a transcrição cobrir mais de uma funcionalidade/módulo independente, avise o usuário que pode fazer sentido dividir em múltiplos arquivos (ex. `REQUIREMENTS_<TEMA>.md`) e pergunte antes de decidir. Nunca divida automaticamente sem confirmação.

9. **Regra removida com relevância externa** — se uma RN deixou de se aplicar mas tem relevância externa (ex.: referenciada em tarefa de tracker do perfil FULL), marque como `[removido]` com uma frase breve do motivo em vez de apagar silenciosamente. Se era puramente interna e nunca referenciada fora do documento, pode remover a linha — na dúvida, marque como removido.

10. **Promoção de PEND a RN** — PEND sai do estado aberta por promoção a RN, por virar CR (altera RN confirmada — registro em `CHANGE_REQUESTS.md`) ou por encerrar como decisão; PEND NUNCA ancora feature (FORMAT §7.3, H1). Ao promover, siga a seção "Promoção de PEND a RN".

## Promoção de PEND a RN

PEND sai do estado aberta SOMENTE por promoção a RN (resolução define
requisito novo/complementar), por virar CR (resolução revela alteração em
RN já confirmada — CR registrado em `CHANGE_REQUESTS.md`) ou por encerrar
como decisão. PEND NUNCA ancora feature (H1) — a feature nasce da
RN.

Ao promover uma PEND-NNN a RN:

1. **Numeração sequencial NOVA** — a RN recebe o próximo número sequencial
   de RNs de REQUIREMENTS.md; NUNCA reutilize o número da PEND.
2. **Conteúdo derivado da PEND** — texto da resolução esclarecida
   (respostas do checkpoint de clarify / decisão do cliente), no padrão
   EARS das demais RNs, com diretiva `> estado: draft` e tag `[confirmado]` + data.
3. **Proveniência obrigatória** — logo abaixo da linha da RN nova, inclua:
   `> Promovida de PEND-NNN em DD/MM/AAAA.`
4. **Marque a PEND de origem** — mantenha a seção da PEND com a
   diretiva `> estado: resolved → RN-XXX` e a marcação
   `**Promovida a RN-XXX em DD/MM/AAAA**` (nunca apague a seção
   silenciosamente — rastreabilidade).
5. **Registre no CHANGELOG.md** — efeito colateral padrão (RN adicionada).
6. **A RN promovida é uma RN comum** — qualquer alteração posterior passa
   por CR (registro em `CHANGE_REQUESTS.md`); nunca edite a RN promovida diretamente.

A RN criada é a âncora legítima da feature: `gm-taskspec` gera a análise
e a Feature.md com a RN como âncora.

**Checkpoint de kit (`post-write: validate`):** após escrever REQUIREMENTS.md
(e DECISIONS.md/CHANGELOG.md), rode `golmet validate` no workspace e corrija
o que apontar — RNs devem estar no vocabulário legal `draft|active|superseded|dropped`
(FORMAT §7.2), PENDs em `open|deferred|resolved|dropped` (§7.3), ADRs em
`proposed|accepted|superseded` (§7.8). Sem kit disponível, anote a pendência
"rode golmet validate" para o próximo passo.

## Template — REQUIREMENTS.md (11 seções)

```markdown
# [Nome do Sistema] — [Subtítulo descritivo]

> **Demanda:** #[numero] (se o perfil usa tracker externo — FULL)
> **Última atualização:** DD/MM/AAAA
> **Objetivo:** [descrição concisa]

---

## 1. Escopo

- [item dentro do escopo]

> Fora do escopo:
> - [item fora]

---

## 2. Atores e Permissões

| Papel | Pode fazer | Não pode fazer |
|---|---|---|

---

## 3. Estrutura da Aplicação

[descrição textual]

---

## 4. Conceitos de Domínio

| Termo | API | Descrição |
|---|---|---|

---

## 5. Regras de Negócio

### [Tema]

## RN-001 — [título curto]
> estado: draft
- `[tipo EARS]` [frase] `[confirmado]` (DD/MM/AAAA)

---

## 6. Dados e Campos

### [Entidade]

| Campo | Obrigatório | Formato | Comportamento |
|---|---|---|---|

---

## 7. Casos de Uso / Fluxos

1. [passo]

---

## 8. Saídas e Integrações

---

## 9. Persistência e Auditoria

| Entidade | Tabela | Finalidade | Rastreabilidade |
|---|---|---|---|

---

## 10. Decisões e Fora do Escopo

| Item | Decisão | Motivo |
|---|---|---|

---

> **Regra de idioma:** Requisitos no idioma do cliente (padrão PT-BR). Títulos de seção, descrições de RNs, UCs e campos no idioma combinado. Apenas notação EARS (shall, must, should), identificadores (RN-001, UC-001) e termos técnicos do domínio permanecem no original.

## PEND-001 — [título da pendência]
> estado: open
- (DD/MM/AAAA) — [descrição + perguntas de follow-up do clarify]

## PEND-002 — [título da pendência]
> estado: resolved → RN-003
- (DD/MM/AAAA) — [descrição] `**Promovida a RN-003 em DD/MM/AAAA**`
```

## Template — Meeting (Checkpoint)

Arquivo: `<Produto>/meetings/YYYY-MM-DD-NN.md` (NN = próximo sequencial)

```markdown
# Requisitos — DD/MM/AAAA

**Resumo:** [descrição concisa do que foi discutido e incorporado]

## Perguntas e Respostas

### [pergunta]

**Resposta:** [resposta]
```

## Template — CHANGELOG.md

```markdown
# Histórico de Mudanças

## [data] — [descrição curta]

### Adicionado
- RN-001: [descrição]

### Alterado
- RN-002: [descrição da mudança]

### Removido
- RN-003: [descrição]
```

## Template — DECISIONS.md

```markdown
# Architecture Decision Records — [Nome do Produto]

## ADR-001 — [título curto]
> estado: proposed

**Data:** DD/MM/AAAA
**Decisor:** [nome do decisor/autor da decisão] | Não informado
**Contexto:** [censo que levou à decisão]
**Decisão:** [o que foi decidido — usar linguagem direta]
**Consequências:** [impactos positivos, negativos, riscos, dívidas técnicas]
```

> **Invariante ADR (FORMAT §7.8):** conteúdo de ADR `accepted` é append-only — para mudar uma decisão aceita, crie um ADR novo com `> estado: superseded → ADR-XXX` no antigo. Nunca edite ADR aceito.

## Exemplo de Commit

```bash
git add REQUIREMENTS.md DECISIONS.md CHANGELOG.md meetings/
git commit -m "docs: update requirements from meeting YYYY-MM-DD

- Add RN-027 \`[evento]\` and RN-028 \`[indesejado]\` for batch validation
- Resolve PEND-001 (promovida a RN-029)
- Record decision on signup workflow (Decisor: [nome])"
```

## Casos de Borda

- **Transcrição sem RNs novas** — atualize apenas o que mudou (escopo, decisões); não gere seções vazias.
- **Contradição com RN existente** — não corrija silenciosamente; use `ask` e, se confirmada, registre CR em `CHANGE_REQUESTS.md` (nunca edite a RN diretamente).
- **Documento ultrapassando ~600-700 linhas** — avise sobre divisão em múltiplos arquivos e pergunte antes.
- **Linguagem ubíqua divergente da transcrição** — use o termo canônico e registre a variante em nota (regra geral).
- **RN com relevância externa deixando de se aplicar** — marque `[removido]` com motivo em vez de apagar.
- **Promoção de PEND a RN** — resolução de PEND define requisito: crie RN sequencial NOVA com proveniência `> Promovida de PEND-NNN em <data>` e resolva a PEND citando a RN destino (nunca reutilize o número da PEND; nunca edite a RN promovida sem CR). Se a resolução altera RN já confirmada, o destino é CR, não RN nova.

---

compatibility: multi-engine

> **Degradação de checkpoints:** o checkpoint `post-write: validate` depende do kit golmet disponível. Sem o kit, a skill continua funcionando — o checkpoint vira "rode `golmet validate` manualmente" após a escrita. Sem `kit.jev`, a calibração de confiança vira julgamento heurístico conservador (regra: na dúvida, rebaixe).
