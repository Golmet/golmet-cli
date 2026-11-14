---
name: gm-legacy-guide
description: Gera docs/legacy/ — a camada humana do legado para migrações e evolutivas (visão geral, mapa de impacto, guias por domínio como superfície de confirmação das CANDs, formulário para o cliente). Lê os artefatos de reverse + o código legado (read-only).
requires: [file, shell]
kit:
  graph: grep-fallback
---

# gm-legacy-guide — Camada Humana do Legado

O reverse produz **substrato de júri** (CANDIDATES com âncoras, COVERAGE,
LEGACY_CONTEXT — legível por máquina e auditoria). Esta skill produz a
**camada humana**: prosa organizada para (a) o operador CONFIRMAR as regras
extraídas e (b) qualquer pessoa entender e EVOLUIR o legado com segurança.

## Regra anti-alucinação (o coração)

**Todo fato técnico citado na camada humana rastreia para uma fonte**:
artefato de reverse (CANDIDATES/REVERSE_REPORT/LEGACY_CONTEXT), DECISIONS
da org, ou código real com path. Prosa sem fonte é o mesmo veneno do
reverse sem evidência. **Regra de negócio extraída do legado nasce CAND**
(pending) — nunca RN direta; a graduação só acontece com confirmação
humana (Passo 4).

## Lê (upstream)

- `reverse/CANDIDATES.md` (todas as CANDs: approved, pending, rejected)
- `reverse/REVERSE_REPORT.md` + `reverse/COVERAGE.md` + `reverse/UNKNOWNS.md`
- `support/LEGACY_CONTEXT.md` (stack, integrações, armadilhas)
- `DECISIONS.md` da org (decisões com dente: DEC-002 UTC-3 etc.)
- Código legado (read-only — agrupamento de domínios, paths reais)

## Gera (downstream) — `reverse/guide/` (a camada humana mora na pasta do legado; **nomes de ARQUIVO em inglês**, conteúdo no idioma do cliente)

| Artefato | Papel | Público |
|---|---|---|
| `guide/OVERVIEW.md` | Orientação: o que o sistema é, domínios (derivados do código), como se conversam, por onde começar | todos |
| `guide/IMPACT_MAP.md` | O mapa de minas: armadilhas técnicas globais + regras **confirmadas** com dente (penalidade/dinheiro/tenants), classificadas 🔴🟡🟢 | quem vai mexer (obrigatório antes) |
| `guide/GUIDES/<domínio>.md` | **A superfície de confirmação + evolutiva**: narrativa do domínio + TODAS as regras com ciclo de vida (✅ graduada RN-NNN / ⏳ CAND pendente) + arquivos reais + cenários de evolutiva | operador + implementador |
| `guide/FORMS/<domínio>.md` | O formulário do cliente: cada CAND pendente como pergunta em linguagem de cliente com **RESPONSE LIVRE** (sem múltipla escolha — o cliente escreve; o agente classifica no write-back) | o cliente/produto owner |

## Nunca edita

- `reverse/*` e `CANDIDATES.md` (o ledger é fonte do ESTADO — a skill lê e
  aponta, o write-back de confirmação é feito pelo workflow na conversa)
- REQUIREMENTS.md, charter, código legado

## Capacidades

- `file` — ler substrato e código, escrever docs/legacy/
- `shell` — glob/grep no código legado (agrupamento de domínios, paths reais)
- `ask` — modo de confirmação (conversa) e decisões de escopo

## Os 3 modos de confirmação (pergunte no Passo 2)

1. **Operador na conversa**: lote por domínio ou um a uma
2. **Cliente via formulário**: gera `guide/FORMS/<domínio>.md` para o cliente responder e devolver
3. **Híbrido (default recomendado)**: operador pré-triagem (regras óbvias → lote;
   regras que só o cliente sabe → formulário)

## O write-back do formulário (mapeamento fixo — o ciclo de vida da CAND)

| Resposta do cliente | Write-back |
|---|---|
| **Confirma** | CAND `approved` → gradua **RN-NNN** no REQUIREMENTS.md |
| **Confirma com correção** | RN nasce com o **conteúdo corrigido pelo cliente** (a palavra dele vence o código) + divergência código↔intenção vira nota — e se exigir mudança de código, o agente pergunta se registra como **CR** |
| **Não é regra** | CAND `rejected` + pergunta: o sistema faz X por acidente — registra como **CR** (bug/decisão-cliente)? |
| **Não sei** | CAND → **PEND-XXX** no REQUIREMENTS.md (incerteza registrada, nunca perdida) |

O formulário usa **vocabulário parseável fixo** (Confirma / Confirma com
correção / Não é regra / Não sei) para o read-back ser determinístico.

## Workflow

### Passo 1 — Derivar os domínios do código (nunca inventar)

Agrupe os controllers/features do código legado por prefixo/pasta
(ex.: `Agendamento*`, `WhatsApp*`, `Assinatura*` → domínio Agendamento).
**O domínio existe se e somente se existe código real para ele.**

### Passo 2 — Join das CANDs por domínio

Para cada domínio, junte as CANDs (qualquer status) cuja evidência
aponta para os arquivos do domínio. CANDs de arquivos não mapeados
ficam num bloco "sem domínio atribuído" no fim (nunca descartadas).

### Passo 3 — VISAO_GERAL + MAPA_IMPACTO

VISAO_GERAL: narrativa por domínio + mapa de conversação (só regras já
confirmadas alimentam o "como se conversam" — pendentes não afirmam).
MAPA_IMPACTO: armadilhas técnicas (LEGACY_CONTEXT + DECISIONS) + regras
**confirmadas com dente**, classificadas 🔴🟡🟢.

### Passo 4 — GUIAS: detalhe regra por regra (a superfície de confirmação)

Por domínio: narrativa (fluxo ponta a ponta em prosa) → arquivos reais →
**regras em 2 blocos**: ✅ já confirmadas (graduadas RN-NNN, prosa +
evidência + data) e ⏳ pendentes (CAND detalhada em prosa de negócio +
evidência + prompt de decisão). **Cenários de evolutiva** por domínio.

### Passo 5 — FORMS (se o modo for cliente/híbrido)

Por domínio: cada CAND pendente como pergunta em linguagem de cliente
(prosa de negócio; a âncora técnica vai para o apêndice), com **RESPONSE
LIVRE** — sem múltipla escolha (o vocabulário Confirma/Correção/Rejeita é
JARGÃO do workflow: nunca empurre para o cliente). O campo de resposta é
aberto; o agente classifica no write-back.

### Passo 6 — Write-back e regeneração (o agente classifica a resposta livre)

Leia a resposta livre do cliente e CLASSIFIQUE-A no vocabulário do
workflow (o cliente escreve à vontade; a decisão de classe é do agente,
documentada):
- resposta positiva, eventualmente corrigindo um detalhe → CAND `approved`
  (RN nasce com o conteúdo corrigido pelo cliente — a palavra dele vence o
  código) + divergência código↔intenção vira nota; se exigir mudança de
  código, pergunte se registra como **CR**
- resposta negativa ("não é assim que deveria ser") → CAND `rejected` +
  pergunte se registra como **CR** (bug/decisão-cliente)
- resposta incerta ("não sei", "preciso verificar") → CAND → **PEND-XXX**
  (incerteza registrada, nunca perdida)
- sem resposta no formulário devolvido → CAND fica pendente (nunca invente)
**Regenere os guias dos domínios afetados** (a regra confirmada muda de
⏳ para ✅ RN-NNN). Delta: reverse ganhou rodada nova → só os domínios
afetados regeneram.

## Escopo negativo

- Não documenta feature nova (isso é gm-howto pós-`[done]`)
- Não substitui o júri (CANDIDATES/COVERAGE ficam como estão)
- Não cria domínio sem código real, nem regra sem CAND (toda regra na
  camada humana nasceu CAND — a prosa narra, o ledger governa)

## Formato de commit

```
docs: camada humana do legado para <produto> (visão geral, mapa de impacto, guias)
```
