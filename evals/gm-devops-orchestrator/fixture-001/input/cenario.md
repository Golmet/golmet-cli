# Cenário — triagem de DevOps no orchestrator

O primary `gm-devops` convoca o orchestrator com dois pedidos simultâneos:

> "1) VPS nova do zero para hospedar o produto.
>  2) E agora o deploy de hoje está falhando — container unhealthy."

## O que o e2e valida (roteamento do orchestrator)

1. O orchestrator **roteia, não executa** — valida pré-condições e delega
2. VPS nova (sem sintoma) → `gm-vps-setup`
3. **Sintoma pós-deploy em VPS configurada → SEMPRE `gm-vps-troubleshoot`**
   (nunca o setup; nunca o próprio orchestrator)
