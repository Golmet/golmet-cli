---
name: gm-criteria
description: Gera specs/ACCEPTANCE_CRITERIA.md a partir das RNs em REQUIREMENTS.md — cada RN vira um ou mais critérios Dado-Quando-Então (Gherkin).
requires: [file, ask]
degrades:
  ask: chat-stop
---
# gm-criteria — Critérios de Aceitação

Gera `specs/ACCEPTANCE_CRITERIA.md` no workspace com critérios de aceitação
derivados das regras de negócio no formato Dado-Quando-Então (Gherkin).

## Referências

- Entidade RN (seções `## RN-NNN — <título>`): FORMATO §7.2
- IDs RN-XXX: FORMATO §4

## Lê (upstream)

- `<Produto>/REQUIREMENTS.md` — todas as RNs (seções `## RN-NNN — <título>`)
- `<Produto>/glossaries/` (se existir, para consistência terminológica; fallback: raiz do workspace)

## Gera (downstream)

- `<Produto>/specs/ACCEPTANCE_CRITERIA.md`

## Nunca edita

- `REQUIREMENTS.md`
- `golmet.index.json` (único escritor é o kit)

## Capacidades

- `file` — ler REQUIREMENTS.md e escrever ACCEPTANCE_CRITERIA.md
- `ask` — para ambiguidades de critério (sem ask nativo: pergunte no chat e PARE — nunca adivinhe)

## Modo de erro

`skip` — tier completo, não bloqueia.

## Algoritmo

1. Leia `REQUIREMENTS.md` e liste todas as RNs (seções `## RN-NNN — <título>`).
2. Para cada RN, classifique o tipo EARS (ubíqua, evento, estado, opcional, indesejado) usando a tabela "Regras de Derivação: EARS → Gherkin".
3. Derive um ou mais cenários Gherkin (Dado-Quando-Então) por RN, cobrindo o fluxo principal e as variações previstas na RN.
4. Em caso de ambiguidade de critério, use `ask` antes de escrever o cenário.
5. Escreva `specs/ACCEPTANCE_CRITERIA.md` com a estrutura do template — uma seção `## RN-XXX` por regra, com os cenários abaixo dela.

## Formato de commit

```
docs: add acceptance criteria for <produto>
```

## Regras de Derivação: EARS → Gherkin

| Tipo EARS (pt-BR) | Tradução Gherkin |
|---|---|
| Ubíqua (`O sistema deve...`) | Cenário: sempre válido. Dado que o sistema está operacional, Quando... |
| Evento (`Quando..., então o sistema deve...`) | Cenário: trigger. Quando [evento]... |
| Estado (`Enquanto..., o sistema deve...`) | Background: Dado que o sistema está em [estado]... |
| Opcional (`Caso..., o sistema deve...`) | Cenário: condicional. Dado que [funcionalidade] está habilitada... |
| Indesejado (`Se..., então o sistema deve...`) | Cenário: exceção. Quando [indesejado]... Então [resposta] |

## Template — ACCEPTANCE_CRITERIA.md

```markdown
# Acceptance Criteria — [Nome do Sistema]

> **Derivado de:** REQUIREMENTS.md
> **Última atualização:** DD/MM/AAAA

---

## RN-001 — [título da RN]

### Cenário: [nome do cenário]

- Dado que [contexto]
- Quando [ação]
- Então [resultado esperado]

### Cenário: [variação]

- Dado que [contexto]
- E [condição adicional]
- Quando [ação]
- Então [resultado]

---

## RN-002 — [título da RN]

...
```

## Casos de Borda

- **RN ubíqua sem gatilho** (`O sistema deve...`): gere um único cenário "sempre válido" — não invente evento que a RN não prevê.
- **RN com múltiplas variações** (opcional + indesejado na mesma regra): gere um cenário por variação, todos sob a mesma seção `## RN-XXX`.
- **RN de estado** (`Enquanto...`): use `Background` para a pré-condição e cenários separados para o comportamento decorrente.
- **REQUIREMENTS.md sem RNs**: não gere arquivo vazio — reporte a ausência e encerre (modo `skip`).
- **Termo ambíguo no critério** (ex: "rápido", "adequado"): use `ask` antes de escrever — não transporte a ambiguidade para o Gherkin. Sem ask nativo: pergunte no chat e PARE — nunca adivinhe.

---

compatibility: multi-engine

> **Degradação de checkpoints:** esta skill não declara checkpoints de kit — funciona integralmente sem o kit presente.
