# gm-stack-rust — Playbook de Stack Rust

Playbook de conhecimento para executors de dev do workflow que implementam
código em repos Rust (projetos CORE). Contém o **como** da stack —
comandos, estrutura e padrões. O **o quê** (task, contrato, RNs) vem de
TASKS.md/API_CONTRACTS.md/REQUIREMENTS.md; as convenções específicas do
repo vêm do `AGENTS.md` do repo (precedência sobre este playbook).

## Escopo

- **Aplica-se a:** repos CORE Rust (Cargo.toml)
- **Não se aplica a:** repos que misturam Rust com outra stack principal (ex: Rust FFI em projeto C#)
- **Camada sobre:** `gm-dev-backend` — carrega este playbook quando detecta `Cargo.toml` no repo alvo

## Lê (upstream)

- `AGENTS.md` do repo alvo — convenções específicas do repo (precedência sobre este playbook)
- `Cargo.toml` (e `[workspace]`, quando houver) — edition, `rust-version`, dependências e features
- Código-fonte do repo (read-only) — `src/`, `tests/`, `benches/`, `migrations/`
- Artefatos de spec do módulo referenciados pela task: TASKS.md, API_CONTRACTS.md, ERROR_CATALOG.md, ACCEPTANCE_CRITERIA.md, REQUIREMENTS.md

## Gera (downstream)

- Nenhum artefato de módulo — este playbook só orienta a implementação aplicada pelos executors de dev (`gm-dev-backend`): código Rust, testes (unit/integração/doc-tests) e relatório da task

## Nunca edita

- Artefatos de spec do módulo (REQUIREMENTS.md, DECISIONS.md, DATA_MODEL.md, API_CONTRACTS.md, ERROR_CATALOG.md, ACCEPTANCE_CRITERIA.md, COMPLIANCE.md, TASKS.md) — read-only (exceto o que as docs vivas autorizarem explicitamente)
- `AGENTS.md` do repo alvo
- Scripts/ pipelines de CI alheios (`.github/workflows` e similares de outros repos ou não relacionados à task)

## Comandos

| Objetivo | Comando |
|----------|---------|
| Build | `cargo build` |
| Build release | `cargo build --release` |
| Testar tudo | `cargo test` |
| Testar unitários | `cargo test --lib` |
| Testar integração | `cargo test --test <nome>` |
| Lint | `cargo clippy -- -D warnings` |
| Formatação | `cargo fmt --check` |
| Formatar | `cargo fmt` |
| Benchmarks | `cargo bench` |
| Documentação | `cargo doc --open` |
| Verificar (build+test+lint) | `cargo clippy -- -D warnings && cargo test && cargo fmt --check` |
| Check sem gerar binário | `cargo check` |
| Atualizar dependências | `cargo update` |
| Auditar segurança | `cargo audit` (se tool instalado) |

**Pré-verificação de ambiente:** confirme `rustc --version` ≥ versão do
`rust-version` em Cargo.toml (ou `edition`). Toolchain incorreta é pitfall
comum — ajuste com `rustup override set <toolchain>`.

## Estrutura de projeto típica

```
<repo>
├── Cargo.toml
├── Cargo.lock
├── src/
│   ├── main.rs              # binário principal (ou lib.rs para library)
│   ├── lib.rs               # library root (quando o crate é lib)
│   ├── config.rs            # configuração (env, args, arquivos)
│   ├── domain/              # modelos de domínio, structs, enums
│   │   ├── mod.rs
│   │   ├── <entidade>.rs
│   │   └── value_objects.rs
│   ├── infra/               # adaptadores de infraestrutura (DB, HTTP, fila)
│   │   ├── mod.rs
│   │   ├── db/              # sqlx queries, migrations, repos
│   │   ├── http/            # actix-web handlers, client externo
│   │   └── queue/           # message broker
│   ├── application/         # casos de uso, serviços, orquestração
│   │   ├── mod.rs
│   │   └── <use_case>.rs
│   └── error.rs             # tipos de erro globais (thiserror)
├── tests/                   # testes de integração
│   └── <mod>_test.rs
├── benches/                 # benchmarks (criterion)
│   └── <mod>_bench.rs
└── migrations/              # sqlx migrations (se aplicável)
    └── <YYYYMMDD>_<desc>.sql
```

> **Alternativa legítima — layout por features:** CLIs e engines Rust
> frequentemente organizam `src/` por feature (ex: `src/ingest/`,
> `src/extract/`, `src/graph/`, `src/rag/`) em vez das camadas
> domain/infra/application — que fazem sentido para serviços web
> (actix). Siga o layout existente do repo; o `AGENTS.md` do repo tem
> precedência sobre este playbook.

## Padrões dominantes

### Web framework (actix-web)
- Servidor HTTP com `HttpServer`, `App`, `scope` para versionamento
- Handlers como funções async que retornam `HttpResponse` ou `impl Responder`
- Extractors: `Json<T>`, `Path<T>`, `Query<T>`, `State<AppState>`
- Error handling: implementar `ResponseError` do actix-web nos erros de domínio
- Middleware: `Logger`, `Compress`, CORS via `actix-cors`

### Async runtime (tokio)
- `#[tokio::main]` no entry point, `tokio::spawn` para tarefas concorrentes
- `tokio::sync` para canais (`mpsc`, `oneshot`, `watch`) e locks (`Mutex`, `RwLock`)
- Evitar `std::sync::Mutex` em contexto async — preferir `tokio::sync::Mutex`
- Streams: `tokio_stream`, `futures::StreamExt` para processamento reativo

### Serialização (serde)
- `#[derive(Serialize, Deserialize)]` em todos os DTOs
- `#[serde(rename_all = "camelCase")]` para API JSON
- `#[serde(skip_serializing_if = "Option::is_none")]` em campos opcionais
- `#[serde(default)]` para campos com valor padrão
- Validar DTOs de entrada com `validator` crate ou `#[serde(deserialize_with)]`

### Banco de dados (sqlx)
- `sqlx::query_as!` para queries compile-time checks (exige migration runtime)
- `sqlx::FromRow` para mapear resultados em structs
- Pool de conexões: `PgPool` ou `SqlitePool` configurado no startup
- Migrations: SQL versionado em `migrations/`, executado via `sqlx migrate run`
- Transações: `pool.begin().await` + `tx.commit()` / `tx.rollback()`

### Banco de grafos (neo4rs)
- `Graph::new(uri, user, password).await` no startup — o driver gerencia o pool
- Schema versionado: função própria que aplica Cypher versionado (ex: `apply_migrations(graph, dir, repo)`) — é o análogo das migrations sqlx
- Queries: params via `query().param()` — **nunca** interpolar strings no Cypher (injection)
- Testes de integração exigem Neo4j real (docker) — a suíte unitária (`cargo test --lib`) é a green mínima

### CLI (clap)
- `#[derive(Parser)]` para struct de argumentos
- Subcomandos com `#[derive(Subcommand)]`
- `--help` automático, `--version` do Cargo.toml
- Variáveis de ambiente como fallback para args não fornecidos

### Error handling (thiserror + anyhow)
- Erros de domínio: enum com `#[derive(thiserror::Error)]`
- Erros de infraestrutura: `anyhow::Result<T>` com `.context()` para wrapping
- Conversão `From<infra_error> for domain_error` nas fronteiras

## Testes

- **Unitários:** `#[cfg(test)] mod tests { ... }` no mesmo arquivo ou em `tests/`
- **Integração:** arquivos em `tests/` (cada um é um crate separado)
- **Doc-tests:** ```rust no doc comments (executados com `cargo test`)
- **Mocks:** `mockall` crate para traits, ou structs mock manuais
- **Banco:** sqlx `query!` macro exige DB real; usar `sqlx::test` ou container Docker
- **Benchmarks:** `criterion` crate em `benches/`
- **Cobertura:** `tarpaulin` (`cargo tarpaulin`) para relatório de cobertura

**Verificação completa:** `cargo clippy -- -D warnings && cargo test && cargo fmt --check` — a suíte inteira precisa passar, não só os testes novos.

**Com integração dependente de DB externo:** `cargo test --lib` é a
suíte mínima green (unitários); os testes de integração viram gate
separado, executado com o DB no ar (docker compose). Nunca use o gate
de integração quebrado como desculpa para pular a verificação.

## Pitfalls

1. **Toolchain errada** — `rustc --version` diferente do `edition` no Cargo.toml
2. **async trait sem `#[async_trait]`** — traits com métodos async precisam do crate `async-trait` (ou `impl Trait` notação noturna)
3. **`Mutex` síncrono em contexto async** — `std::sync::Mutex` segura thread; usar `tokio::sync::Mutex`
4. **Lifetime mal resolvido** — retornar referências de funções que criam dados? retornar `String` ou `Arc<T>` em vez de `&str`
5. **`cargo test` compila tudo** — diferentemente de `mvn test` que só compila testes, `cargo test` compila o crate inteiro de novo
6. **sqlx `query!` exige DB em build time** — `DATABASE_URL` precisa estar setada mesmo para `cargo build` se houver macros sqlx
7. **Overflow em release** — debug tem overflow checks, release não; usar `.checked_add()`, `.saturating_sub()`
8. **FFI com C** — `#[no_mangle]`, `extern "C"`, `unsafe`, CString sem vazamento de memória
9. **WASM target** — `wasm-pack build`, sem std IO, `wasm-bindgen` para interop JS
10. **Testes de integração stale quebram tudo** — `cargo test` compila TODOS os binários de teste; se testes de integração exigem DB real e ficaram stale após mudança de assinatura, a suíte inteira falha em compilação. Leia o AGENTS.md do repo: testes com erros conhecidos têm regra própria (ex: "não consertar às cegas") — use `cargo test --lib` como suíte mínima e trate integração como gate separado.

## Quality Checks

### Automação

| Check | Comando | O que verifica |
|-------|---------|----------------|
| Lint | `cargo clippy -- -D warnings` | Lint, common mistakes, idiomatic code |
| Formatação | `cargo fmt --check` | Estilo de código (rustfmt) |
| Build | `cargo build` | Compilação sem erros |
| Testes | `cargo test` | Suíte completa verde |
| Auditoria segurança | `cargo audit` | Vulnerabilidades em dependências |

### Padrões da stack

- **Naming:** `snake_case` para funções/variáveis, `PascalCase` para types/traits, `SCREAMING_SNAKE_CASE` para constants
- **Testes:** `#[cfg(test)] mod tests` no mesmo arquivo, ou `tests/` para integração
- **Testes naming:** `fn test_<scenario>_<expected>` ou `fn <function_name>_<scenario>()`
- **Estrutura:** `src/domain/`, `src/infra/`, `src/application/`, `src/error.rs`
- **Error handling:** `Result<T, E>` com `thiserror` para domínio, `anyhow` para infra

### Limites

| Check | Limite | Severidade |
|-------|--------|------------|
| Função | ≤ 30 linhas | error |
| Struct | ≤ 50 campos | warning |
| Complexidade ciclomática | ≤ 10 | warning |
| `unsafe` blocks | 0 sem justificativa | error |
| Clippy warnings | 0 | error |

## Algoritmo

1. **Detectar stack:** confirme `Cargo.toml` na raiz do repo alvo. Sem ele → `warn`.
2. **Carregar convenções:** leia o `AGENTS.md` do repo (precedência sobre este playbook) e confira edition/rust-version no `Cargo.toml`.
3. **Contextualizar a task:** leia os artefatos de spec referenciados (TASKS.md, API_CONTRACTS.md, ERROR_CATALOG.md, COMPLIANCE.md).
4. **Aplicar na implementação:** use os padrões deste playbook ao codar.
5. **Verificar:** rode `cargo clippy -- -D warnings && cargo test && cargo fmt --check` com a suíte inteira verde.
6. **Relatar:** arquivos tocados, testes verdes, decisões tomadas, RNs cobertas.

## Modo de erro

`warn` — divergências entre o playbook e o AGENTS.md do repo resolvem a
favor do repo (registre a decisão no relatório). Só é `strict` na
verificação: sem `Cargo.toml` (Passo 1) ou com `cargo clippy/test/fmt`
falhando no Passo 5, a task não é reportada como concluída.

## Tools

- Capacidade `kit.graph` (política graph-first — FORMATO §10/DR-020): com o grafo disponível e índice fresco, mapeie o repo por navegação estruturada (`get_architecture`, `search_graph`, `trace_path`) ANTES de grep/glob; grep/glob ficam para literais e para o que o grafo não resolve. Índice stale → re-indexar antes de usar. Sem grafo → degradação `grep-fallback` declarada no relatório.

- `file`/`glob`/`grep` — inspecionar código-fonte, `Cargo.toml` e configs (read-only)
- `shell` — executar `cargo build`, `cargo test`, `cargo clippy` e `cargo fmt` (verificação)

## Casos de Borda

- **Repo sem testes (tests/ vazio ou sem #[cfg(test)]):** sinalizar `not-verified.` com motivo
- **Dependência de nightly:** se o projeto usa feature noturna (`#![feature(...)]`), usar `rustup run nightly cargo test`
- **sqlx sem DATABASE_URL:** abortar e pedir config — não inventar URL
- **Benchmarks quebrados:** não bloquear o PR, reportar como warning
- **Testes de integração com erros conhecidos (documentados no AGENTS.md do repo):** NÃO consertar às cegas — usar a suíte canônica do repo (`cargo test --lib`) e registrar no relatório
- **Workspace (Cargo.toml com [workspace]):** `cargo test --workspace`, `cargo build -p <crate>`

## Formato de commit

```
feat(<modulo>): <descricao-curta> — refs RN-XXX
```

(Convenção final pertence ao AGENTS.md do repo — este playbook só
sugere o estilo dominante.)
