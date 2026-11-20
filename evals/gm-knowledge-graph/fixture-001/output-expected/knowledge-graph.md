# Grafo do Conhecimento (T1-T5)

> Gerado por gm-knowledge-graph (scripts/build-knowledge-graph.mjs — determinístico).
> Gerado em 2026-11-20T19:58 · modo: global · escopo: DemoApp

## Contagem por tipo

| Tipo | Nós |
|---|---|
| t1_dominio | 2 |
| t1_indice | 1 |
| t2_pack | 1 |
| t2_pattern | 2 |
| t3_rule | 1 |
| t4_adr | 3 |
| t4_feature | 3 |
| t4_modulo | 1 |
| t4_pend | 2 |
| t4_rn | 2 |
| t4_spec | 3 |
| t5_config | 1 |
| t5_dominio | 1 |
| t5_primary | 1 |
| t5_skill | 1 |
| t5_subagent | 1 |

**Arestas:** 39

**Bases T1 stale:** legado

## Mermaid (top 26 nós por grau)

```mermaid
graph LR
  t4_modulo_DemoApp["DemoApp"]
  t4_rn_DemoApp_RN_002["RN-002 — Pagamentos"]
  t4_rn_DemoApp_RN_001["RN-001 — Pedidos"]
  t5_subagent_gm_demo_backend["gm-demo-backend"]
  t5_primary_gm_demo["gm-demo"]
  t5_skill_gm_demo_skill["gm-demo-skill"]
  t4_adr_DemoApp_ADR_003["ADR-003 — Desconto no gateway"]
  t4_pend_DemoApp_PEND_002["PEND-002 — Desconto progressivo"]
  t4_spec_DemoApp_API_CONTRACTS["API_CONTRACTS.md"]
  t4_spec_DemoApp_DATA_MODEL["DATA_MODEL.md"]
  t1_dominio_demo["demo"]
  t1_indice["domains/INDEX.md (demo)"]
  t2_pack_demo["pack demo"]
  t2_pattern_validacao_dupla["validacao-dupla"]
  t4_adr_DemoApp_ADR_001["ADR-001 — Validador centralizado"]
  t4_adr_DemoApp_ADR_002["ADR-002 — Gateway de pagamento"]
  t4_feature_DemoApp_RN_001["RN-001 — Pedidos"]
  t4_feature_DemoApp_RN_002["RN-002 — Pagamentos"]
  t4_pend_DemoApp_PEND_001["PEND-001 — Fatura"]
  t4_spec_DemoApp_UI_SPEC["UI_SPEC.md"]
  t5_dominio_demo["demo"]
  t1_dominio_legado["legado"]
  t2_pattern_import_legado["import-legado"]
  t3_rule_org["org"]
  t4_feature_DemoApp_PEND_001["PEND-001 — Fatura"]
  t5_config_agents["AGENTS.md (config)"]
  t4_modulo_DemoApp -->|abre_questao| t4_pend_DemoApp_PEND_001
  t4_modulo_DemoApp -->|abre_questao| t4_pend_DemoApp_PEND_002
  t5_dominio_demo -->|agrega| t5_skill_gm_demo_skill
  t4_rn_DemoApp_RN_001 -->|ancora| t4_feature_DemoApp_RN_001
  t4_rn_DemoApp_RN_002 -->|ancora| t4_feature_DemoApp_RN_002
  t2_pattern_validacao_dupla -->|aplica_em| t4_modulo_DemoApp
  t4_adr_DemoApp_ADR_001 -->|artefato| t4_modulo_DemoApp
  t4_adr_DemoApp_ADR_002 -->|artefato| t4_modulo_DemoApp
  t4_adr_DemoApp_ADR_003 -->|artefato| t4_modulo_DemoApp
  t4_feature_DemoApp_PEND_001 -->|artefato| t4_modulo_DemoApp
  t4_feature_DemoApp_RN_001 -->|artefato| t4_modulo_DemoApp
  t4_feature_DemoApp_RN_002 -->|artefato| t4_modulo_DemoApp
  t4_spec_DemoApp_API_CONTRACTS -->|artefato| t4_modulo_DemoApp
  t4_spec_DemoApp_DATA_MODEL -->|artefato| t4_modulo_DemoApp
  t4_spec_DemoApp_UI_SPEC -->|artefato| t4_modulo_DemoApp
  t4_rn_DemoApp_RN_001 -->|cobre| t4_spec_DemoApp_API_CONTRACTS
  t4_rn_DemoApp_RN_001 -->|cobre| t4_spec_DemoApp_DATA_MODEL
  t4_rn_DemoApp_RN_002 -->|cobre| t4_spec_DemoApp_API_CONTRACTS
  t4_rn_DemoApp_RN_002 -->|cobre| t4_spec_DemoApp_DATA_MODEL
  t4_rn_DemoApp_RN_002 -->|cobre| t4_spec_DemoApp_UI_SPEC
  t4_adr_DemoApp_ADR_001 -->|decide| t4_rn_DemoApp_RN_001
  t4_adr_DemoApp_ADR_002 -->|decide| t4_pend_DemoApp_PEND_001
  t4_adr_DemoApp_ADR_003 -->|decide| t4_pend_DemoApp_PEND_002
  t4_adr_DemoApp_ADR_003 -->|decide| t4_rn_DemoApp_RN_002
  t4_modulo_DemoApp -->|define| t4_rn_DemoApp_RN_001
  t4_modulo_DemoApp -->|define| t4_rn_DemoApp_RN_002
  t5_primary_gm_demo -->|delega| t5_subagent_gm_demo_backend
  t5_primary_gm_demo -->|disponivel_para| t5_subagent_gm_demo_backend
  t5_skill_gm_demo_skill -->|fundado_em| t1_dominio_demo
  t1_indice -->|indice| t1_dominio_demo
  t1_indice -->|indice| t1_dominio_legado
  t5_config_agents -->|indice_lazy| t3_rule_org
  t5_subagent_gm_demo_backend -->|invoca| t5_skill_gm_demo_skill
  t5_primary_gm_demo -->|invoca_direta| t5_skill_gm_demo_skill
  t5_subagent_gm_demo_backend -->|orquestra| t5_dominio_demo
  t5_subagent_gm_demo_backend -->|owner| t5_primary_gm_demo
  t2_pattern_import_legado -->|pertence| t2_pack_demo
  t2_pattern_validacao_dupla -->|pertence| t2_pack_demo
  t4_pend_DemoApp_PEND_002 -->|promove| t4_rn_DemoApp_RN_002
```

## Avisos (1)

- dir ausente para fundado_em: config/agents
