# Cenário — Figma: prompt para protótipo no Figma Make

O primary `gm-req` (módulo Contatos, specs fechadas) convoca o
`gm-figma-orchestrator`:

> "Preciso do prompt para gerar o protótipo das telas no Figma Make."

## O que o e2e valida (roteamento do orchestrator)

1. **Prompt para protótipo no Figma Make** → `gm-figma-prompt`
2. Se o contexto de design (identidade/estilo) ainda não existir →
   `gm-design-brief` primeiro
3. "Tasks a partir do design real" → `gm-figma-tasks` (fase posterior,
   com fileKey)
4. Roteia, não executa
