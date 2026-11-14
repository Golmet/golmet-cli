# Relato de Incidente — indisponibilidade MeuApp-API (01/01/2026)

> Registrado pelo operador de plantão na reunião de triagem.

## Detalhes coletados

- **Detecção:** 08:10 — alerta de latência do monitoramento (p95 > 30s em POST /pedidos)
- **Sintoma:** clientes do MeuApp não conseguem finalizar pedidos; timeout após 30s
- **Severidade percebida:** alta
- **Módulo afetado:** MEUAPP (MeuApp-API)
- **Funcionalidades:** RN-001 (finalização de pedidos) indisponível; RN-002 (baixa de estoque) degradada
- **Mudança recente:** deploy v1.8.0 da MeuApp-API às 07:45
- **Causa percebida:** logs exibem timeout ao obter conexão do pool após o deploy
- **Ações imediatas:** rollback para v1.7.9 às 09:20; normalização confirmada às 09:50
- **Duração:** 08:10 → 09:50 (1h 40min)
- **Usuários afetados:** todos os operadores do painel web no período
- **Lições preliminares:** deploy sem verificação de carga em rota crítica
