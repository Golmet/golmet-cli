# Paginação offset/cursor

**Categoria:** api-design
**Tipo:** codigo
**Projetos que usaram:** MeuApp (2026-04)
**Stacks aplicáveis:** all
**Maturidade:** validado
**Última revisão:** 2026-05-02
**Status:** ativo
**Superado por:** —
**Evidência:** MeuApp-API PR #31

## Cenário

Quando um endpoint de listagem pode crescer sem limite prático e o
frontend precisa de navegação por páginas sem carregar tudo.

## Descrição

Paginar endpoints de listagem com parâmetros de offset e limite, devolvendo
o total e o intervalo na mesma resposta para o frontend montar a navegação.

## Implementação

### .NET C#

```csharp
public record Pagina<T>(IReadOnlyList<T> Itens, int Total, int Offset, int Limite);
```

## Quando NÃO usar

Conjuntos pequenos e estáticos — devolver a lista inteira é mais simples.

## Variações

Cursor quando a ordenação precisa ser estável entre páginas.

## Referências

- Retrospectiva: MeuApp/RETROSPECTIVE.md
- Evidência de aplicação: MeuApp-API PR #31
