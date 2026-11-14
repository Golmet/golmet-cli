# Cenário — Governança: reestruturação de projeto legado

O operador abre sessão no `gm-governance-orchestrator`:

> "Tenho um projeto antigo na estrutura docs/projects/<nome>/ que precisa
> ser migrado para a estrutura de módulo atual. E de passada, revisa a
> CONSTITUTION que está desatualizada."

## O que o e2e valida (roteamento do orchestrator)

1. **Migrar estrutura legada** → `gm-restructure`
2. **Revisar CONSTITUTION/PRODUCT_GOAL** → `gm-governance` (baixa frequência
   — só quando os princípios precisam atualizar)
3. Roteia, não executa
