# Retrospectiva — golmet-cli (lançamento público v1.0.0 → v1.1.1)

**Data:** 17/11/2026
**Projeto:** golmet-cli (`Golmet/golmet-cli`)
**Duração:** ~2 dias de janela intensiva (publicação v1.0.0 → pipeline automática v1.1.1)
**Skills usadas:** gm-skill-eval (corpus), gm-project-retrospective (esta)

## O que funcionou

- **Pipeline de release automática**: tag por Conventional Commits no merge
  da main → build cross-compile ×5 → GitHub Release idempotente → npm ×6.
  A v1.1.1 nasceu sem toque humano — o teste final do ciclo.
- **Gates de PR provaram-se ao vivo**: o gate `conventional-commits` pegou o
  título fora do padrão ("sync:") no próprio PR que o introduziu;
  `release-readiness` passou de primeira; `branch-naming` fechou a convenção.
- **Checador determinístico em Go**: corpus self-score 73/88 fixtures + 15
  orquestradores skipados com motivo — a rede de proteção dos evals virou
  realidade no CI.
- **Gate vermelho com lista nomeada** (`FAILED_PKGS`): a falha de publish
  deixou de ser silenciosa — o erro nomeia cada pacote e a remediação.
- **Trunk-based com nomenclatura semântica**: `hotfix/` e `feature/` da
  main, branches curtas, integração por PRs com gates.

## O que não funcionou

- **A saga npm** (maior custo do período): organização bloqueada pelo
  formulário React, token Granular sem permissão de write, EOTP por 2FA de
  escrita (3 vezes), escopo inexistente (E404), dispatch 403 por permissão
  ausente. Nenhum checklist prévio antecipou qualquer um desses passos.
- **IMPs não registrados**: o trabalho de pipeline/docs do período não
  passou pelo improver — zero IMPs formais (dívida de processo).
- **15 orquestradores sem baseline**: fixtures `.self-score-skip` herdadas
  do runtime antigo aguardam adaptação.
- **Coordenação entre sessões paralelas no mesmo clone**: registry sujo
  gerou falsos vermelhos de suite e exigiu isolamento por path explícito.

## Padrões extraídos

- **Tag-fonte-de-verdade**: a versão vem da tag; o `package.json` é
  patchado no publish — evita commit de bot na main protegida e mantém
  registry nunca atrás da tag → ver `patterns/` (extração automática)
- **Gate vermelho listado**: falha agregada com o nome de cada item falho
  e a remediação inline (substitui tolerância cega)
- **Fixture negativa**: `mode: manual` sem `output-expected/` — a ausência
  de artefato É a asserção de recusa (não inventa, não sobrescreve, não
  promove pendente)

## Métricas do projeto

| Métrica | Valor |
|---------|-------|
| Skills usadas (invocação formal) | 2 (gm-skill-eval via corpus, gm-project-retrospective) |
| Issues de qualidade rastreadas | 5 (org npm, token sem write, EOTP ×3, E404 escopo, 403 dispatch) |
| Releases publicadas | 3 (v1.0.0, v1.1.0, v1.1.1) |
| Pacotes Go verdes | 25 |
| Corpus de evals | 88 fixtures · 73 self-score PASS · 15 skipadas com motivo |
| Checks do corpus | ~800 checks determinísticos |
| PRs merged | 3 (#8, #9, #11) |
| IMPs formais no período | 0 (dívida — ver Ações) |
| ADRs formais no período | 0 (histórico: 18 ADRs) |

## Ações para o workflow

| Ação | Tipo | Prioridade |
|------|------|------------|
| Skill de onboarding npm (org → token Automation → 2FA auth-only → secret → smoke) | IMP | alta |
| Registrar os IMPs do período (pipeline/docs/evals) via improver | IMP | média |
| Adaptar as 15 fixtures de orquestrador ao formato v1 (tirar `.self-score-skip`) | IMP | média |
| Revisão humana dos 2 seeds do golden-human (`status: revisado`) | IMP | média |

## Scorecard do Workflow (IMP-048)

| Segmento | Tipo | Nota | Evidência |
|----------|------|:----:|-----------|
| Intake | fase | N/A | sem uso — o repo do kit não roda o fluxo de charter/módulo sobre si |
| Requisitos & cliente | fase | N/A | sem REQUIREMENTS.md próprio; decisões por conversa direta |
| Tradução técnica | fase | N/A | sem módulo documentado — o produto É o workflow |
| Entrega/tasks | fase | N/A | sem `delivery/` no repo |
| Desenvolvimento | fase | 9 | 25 pacotes Go verdes · workflows versionados e endurecidos · checador determinístico · 500+ testes |
| Verificação & qualidade | fase | 9 | gates de CI provados ao vivo (título "sync:" pegado) · corpus self-score 73/88+15 · preflight npm · gate vermelho com FAILED_PKGS |
| Suporte & pós-produção | fase | 7 | diagnósticos EOTP/E404/403 rápidos e documentados no PR; porém zero registro em KNOWN_ERRORS |
| Governança da config | transversal | 8 | gates de convenção (commits, branch) enforced no CI · registry com 126 skills |
| Meta-melhoria | transversal | 7 | evals aprofundados (P3-P5) + CONTRIBUTING + versioning doc; mas fluxo do improver pulado (IMPs=0) |
| Observabilidade & eficiência | transversal | 8 | preflight fail-fast, gate agregador, self-score no CI, contagens no log |
| Padrões & aprendizado | transversal | 9 | 3 padrões extraídos para a library · corpus com 10 fixtures novas de variante/erro |
| Segurança | transversal | 8 | neutrality scan + pre-push hook · token Automation no secret (nunca no tree) · actions pinadas |
| Integrações externas | transversal | 7 | npm + GitHub Releases + Actions funcionando com idempotência; porém o caminho foi a maior dor do período |
| Portabilidade & onboarding | transversal | 8 | CONTRIBUTING bilíngue · install npm global testado · binários ×5 no release |

**Nota geral (fases com evidência): 8.3** — (Desenvolvimento 9 + Verificação 9 + Suporte 7) ÷ 3
