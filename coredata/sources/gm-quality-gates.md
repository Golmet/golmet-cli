# gm-quality-gates — Checklist Universal de Qualidade de Código

Skill transversal de qualidade para qualquer stack backend e frontend
Angular. Verifica princípios de código, padrões de teste, segurança e
performance. **Não substitui** o playbook da stack — complementa com
verificações transversais que se aplicam a qualquer linguagem.

## Lê (upstream)

- Código-fonte do repo alvo (read-only) — arquivos modificados pela task
- Playbook da stack carregado (se existir) — para comandos de verificação
- `AGENTS.md` do repo — convenções específicas
- Base de conhecimento do domínio (T1): `knowledge/security-review/KNOWLEDGE.md`
  no config do engine — fonte canônica dos critérios de segurança
  (seção D — OWASP ASVS 5.0.0 e Top 10:2025)

## Gera (downstream)

- Nenhum artefato de módulo — esta skill gera checklist e métricas para
  o relatório do `gm-dev-backend`

## Nunca edita

- Código-fonte (read-only)
- Artefatos de spec do módulo

## Tools

- `file`/`glob`/`grep` — inspecionar código (read-only)
- `shell` — executar comandos de formatação/lint da stack

## Modo de erro

`warn` — ferramenta de automação indisponível ou com falha não bloqueia a
auditoria: pule o comando, registre como "não executado" no relatório e
siga com a verificação manual das seções A-E. Quality gate `FAIL` não
aborta a skill — apenas bloqueia o `[done]` da task no `gm-dev-backend`.

## A — Princípios de Código

### KISS (Keep It Simple, Stupid)

> Manter soluções o mais simples possível. Se uma solução tem mais de 2
> alternativas, escolha a mais direta e legível.

| Regra | Verificação | Severidade |
|-------|-------------|------------|
| Método com ≤ 1 nível de abstração | Cada função faz UMA coisa | error |
| Método com ≤ 30 linhas | `wc -l` ou contagem manual | error |
| Solução mais direta escolhida | Sem frameworks complexos para problemas simples | warning |
| Sem over-engineering | Classes/interfaces que não são usadas pela task atual | warning |

### YAGNI (You Ain't Gonna Need It)

> Não implemente funcionalidade futura. Só code o que a spec pede HOJE.

| Regra | Verificação | Severidade |
|-------|-------------|------------|
| Sem código "para o futuro" | Classes/métodos que não são chamados pela task | error |
| Sem abstrações genéricas prematuras | Interfaces com 1 implementação | warning |
| Sem features não solicitadas | Tudo que foi implementado mapeia para RN | error |
| Sem parametrização excessiva | Config options que ninguém usa | warning |

### DRY (Don't Repeat Yourself)

> Extraia lógica duplicada para métodos, services ou herança.

| Regra | Verificação | Severidade |
|-------|-------------|------------|
| Code duplication < 3% | Blocos > 10 linhas duplicados | error |
| Lógica de negócio não duplicada | Mesma regra em 2+ lugares → extrair | error |
| Validação não duplicada | Usar framework de validação (FluentValidation, etc.) | warning |
| Config não duplicada | Constantes centralizadas, não hardcoded em vários arquivos | warning |

### SOLID

| Princípio | Regra | Verificação | Severidade |
|-----------|-------|-------------|------------|
| **SRP** | 1 classe = 1 responsabilidade | Max 5 métodos públicos, max 3 concerns | error |
| **OCP** | Aberto para extensão, fechado para modificação | Strategy para > 3 branches no mesmo tipo | warning |
| **LSP** | Subtipos são substituíveis | Sem `NotImplementedException` em overrides | error |
| **ISP** | Interfaces pequenas | Max 5 métodos por interface | warning |
| **DIP** | Depender de abstrações | Constructor injection, nenhum `new` de serviço | error |

### Clean Code

| Regra | Verificação | Severidade |
|-------|-------------|------------|
| Nomes revelam intenção | `GetOverdueOrders()` não `GetOrders()` | warning |
| Parâmetros ≤ 4 | Mais que 4 → usar record/objeto | warning |
| Early return | Evitar if/else aninhado > 2 níveis | warning |
| Funções puras quando possível | Sem side effects em lógica de negócio | warning |
| Constantes nomeadas | Sem números mágicos (`MAX_RETRY = 3` não `if (count > 3)`) | error |

## B — Checklist de Código

### Erros (error — bloqueante)

- [ ] Método > 30 linhas → refatore (Extract Method)
- [ ] Classe > 300 linhas → divida por responsabilidade
- [ ] Catch vazio (`catch { }`) → registre log ou re-throw
- [ ] `Console.WriteLine` / `print` em código de produção → remova ou use logger
- [ ] `new` direto de serviço (DI container) → use constructor injection
- [ ] `async void` (exceto event handlers) → mude para `async Task`
- [ ] TODO sem ticket → `TODO(autor): descrição [TICKET-123]`
- [ ] Código comentado removido → use git history
- [ ] Números mágicos → extraia para constante nomeada
- [ ] SQL concatenado → use parameterized queries / ORM

### Warnings (warning — informativo)

- [ ] Método com muitos níveis de abstração → extraia helpers
- [ ] Classe com muitos dependencies → aplique ISP (interfaces menores)
- [ ] Parâmetros > 4 → use record/DTO
- [ ] Método faz muita coisa → divida (SRP)
- [ ] Código duplicado em 2+ arquivos → extraia para service/shared
- [ ] Validação inline → use framework (FluentValidation, etc.)
- [ ] Magic strings → extraia para constants ou enums

## C — Padrões de Teste

### Pirâmide de Testes

```
        /\
       / E2E \          ~10% — lento, caro, frágil
      /--------\
     /Integração\       ~20% — velocidade moderada
    /--------------\
   /  Unitários     \   ~70% — rápido, isolado, barato
  /------------------\
```

### AAA (Arrange, Act, Assert)

```csharp
[Fact]
public void CalculateTotal_WithValidItems_ReturnsSum()
{
    // Arrange
    var items = new[] { new Item(10), new Item(20) };
    var calculator = new OrderCalculator();

    // Act
    var result = calculator.CalculateTotal(items);

    // Assert
    Assert.Equal(30, result);
}
```

### Naming de Testes

| Stack | Formato | Exemplo |
|-------|---------|---------|
| .NET | `Method_Scenario_ExpectedResult` | `GetUser_WithValidId_ReturnsUser` |
| Java | `should ExpectedBehavior when Condition` | `shouldReturnUserWhenIdIsValid` |
| Rust | `fn test_<scenario>_<expected>` | `fn test_get_user_valid_id_returns_user` |

### Checklist de Testes

- [ ] 1 teste por RN mínimamente
- [ ] Cenários de erro cobertos (contrato)
- [ ] Testes independentes (ordem não importa)
- [ ] Sem dependência de estado externo (mock/in-memory)
- [ ] Determinísticos (sem random, sem time-dependent)
- [ ] Nome descreve comportamento testado
- [ ] AAA pattern (sem lógica no teste)
- [ ] Assertions ≤ 3 por teste (ideal: 1 por behavior)

## D — Segurança

Fonte canônica dos critérios: base T1 `knowledge/security-review/KNOWLEDGE.md`
(OWASP ASVS 5.0.0 + Top 10:2025 + Secure Code Review Cheat Sheet) —
tabela abaixo atualizada da lista 2021 para a **Top 10:2025** (A03
Software Supply Chain Failures e A10 Mishandling of Exceptional
Conditions são novas; Misconfiguration subiu para A02; Injection caiu
para A05; SSRF foi absorvido em A01). Papéis: esta tabela é referência
rasa no gate; a **triagem profunda de findings de segurança** é da skill
`gm-security-audit` (consumida pelo `gm-dev-reviewer`, eixo de segurança).

### OWASP Top 10:2025 — Checklist

| # | Risco (2025) | Verificação | Severidade |
|---|--------------|-------------|------------|
| A01 | Broken Access Control (SSRF absorvido) | Auth + AuthZ server-side por objeto, default deny, sem bypass por API direta (P-03) | error |
| A02 | Security Misconfiguration | Defaults seguros por ambiente, headers/TLS, debug off em prod, segredos fora do código (P-05) | warning |
| A03 | Software Supply Chain Failures | Triagem de CVE, versões pinadas, verificação de integridade de artefato/hash (P-06) | error |
| A04 | Cryptographic Failures | Algoritmos modernos, hash de senha dedicado + salt, CSPRNG, IV/nonce correto (P-07) | error |
| A05 | Injection | Queries parametrizadas / ORM, encode contextual na saída, allowlist (P-02) | error |
| A06 | Insecure Design | Abuse cases em features novas, rate limit/quota em fluxos sensíveis (P-10) | warning |
| A07 | Authentication Failures | Rate limiting, lockout, hash de senha dedicado, gestão de sessão (P-03/P-07) | error |
| A08 | Software or Data Integrity Failures | Pinning/assinatura de artefatos e updates, pipeline protegido (P-06) | warning |
| A09 | Security Logging & Alerting Failures | Eventos de segurança logados sem PII e com alerta correlato (P-08) | warning |
| A10 | Mishandling of Exceptional Conditions | Fail closed, erros sem disclosure de stack/dado sensível (P-04) | error |

### Validação de Input

- Usar framework de validação (FluentValidation, Jakarta Validation, etc.)
- Validar no handler/command, não no controller
- Limits de tamanho em strings (MaximumLength)
- Tipos primitivos validados (não confiar em casting)

### Secret Management

- [ ] Nenhum segredo hardcoded no código
- [ ] Segredos via environment variables ou secret manager
- [ ] `appsettings.json` sem segredos reais
- [ ] `.gitignore` exclui arquivos com segredos

## E — Performance

### N+1 Queries

```csharp
// ERRADO: N+1 queries
var orders = await context.Orders.ToListAsync();
foreach (var order in orders)
{
    order.Customer = await context.Customers.FindAsync(order.CustomerId); // N queries!
}

// CORRETO: Eager loading
var orders = await context.Orders
    .Include(o => o.Customer)
    .ToListAsync(); // 1 query
```

| Check | Severidade |
|-------|------------|
| Queries em loop → eager loading | error |
| `ToList()` antes de filtro → filtrar antes | warning |
| `AsNoTracking()` em queries de leitura | warning |
| `Include()` faltando → N+1 | error |

### Async/Await

| Check | Severidade |
|-------|------------|
| `.Result` / `.Wait()` / `.GetAwaiter().GetResult()` → deadlock | error |
| `Task.Run()` sem necessidade → thread pool starvation | warning |
| CancellationToken não propagado → memória leak | error |
| `ConfigureAwait(false)` em libraries | warning |

### Memória

| Check | Severidade |
|-------|------------|
| IDisposable sem `using` / `await using` | error |
| Coleções estáticas crescendo ilimitado | warning |
| Streams não disposed | error |
| Event handlers não unsubscribed | warning |

### Conexões

| Check | Severidade |
|-------|------------|
| Connection pooling habilitado | warning |
| Connection string com pooling params | warning |
| HttpClient sem IHttpClientFactory | error |

## F — Automação por Stack (fonte única: playbook da stack)

Os comandos de automação **não são duplicados nesta skill** — consulte a
seção "Comandos de build e verificação" / "Quality Checks" do playbook da
stack carregado (`gm-stack-dotnet-csharp`, `gm-stack-java-quarkus`,
`gm-stack-angular`, `gm-stack-rust`), que é a **fonte única**.
Sem playbook carregado, aplique apenas o checklist manual (A-E) e registre
automação como "não executada".

**Interpretação de resultados frontend Angular:** budget de bundle
excedido (`angular.json` budgets) = warning; erro de compilação
TypeScript = error; componente sem spec = warning.

## G — Métricas

Colete e reporte as seguintes métricas:

### Complexidade

| Métrica | Limite | Ação |
|---------|--------|------|
| Complexidade ciclomática | ≤ 10 por método | Refatorar se > 10 |
| Complexidade cognitiva | ≤ 15 por método | Refatorar se > 15 |
| Linhas por método | ≤ 30 | Extract Method |
| Linhas por classe | ≤ 300 | Dividir responsabilidade |

### Cobertura

| Métrica | Mínimo | Ação |
|---------|--------|------|
| Line coverage (novo código) | ≥ 80% | Adicionar testes |
| Branch coverage | ≥ 75% | Testar todos os caminhos |
| Mutation score | ≥ 70% | Mutation testing |

### Duplicação

| Métrica | Limite | Ação |
|---------|--------|------|
| Code duplication rate | < 3% | Extrair método compartilhado |
| Blocos duplicados > 10 linhas | 0 | Refatorar imediatamente |

### Technical Debt

| Métrica | Limite | Ação |
|---------|--------|------|
| Debt ratio | < 5% | Corrigir issues acumuladas |
| Issue density | < 1/1000 linhas | Resolver issues críticas |

## G2 — Triagem de Severidade com Jev (opcional, não bloqueante)

Issues ambíguas entre níveis (ex.: número mágico que pode ser regra de
negócio oculta × simples estilo) podem receber uma **segunda opinião
calibrada** do modelo Jev antes de classificar. Uso opcional e por exceção:
apenas quando a severidade não for óbvia pelas tabelas A-F. Executor:
quem a tem** (ex.: primary `gm-dev` executando inline) — subagents como o
`gm-dev-backend` usam o curl diretamente abaixo; os dois caminhos são
equivalentes.

**Chamada (PowerShell):**

```powershell
$key = [System.Environment]::GetEnvironmentVariable("TYPESAFE_API_KEY", "User")
$state = "Issue: <descricao da issue com trecho do codigo>`nContexto: <RN ou regra relacionada, se houver>"
$body = @{
  state = $state; model = "jev-latest"
  questions = @{
    severidade = @{ type = "choice"; criteria = @{ bloqueia_release = "Violacao de regra de negocio ou risco de bug em producao"; warning = "Divida tecnica ou legibilidade, sem quebra de regra"; info = "Estilo ou sugestao, sem efeito funcional" } }
    evidencia = @{ type = "noul"; instructions = "A issue tem evidencia objetiva no codigo, sem depender de contexto adicional?" }
  }
} | ConvertTo-Json -Depth 10
$r = Invoke-RestMethod -Uri "https://api.typesafe.ai/v1/systemone" -Method Post -ContentType "application/json" -Headers @{ Authorization = "Bearer $key" } -Body ([System.Text.Encoding]::UTF8.GetBytes($body))
# $r.answers.severidade.choice + .confidence ; $r.answers.evidencia.noul
```

**Regras:**

- Sem chave de API, sem rede ou confidence < 0.5 → classifique pelas
  tabelas A-F e siga (a triagem Jev não aconteceu — não mencione no relatório)
- Confidence ≥ 0.5 → use a severidade sugerida e registre no relatório:
  `severidade: <nivel> (Jev, conf. <0.00-1.00>)`
