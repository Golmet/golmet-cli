# API Contracts - Consulta

## Saldo

### GET /saldo

Saldo disponível por safra do produtor autenticado.

**Response 200:** `{ "safra": string, "saldo": number }`
**Trace:** RN-001

## Boletos

### GET /parcelas/{id}/boleto

Boleto da parcela em PDF.

**Response 200:** application/pdf
**Erros:** 404 (parcela sem boleto)
**Trace:** RN-002
