---
name: gm-reverse
description: Gera LEGACY_CONTEXT.md a partir da análise de sistema/banco/API existente — só executa se o charter contiver legado: true ou sistema_existente: true.
requires: [file, shell]
kit:
  graph: grep-fallback
---
# gm-reverse — Análise de Legado

Gera `reverse/LEGACY_CONTEXT.md` no workspace com a análise do sistema,
banco de dados ou API existente que será evoluído, migrado **ou apenas
documentado como está** (sistema vivo/em produção).

## Referências

- Campos opcionais do charter `legado` / `sistema_existente`: FORMATO §7.1
- Pasta `reverse/` do workspace: FORMATO §2

## Lê (upstream)

- `<Produto>/PROJECT_CHARTER.md` (para confirmar `legado: true` ou `sistema_existente: true`)

## Gera (downstream)

- `<Produto>/reverse/LEGACY_CONTEXT.md`

## Nunca edita

- `PROJECT_CHARTER.md`
- `golmet.index.json` (único escritor é o kit)
- Código-fonte do sistema legado (read-only — análise nunca altera o legado)

## Capacidades

- `file` — ler o charter e escrever LEGACY_CONTEXT.md
- `shell` — inspecionar código-fonte, schemas de banco e APIs existentes (grep/glob/buscas)
- `kit.graph` (opcional — FORMATO §10): com o grafo de código disponível, use navegação estruturada (arquitetura, símbolos, caminhos de chamada); **sem kit.graph, use grep/glob no código legado** — degradação `grep-fallback` declarada e nunca bloqueante
- Pergunte ao usuário no chat (ou via `ask`, se disponível) sobre pontos não encontrados na análise — nunca adivinhe

## Modo de erro

`skip` — se o legado não for encontrado, pula com log warning.

## Algoritmo

1. Confirme no PROJECT_CHARTER.md que `legado: true` ou `sistema_existente: true` — se não, encerre (skill condicional)
2. Inspecione o sistema existente: stack, arquitetura, banco de dados (tabelas, volumes), APIs e regras vigentes
3. Use buscas no código e schemas legados (via kit.graph se disponível, senão grep/glob); pergunte no chat para pontos não encontrados na análise
4. Preencha as seções do template (stack, arquitetura, banco, APIs, regras, armadilhas, glossário de equivalência)
5. Escreva `reverse/LEGACY_CONTEXT.md`

## Formato de commit

```
docs: add legacy context for <produto>
```

## Template — LEGACY_CONTEXT.md

> **Regra de idioma:** Documento no idioma do cliente (padrão PT-BR). Títulos, descrições de tabelas, fluxos e regras no idioma combinado. Apenas nomes de classes, colunas de banco e identificadores técnicos permanecem no original.

```markdown
# Contexto Legado — [Sistema Existente]

## 1. Stack Tecnológica

| Componente | Tecnologia | Versão |
|---|---|---|
| Backend | | |
| Frontend | | |
| Banco de dados | | |
| Servidor | | |

## 2. Arquitetura Atual

[Descrição da arquitetura, diagramas se necessário]

## 3. Banco de Dados

### 3.1 Tabelas Principais

| Tabela | Finalidade | Estimativa de linhas |
|---|---|---|
| | | |

### 3.2 Queries Relevantes

```sql
-- queries que serão reutilizadas ou adaptadas
```

## 4. APIs e Endpoints

| Endpoint | Método | Finalidade | Reutilizar? |
|---|---|---|---|
| | | | Sim/Não/Adaptar |

## 5. Regras de Negócio Existentes

| Código legado | Descrição | Vai mudar? |
|---|---|---|
| | | |

## 6. Armadilhas Conhecidas

- [ponto de atenção, risco de migração]

## 7. Glossário de Equivalência

| Termo legado | Termo novo | Observação |
|---|---|---|
| | | |
```

## Casos de Borda

- **`legado: false` e `sistema_existente: false`** — skill não se aplica; encerre com `skip` (Modo de erro).
- **Legado sem acesso a código/banco** — documente a partir de fontes secundárias (docs, relato do usuário) e marque as seções não verificadas.
- **Sistema em produção (`sistema_existente`)** — destaque na seção de Armadilhas os riscos de mudar comportamento vigente; a análise documenta, não substitui.
- **Tabela/API legada reaproveitada** — marque na coluna "Reutilizar?" (`Sim/Não/Adaptar`); o `gm-datamodel` usa essa marcação para FKs e constraints herdados.
- **Navegação de código sem kit.graph** — use grep/glob (degradação `grep-fallback`); em codebases grandes, comece pelos pontos de entrada e schemas de banco, e pergunte ao usuário onde concentrar a análise.

---

compatibility: multi-engine

> **Degradação de checkpoints:** esta skill não declara checkpoints de kit — funciona integralmente sem o kit presente. A capacidade `kit.graph` é opcional: sem o grafo de código, a navegação degrada para grep/glob (`grep-fallback`) sem bloquear a análise.
