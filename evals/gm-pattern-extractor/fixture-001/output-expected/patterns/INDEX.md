# Pattern Library — meuapp (accumulation/patterns)

> Padrões reutilizados extraídos de retrospectivas de projetos.
> Atualizado automaticamente por gm-pattern-extractor.
> Staleness (MP-03): idade (data_ultima_revisao > 6m) = GATILHO de revisão,
> nunca veredito de verdade (P-05). Verdade resolve-se por invalidação por
> contradição: conflito detectado marca o antigo status: superado +
> superado_por — histórico preservado, nunca delete (MP-13).

## Índice

| Padrão | Tipo | Categoria | Projetos | Maturidade | data_ultima_revisao | status |
|--------|------|-----------|----------|------------|---------------------|--------|
| [paginacao-offset](codigo/paginacao-offset/PATTERN.md) | codigo | api-design | MeuApp | validado | 2026-05-02 | ativo |
| [envelope-erro-api](codigo/envelope-erro-api/PATTERN.md) | codigo | erros | MeuApp, AppIrmao | validado | 2026-11-12 | ativo |
| [checklist-deploy-vps](processo/checklist-deploy-vps/PATTERN.md) | processo | deployment | MeuApp | novo | 2026-11-12 | ativo |

## Por Tipo

### Processo
- [checklist-deploy-vps](processo/checklist-deploy-vps/PATTERN.md) — checklist de deploy em VPS antes de atualizar containers (revisão: 2026-11-12)

### Arquitetura
- <padrão> — <descrição> (revisão: AAAA-MM-DD)

### Código
- [paginacao-offset](codigo/paginacao-offset/PATTERN.md) — Paginação offset/cursor (revisão: 2026-05-02)
- [envelope-erro-api](codigo/envelope-erro-api/PATTERN.md) — Envelope único de erro HTTP para todas as APIs (revisão: 2026-11-12)

### UI
- <padrão> — <descrição> (revisão: AAAA-MM-DD)

### Teste
- <padrão> — <descrição> (revisão: AAAA-MM-DD)

### Dados
- <padrão> — <descrição> (revisão: AAAA-MM-DD)
