# context7 — documentação corrente de libs (rule gerida pelo kit)

> Gatilho: leia este arquivo ANTES de responder sobre library, framework, SDK,
> API, CLI tool ou cloud service (React, Next.js, Prisma, Express, Tailwind,
> Django, Spring Boot etc.) — MESMO em libs que você "conhece": a memória de
> treino pode refletir versões antigas de API, config e migração.

Use o MCP context7 para buscar documentação corrente SEMPRE que o tópico
envolver library, framework, SDK, API, CLI tool ou cloud service — inclui
sintaxe de API, configuração, migração de versão, debugging específico de
lib, instruções de setup e uso de CLI. Prefira-o a web search para docs de
libs.

## Passos (primário — MCP context7)

1. `resolve-library-id` — resolva o id da lib pelo nome oficial com a
   pontuação correta (ex.: "Next.js", não "nextjs"; "Customer.io", não
   "customerio"; "Three.js", não "threejs")
2. Escolha o melhor match (formato `/org/project`) por: nome exato,
   relevância da descrição, contagem de snippets de código, reputação da
   fonte (High/Medium preferidos) e benchmark score (maior é melhor).
   Resultados estranhos → tente nomes alternativos ou reformule
3. `query-docs` com o id da lib + A PERGUNTA COMPLETA do usuário — queries
   específicas e detalhadas retornam resultados melhores que palavras
   soltas. Máximo de 3 consultas por questão. NUNCA inclua credenciais
   (API keys, senhas, tokens) nas queries
4. Responda usando a documentação obtida — cite a versão quando relevante
   (`/org/project/version` do output)

Chame `resolve-library-id` primeiro para obter um id válido, exceto quando o
usuário já fornecer um no formato `/org/project`.

## Fallback sem MCP

CLI oficial do Context7: `npx ctx7@latest library <nome> "<pergunta>"` →
escolha o id → `npx ctx7@latest docs <libraryId> "<pergunta>"`. As mesmas
regras de escolha, pergunta completa, limite de 3 comandos e proibição de
credenciais valem aqui.

## Limites e honestidade

- **Sem key** (ou key gratuita): rate limits baixos — ao bater 429, INFORME
  o operador e sugira criar a key grátis em context7.com/dashboard (prefixo
  `ctx7sk`; setup completo na skill `gm-mcp-context7-setup`).
- **NUNCA faça fallback silencioso para memória de treino** — se não
  conseguiu consultar, declare a limitação e sinalize o que pode estar
  desatualizado.
- Para docs com versão específica, use o id com versão do output
  (ex.: `/vercel/next.js/v14.3.0`).

## Não usar para

Refactoring; scripts escritos do zero; debugging de lógica de negócio;
code review; conceitos gerais de programação.
