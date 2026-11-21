# API Contracts - Chamados de Assistência

## Chamados

### POST /chamados/{id}/atribuicao

Atribui um chamado aberto a um técnico de campo.

**Request:** `{ "tecnico_id": number }`
**Response 200:** `{ "id": number, "tecnico_id": number, "data_atribuicao": "date-time" }`
**Erros:** 404 (chamado inexistente), 422 (chamado não está aberto)

### POST /chamados/{id}/conclusao

Conclui um chamado em atendimento.

**Request:** `{ "fotos": [string], "relatorio": string }`
**Response 200:** `{ "id": number, "status": "concluido" }`
**Erros:** 400 (sem foto ou relatório — RN-001), 422 (status inválido)

### GET /chamados/{codigo}/status-publico

Consulta pública do status por código do chamado (RN-003).

**Response 200:** `{ "codigo": string, "status": string, "atualizado_em": "date-time" }`
**Erros:** 404 (código inexistente)
