---
project: ASSIST-DOCS
module: spec-ui
audience: [tecnica]
status: draft
---

# Especificação de UI - Chamados de Assistência

> **Derivado de:** REQUIREMENTS.md (RN-001, RN-002, RN-003), specs/API_CONTRACTS.md

---

## 1. Atores e Telas

### Técnico de Campo

| Tela | Casos de Uso | RNs |
|---|---|---|
| Meus chamados atribuídos | UC-01, UC-02 | RN-001, RN-002 |
| Executar atendimento | UC-01 | RN-001 |

### Supervisor

| Tela | Casos de Uso | RNs |
|---|---|---|
| Fila de chamados abertos | UC-02 | RN-002 |

### Cliente

| Tela | Casos de Uso | RNs |
|---|---|---|
| Acompanhar chamado | UC-03 | RN-003 |

---

## 2. Telas

### 2.1 Executar atendimento (Técnico de Campo)

**Formulário (conclusão do chamado):**

| Campo | Tipo | Obrigatório | Validação frontend | Contrato |
|---|---|---|---|---|
| Fotos do equipamento | upload múltiplo | Sim | Ao menos 1 imagem | fotos[] |
| Relatório | textarea (2000) | Sim | Obrigatório, máx. 2000 caracteres | relatorio |

> RN-001: a conclusão é bloqueada no frontend enquanto fotos e relatório
> não estiverem preenchidos.

### 2.2 Fila de chamados abertos (Supervisor)

**Tabela:** chamados com status aberto, ações por linha.

| Ação | Contrato |
|---|---|
| Atribuir a técnico (select) | POST /chamados/{id}/atribuicao |

> RN-002: ao atribuir, o frontend exibe a confirmação com a data registrada
> pelo backend.

### 2.3 Acompanhar chamado (Cliente)

**Consulta pública:** campo de código do chamado.

| Elemento | Comportamento | Contrato |
|---|---|---|
| Código do chamado | entrada simples, sem login | GET /chamados/{codigo}/status-publico |

> RN-003: sem autenticação completa — apenas o código do chamado.

---

## 3. Fluxo de Navegação

```
Cliente (código) → status público
Técnico → meus chamados → executar atendimento → conclusão (RN-001)
Supervisor → fila aberta → atribuição (RN-002) → notificação ao técnico
```

---

## 4. Regras de UI Globais

- Todos os formulários exibem erro inline abaixo do campo.
- Upload de foto aceita JPG/PNG até 5 MB por arquivo.
- Tabelas paginam a cada 20 registros.
