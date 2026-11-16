---
name: gm-arch
description: Gera specs/ARCHITECTURE.md e specs/SECURITY.md a partir do DATA_MODEL.md e das RNs técnicas — decisões arquiteturais, stack, padrões e análise de segurança.
requires: [file, ask]
degrades:
  ask: chat-stop
kit:
  checkpoints: post-write validate
---
# gm-arch — Arquitetura e Segurança

Gera `<Produto>/specs/ARCHITECTURE.md` e `<Produto>/specs/SECURITY.md`
com as decisões arquiteturais e análise de segurança do sistema.

## Referências

- ADR (decisões arquiteturais registradas): FORMAT §7.8
- Layout do workspace (pasta `specs/`): FORMAT §2

## Resolução de caminhos

Os artefatos do workspace seguem o layout do FORMAT §2 — a saída fica em
`specs/ARCHITECTURE.md` e `specs/SECURITY.md`.

**Regra de fallback (transição):** ao **LER** um artefato, verifique primeiro
o caminho novo (ex: `specs/DATA_MODEL.md`); se não existir, use o caminho
legado na raiz do workspace (ex: `DATA_MODEL.md`). Ao **ESCREVER/GERAR**, use
sempre o caminho novo. Workspaces ainda não adotados continuam legíveis
durante a transição.

## Lê (upstream)

- `<Produto>/specs/DATA_MODEL.md` (tabelas, colunas, PKs, FKs, relacionamentos; fallback: raiz do workspace)
- `<Produto>/REQUIREMENTS.md` §2 (atores e permissões), §3 (estrutura)
- `<Produto>/DECISIONS.md` (se existir)
- `<Produto>/specs/API_CONTRACTS.md` (se existir; fallback: raiz do workspace)
- `<Produto>/reverse/LEGACY_CONTEXT.md` (se existir, para stack legada; fallback: raiz do workspace)

## Gera (downstream)

- `<Produto>/specs/ARCHITECTURE.md`
- `<Produto>/specs/SECURITY.md`

## Nunca edita

- `REQUIREMENTS.md`
- `DATA_MODEL.md`
- `DECISIONS.md`
- `golmet.index.json` (único escritor é o kit)

## Capacidades

- `file` — ler documentos upstream e escrever ARCHITECTURE.md, SECURITY.md
- `ask` — para decisões arquiteturais não definidas (sem ask nativo: pergunte no chat e PARE — nunca adivinhe)

## Modo de erro

`skip` — se falhar, loga warning. Arquitetura é tier completo, não bloqueia o fluxo básico.

## Algoritmo

1. Leia `specs/DATA_MODEL.md` (fallback: raiz) — tabelas, PKs, FKs e relacionamentos definem volume e complexidade do backend.
2. Leia `REQUIREMENTS.md` §2 (atores e permissões) e §3 (estrutura) — definem autenticação, autorização e módulos.
3. Leia `DECISIONS.md`, `specs/API_CONTRACTS.md` e `reverse/LEGACY_CONTEXT.md` (se existirem) — ADRs, contratos e stack legada restringem as escolhas.
4. Defina stack tecnológica, diagrama de arquitetura, padrões, módulos e fluxo de dados (template ARCHITECTURE.md).
5. Derive a análise de segurança dos atores/permissões e dos dados do modelo (template SECURITY.md: autenticação, autorização, dados sensíveis, auditoria).
6. Em decisão arquitetural não definida, use `ask` antes de escrever.
7. Escreva `specs/ARCHITECTURE.md` e `specs/SECURITY.md`.
8. Rode `golmet validate` (checkpoint post-write) — ou anote a pendência se o kit não estiver disponível.
9. Aponte o próximo passo natural: fechado o conjunto de specs do workspace, a camada seguinte é a de features — proponha gerá-la com `gm-taskspec` (análise `analysis/RN-NNN — <título>/` + feature `delivery/<grupo>/RN-NNN — <título> — Feature.md`) ou confirme a ordem com o roteamento do perfil (se disponível).

## Propagação de confiança (legado)

- Afirmações derivadas de RN induzida de engenharia reversa **herdam a marca de confiança da RN de origem** (campo `**Confiança:**` em REQUIREMENTS.md).
- Escala: Alta → 🟢 CONFIRMADO · Média → 🟡 INFERIDO · Baixa → 🔴 GAP.
- **Regra de ouro:** na dúvida, nível mais baixo.
- Quando disponível, anexe a evidência da análise reversa (`[arquivo:linha @hash]`) — **afirmação sem evidência nunca recebe 🟢**.

## Formato de commit

```
docs: add architecture and security for <produto>
```

## Template — ARCHITECTURE.md

```markdown
# Architecture — [Nome do Sistema]

> **Última atualização:** DD/MM/AAAA

---

## 1. Stack Tecnológica

| Camada | Tecnologia | Versão |
|---|---|---|
| Backend | | |
| Frontend | | |
| Banco | | |
| Cache | | |

## 2. Diagrama de Arquitetura

[Descrição textual ou Mermaid diagram]

## 3. Padrões

- [padrão de arquitetura, ex: Clean Architecture]
- [padrão de código, ex: Repository + Service]

## 4. Módulos

| Módulo | Responsabilidade | Depende de |
|---|---|---|

## 5. Fluxo de Dados

[Descrição do fluxo de dados entre módulos/sistemas externos]
```

## Template — SECURITY.md

```markdown
# Security — [Nome do Sistema]

> **Última atualização:** DD/MM/AAAA

---

## 1. Autenticação

| Método | Provedor | Escopo |
|---|---|---|

## 2. Autorização

| Papel | Permissões | RN |
|---|---|---|

## 3. Dados Sensíveis

| Campo | Classificação | Proteção |
|---|---|---|

## 4. Auditoria

| Tabela | O que audita | Retenção |
|---|---|---|
```

## Casos de Borda

- **Workspace sem backend próprio** (só consome APIs existentes): documente a stack de frontend e o consumo das APIs; não invente camada de backend — marque a dependência no fluxo de dados.
- **Monolito vs micro-frontends**: se o projeto já usa micro-frontends (projetos irmãos), siga o padrão de federação existente; se for monolito legado, documente a estratégia de convivência/migração.
- **Stack legada conflitante com a stack alvo (LEGACY_CONTEXT.md)**: registre a estratégia (ex: Strangler Fig) na seção de padrões — não omita o conflito.
- **ADR conflitante sobre stack em DECISIONS.md**: resolva com `ask` antes de escrever e registre a decisão final no ARCHITECTURE.md.
- **Dado sensível sem classificação em DATA_MODEL.md**: derive a classificação das RNs de §2/§3; se permanecer ambíguo, pergunte via `ask`.

---

compatibility: multi-engine

> **Degradação de checkpoints:** o checkpoint `post-write: validate` depende do kit golmet disponível. Sem o kit, a skill continua funcionando — o checkpoint vira "rode `golmet validate` manualmente" após a escrita.
