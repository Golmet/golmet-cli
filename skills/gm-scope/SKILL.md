---
name: gm-scope
description: Gera delivery/SCOPE_AGREEMENT.md — acordo de escopo com o cliente em linguagem de negócio, unificando escopo, RNs confirmadas, mudanças e status de aprovação.
requires: [file]
kit:
  checkpoints: post-write validate
---
# gm-scope — Acordo de Escopo com o Cliente

Gera `<Produto>/delivery/SCOPE_AGREEMENT.md` — um documento vivo em
linguagem de negócio que serve como contrato de comprometimento entre
a equipe técnica e o cliente.

## Referências

- Entidades RN/CR/PEND (fonte do conteúdo): FORMAT §7.2/§7.4/§7.3
- Layout do workspace (pasta `delivery/`): FORMAT §2

## Resolução de caminhos

Os artefatos do workspace seguem o layout do FORMAT §2 — a saída fica em
`delivery/SCOPE_AGREEMENT.md`.

**Regra de fallback (transição):** ao **LER** um artefato, verifique primeiro
o caminho novo (ex: `specs/DATA_MODEL.md`); se não existir, use o caminho
legado na raiz do workspace (ex: `DATA_MODEL.md`). Ao **ESCREVER/GERAR**, use
sempre o caminho novo. Workspaces ainda não adotados continuam legíveis
durante a transição.

## Lê (upstream)

- `<Produto>/PROJECT_CHARTER.md` (identificação, escopo, riscos)
- `<Produto>/REQUIREMENTS.md` (RNs, PENDs, decisões)
- `<Produto>/CHANGE_REQUESTS.md` (mudanças solicitadas e aprovadas; fallback: raiz)
- `<Produto>/CHANGELOG.md` (histórico de adições/alterações)
- Documentos de linguagem do usuário (se existirem, para vocabulário)
- `<Produto>/delivery/SCOPE_AGREEMENT.md` (se existir, para merge incremental; fallback: raiz)

## Gera (downstream)

- `<Produto>/delivery/SCOPE_AGREEMENT.md`

## Nunca edita

- `REQUIREMENTS.md`
- `PROJECT_CHARTER.md`
- Nenhum documento técnico ou de requisitos
- `golmet.index.json` (único escritor é o kit)

## Capacidades

- `file` — ler documentos upstream e escrever SCOPE_AGREEMENT.md
- **Esta skill NÃO declara `ask`:** sem pergunta interativa, dúvidas de conteúdo são registradas como pendência no próprio documento (seção §6) — nunca invente aprovação ou conteúdo.

## Modo de erro

`continue` — se falhar, loga warning. É documento de comunicação, não
bloqueia o fluxo técnico.

## Algoritmo

1. Leia REQUIREMENTS.md, `CHANGE_REQUESTS.md`, CHANGELOG.md e o SCOPE_AGREEMENT.md existente (se houver)
2. Traduza RNs e mudanças para linguagem de negócio — sem IDs internos visíveis no texto principal (IDs só na seção de rastreamento)
3. Se o documento existe, faça merge incremental (seções adicionam itens, CRs aprovados refletem, pendências resolvidas saem); se não, gere todas as seções do template
4. Atualize a data e o status da versão (`proposta` | `vigente`)
5. Escreva `delivery/SCOPE_AGREEMENT.md`
6. Rode `golmet validate` (checkpoint post-write) — ou anote a pendência se o kit não estiver disponível

## Formato de commit

```
docs: update scope agreement for <produto>
```

## Regras

1. **Linguagem de negócio** — NUNCA use EARS, termos técnicos ou IDs
   internos (RN-XXX) no texto principal. Use paráfrases claras.
   Os IDs internos devem ficar em nota de rodapé ou coluna separada
   apenas na seção de rastreamento.

2. **Mercado-alvo da linguagem:** o stakeholder de negócio que aprovou
   o projeto — não o usuário final, não o desenvolvedor.

3. **Última versão sempre substitui a anterior** — o SCOPE_AGREEMENT.md
   é o documento mais atualizado. Não mantém versões históricas no
   mesmo arquivo (o CHANGELOG.md cuida disso).

4. **Se existir, faça merge incremental** — preserve seções que o
   cliente já aprovou, adicione apenas o novo lote.

## Template — SCOPE_AGREEMENT.md

```markdown
# Acordo de Escopo — [Nome do Sistema]

> **Cliente:** [nome do contratante]
> **Última atualização:** DD/MM/AAAA
> **Status da versão:** proposta | vigente

---

## 1. Objetivo do Projeto

[parágrafo em linguagem de negócio — o problema que o sistema resolve]

## 2. O Que Será Entregue

### 2.1 [Tema/Funcionalidade 1]

- Item de escopo em linguagem clara
- Item de escopo em linguagem clara

### 2.2 [Tema/Funcionalidade 2]

- Item de escopo em linguagem clara

## 3. O Que Não Está no Escopo

- Item fora do escopo
- Item fora do escopo

## 4. Regras de Negócio Confirmadas

| # | Regra | Status |
|---|---|---|
| 1 | [descrição em linguagem de negócio] | ✅ confirmada |
| 2 | [descrição em linguagem de negócio] | 🔄 alterada em DD/MM |

> Os detalhes técnicos de cada regra estão em REQUIREMENTS.md.

## 5. Mudanças em Andamento

| # | Mudança | Status | Data |
|---|---|---|---|
| — | Nenhuma mudança pendente | — | — |

## 6. Pendências e Definições Futuras

| Item | Previsão |
|---|---|
| [pendência](/dúvida) | [data ou vazio] |

## 7. Riscos e Premissas

- [premissa/risco relevante para o cliente]

---

> Este documento é enviado ao cliente para leitura e aprovação verbal.
> Não substitui contrato formal — é uma ferramenta de alinhamento contínuo.
```

## Seções que mudam a cada execução

Na primeira execução, gere todas as seções do template.
Nas execuções seguintes (merge incremental):

- **§2 e §4** — adicione novos itens sem remover os existentes.
  Se uma RN foi alterada por CR, marque como `🔄 alterada` em vez de
  reescrever.
- **§5** — CRs com status `pending` entram como pendentes; CRs
  `approved` entram como "em implementação"; CRs `implemented` são
  movidos para o CHANGELOG (seção §4 com status `✅ confirmada`).
- **§6** — PENDs resolvidas desde a última execução são removidas
  (diretiva `> estado: resolved → ...` na fonte).

## Exemplo de §4 (RNs em linguagem de negócio)

```markdown
## 4. Regras de Negócio Confirmadas

| # | Regra | Status |
|---|---|---|
| 1 | Todo cliente cadastrado deve ter um responsável vinculado | ✅ confirmada |
| 2 | O sistema deve validar o documento do responsável no cadastro | ✅ confirmada |
| 3 | Quando um atendimento é agendado, o cliente recebe notificação com a data | ✅ confirmada |
| 4 | Atendimentos podem ser registrados offline e sincronizam automaticamente | 🔄 alterada em DD/MM |
```

## Casos de Borda

- **Primeira execução** — gere todas as seções do template; não deixe seções vazias sem o placeholder.
- **CR implementado** — mova a mudança para a seção de mudanças confirmadas com status `✅ confirmada` e remova das pendentes.
- **CR pendente** — mantenha como pendente; não antecipe aprovação.
- **RN alterada por CR** — marque `🔄 alterada em DD/MM` em vez de reescrever a linha.
- **PEND resolvida desde a última execução** — remova da seção de pendências.

---

compatibility: multi-engine

> **Degradação de checkpoints:** o checkpoint `post-write: validate` depende do kit golmet disponível. Sem o kit, a skill continua funcionando — o checkpoint vira "rode `golmet validate` manualmente" após a escrita.
