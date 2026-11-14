# PEDIDO — Criar primary agent gm-bar (fixture de eval)

> **Contexto:** projeto fictício MeuApp. Requisitos já coletados via
> questionário (3 blocos) — a skill não precisa perguntar novamente.

## Bloco 1 — Identificação

- **Nome:** `gm-bar`
- **Domínio de atuação:** auditoria de qualidade de cobertura — conduz a
  sessão principal quando o trabalho é diagnóstico de cobertura por RN,
  lacunas de teste e relatórios de qualidade do módulo.

## Bloco 2 — Subagents e roteamento

- **Subagents existentes que invoca:** `@gm-foosub` (já existe em
  `agents/gm-foosub.md` — diagnóstico read-only de cobertura).
- **Subagents novos:** nenhum.
- **Skills por subagent:** `gm-foosub` não invoca skills.

## Bloco 3 — Comportamento

- **Regras de sessão:** iniciar verificando o estado dos artefatos de
  qualidade do módulo; comportamento padrão de diagnóstico antes de
  edição; limite — confirmar com o usuário antes de commit/push/PR.
- **Permissão:** ampla (padrão do Quadri — igual `gm-req`/`gm-doc`).
