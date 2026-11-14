---
project: MEUAPP
module: requisitos
status: draft
---

# MeuApp — Requisitos de Agendamento

## 5. Regras de Negócio

### RN-001 [confirmado] (01/10/2026)
Quando uma reserva for criada para uma sala já ocupada no período, então o sistema deve recusar a reserva com mensagem de conflito.

### RN-002 [confirmado] (01/10/2026)
Se o usuário cancelar uma reserva confirmada, então o sistema deve liberar o período e registrar o cancelamento no histórico da sala.

## 11. Pendências e Dúvidas em Aberto

- **PEND-001** (01/10/2026) — Definir política de reservas recorrentes (aguardando decisão do cliente).
