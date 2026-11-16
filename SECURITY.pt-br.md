<p align="center"><a href="SECURITY.md">English</a> | <a href="README.pt-br.md">Português (Brasil)</a></p>

# Política de Segurança

O GOLMET (`golmet-cli`) materializa workflows e specs de fontes externas — cores e packs baseados em Git — em configurações de engines (opencode, claude-code e outros). Como ele instala conteúdo escrito por terceiros em configs consumidos por agentes de IA, tratamos os limites de validação dele (contratos de pack, contenção de paths, redação de credenciais, gating de mutações MCP) como superfície sensível de segurança.

## Versões Suportadas

| Versão | Suportada |
| ------- | --------- |
| 1.0.x   | ✓ Sim    |
| < 1.0   | ✗ Não     |

Apenas a release 1.0.x mais recente recebe correções de segurança. Se você está em uma release antiga, atualize primeiro — não faremos backport de patches para versões não suportadas.

## Reportando uma Vulnerabilidade

**Canal primário e oficial: GitHub Private Vulnerability Reporting** neste
repositório (aba Security — "Report a vulnerability"). Este é o único canal
que nos comprometemos a monitorar atualmente. Por favor, **não** abra uma
issue pública para nada que você considere explorável.

> Nota do mantenedor: quando houver um endereço de e-mail dedicado para
> relatos de segurança, ele será listado aqui. Até lá, o GitHub Private
> Vulnerability Reporting é o canal oficial — não envie relatos para outros
> endereços.

Por favor, evite divulgação pública até que uma correção seja lançada e tenhamos
coordenado uma janela de disclosure com você.

## O que Incluir

Para triagem rápida, por favor inclua:

- **Versão do GOLMET** (`golmet version` ou a tag da release) e o SHA de commit, se compilado da fonte.
- **SO e engine afetado** (ex.: Windows 11 + opencode, Linux + claude-code), indicando se o fluxo passou pelo CLI ou pelas tools do servidor MCP.
- **Passos de reprodução** — comandos exatos e qualquer `golmet.yaml` / `pack.yaml` usados.
- **Um PoC de pack ou core malicioso**, quando aplicável — um repositório Git mínimo cujo conteúdo dispara o problema (ex.: um `pack.yaml` cujas entradas de `provides` tentam traversal, um symlink/junction em um pack, conteúdo criado para prompt injection que é materializado). Anexe ou linkie o repo; não envie credenciais reais.
- **Comportamento esperado vs. real**, incluindo qualquer output de erro estruturado.

## Escopo

### Dentro do escopo

- **Execução de código ou corrupção de config via conteúdo materializado** — packs/cores maliciosos que abusam do pipeline de instalação para injetar instruções (prompt injection aparecendo em arquivos de skill/agent materializados) ou sobrescrever arquivos que não deveriam.
- **Escape de path durante a materialização** — entradas de `provides` do `pack.yaml` burlando a allowlist de `agents/`/`skills/`, path traversal, ou escape de symlink/junction do cache de fonte ou da raiz de destino.
- **Falhas de redação** — credenciais Git, tokens ou segredos de source URLs vazando em manifests, payloads, relatórios ou mensagens de erro.
- **Bypasses de validação de contrato de pack** — incluindo validação de `requires-core` (e `no-override`) que possa ser evadida, deixando instalar um pack incompatível ou hostil.
- **O servidor MCP (transporte `stdio`)** — gating de mutações (enforcement de `confirm=true`), efeitos colaterais de `dry_run` e tratamento de argumentos de tools.

### Fora do escopo

- **Vulnerabilidades nos engines hospedeiros** (opencode, claude-code, editores, o próprio `git`). Reporte aos projetos respectivos; ficamos felizes em ajudar a coordenar se o problema exigir mudanças do nosso lado.
- **Configuração incorreta do operador que desativa validações** — ex.: usar explicitamente `--force` para sobrescrever arquivos geridos, remover checks das configs ou apontar fontes para repos não confiáveis com validações removidas. As garantias da ferramenta assumem que os caminhos de validação embarcados estão em uso.
- **Força bruta de credenciais Git** — força de credenciais, rate limiting ou quebra de provedores de hospedagem Git está fora do escopo.
- Engenharia social para induzir usuários a instalar packs em que não confiam (veja a nota de confiança abaixo).

## Nota de Modelo de Confiança

O GOLMET valida a segurança *estrutural* de packs e cores (paths na allowlist, contenção, conformidade de contrato) mas **não** audita o conteúdo *semântico* de packs de terceiros. Um pack estruturalmente válido ainda pode conter texto de prompt-injection ou instruções ruins que um agente de IA vai ler. Instalar um pack significa confiar no autor no nível de conteúdo. Trate as fontes de pack como você trata dependências em qualquer gerenciador de pacotes: pune-as (o GOLMET registra o SHA resolvido no manifest), revise o que você instala e reporte ataques de nível de conteúdo como vulnerabilidades quando eles explorarem uma brecha de validação, e não mera escolha do autor.

## Safe Harbor

Consideramos pesquisa de segurança de boa-fé autorizada e bem-vinda, desde que você:

- Teste apenas contra repositórios e configs que você possui ou tenha permissão explícita para modificar;
- Não exfiltre dados, degrade serviços ou ataque infraestrutura que você não controla (incluindo a infraestrutura de provedores de hospedagem Git);
- Use PoCs benignos que demonstrem o problema sem causar dano;
- Reporte privadamente pelo canal oficial e nos dê tempo razoável para corrigir antes de qualquer divulgação pública;
- Não exija pagamento ou faça ameaças atreladas à divulgação.

Não tomaremos ação legal por pesquisa que respeite esses limites, mesmo que cause disrupção temporária e reversível nos seus próprios ambientes de teste. Se você não tem certeza se uma ação se enquadra nesta política, pergunte primeiro pelo canal de reporte.

## Expectativas de Resposta (SLA Honesto)

O GOLMET é um projeto jovem mantido com recursos limitados, então nos comprometemos com processo, não com prazos:

- **Reconhecimento de triagem: dentro de 7 dias** do seu relatório, você receberá uma resposta confirmando o recebimento e uma avaliação inicial de severidade e validade.
- **Sem prazo fixo de correção.** Vamos trabalhar com você em um plano de disclosure coordenado, mas não prometeremos uma data de patch específica que não possamos cumprir. Correções complexas (ex.: redesign de contenção no tratamento de junctions do Windows) podem levar mais tempo que as simples.
- Vamos mantê-lo informado do progresso em cada etapa significativa e creditá-lo nas release notes (a menos que prefira permanecer anônimo).
- Relatos fora do escopo serão respondidos com uma explicação e, quando possível, um apontamento para o projeto ou mudança de configuração corretos.
