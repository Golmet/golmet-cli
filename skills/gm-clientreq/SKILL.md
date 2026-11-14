---
name: gm-clientreq
description: Gera ou sincroniza delivery/CLIENT_REQUIREMENTS.md a partir do REQUIREMENTS.md — vista de negócio para o cliente, sem identificadores técnicos nem pendências internas.
requires: [file]
kit:
  checkpoints: post-write validate
---
# gm-clientreq — Requisitos do Cliente

Leio `REQUIREMENTS.md` (a fonte da verdade técnica) e gero ou atualizo
`delivery/CLIENT_REQUIREMENTS.md` — uma vista filtrada e reescrita para
consumo do cliente/stakeholder de negócio, sem vocabulário de implementação.

`REQUIREMENTS.md` é sempre a fonte. `CLIENT_REQUIREMENTS.md` é sempre derivado
dele. Nunca edito `REQUIREMENTS.md` a partir desta skill, e nunca aceito
informação nova que não esteja lá.

Ambos os documentos são versionados em Git. Para descobrir o que mudou desde
a última sincronização, uso o histórico do Git sobre `REQUIREMENTS.md`
(`git log` / `git diff` no arquivo) como fonte de verdade de mudança — não
existe changelog interno em nenhum dos dois arquivos.

## Resolução de caminhos

Os artefatos do workspace seguem o layout do FORMATO §2 — a saída fica em
`delivery/CLIENT_REQUIREMENTS.md`.

**Regra de fallback (transição):** ao **LER** um artefato, verifique primeiro
o caminho novo (ex: `specs/DATA_MODEL.md`); se não existir, use o caminho
legado na raiz do workspace (ex: `DATA_MODEL.md`). Ao **ESCREVER/GERAR**, use
sempre o caminho novo. Workspaces ainda não adotados continuam legíveis
durante a transição.

## Quando me usar

Use esta skill sempre que `REQUIREMENTS.md` tiver sido criado ou atualizado e o documento voltado ao cliente precisar refletir isso. Cobre os dois casos automaticamente:

- **`CLIENT_REQUIREMENTS.md` não existe** → gero do zero a partir do `REQUIREMENTS.md` atual (salvo em `delivery/`).
- **`CLIENT_REQUIREMENTS.md` já existe** (em `delivery/` ou na raiz) → sincronizo apenas o que mudou em `REQUIREMENTS.md` desde a última sincronização, preservando o restante.

Entrada esperada: localizar e ler `REQUIREMENTS.md`. Se não existir, avise que não há fonte e pare. Se houver acesso ao repositório Git, verifique o histórico de commits do arquivo desde a data da última sincronização registrada no `CLIENT_REQUIREMENTS.md` (ver Header) para identificar precisamente o que mudou, em vez de comparar o documento inteiro do zero.

## Estrutura do CLIENT_REQUIREMENTS.md

1. **Header** — título, data da última sincronização, objetivo de negócio em 1-3 frases (sem sistemas internos, siglas técnicas ou nomes de tabelas).
2. **Scope** — o que está incluído, em linguagem de negócio.
3. **Actors and Permissions** — mantido como no `REQUIREMENTS.md`.
4. **Business Rules** — reescritas sem identificadores técnicos e sem os IDs internos (`RN-XXX`) visíveis no corpo do texto — ver regras de filtragem abaixo.
5. **Use Cases** — fluxos do ponto de vista de quem usa, sem falar em módulos, camadas, ou arquitetura interna.
6. **Outputs** — documentos, comunicações ou resultados visíveis ao cliente.
7. **Data Guarantees** — "Persistence and Audit" reescrito em garantia de negócio, sem tabelas, chaves ou estrutura de banco.
8. **Decisions and Out of Scope** — mantido como no `REQUIREMENTS.md`.
9. **Open Questions for the Client** — apenas pendências (`PEND-XXX`) que dependem de decisão ou resposta do cliente. Pendências técnicas internas não entram — ver regra de triagem.
10. **Summary of Changes** — resumo em linguagem de negócio do que mudou desde a última sincronização, derivado do histórico de Git sobre `REQUIREMENTS.md`, não de um changelog manual.

Omita uma seção se não houver conteúdo correspondente relevante ao cliente.

## Regras de filtragem e reescrita

- **Nunca inclua identificadores técnicos.** Campo, tabela, enum, classe, endpoint, ou qualquer coisa em `code formatting` no `REQUIREMENTS.md` vira linguagem natural de negócio aqui.
- **Nunca exponha os IDs internos (`RN-XXX`, `PEND-XXX`) no corpo do texto do cliente** — eles são instrumentação interna de rastreabilidade, não algo que o cliente precisa ver. Internamente (nos seus próprios registros de trabalho, se precisar rastrear a correspondência), pode manter a referência, mas não no arquivo entregue ao cliente.
- **Nunca inclua arquitetura interna** — módulos, camadas, serviços, integrações técnicas.
- **Preserve o significado da regra, não a forma.** Remover jargão técnico não pode enfraquecer nem simplificar demais a regra em si.
- **Triagem de pendências:** para cada `PEND-XXX`, pergunte "isso depende de decisão do cliente, ou é escolha técnica que o time resolve sozinho?". Só o primeiro tipo entra em "Open Questions for the Client". Na dúvida, não inclua.

## Regras de sincronização

- **Não regenere o documento inteiro a cada sincronização.** Use o histórico do Git sobre `REQUIREMENTS.md` desde a última sincronização para identificar exatamente quais regras/seções mudaram, e atualize só essas.
- **Mudança técnica sem efeito percebido pelo cliente** não gera alteração no `CLIENT_REQUIREMENTS.md` nem entra em "Summary of Changes".
- **Pendência resolvida em `REQUIREMENTS.md`:** se estava em "Open Questions for the Client" e agora tem decisão registrada como Business Rule, mova para a seção definitiva e remova de pendências.
- **Nunca deixe o `CLIENT_REQUIREMENTS.md` divergir silenciosamente do `REQUIREMENTS.md`.** Se uma regra mudou na fonte, a versão do cliente reflete a nova regra, e isso aparece no Summary of Changes.
- Se a mudança em `REQUIREMENTS.md` for ambígua quanto a impactar ou não a experiência do cliente, aplique a decisão conservadora (regra abaixo) e registre a dúvida no retorno.

## Saída esperada

Produza **o arquivo `delivery/CLIENT_REQUIREMENTS.md` completo**, pronto para salvar ou substituir o anterior — não um diff. Não inclua meta-comentário sobre o processo de triagem no corpo do documento (o "Summary of Changes" é a única exceção, por ser parte do arquivo). Ao final da resposta, sugira uma mensagem de commit objetiva (ex. `docs: sync CLIENT_REQUIREMENTS.md with rule update`). Se `REQUIREMENTS.md` não puder ser localizado, informe isso e não prossiga.

**Checkpoint de kit (`post-write: validate`):** após escrever, rode `golmet validate` no workspace e corrija o que apontar antes de seguir. Sem kit disponível, anote a pendência "rode golmet validate" para o próximo passo.

## Lê (upstream)

- `REQUIREMENTS.md` (raiz do workspace) — fonte única da verdade técnica
- Histórico Git sobre `REQUIREMENTS.md` (`git log` / `git diff`) — delta desde a última sincronização
- `delivery/CLIENT_REQUIREMENTS.md` (ou raiz, pré-adoção) — versão anterior para sync

## Gera (downstream)

- `<Produto>/delivery/CLIENT_REQUIREMENTS.md` — cria do zero ou sincroniza apenas o que mudou

## Nunca edita

- `REQUIREMENTS.md` — fonte, somente leitura
- Artefatos técnicos de spec (DATA_MODEL.md, API_CONTRACTS.md, Features em delivery/, etc.)
- `golmet.index.json` (único escritor é o kit)

## Capacidades

- `file` — localizar e ler REQUIREMENTS.md e artefatos do workspace; gravar `delivery/CLIENT_REQUIREMENTS.md`
- **Esta skill NÃO declara `ask`:** sem pergunta interativa, aplique a decisão conservadora nas ambiguidades — na dúvida sobre impactar o cliente, omita do documento do cliente e registre a dúvida no seu retorno ao operador (nunca invente).

## Modo de erro

`strict` — sem `REQUIREMENTS.md` localizável, informe e não prossiga; mudança ambígua quanto a impacto no cliente → decisão conservadora com registro (regra de Capacidades).

## Algoritmo

1. Localize e leia `REQUIREMENTS.md` do workspace — sem ele, informe e encerre (Modo de erro)
2. Identifique o delta: com Git, use `git log`/`git diff` sobre `REQUIREMENTS.md` desde a data registrada no Header do `CLIENT_REQUIREMENTS.md`; sem Git, compare o documento inteiro
3. Reescreva o conteúdo em linguagem de negócio aplicando as regras de filtragem (sem identificadores técnicos, IDs internos ou arquitetura interna)
4. Faça a triagem das pendências (`PEND-XXX`) — só as que dependem de decisão do cliente entram em "Open Questions for the Client"; na dúvida, omita e registre no retorno
5. Aplique as regras de sincronização: atualize apenas o que mudou; Summary of Changes coerente com o delta
6. Escreva `delivery/CLIENT_REQUIREMENTS.md` completo
7. Rode `golmet validate` (checkpoint post-write) — ou anote a pendência se o kit não estiver disponível

## Formato de commit

```
docs: sync CLIENT_REQUIREMENTS.md (<produto>)
```

## Casos de Borda

- **CLIENT_REQUIREMENTS.md inexistente** — gere do zero a partir do REQUIREMENTS.md atual.
- **CLIENT_REQUIREMENTS.md na raiz (pré-adoção do layout)** — sincronize e reloqueie para `delivery/`.
- **Mudança técnica sem efeito percebido pelo cliente** — não altera o documento nem o Summary of Changes.
- **PEND técnica interna** — não entra em "Open Questions for the Client" (regra de triagem).
- **PEND resolvida** — mova para a seção definitiva e remova das pendências (diretiva `> estado: resolved → ...` na fonte).

---

compatibility: multi-engine

> **Degradação de checkpoints:** o checkpoint `post-write: validate` depende do kit golmet disponível. Sem o kit, a skill continua funcionando — o checkpoint vira "rode `golmet validate` manualmente" após a escrita.
