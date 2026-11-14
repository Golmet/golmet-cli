---
name: gm-charter
description: Cria PROJECT_CHARTER.md a partir da primeira conversa com o cliente — modos completo (novo) e simplificado (projeto existente), com campos legado/sistema_existente/compliance.
requires: [file, ask]
degrades:
  ask: chat-stop
kit:
  checkpoints: post-create validate
---
# gm-charter — Project Charter

Gera `PROJECT_CHARTER.md` na raiz do workspace do produto a partir da
primeira conversa com o cliente.

## Referências

- Contrato do Charter (estados, pré-condições): FORMATO §7.1
- Layout do workspace: FORMATO §2

## Lê (upstream)

- Nenhum (primeiro documento do workspace)
- Padrões acumulados de organizações (overlay de pack, se configurado no
  `golmet.yaml` — condicional): ao levantar o escopo, consultar padrões
  aplicáveis — especialmente padrões de **processo** (ex: existe produto-irmão
  de domínio cujas features podem ser adaptadas?)

## Gera (downstream)

- `PROJECT_CHARTER.md` (raiz do workspace `<Produto>/`)

## Nunca edita

- `golmet.index.json` — escrito só pelo kit
- Nenhum documento anterior (charter é o primeiro documento)

## Capacidades

- `ask` — coletar informações do cliente (sem ask nativo: pergunte no chat e
  PARE — nunca adivinhe)
- `file` — criar e ler arquivos do workspace

## Modo de erro

`strict` — se o charter não for gerado, o projeto não pode começar.

## Formato de commit

```
docs: add project charter for <nome>
```

## Estratégia de Perguntas

### Regras gerais

1. **Agrupe perguntas relacionadas** — nunca pergunte um campo por vez.
   Máximo de 3 blocos de perguntas para todo o intake.

2. **NUNCA inclua "Digitar agora", "Outro" ou "Custom" como opção.**
   Interfaces de ask nativas já oferecem resposta livre; sem ask nativo,
   pergunte no chat em formato aberto e PARE pela resposta.

3. **Para campos de texto livre** (nome do projeto, cliente, patrocinador,
   contexto, objetivo): forneça 2-3 exemplos ilustrativos como opções
   para guiar o usuário, não para limitar. Ex:

   ```
   // BOM: exemplos genéricos que orientam o formato
   ask:
     header: "Nome do projeto"
     options:
       - { label: "MeuApp", description: "Sistema de gestão interna" }
       - { label: "MinhaOrg", description: "Plataforma da organização" }
   // Se o usuário quiser outro nome, ele digita livremente

   // RUIM: "Digitar agora" como opção
   // RUIM: pergunta única "Qual o nome?" sem contexto
   ```

4. **Para perguntas categóricas** (tipo de projeto, sim/não): forneça
   opções reais e mutuamente exclusivas. Ex:

   ```
   ask:
     header: "Tipo de projeto"
     options:
       - { label: "Greenfield", description: "Sistema novo, sem legado" }
       - { label: "Migração", description: "Substituição de sistema existente" }
   ```

5. **Use descrições contextuais** nas opções — em vez de "Sim"/"Não",
   explique o que cada escolha significa.

### Modo simplificado para projetos existentes

Quando o usuário já tem um projeto com código implementado ou em
desenvolvimento, use este fluxo alternativo que pula os blocos 2 e 3
(contexto/escopo e stakeholders) — eles serão preenchidos naturalmente
durante a geração de REQUIREMENTS.md.

#### Detecção

Se o usuário pedir "iniciar documentação de projeto existente", "aplicar
GOLMET em projeto que já existe", "tenho código pronto", "meu projeto já
está em desenvolvimento", ou similar, ative o modo simplificado.

#### Fluxo simplificado (2 blocos no máximo)

**Bloco 1 — Identificação rápida** (1 pergunta):
- Nome do projeto (com exemplos)
- Repositório existente (URL ou nome)
- Já existe REQUIREMENTS.md ou outros docs GOLMET? (Sim / Não)

**Bloco 2 — Contexto mínimo** (1 pergunta):
- O que já foi implementado (resumo rápido, 1-2 linhas)
- O que falta documentar
- Se há sistema legado a documentar (produção / legado / novo)
- Se o produto trata dado pessoal (Sim / Não — se sim, `compliance: true`
  no charter)

#### Resultado

- Crie o diretório do workspace `<Produto>/`
- Gere PROJECT_CHARTER.md com `sistema_existente: true` no front matter
- Se há sistema legado, recomende `gm-reverse` como próximo passo
- Se não há, vá direto para `gm-requirements`

#### Exemplo de Bloco 1 (modo simplificado)

```
ask:
  header: "Projeto existente"
  question: "Qual o nome do produto e quais repositórios ele possui?"
  options:
    - { label: "MeuApp", description: "Sistema de agendamento — repos: MeuApp-API, MeuApp-FE" }
    - { label: "MinhaOrg", description: "Plataforma — repos: MinhaOrg-App, MinhaOrg-Web" }
```

#### Exemplo de Bloco 2 (modo simplificado)

```
ask:
  header: "Contexto do projeto"
  question: "O que já existe nesse projeto e o que falta documentar?"
  options:
    - { label: "API pronta", description: "Backend implementado, falta documentar requisitos e contratos" }
    - { label: "Sistema em produção", description: "Sistema vivo, preciso documentar o que existe para evolutiva" }
    - { label: "Começando agora", description: "Tenho repo vazio, quero definir requisitos antes de codar" }
```

### Blocos de perguntas (3 no máximo — fluxo completo)

**Bloco 1 — Identificação e tipo** (1 pergunta):
- Nome do projeto (com exemplos)
- Cliente / contratante (texto livre)
- Tipo de projeto (greenfield / migração / integração / documentação de sistema existente)
- **Demanda registrada em tracker externo** (ex: `#123123`) — obrigatório
  se o perfil usa tracker (recurso FULL; no perfil BASE registre `—`)
- **Já existe repositório do projeto?** — se sim, registrar nome e URL
- **O workspace documenta algo que já existe?** — sistema vivo/em produção,
  código já implementado (Sim / Não — greenfield puro)
- **O produto trata dado pessoal?** — coleta dados de usuários/cidadãos
  (cadastro, contato, localização, pagamento, saúde...)? (Sim / Não).
  **Se sim, marque `compliance: true` no charter** — gatilho do checklist
  de proteção de dados via `gm-compliance`

**Bloco 2 — Contexto e escopo** (1 pergunta):
- Descrição do problema (texto livre com exemplos)
- Principais funcionalidades (lista curta)
- O que está fora do escopo

**Bloco 3 — Stakeholders e riscos** (1 pergunta):
- Patrocinador (sponsor), PO, lead técnico
- Premissas importantes
- Riscos percebidos

### Origem do projeto (repositório / sistema existente)

Se o usuário informar repositório existente ou que o workspace documenta algo
já implementado (sistema vivo/em produção):

1. **Registre TODOS os repositórios na §1 do charter** — tabela
   `| Repo | Branch | HEAD |` (FORMATO §7.1: cada repo do produto com path,
   branch e HEAD; o HEAD é congelado na confirmação do charter)
2. **Marque na §5** a opção **Documentação de sistema existente** (não
   greenfield puro)
3. **Recomende `gm-reverse`** — o gatilho agora é `legado: true`
   **ou** `sistema_existente: true` no front matter. Distinção:
   - `legado` = sistema que será **substituído/migrado**
   - `sistema_existente` = documentar sistema **vivo/em produção como está**
4. **Nunca crie pastas por camada** — se o usuário pedir "criar doc da
   MeuApp-API", pergunte se há frontend e crie o workspace `MeuApp/` unificado

### Exemplo completo de Bloco 1

```
ask:
  header: "Identificação"
  question: "Qual o nome do produto e o cliente?"
  options:
    - { label: "MeuApp", description: "Sistema de agendamento — repos: MeuApp-API, MeuApp-FE" }
    - { label: "MinhaOrg", description: "Plataforma central — repos: MinhaOrg-API, MinhaOrg-FE" }
```

Após obter as respostas dos 3 blocos, escreva o charter.
Não invente dados não fornecidos pelo usuário.

## Regra de Escopo — Unidade é o Produto

> **⚠️ REGRA FUNDAMENTAL:** Crie UM charter por **produto**, não por
> **camada técnica**. Um produto pode ter N repos (API, FE, CORE, BATCHS,
> etc.), mas as regras de negócio são do produto inteiro. Nunca criar
> `MeuApp-API/` e `MeuApp-FE/` como projetos separados — o correto é
> `MeuApp/` com todos os repos listados.

### Como aplicar

1. Se o usuário pedir "criar doc da API X" ou "criar doc do frontend Y", pergunte se faz parte de um produto maior
2. Se sim, crie o charter do **produto** (ex: `MeuApp/`) listando todos os repos
3. Se não (projeto isolado, sem par), crie como produto único (ex: `Rapido/`)
4. A §1 do charter deve listar todos os repos do produto

### Exemplo

```
Usuário: "criar doc do MeuApp API"
Skill: "O MeuApp API faz parte de um produto maior? Há frontend ou outros componentes?"
Usuário: "Sim, tem o MeuApp-FE também"
Skill: cria MeuApp/PROJECT_CHARTER.md com a §1 listando MeuApp-API e MeuApp-FE
```

## Workflow

1. **Detecte o modo:** Se o projeto já existe (usuário menciona código pronto, repo existente, "aplicar GOLMET"), use o **modo simplificado** (2 blocos). Caso contrário, use o **fluxo completo** (3 blocos).
2. **Verifique se é produto composto:** Se o usuário mencionar API + FE (ou outras camadas) para o mesmo produto, crie UM charter unificado. Pergunte se há mais componentes antes de criar.
3. Pergunte usando os blocos correspondentes ao modo detectado (via `ask`; sem ask nativo, pergunte no chat e PARE — nunca adivinhe)
4. Crie a pasta do workspace `<Produto>/` (nome do PRODUTO, não da camada)
5. Escreva `PROJECT_CHARTER.md` com todos os repos listados na §1
6. Liste os arquivos em `<Produto>/meetings/`, encontre o maior número sequencial no padrão `YYYY-MM-DD-NN.md` e escreva `<Produto>/meetings/YYYY-MM-DD-NN.md` (NN = próximo sequencial) com o resumo das informações coletadas + perguntas e respostas
7. **Checkpoint de kit (`post-create: validate`):** após criar o charter, rode `golmet validate` no workspace e corrija o que apontar antes de seguir (o charter entra com `status: draft` — estado legal §7.1). Sem kit disponível, anote a pendência "rode golmet validate" para o próximo passo.

## Template — PROJECT_CHARTER.md

```yaml
---
status: draft
# campos opcionais conforme respostas (FORMATO §7.1):
# legado: true
# sistema_existente: true
# compliance: true
---
```

> **Regra de idioma:** Documentos em português brasileiro (PT-BR) por padrão;
> siga o idioma do cliente. Títulos, descrições, perguntas e conteúdo do
> charter no idioma combinado. Apenas identificadores técnicos (API, SOAP)
> permanecem no original.

```markdown
# Termo de Abertura — [Nome do Sistema]

## 1. Identificação

| Campo | Valor |
|---|---|
| Nome do projeto | |
| Cliente / contratante | |
| Demanda (tracker externo, se houver) | — |
| Repositórios | \`repo1\`, \`repo2\` — listar todos os repos do produto |
| Data de início | DD/MM/AAAA |

### §1.1 Repos (FORMATO §7.1 — HEAD congelado no `confirmed`)

| Repo | Branch | HEAD |
|---|---|---|
| ../MeuApp-API | main | |

## 2. Contexto

[Descrição do problema ou oportunidade, em linguagem de negócio]

## 3. Objetivo

[O que o projeto entrega, critério de sucesso, métricas]

## 4. Escopo

Dentro do escopo:

- [item]

Fora do escopo:

- [item]

## 5. Tipo de projeto

- [ ] Novo sistema (greenfield)
- [ ] Evolutiva / migração de sistema existente
- [ ] Integração com terceiros
- [ ] Documentação de sistema existente (sistema vivo/em produção, sem migração)

**Legado:** ✅ Sim — sistema existente será analisado (`legado: true`)
**Legado:** ❌ Não — projeto novo, sem sistema anterior
**Sistema existente:** ✅ Sim — documentação de sistema vivo/em produção (`sistema_existente: true`)
**Sistema existente:** ❌ Não — greenfield puro

**Compliance:** ✅ Sim — `compliance: true` (produto trata dado pessoal — checklist de proteção de dados obrigatório: `specs/COMPLIANCE.md`)
**Compliance:** ❌ Não — produto não trata dado pessoal

## 6. Stakeholders

| Papel | Nome | Contato |
|---|---|---|
| PO / Negócio | | |
| Lead técnico | | |

## 7. Premissas

- [premissa]

## 8. Riscos iniciais

- [risco, probabilidade, impacto, mitigação]
```

## Template — Meeting (Checkpoint)

Arquivo: `<Produto>/meetings/YYYY-MM-DD-NN.md` (NN = próximo sequencial)

```markdown
# Intake — DD/MM/AAAA

**Resumo:** [descrição concisa do projeto e objetivos]

## Perguntas e Respostas

### [pergunta]

**Resposta:** [resposta]
```

## Regra de Localização de Artefatos

**⚠️ REGRA ABSOLUTA: Artefatos GOLMET NUNCA devem ser criados em pastas de código fonte.**

### Contexto

Em workspaces com múltiplos repositórios (código e docs misturados), a sessão
pode estar posicionada em qualquer repo. A skill DEVE detectar isso e
redirecionar os artefatos para a pasta correta de documentação (o **workspace
do produto** — `<Produto>/`, que pode conviver com os repos em uma pasta-mãe
da organização).

### Detecção de Contexto

Ao iniciar o intake, verifique se o diretório atual é:
- Um repositório de código fonte (contém `src/`, `pom.xml`, `package.json`, `.csproj`, etc.)
- Uma pasta-mãe com múltiplos repos (contém subpastas com nomes de projetos/repos)
- Uma pasta dedicada de documentação (workspace de produto já existente, com `golmet.yaml`)

### Pergunta de Localização (quando aplicável)

Se o diretório NÃO for uma pasta de documentação dedicada, pergunte:

```
ask:
  header: "Localização da documentação"
  question: "Onde devo salvar os artefatos GOLMET deste projeto?"
  options:
    - { label: "<pasta-de-produtos>/<produto>/", description: "Workspace dedicado de docs (recomendado)" }
    - { label: "docs/<produto>/", description: "Pasta docs dentro do repo atual" }
```

### Regras de Salvamento

| Cenário | Localização correta |
|---------|---------------------|
| Diretório é pasta-mãe de produtos | `<pasta-de-produtos>/<Produto>/` |
| Diretório é repo de código (ex: `MeuApp-API/`) | `<pasta-de-produtos>/<Produto>/` (unificar com outros repos do mesmo produto) |
| Diretório já é workspace de produto (tem `golmet.yaml`) | `<Produto>/` (raiz — charter na raiz do workspace, FORMATO §2) |

> **Lembrete:** A unidade é o **produto**, não a camada. Se o diretório é `MeuApp-API/` mas existe `MeuApp-FE`, crie o workspace `MeuApp/`.

### Nunca faça

- ❌ Criar `PROJECT_CHARTER.md` na raiz de um repo de código
- ❌ Criar `meetings/` na raiz de um repo de código
- ❌ Misturar artefatos GOLMET com código fonte
- ❌ Assumir que o diretório atual é o local correto para docs
- ❌ Criar pastas separadas por camada (ex: `MeuApp-API/` e `MeuApp-FE/` como projetos distintos)
- ❌ Escrever `golmet.index.json` (único escritor é o kit)

### Exemplo de fluxo correto

```
Usuário está em: <pasta-de-produtos>/MeuApp-API/
Usuário pede: "iniciar doc do MeuApp"

Skill detecta: diretório é repo de código (tem pom.xml)
Skill pergunta: "O MeuApp API faz parte de um produto maior? Há frontend?"
Usuário responde: "Sim, tem o MeuApp-FE"
Skill pergunta: "Onde salvar?"
Usuário escolhe: "<pasta-de-produtos>/MeuApp/"

Skill cria: <pasta-de-produtos>/MeuApp/PROJECT_CHARTER.md
Skill cria: <pasta-de-produtos>/MeuApp/meetings/YYYY-MM-DD-NN.md
Skill roda: golmet validate (checkpoint post-create)
```

## Casos de Borda

- **PROJECT_CHARTER.md já existe no workspace** — não recrie; informe e sugira atualizar via `gm-requirements` (novos requisitos) ou `gm-change-request` (mudança em requisito confirmado — se disponível no perfil).
- **Usuário não sabe responder um campo** — não invente; registre como pendência no charter e siga (o campo pode ser preenchido depois).
- **Nome de pasta duplicado** — pergunte uma variação de nome antes de criar `<Produto>/`.
- **Workspace documenta sistema existente** — marque `sistema_existente: true` e recomende `gm-reverse` como próximo passo.
- **Cliente sem demanda em tracker** — registre `—` (não informado) em vez de deixar vazio.
- **Operador em dúvida sobre dado pessoal** — não marque `compliance: true` por omissão; registre como pendência no charter e recomende `gm-compliance` para decidir com o checklist.
- **Projeto existente sem REQUIREMENTS.md** — ative modo simplificado; vá direto para `gm-requirements` após o charter.
- **Projeto existente com REQUIREMENTS.md parcial** — ative modo simplificado; recomende `gm-requirements` para merge incremental.
- **Diretório é repo de código fonte** — NUNCA crie artefatos GOLMET na raiz. Pergunte localização ou sugira `<pasta-de-produtos>/<Produto>/`.
- **Usuário pede para criar por camada (API/FE)** — pergunte se faz parte de um produto maior. Se sim, crie o produto unificado. Se não (projeto isolado), crie como produto único.

---

compatibility: multi-engine

> **Degradação de checkpoints:** o checkpoint `post-create: validate` depende do kit golmet disponível no workspace. Sem o kit, a skill continua funcionando — o checkpoint vira "rode `golmet validate` manualmente" e a pendência é anotada para o próximo passo.
