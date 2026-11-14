# Envelope único de erro HTTP nas APIs

**Categoria:** erros
**Tipo:** codigo
**Projetos que usaram:** MeuApp (2026-10), AppIrmao (2026-08)
**Stacks aplicáveis:** dotnet
**Maturidade:** validado
**Última revisão:** 2026-11-12
**Status:** ativo
**Superado por:** —
**Evidência:** MeuApp-API PR #42

## Cenário

Quando o frontend precisa tratar toda resposta de erro da API com a mesma
forma, sem que cada endpoint invente seu próprio formato de mensagem.

## Descrição

Toda resposta de erro HTTP carrega o mesmo envelope: código de negócio,
mensagem pronta para exibição e detalhes técnicos opcionais. O frontend
implementa um único tratamento de erro.

## Implementação

### .NET C#

```csharp
public record ErroApi(string Codigo, string Mensagem, IReadOnlyList<string>? Detalhes = null);
```

## Quando NÃO usar

APIs públicas com contrato de erro já definido por terceiros.

## Variações

Envelope com trace-id quando há observabilidade distribuída no produto.

## Referências

- Retrospectiva: MeuApp/RETROSPECTIVE.md
- Evidência de aplicação: MeuApp-API PR #42
