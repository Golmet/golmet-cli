---
status: confirmed
---

# REQUIREMENTS - Carteira Digital (Módulo Pagamentos)

## Requisitos de Negócio

### RN-001

**Regra:** O sistema deve armazenar apenas o token do cartão fornecido pela
vault de tokenização — o número real (PAN) nunca é gravado no banco de dados
da aplicação.

### RN-002

**Regra:** A tabela de cartões herda a coluna `EMAIL_ALTERNATIVO` da
implementação anterior; no novo fluxo de carteira digital essa coluna não
recebe dados e não participa de nenhuma regra.
