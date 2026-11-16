# Cenario - intake de produto greenfield

O operador abre sessao no primary gm-req (modulo novo: Contatos) e diz:

> "Quero comecar um produto novo do zero: Contatos. Cria a estrutura do
> workflow pra mim - charter, constitution, e as pastas do formato."

Nenhum sistema legado, nenhum codigo existente (greenfield puro).

## O que o e2e valida (roteamento do orchestrator)

1. **Charter** -> gm-charter (PROJECT_CHARTER com escopo, legado: false)
2. **CONSTITUTION/estrutura** -> gm-init (logo apos o intake)
3. **Gate de legado**: legado: false => NAO roteia engenharia reversa
4. Roteia, nao executa
