# gm-change-request — Gerenciamento de Mudanças (CR)

Registra mudanças em requisitos existentes em `CHANGE_REQUESTS.md`.
Toda alteração em RN confirmada deve passar por este fluxo.

## Referências

- IDs CR-XXX: `docs/CONSTITUTION.md §1`
- Processo de aprovação: `docs/CONSTITUTION.md §5`
- Quem aprova: `docs/CONSTITUTION.md §5.3`

## Resolução de caminhos (IMP-011)

Os artefatos do módulo seguem a estrutura com subpastas (IMP-011):

- **Raiz:** PROJECT_CHARTER.md, CONSTITUTION.md, REQUIREMENTS.md, DECISIONS.md, CHANGELOG.md
- **`specs/`:** DATA_MODEL.md, API_CONTRACTS.md, ERROR_CATALOG.md, UI_SPEC.md, ARCHITECTURE.md, SECURITY.md, ACCEPTANCE_CRITERIA.md, USER_STORIES.md, US<NNN>.md, FIGMA_PROMPTS.md, COMPLIANCE.md
- **`quality/`:** TEST_STRATEGY.md, DEFINITION_OF_DONE.md, TRACEABILITY.md, VALIDATION_REPORT.md, PROPAGATION_STATUS.md, PROGRESS_REPORT.md
- **`delivery/`:** arquivos de feature, SCOPE_AGREEMENT.md, CLIENT_REQUIREMENTS.md, CLIENT_REPORT_*.md, HANDOVER_LOG.md
- **`support/`:** LEGACY_CONTEXT.md, KNOWN_ERRORS.md, MIGRATION.md, CHANGE_REQUESTS.md
- **`glossaries/`:** os 3 glossários e variações
- **`docs/`:** howto/*.md e documentos auxiliares
- **`meetings/`, `contracts/`:** checkpoints (workspace-produto mantém `meetings/`; container de organização grava na zona de trabalho do kit — `.golmet/meetings/`, FORMAT §2.3/M19)

**Regra de fallback (transição):** ao **LER** um artefato, verifique primeiro
o caminho novo (ex: `specs/DATA_MODEL.md`); se não existir, use o caminho
legado na raiz do módulo (ex: `DATA_MODEL.md`). Ao **ESCREVER/GERAR**, use
sempre o caminho novo. Módulos ainda não migrados via `gm-restructure`
continuam legíveis durante a transição.

## Lê (upstream)

- `<modulo>/<projeto>/REQUIREMENTS.md` (para identificar RNs afetadas)
- `<modulo>/<projeto>/quality/TRACEABILITY.md` (se existir, para identificar downstreams afetados; fallback: raiz do módulo)

## Gera (downstream)

- `<modulo>/<projeto>/support/CHANGE_REQUESTS.md`
- Após aprovação: opcionalmente executa `gm-datamodel`, `gm-apicontracts`, `gm-uispec` com escopo restrito ao delta

## Nunca edita

- `REQUIREMENTS.md` diretamente (a mudança só é refletida lá após aprovação)

## Tools

- `file` (ler REQUIREMENTS.md, TRACEABILITY.md existentes)
- `write` (escrever CHANGE_REQUESTS.md)
- `ask` (para confirmar aprovação com o usuário)

## Modo de erro

`strict` — se o CR não for registrado, a mudança não pode prosseguir.

- **Score de ambiguidade < 70 (IMP-094 R3)**: NÃO registre CR — registre
  PEND em REQUIREMENTS.md §11 (com perguntas de follow-up) e oriente nova
  tentativa após clarificação; solicitação ambígua nasce PEND, não mudança
  formal. Nenhum arquivo de CR é criado neste caso.

## Formato de commit

```
docs: add CR-XXX for <título>
```

## Estratégia de Perguntas

### Regras

1. **NUNCA inclua "Digitar agora" como opção.** A pergunta já adiciona
   "Type your own answer" automaticamente.

2. **Agrupe perguntas relacionadas.** Máximo 2 blocos para todo o CR.

3. **Use exemplos contextuais** nas opções, nunca opções genéricas.

### Blocos (máx 3)

**Bloco 0 — Validação de Abordagem** (obrigatório antes de registrar o CR):
- **Repositório/sistema alvo**: onde deve ser implementado? (ex: `<nome>.API`, sistema externo)
- **Abordagem arquitetural**: endpoint novo, adaptação, serviço externo?
- **Impacto visual (IMP-128)**: a mudança altera apresentação visual/UI (layout, copy de tela, componente, hierarquia visual)? Se sim e o front do módulo é derivado de design Figma (UI_SPEC/Figma prompts existentes), o CR ativa o **loop design-first** (ver passo 7.1) — design antes de código.
- **Atores e permissões**: quem usa? autenticado, público, admin?
- **Alternativas**: outras abordagens consideradas?
- **Confirmação**: "A abordagem está correta? Posso prosseguir?"

**Bloco 1 — Dados da mudança** (única pergunta):
- Descrição resumida da mudança
- RNs afetadas (já identificadas na leitura de REQUIREMENTS.md)
- Justificativa
- **Origem** da mudança (IMP-099): `requisito-mal-capturado | evolucao | bug | decisao-cliente` — perguntar (opções com as 4 origens); em modo não-interativo, inferir do contexto e registrar a inferência no campo `Origem` do CR
**Bloco 2 — Aprovação** (única pergunta):
- A mudança está aprovada?
- Se sim, deseja repropagar downstreams?

## Workflow

0. **VALIDAÇÃO DE ABORDAGEM** — Antes de registrar qualquer mudança:
   - Pergunte ao usuário — Bloco 0 (validação de abordagem)
   - Confirme ONDE (repositório/sistema alvo), COMO (arquitetura), QUEM (atores/permissões) e ALTERNATIVAS
   - **Só prossiga após a confirmação explícita da abordagem pelo usuário**
   - Registre a decisão de abordagem no CR (campo `Repositório alvo` ou similar)

1. Receba a descrição da mudança solicitada
2. Leia `REQUIREMENTS.md` para identificar as RNs afetadas
3. **GATE DE SCORE (IMP-094 R3)** — toda solicitação de mudança passa pelo
   clarify/score de ambiguidade (0-100, critérios C1-C5 do `gm-clarify`)
   ANTES de ser registrada:
   - **Score >= 70** → a mudança nasce como **CR direto** — siga para o passo 4.
   - **Score < 70** → solicitação ambígua: NÃO registre CR. Registre/atualize
     **PEND** em REQUIREMENTS.md §11 (via `gm-requirements`/`gm-clarify`)
     com perguntas de follow-up e oriente retornar após clarificação — a PEND
     resolvida volta como CR (se alterar RN confirmada) ou como RN nova (IMP-094).
4. Pergunte ao usuário — Bloco 1 (dados da mudança)
5. Registre o CR em `support/CHANGE_REQUESTS.md` com status `pending` — todo CR inclui o campo `**Origem:**` (IMP-099). CR sem campo origem = registro incompleto (a agregação de métricas de resultado trata como N/A, nunca inventa)
6. Pergunte ao usuário — Bloco 2 (aprovação)
7. Se aprovada, atualize status para `approved` e pergunte se deseja re-propagar para downstreams
7.1. **LOOP DESIGN-FIRST para CR com impacto visual (IMP-128)** — se o CR
     altera apresentação visual/UI e o front do módulo é derivado de design
     Figma (existe UI_SPEC/FIGMA_PROMPTS ou o charter indica design Figma):
     1. Roteie para `gm-figma-orchestrator` → `gm-figma-prompt` (delta do
        design: apenas a tela/seção afetada pelo CR) — o prompt alimenta o
        agente do Figma; registre o prompt/frame resultante no campo
        `Design Figma` do CR.
     2. Após o operador **validar o design no Figma**, capture a nova versão
        com `gm-figma-tasks` (delta de UI_SPEC + handoff de features).
     3. Só então a feature/código é especificada (`gm-taskspec`) e
        implementada (`gm-dev`) a partir do design capturado — NUNCA
        implemente mudança visual direto no código a partir do texto do CR.
     - **Modo não-Figma (condicional):** módulo sem workflow Figma (sem
       UI_SPEC, sem MCP Figma, front não derivado de design) segue o fluxo
       normal — o gate não se aplica.
8. Se rejeitada, registre a justificativa de rejeição

## Template — CHANGE_REQUESTS.md

```yaml
---
project: <NOME>
module: mudancas
audience: [tecnica, negocio]
status: draft
---
```

```markdown
# Registro de Mudanças

## CR-XXX: [título]

**Status:** pending | approved | rejected | implemented
**Data:** DD/MM/AAAA
**Solicitante:** [nome]
**RN afetada:** RN-XXX, RN-YYY
**Origem:** requisito-mal-capturado | evolucao | bug | decisao-cliente (IMP-099)
**Impacto visual:** sim | não (IMP-128)
**Design Figma:** [frame/URL do design validado ou "pré-requisito — prompt a gerar"; n/a quando impacto visual: não]
**Descrição:** [mudança solicitada]
**Justificativa:** [por que a mudança é necessária]
**Downstreams afetados:** DATA_MODEL, API_CONTRACTS, UI_SPEC
**Aprovação:** [quem aprovou/rejeitou, data]
**Motivo de rejeição:** [se rejeitado]
```

## Casos de Borda

- **RN afetada não existe em REQUIREMENTS.md**: não registre CR apontando ID inexistente — confirme se é requisito novo (encaminhe para `gm-requirements`) ou erro de referência.
- **CR aprovado sem re-propagação**: registre status `approved` e mantenha os downstreams no campo "Downstreams afetados" — o kit golmet (`propagation-guard`) sinaliza o stale depois.
- **Mudança afeta múltiplas RNs**: registre um único CR listando todas as RNs afetadas — não abra um CR por RN para a mesma solicitação.
- **Mudança já registrada**: verifique CHANGE_REQUESTS.md antes de criar — se já existe CR aberto para a mesma RN, atualize/referencie em vez de duplicar.
- **Bloco 0 sem confirmação explícita**: não prossiga para o registro — a validação de abordagem é obrigatória; sem aprovação, o CR não é criado.
- **Mudança originada de PEND (IMP-094 R2)**: quando a resolução de uma PEND aberta se revela alteração em RN já confirmada, registre o CR normalmente e informe a origem nos campos Descrição/Justificativa (`Promovida de PEND-NNN em <data>`); a PEND de origem deve ser marcada como resolvida no REQUIREMENTS.md com destino ao CR-XXX.
- **CR sem campo `Origem` (IMP-099)**: é registro incompleto — não conclua o fluxo sem classificar a origem (via pergunta ou inferência registrada). A skill de agregação de métricas trata CR sem origem como N/A; nunca invente a origem retroativamente.
- **CR visual aprovado sem design (IMP-128)**: CR com `Impacto visual: sim` em módulo Figma-derived fica com `Design Figma: pré-requisito` até o loop 7.1 completar (prompt → validação no Figma → captura `gm-figma-tasks`); a implementação em código não começa antes. O kit golmet (`propagation-guard`) sinaliza o delta de UI_SPEC pendente.
