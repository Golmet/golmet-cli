# Cenário — engenharia reversa multi-repo com fonte congelada

O primary `gm-doc` convoca o `gm-reverse-orchestrator` com:

> "Produto: <cliente>. Repos: <cliente>-API (backend .NET) e <cliente>-FE (frontend
> Angular), ambos em produção. Fonte congelada: <cliente>-API @ HEAD `a1b2c3d`,
> <cliente>-FE @ HEAD `d4e5f6a`. Preciso da análise completa: modelos, contratos,
> regras e dados estruturados para eu consolidar."

Repos grandes (varredura pesada). O orchestrator NÃO executa git (shell: ask
— sincronização é papel do gm-doc).

## O que o e2e valida (roteamento do orchestrator)

1. **Stack por repo identificada primeiro (Fatia 1)** antes do conteúdo
2. **Sharding em fatias** (6 fatias) para varredura pesada
3. Extração na **tríade O/I/U** (fonte de dados, interface, uso)
4. **Ledger UNKNOWNS** para pontos não encontrados
5. Saída **estruturada para o gm-doc consolidar** — o orchestrator não consolida
6. **Não executa git** — sincronização/snapshot solicitados ao gm-doc
