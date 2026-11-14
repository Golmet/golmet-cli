---
project: MEUAPP
module: agent-health-report
audience: [governanca, tecnica]
status: draft
generated_at: "2026-01-01T08:00:00Z"
---

# Agent Health Report — MeuApp

> **Gerado em:** 01/01/2026 08:00
> **Registry consultado:** config/registry.yaml (2 skills, 1 subagent)
> **Resultado Geral:** ⚠️ Atenção necessária

---

## Resumo

| Verificação | Status | Detalhes |
|-------------|--------|----------|
| A. Subagents | ✅ | 1/1 conformes, 0 referências quebradas |
| B. Skills | ❌ | 1 skill com violações (calc-frete) |
| C. Artefatos Órfãos | ✅ | 0 órfãos |
| D. Staleness | N/A | Não avaliado — sandbox sem acesso a Git |
| E. Propagation | ⚠️ | 1 CR pendente antigo (CR-001, 17 dias) |
| F. Conformidade (checklists) | ❌ | 3 conformes / 1 componente com violações |
| G. Índice lazy (AGENTS.md) | N/A | AGENTS.md global fora do escopo do sandbox |
| H. Coerência PERMISSIONS.md | ⚠️ | PERMISSIONS.md ausente — matriz não mantida |

---

## A. Subagents Registrados

| Subagent | Name | Description | Permission | Status |
|----------|------|-------------|------------|--------|
| config/agents/revisor-pedidos.md | revisor-pedidos | Presente | Presente, restritivo | ✅ |

**Referências quebradas:** nenhuma

---

## B. Skills

| Skill | Front Matter | Seções | Placeholders | Status |
|-------|-------------|--------|--------------|--------|
| config/skills/calc-frete/SKILL.md | ✅ | ❌ | 1 | ❌ |
| config/skills/calc-imposto/SKILL.md | ✅ | ✅ | 0 | ✅ |

**Skills com problemas:** calc-frete — seções obrigatórias ausentes
(`## Capacidades` e `## Modo de erro`) e 1 placeholder de bloco por
preencher no corpo. Registry coerente com as pastas existentes (0 drift).

---

## C. Artefatos do Módulo

| Artefato | Existe | Referenciado | Última atualização | Staleness |
|----------|--------|-------------|--------------------|-----------|
| PROJECT_CHARTER.md | Sim | Sim (registry e relatório) | N/A (sem Git) | N/A |
| quality/PROPAGATION_STATUS.md | Sim | Sim (este relatório) | N/A (sem Git) | N/A |

**Artefatos órfãos:** nenhum

---

## D. Staleness

Não avaliado — sandbox sem acesso a Git (sem histórico de commits).

---

## E. Propagation Status

| CR | RNs afetadas | Aprovado em | Dias pendente | Situação |
|----|--------------|-------------|---------------|----------|
| CR-001 | RN-003 | 15/12/2025 | 17 | ⚠️ antigo (> 7 dias) |

---

## F. Conformidade com CHECKLISTS.md

| Componente | Tipo | Front Matter | Permissões | Portabilidade | Seções | Registro | Status |
|------------|------|--------------|------------|---------------|--------|----------|--------|
| calc-imposto | skill | ✅ | ✅ | ✅ | ✅ | ✅ registry | ✅ |
| calc-frete | skill | ✅ | ✅ | ✅ | ❌ | ✅ registry | ❌ |
| revisor-pedidos | subagent | ✅ | ✅ | ✅ | ✅ | ✅ registry | ✅ |

**Violações:** calc-frete — item da checklist de seções obrigatórias
(`## Capacidades`/`## Modo de erro` ausentes) e placeholder no corpo;
corrigir o SKILL.md completando as seções e removendo o marcador.
**Refs de tools/MCP pendentes (não-bloqueante):** nenhuma
**Não automatizáveis (revisão humana):** smoke test das duas skills em sessão real

---

## G. Índice lazy do AGENTS.md global

N/A — AGENTS.md global fora do escopo do sandbox (verificação roda
somente com a config completa do engine instalada).

---

## H. Coerência PERMISSIONS.md

PERMISSIONS.md ausente no sandbox — matriz não mantida (⚠️). Criar a
matriz agent × permissão com justificativa antes do próximo release.

## Recomendações

1. Corrigir calc-frete: adicionar `## Capacidades` e `## Modo de erro` e remover o placeholder de bloco
2. Criar PERMISSIONS.md e alinhar a matriz com os front matters existentes
3. Tratar CR-001 (RN-003) pendente há 17 dias — propagar ou arquivar com justificativa
