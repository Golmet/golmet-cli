# Cenário — Suporte: incidente em produção

O primary `gm-support-orchestrator` recebe do operador:

> "Tivemos um incidente em produção ontem: o sistema ficou fora do ar por
> 2 horas. Preciso do post-mortem documentado e registrar isso como known
> error para o suporte."

## O que o e2e valida (roteamento do orchestrator)

1. **Post-mortem do incidente** → `gm-incident-postmortem` (timeline, causa
   raiz, ações corretivas)
2. **Known error** → `gm-known-error` (registrar em KNOWN_ERRORS.md para o
   suporte)
3. Roteia, não executa
