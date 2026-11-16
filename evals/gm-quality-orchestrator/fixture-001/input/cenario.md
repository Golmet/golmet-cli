# Cenario - camada de qualidade antes das features

O primary gm-req (modulo Contatos) tem as specs completas (DATA_MODEL,
API_CONTRACTS, UI_SPEC) e diz:

> "Specs fechadas. Antes de gerar as features, preciso da camada de
> qualidade: estrategia de testes, rastreabilidade e validacao da DAG."

## O que o e2e valida (roteamento do orchestrator)

1. **Test strategy + DoD** -> gm-test-strategy (T1/T2/T3/T7 existem)
2. **Matriz RN -> artefatos** -> gm-traceability (multiplos artefatos)
3. **Integridade da DAG** -> gm-validate (valida e gera VALIDATION_REPORT)
4. **Wiki-lint antes de entregar** -> gm-wiki-lint (contradicoes, orfaos, links)
5. Roteia, nao executa
