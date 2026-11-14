# Formulário — Domínio Estoque

> Para o cliente/produto owner responder. Resposta LIVRE — escreva com
> suas palavras; não há múltipla escolha. O agente classifica a resposta
> no write-back e atualiza o REQUIREMENTS (RN/CR/PEND).

## Pergunta 1 — Tempo de reserva (CAND-002)

Quando um cliente separa um item no checkout, a reserva expira em 30
minutos e o item volta ao estoque. O prazo está correto?

**Detalhe técnico (apêndice):** evidência [legado/src/EstoqueService.cs:6 @f3e4a5b]

### Sua resposta

```
(escreva aqui)
```

---

> **Nota para o agente (write-back — vocabulário parseável fixo):**
> - **Confirma** → CAND-002 approved → gradua RN no REQUIREMENTS.md
> - **Confirma com correção** → RN nasce com o conteúdo corrigido pelo
>   cliente (a palavra dele vence o código); divergência vira nota; se
>   exigir mudança de código, pergunte sobre registro como CR
> - **Não é regra** → CAND-002 rejected + perguntar sobre CR (bug/decisão)
> - **Não sei** → CAND-002 → PEND-XXX no REQUIREMENTS.md (incerteza
>   registrada, nunca perdida)
