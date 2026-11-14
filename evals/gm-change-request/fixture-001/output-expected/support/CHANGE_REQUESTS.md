---
project: MeuApp
module: mudancas
audience: [tecnica, negocio]
status: draft
---

# Registro de Mudanças

## CR-001: Totalizador no rodapé do PDF

**Status:** implemented
**Data:** 28/09/2026
**Solicitante:** Ana (PO)
**RN afetada:** RN-001
**Origem:** requisito-mal-capturado
**Impacto visual:** não
**Design Figma:** n/a
**Descrição:** Incluir totalizador de registros no rodapé de cada página do PDF exportado.
**Justificativa:** Quem recebe o relatório impresso não identificava a quantidade de linhas ao paginar documentos longos.
**Downstreams afetados:** API_CONTRACTS, UI_SPEC
**Aprovação:** aprovado por Bruno (líder técnico) em 29/09/2026

---

## CR-002: Filtro por período relativo nas exportações

**Status:** pending
**Data:** 12/11/2026
**Solicitante:** Ana (PO)
**RN afetada:** RN-002
**Origem:** evolucao
**Impacto visual:** não
**Design Figma:** n/a
**Descrição:** Aceitar períodos relativos (últimos 7 dias e últimos 30 dias) no filtro de exportação, além das datas inicial/final explícitas. Validação de abordagem confirmada (Bloco 0): repositório alvo MeuApp.API, implementação como adaptação do endpoint de exportação existente (novo campo opcional no contrato), atores mantidos autenticados. Alternativa descartada: endpoint novo dedicado a períodos relativos.
**Justificativa:** Operadores consultam recorrentemente o mês corrente; digitar as datas manualmente é lento e gera erros de digitação (EXP-422).
**Downstreams afetados:** API_CONTRACTS, UI_SPEC
**Aprovação:** —
