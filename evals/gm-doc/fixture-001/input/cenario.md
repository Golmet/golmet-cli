# Cenário — engenharia reversa multi-repo no gm-doc

O operador abre sessão no primary `gm-doc` (módulo do produto) e solicita:

> "Documente o sistema legado do produto por engenharia reversa: os repos
> da API e do front estão em produção, implementados fora do workflow.
> Quero o contexto do legado e as specs atualizadas."

Código multi-repo existente, varredura pesada necessária.

## O que o e2e valida (roteamento do primary)

1. A **varredura pesada de código** é delegada ao `gm-reverse-orchestrator`
   (nunca inline no primary)
2. O prompt do delegado inclui **fonte congelada** (hashes/HEAD informados)
3. O produto do ciclo: `LEGACY_CONTEXT.md` + specs atualizadas
4. O primary consolida e documenta — não varre código ele mesmo
