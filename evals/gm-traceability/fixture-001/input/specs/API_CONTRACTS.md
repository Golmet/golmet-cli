---
project: MeuApp
module: contratos-api
audience: [tecnica]
status: draft
---

# API Contracts — MeuApp.API (módulo Relatórios)

> **Última atualização:** 05/10/2026

## Convenções

- Erros previstos em contrato: ver specs/ERROR_CATALOG.md.
- Todo endpoint exige usuário autenticado.

## Endpoints

### POST /api/relatorios/{id}/exportacoes

Gera o PDF do relatório (RN-001).

| Campo | Tipo | Obrigatório | Descrição |
|---|---|---|---|
| dataInicial | date | Sim | Início do período (RN-002) |
| dataFinal | date | Sim | Fim do período (RN-002) |
| incluirTotalizador | boolean | Não | Totalizador no rodapé do PDF (CR-001) |

Resposta: `202 Accepted` com o identificador da exportação e o link de
download válido por 24 horas (RN-001).

### POST /api/relatorios/{id}/agendamentos

Cria agendamento de envio automático por e-mail (RN-003).

| Campo | Tipo | Obrigatório | Descrição |
|---|---|---|---|
| frequencia | enum | Sim | diaria, semanal ou mensal (RN-003) |
| horario | time | Sim | Horário do envio (RN-003) |
| destinatarios | string[] | Sim | E-mails dos destinatários (RN-003) |
