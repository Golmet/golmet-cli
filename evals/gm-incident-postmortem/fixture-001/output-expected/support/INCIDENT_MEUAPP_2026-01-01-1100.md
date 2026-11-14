---
project: MEUAPP
module: incident-postmortem
audience: [suporte, tecnica]
status: draft
incident_date: "2026-01-01"
severity: alta
duration: 1h 40min
---

# Post-Mortem — Timeouts na finalização de pedidos (MeuApp-API)

> **Data do incidente:** 01/01/2026 08:10
> **Severidade:** Alta
> **Módulo afetado:** MEUAPP (MeuApp-API)
> **Duração total:** 1h 40min (detecção → resolução)
> **Status:** Resolvido

---

## 1. Resumo Executivo

Após o deploy v1.8.0, o endpoint POST /pedidos passou a esgotar o pool
de conexões do banco e os clientes não conseguiam finalizar pedidos
(RN-001), com a baixa de estoque (RN-002) degradada no mesmo período.
O rollback para v1.7.9 normalizou o serviço em 30 minutos; a correção
definitiva foi aplicada na v1.8.1 com eager loading e pool ampliado.

---

## 2. Timeline do Incidente

| Hora | Evento | Responsável |
|------|--------|-------------|
| 07:45 | Deploy v1.8.0 da MeuApp-API | Time de dev |
| 08:10 | Detecção — alerta de latência p95 > 30s em POST /pedidos | Monitoramento |
| 08:25 | Diagnóstico — logs com timeout ao obter conexão do pool | Operador de plantão |
| 09:20 | Contenção — rollback para v1.7.9 | Operador de plantão |
| 09:50 | Resolução/Verificação — latência normalizada, pedidos fluindo | Operador de plantão |

---

## 3. Impacto

**Usuários afetados:**
- Todos os operadores do painel web MeuApp entre 08:10 e 09:50

**Funcionalidades impactadas:**
- RN-001 — finalização de pedidos indisponível
- RN-002 — baixa de estoque degradada (filas atrasadas)

**Período de impacto:**
- De 08:10 até 09:50 (total: 1h 40min)

**Dados afetados:**
- Nenhum — sem perda ou corrupção; pedidos em timeout foram reemitidos manualmente

---

## 4. Causa Raiz

**Técnica:**
A v1.8.0 introduziu uma consulta em loop (padrão N+1) ao listar itens
do pedido: para cada item, uma query separada buscava o cliente. Sob
carga real, as conexões do pool se esgotaram (max pool atingido) e as
requisições seguintes ficaram em fila até o timeout de 30s. O deploy
não passou por verificação de carga na rota crítica.

**Organizacional (se aplicável):**
Pipeline de release sem gate de teste de carga para rotas de escrita e
sem rollback automático por erro de latência pós-deploy.

---

## 5. Ações Corretivas

### Curto prazo (até 1 semana)

| # | Ação | Responsável | Prazo | Status |
|---|------|-------------|-------|--------|
| 1 | Corrigir N+1 com eager loading no carregamento de itens | Time de dev | 05/01/2026 | Pendente |
| 2 | Ampliar pool de conexões com limite monitorado | Time de dev | 05/01/2026 | Pendente |
| 3 | Release v1.8.1 com teste de carga em POST /pedidos | Time de dev | 07/01/2026 | Pendente |

### Longo prazo (até 1 mês)

| # | Ação | Responsável | Prazo | Status |
|---|------|-------------|-------|--------|
| 1 | Gate de teste de carga para rotas de escrita no pipeline | Time de dev | 25/01/2026 | Pendente |
| 2 | Rollback automático quando p95 exceder 10s por 3 minutos | Time de dev | 31/01/2026 | Pendente |

---

## 6. Lições Aprendidas

**O que funcionou bem:**
- Alerta de latência detectou o problema em menos de 30 minutos após o deploy
- Rollback rápido e ensaiado (procedimento documentado funcionou)

**O que poderia ter sido melhor:**
- Verificação de carga ausente antes do deploy v1.8.0
- Alarme de pool de conexões não existia — diagnóstico dependeu de leitura manual de logs

**Ações preventivas para o futuro:**
- Alarme de utilização do pool com limiar de 80%
- Checklist de release com teste de carga obrigatório em rotas de escrita

---

## 7. Follow-up: Known Error

> **Ação recomendada:** Execute `gm-known-error` para registrar este
> erro como conhecido em KNOWN_ERRORS.md (o arquivo ainda não existe
> no módulo; a primeira entrada criará o registro).

**Entrada sugerida:**

- **Data do incidente:** 01/01/2026
- **Status:** aberto
- **Sintoma:** timeouts de 30s em POST /pedidos após deploy, com erros de esgotamento de pool nos logs
- **Causa:** query N+1 na listagem de itens esgota o pool de conexões
- **Workaround:** rollback para a versão anterior enquanto a correção de eager loading não é aplicada
- **Resolução definitiva:** em andamento — ver ações corretivas (v1.8.1 e gate de carga)
