# REQUIREMENTS — MeuApp (fixture de eval)

> **Propósito:** workspace semeado para eval do corpus do kit — módulo
> Relatórios do produto fictício MeuApp.

## 5. Regras de Negócio

### RN-001 — Exportação de relatórios em PDF

O usuário autenticado pode exportar para PDF qualquer relatório que
esteja visualizando. O arquivo é gerado pelo serviço de relatórios do
MeuApp.API e fica disponível para download por 24 horas.

### RN-002 — Filtro por período nas exportações

A exportação deve refletir exatamente os filtros de data inicial e data
final aplicados na tela de visualização do relatório.

### RN-003 — Agendamento de envio automático de relatórios

O usuário pode agendar o envio automático de um relatório por e-mail nos
intervalos diário, semanal ou mensal, informando horário e destinatários.

## 11. Pendências (PENDs)

### PEND-001 — Retenção dos relatórios agendados

Definir por quanto tempo o PDF gerado por agendamento fica disponível
para download. Follow-up: confirmar política de retenção com a PO (Ana).
