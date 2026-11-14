# Formulário — Domínio Pedidos

> Para o cliente/produto owner responder. Resposta LIVRE — escreva com
> suas palavras; não há múltipla escolha. O agente classifica a resposta
> no write-back e atualiza o REQUIREMENTS (RN/CR/PEND).

## Pergunta 1 — Cancelamento e estoque (CAND-003)

Quando um pedido é cancelado, os itens voltam para o estoque e ficam
disponíveis para venda novamente? Isso está correto, ou o comportamento
é outro (por exemplo, itens perdidos ou retorno manual)?

**Detalhe técnico (apêndice):** evidência [legado/src/PedidoService.cs:13 @f3e4a5b]

### Sua resposta

```
(escreva aqui)
```

---

> **Nota para o agente (write-back — vocabulário parseável fixo):**
> - **Confirma** → CAND-003 approved → gradua RN nova no REQUIREMENTS.md
> - **Confirma com correção** → RN nasce com o conteúdo corrigido pelo
>   cliente (a palavra dele vence o código); divergência vira nota; se
>   exigir mudança de código, pergunte sobre registro como CR
> - **Não é regra** → CAND-003 rejected + perguntar sobre CR (bug/decisão)
> - **Não sei** → CAND-003 → PEND-XXX no REQUIREMENTS.md (incerteza
>   registrada, nunca perdida)
