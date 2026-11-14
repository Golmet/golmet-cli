# Cenário — VPS nova + pipeline de deploy no gm-devops

O operador abre sessão no primary `gm-devops` e solicita:

> "Preciso de uma VPS nova do zero para hospedar o produto e o pipeline
> de deploy do repo. Depois quero monitorar pra saber se cair."

Duas demandas encadeadas: infraestrutura nova + pipeline + observabilidade.

## O que o e2e valida (roteamento do primary)

1. **VPS nova** → `gm-vps-setup` com as **12 fases na ordem** e a regra
   **strict** do tunnel exclusivo
2. **Pipeline do repo** → `gm-cicd-workflow` em **modo 1** (gerar
   `deploy.yml` em dry-run com confirmação)
3. **Observabilidade** → `gm-observability` em config **mínima KISS**
   (healthcheck + watchdog), sem stack de monitoramento
4. **Sintomas pós-deploy** → `gm-vps-troubleshoot` (triagem pela tabela)
