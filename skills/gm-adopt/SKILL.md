---
name: gm-adopt
description: Conduz a adoção do formato GOLMET por um workspace existente — handshake de 7 passos com o kit (dry-run → julgamento → gates → apply → validate tolerant → correção → strict), agente conduz e kit executa.
requires: [file, shell, kit]
degrades:
  ask: chat-stop
kit:
  checkpoints: each-step validate
---
# gm-adopt — Adoção do formato GOLMET por um workspace existente

Conduz a migração de um workspace de produto já existente (documentação
prévia em qualquer arranjo) para o FORMATO GOLMET, usando o handshake do kit.
A regra central: **agente conduz, kit executa** — o agente investiga, propõe
e pergunta gates (julgamento); o kit executa os transforms determinísticos e
valida (único escritor do índice e do scaffold).

## Referências

- Estados e transições de cada entidade: FORMATO §7 (§7.1 Charter — §7.9 Candidate)
- Gramática de evidência: FORMATO §5
- Erros estruturados com `fix:`: FORMATO §11
- Enforcement de dois níveis (H/S): FORMATO §9

## Lê (upstream)

- `<Produto>/PROJECT_CHARTER.md` e demais documentos existentes do workspace (read-only nesta fase)
- Repos de código registrados no charter — para re-pin de evidências (hash atual)
- `golmet.yaml` do workspace (se já existir)

## Gera (downstream)

- `golmet.yaml` + `golmet.index.json` — SEMPRE via kit (`golmet adopt --apply`), nunca escritos diretos pelo agente
- Editações cirúrgicas na prosa (diretivas `> estado:`, evidências com hash, front matter do charter) — escritas pelo agente, validadas pelo kit
- Tabela de acompanhamento da migração por produto (registro do dono e do estado)

## Nunca edita

- `golmet.index.json` — único escritor é o kit (D2); o agente edita a diretiva na prosa e o kit recompõe o índice
- Decisões de negócio sem gate — nenhum estado muda sem confirmação humana quando há dúvida razoável
- A prosa além do que a adoção exige — migração não é reescrita de conteúdo

## Capacidades

- `file` — ler a prosa existente, editar diretivas/evidências/front matter
- `shell` — rodar os comandos do kit (`golmet adopt`, `golmet validate`, `golmet doctor`) e consultar o Git dos repos (hash/HEAD para re-pin de evidência)
- `kit` — o kit é pré-condição operacional desta skill (comandos `adopt`/`validate`/`doctor`); sem o kit instalado, instrua a instalação e interrompa — não improvise transform manual
- `ask` — gates de confirmação humana (sem ask nativo: pergunte no chat e PARE — nunca adivinhe)

## Modo de erro

`strict` — julgamento pendente não é resolvido por inferência; cada gate sem
resposta interrompe o passo com a pendência registrada. Nunca rode `--apply`
com gates não confirmados (modo assistido). Erros estruturados do kit trazem
`fix:` — siga a correção sugerida no loop.

## O handshake (7 passos)

A unidade de migração é o **PRODUTO** (workspace), não a pessoa — cada produto
atravessa quando o dono opta; convivência de workspaces em janela de migração
é estado normal, não exceção.

```
1. KIT     golmet adopt --dry-run      → relatório de conformidade (leitura pura, ZERO escrita)
2. AGENTE  conduz: prioriza lacunas, investiga código para re-pinar evidências,
           propõe estados (superseded por qual? ready pode?) — julgamento
3. HUMANO  confirma gates (ask/PR): charter confirmed? RN vira dropped? normalizar layout?
4. KIT     golmet adopt --apply --charter-status confirmed --keep-legacy-layouts
           → executa o transform determinístico das escolhas DECLARADAS (golmet.yaml + índice)
5. KIT     golmet validate --tolerant  → ponte da janela (regras de evidência degradadas a AVISO)
6. AGENTE  corrige o que o validate apontou — loop do kit (agente escreve, kit checa)
7. CI      validate --strict entra na Action do produto SÓ quando o dono declara a migração
           concluída (gate humano, registrado na tabela de acompanhamento)
```

### Passo 1 — Dry-run (leitura pura)

Rode `golmet adopt --dry-run` no workspace. O relatório lista: charters sem
`status:`, evidências sem hash, RNs active sem evidência parseável, IDs
duplicados, variantes legadas de layout detectadas (ex.: feature dentro de
pasta própria). **Nenhuma escrita acontece neste passo.** Apresente o
relatório ao dono do produto em linguagem direta: o que está conforme, o que
precisa de julgamento, o que o kit resolve sozinho.

### Passo 2 — Condução (julgamento do agente)

Com o relatório em mãos, conduza a priorização:

- **Priorize lacunas** que bloqueiam o `--apply` (front matter do charter
  ausente, IDs duplicados) antes das cosméticas.
- **Re-pin de evidências:** para cada evidência sem hash ou stale, investigue
  o código no repo (Git) e monte a evidência canônica
  `[caminho:linha @hash]` — o kit NUNCA escolhe hash sozinho (re-pin é
  julgamento: qual é a linha certa que sustenta a RN?).
- **Proponha estados:** para cada RN/PEND/CR/ADR da prosa antiga, proponha a
  diretiva inicial (`> estado: draft`, `> estado: open`, ...) e, onde houver
  regra substituída, proponha `> estado: superseded → RN-XXX` — a escolha do
  sucessor é do agente, a confirmação é humana.
- Marque o que é mecânico (kit resolve no `--apply`) vs o que precisa de
  decisão (vira gate no passo 3).

### Passo 3 — Gates humanos

Confirme via `ask` (sem ask nativo: pergunte no chat e PARE — nunca adivinhe)
— agrupe TODOS os gates pendentes num único bloco de perguntas:

- Charter pode nascer `confirmed`? (pré-condição FORMATO §7.1: §1 registra
  cada repo com path + branch + HEAD congelado)
- RNs fora de vigor: `dropped` (com motivo no corpo) ou `superseded` (citando
  a sucessora)? `dropped` nunca reverte.
- Variantes legadas de layout: normalizar agora ou manter (o validate aceita
  ambas e recomenda a normalização)?
- RNs sem evidência suficiente: permanecem `draft` (exigência H4 só atinge
  `active`) ou ficam na worklist do passo 6?

Nenhum gate confirmado, nenhum `--apply`.

### Passo 4 — Apply (kit executa)

Com os gates confirmados, rode o apply declarando as escolhas:

```
golmet adopt --apply --charter-status confirmed --keep-legacy-layouts
```

O kit executa o transform determinístico das escolhas DECLARADAS: cria
`golmet.yaml` + `golmet.index.json` a partir da prosa. O agente nunca escreve
índice nem scaffold direto — se faltou uma escolha no apply, volta ao passo 3
e re-aplica, nunca edita o índice à mão.

### Passo 5 — Validate tolerant (ponte da janela)

Rode `golmet validate --tolerant`: regras de evidência (H4/H7) ficam
degradadas a AVISO com worklist enquanto o produto está na janela de
migração. Isso permite o produto seguir vivo durante a adoção sem bloquear
todo o fluxo por lacunas ainda não resolvidas.

### Passo 6 — Loop de correção

Para cada item da worklist do tolerant: o agente corrige a prosa (diretiva,
evidência com hash re-pinado, front matter), o kit valida de novo. Repita até
a worklist de itens bloqueantes esvaziar. Erros estruturados trazem `fix:` —
siga-o. **Checkpoint de kit (`each-step: validate`):** cada passo que escreve
prosa é seguido de `golmet validate` — nunca acumule muitas edições sem
validar.

### Passo 7 — Strict (gate de conclusão)

Quando o dono declara a migração concluída (gate humano), o
`golmet validate --strict` entra na Action/CI do produto — a partir daí as
regras de evidência voltam a ser HARD. O kit não impõe prazo: o gate é
humano e registrado na tabela de acompanhamento.

## Modo assistido vs não-assistido

| Modo | Como funciona |
|---|---|
| **Assistido** (padrão) | Agente conduz os 7 passos com gates humanos em cada decisão de julgamento |
| **Não-assistido** | `golmet adopt --apply` sem agente funciona (quem só quer entrar rápido): julgamentos pendentes ficam no relatório do kit e o `validate` lista as lacunas restantes — o dono resolve depois, ainda com gate humano para concluir |

Esta skill opera o modo assistido. Se o operador pedir não-assistido,
aponte o comando direto e não rode gates — mas nunca declare a migração
concluída (passo 7) sem confirmação humana.

## Tabela de acompanhamento por produto

Mantenha (no registro do time/dono da janela de migração) o estado de cada
produto:

| Produto | Dono | Estado (convivendo · migrado · cortado) |
|---|---|---|
| *(produto)* | *(dono)* | convivendo |

- **convivendo** — na janela: tolerant ativo, strict ainda não entrou
- **migrado** — dono declarou conclusão; strict no CI
- **cortado** — legado do formato anterior removido (remoção física do
  workflow anterior só após desuso comprovado, nunca calendário)

## Prova pós-migração

Antes de declarar concluído, rode `golmet doctor` e confirme: paridade de
comportamento com o fluxo anterior (mesmos estados, mesmas regras), `format`
version coerente, capacidades de kit (`kit.jev`/`kit.graph`) declaradas
presentes ou degradadas com aviso, e zero erros no validate strict.

## Casos de Borda

- **Workspace sem charter** — o adopt cria o scaffold mínimo; o charter nasce
  `draft` e só sobe a `confirmed` no gate do passo 3 (com §1.1 preenchida).
- **IDs duplicados na prosa antiga** — o kit reporta; a desambiguação é do
  agente (IDs são únicos e imutáveis — H2): renomear seção com ID quebra a
  história; se dois conteúdos dividiam um ID, separe e cite a proveniência.
- **Evidência apontando para repo fora do charter** — H7 exige que o hash
  resolva em exatamente 1 repo do charter §1; corrija o charter (gate) ou
  a evidência antes do apply.
- **Produto sem dono definido** — não comece: a adoção é opt-in por produto
  com dono; registre a pendência e siga para o próximo produto.
- **Kit ausente ou versão antiga** — interrompa com instrução de
  instalação/atualização (`golmet update`); nunca simule o transform na mão.
- **Perguntas sem resposta** — registre a pendência e interrompa o passo;
  retomada recomeça do dry-run (idempotente, leitura pura).

## Formato de commit

```
docs: adopt golmet format for <produto> (dry-run → apply → tolerant)
```

---

compatibility: multi-engine

> **Degradação de checkpoints:** os checkpoints `each-step: validate` dependem do kit golmet disponível — sem o kit esta skill NÃO opera (o transform determinístico é papel exclusivo do kit; o agente nunca o simula). Instale/atualize o kit (`golmet update`) antes de conduzir a adoção.
