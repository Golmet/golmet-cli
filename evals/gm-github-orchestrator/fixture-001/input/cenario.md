# Cenário — GitHub: issues das tasks + PR da task aprovada

O primary `gm-dev` convoca o `gm-github-orchestrator` com dois pedidos:

> "1) Publica as tasks do módulo no GitHub como issues.
>  2) E a task RN-002 já aprovada pelo reviewer — abre o PR."

Além disso: o `gh` CLI não está autenticado nesta máquina (pré-condição falha).

## O que o e2e valida (roteamento do orchestrator)

1. "Publicar tasks" → `gm-gh-issue-creator`
2. "PR da feature aprovada" → `gm-gh-pr-creator`
3. Pré-condição `gh` falha → `gm-gh-setup`
4. Roteia, não executa
