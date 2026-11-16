# gm-stack-boilerplate — Bootstrap de Projetos a partir de Boilerplates

Template engine para bootstrapping de projetos a partir de boilerplates
pré-definidos por stack. Lê o registry T3 de templates por stack
(pack-first com fallback: `packs/<pack-ativo>/rules/stack-templates.yaml`
— registry da org IN-PLACE no state pack; ausente → fallback
`profile/stack-templates.yaml`, canônico do kit — lazy-loaded via AGENTS.md),
clona o boilerplate correspondente (em ref fixa quando o template declara
`ref` — determinismo do bootstrap), executa o setup pós-clone
(rename, env, restore; script do próprio boilerplate quando o template
declara `post_setup_script`) e prepara o projeto para receber implementação.
Invocada por `gm-dev-backend` e `gm-dev-frontend` ao detectar repo vazio
(sem código-fonte, só README) ou no início de um novo projeto via
`gm-intake-orchestrator`. Evita scaffold genérico (`dotnet new webapi`)
— usa um repo curado da organização (`boilerplate_org` do perfil do
cliente).

## Escopo

- **Entrada:** um repositório vazio (só README) + stack detectada (ex.: `.NET`)
- **Saída:** repositório populado com o boilerplate da stack + renomeado para o projeto
- **Quando carregar:** no passo 0 do `gm-dev-backend`/`gm-dev-frontend`, ANTES de implementar qualquer task — se o repo for vazio, carregar este stack-boilerplate primeiro
- **Fallback:** sem boilerplate no registry → executor gera scaffold genérico (`dotnet new webapi`)

## Lê (upstream)

- Registries de boilerplates (T3), na ordem: 1) `packs/<pack-ativo>/rules/stack-templates.yaml` — registry da org IN-PLACE no state pack; se existir, é a fonte primária e SOBREPOE o kit por stack (mesma chave = o pack manda; stacks só no kit = mantidas); múltiplos packs com registry → `ask`; 2) fallback `profile/stack-templates.yaml` — canônico do kit (lazy-loaded via AGENTS.md). Por template: `boilerplate` (relativo à org do `boilerplate_org` — null = stack sem boilerplate), `detect`, `description`, `pre_setup`/`post_setup` (prosa) e os campos OPCIONAIS de determinismo: `ref` (tag/branch fixa a clonar; ausente = HEAD/main) e `post_setup_script` (path relativo AO BOILERPLATE de um script executável que o boilerplate traz, ex.: `scripts/setup.ps1`; presente → execução determinística pós-clone e o `post_setup` vira checklist de verificação)
- `AGENTS.md` do repo alvo — convenções específicas do projeto

## Gera (downstream)

- O repositório alvo populado com o boilerplate + renomeado + configurado
- Relatório resumido: boilerplate usado, steps executados, pendências pós-setup

## Nunca edita

- Registries de boilerplates (`profile/stack-templates.yaml` e `packs/<pack-ativo>/rules/stack-templates.yaml` — adicionar stacks/boilerplates é via pack da org ou CR, nunca por este skill)
- Boilerplate repos (são lidos como template, nunca modificados)
- Código existente no repo alvo (só escreve em repo vazio)

## Tools

- `file` — ler o registry YAML e o AGENTS.md do repo
- `shell` — `git clone` (com/sem `--branch <ref>`), execução do `post_setup_script` (`.ps1` → powershell, `.sh` → bash), `powershell rename-project.ps1`, `dotnet restore`
- `glob` — detectar stack no repo alvo
- `ask` — confirmar stack quando ambígua ou sem boilerplate

## Modo de erro

`warn` — repo já tem código-fonte → pular boilerplate (repo existente, não é bootstrap).
Boilerplate não encontrado ou repo privado sem acesso → orientar fallback a scaffold genérico.

## Algoritmo

1. **Detectar stack no repo alvo:** use `glob` para os padrões em `templates.<stack>.detect`.
   Se mais de uma stack combinar, pergunte (`ask`). Se nenhuma, reporte que a stack não tem
   boilerplate no registry e encerre (deixando o executor cair no scaffold genérico).

2. **Carregar o registry (pack-first com fallback):** se o pack ativo da org
   tiver `packs/<pack-ativo>/rules/stack-templates.yaml`, ele é a fonte
   primária e SOBREPOE o kit por stack — mesma chave = o pack manda; stacks
   que só existem no kit são mantidas. Múltiplos packs com registry →
   pergunte (`ask`). Registry do pack ausente → fallback
   `profile/stack-templates.yaml` (canônico do kit). Busque o template da
   stack detectada no registry efetivo.

3. **Verificar boilerplate disponível:** se `templates.<stack>.boilerplate` for `null`,
   reporte que não há boilerplate para esta stack e encerre (executor usará scaffold genérico).

4. **Verificar se o repo já tem código:** se o repo alvo tiver mais que README.md e arquivos
   de config mínima (`.gitignore`, `.env.example`, `LICENSE`), pule — repo não é vazio.

5. **Clonar boilerplate na pasta correta:**

   ```bash
   # Com `ref` declarada no template — estado fixo (determinismo):
   git clone --branch <ref> --depth 1 https://github.com/<Org>/<Boilerplate> /tmp/<boilerplate>-tmp
   # Sem `ref` — como sempre (HEAD/main):
   git clone --depth 1 https://github.com/<Org>/<Boilerplate> /tmp/<boilerplate>-tmp
   ```

   Ref inexistente no repo remoto → erro claro: "crie a tag/branch `<ref>` no
   boilerplate ou remova o campo `ref` do template".

6. **Executar o `post_setup_script` (quando o template declara um):** rode o
   script DENTRO do dir clonado, escolhendo o shell pela extensão
   (`.ps1` → `powershell -File`, `.sh` → `bash`):

   ```bash
   cd /tmp/<boilerplate>-tmp
   powershell -File scripts/setup.ps1 <ProjectName>   # exemplo .ps1
   # bash scripts/setup.sh <ProjectName>              # exemplo .sh
   ```

   - **Falha do script = ERRO claro com a saída integral** — não prossiga
     com a cópia; o bootstrap é determinístico, não "aproximado".
   - Script ausente no boilerplate clonado (o template declara o path mas o
     repo não o tem) → erro claro: "o boilerplate deveria trazer
     `<post_setup_script>` — crie-o no boilerplate ou remova o campo do template".
   - Com script executado com sucesso, o campo `post_setup` (prosa) vira
     **checklist de VERIFICAÇÃO**: confirme cada item (ex.: env criado,
     connection string apontando para `<ProjectName>`) antes de copiar.
   - Sem `post_setup_script` → siga o `post_setup` como prosa guiada
     (comportamento original, passo a passo).

7. **Renomear para o nome do projeto:** execute o script de rename do boilerplate (se existir).

   ```bash
   cd /tmp/<boilerplate>-tmp
   powershell -File rename-project.ps1 <ProjectName>
   ```

   O nome do projeto (`<ProjectName>`) é derivado do nome do repo alvo:
   - Se o repo termina com `-API`, `-API` é extraído (ex.: `Candela-API` → `Candela`)
   - Caso contrário, usa o nome completo do repo.
   - Pergunte via `ask` se a derivação não for clara.

8. **Copiar conteúdo para o repo alvo:**

   ```bash
   # Remove .git do boilerplate
   Remove-Item -Recurse -Force /tmp/<boilerplate>-tmp/.git
   # Copia para o repo alvo
   Copy-Item -Recurse -Force /tmp/<boilerplate>-tmp\* <repo-target>\
   ```

9. **Configurar pós-clone:**
   - Copiar `.env.example` para `.env`
   - Rodar `dotnet restore` (se .NET)
   - Ajustar connection string do banco para `<ProjectName>` (em `appsettings*.json` ou `.env`)

10. **Registrar no relatório:** boilerplate usado, ref clonada (ou "HEAD/main"), script de pós-clone executado (ou prosa guiada), rename aplicado, steps executados, quebra de compatibilidade com
   o playbook (ex.: versão de dependências diferente da documentada), pendências pós-setup
   (ex.: configurar secrets, subir migrations, ajustar docker-compose ports).

## Relatório de saída

```
═══ gm-stack-boilerplate — .NET ═══
Boilerplate: <boilerplate_org do perfil>/<Boilerplate>.NET-API (ex.: Acme-Group/Boilerplate.NET-API)
Ref: v1.2.0 (ou "HEAD/main" — template sem ref)
Post-setup: scripts/setup.ps1 executado (ou "prosa guiada" — template sem post_setup_script)
Rename: <ProjectName> (Candela)
Steps executados:
  - git clone --branch v1.2.0 --depth 1
  - scripts/setup.ps1 Candela (OK)
  - Checklist de verificação do post_setup confirmado
  - Copiado para repo alvo
  - .env criado a partir de .env.example
  - dotnet restore (OK)
Quebras de compatibilidade:
  - (nenhuma — boilerplate atualizado)
Pendências pós-setup:
  - Configurar secrets do GitHub
  - Ajustar docker-compose ports se necessário
```

## Formato de commit

```
(Não commita diretamente — o executor (gm-dev-backend) commita o bootstrap)
```

## Casos de Borda

- **Repo já tem código:** pular silenciosamente (repo existente, em desenvolvimento). Reporte "pulado — repo não vazio".
- **Boilerplate não encontrado:** oriente fallback ao scaffold genérico (`dotnet new webapi`). Registry não precisa ter entry para toda stack.
- **`ref` inexistente no repo remoto:** erro claro com a remediação — "crie a tag/branch `<ref>` no boilerplate ou remova o campo `ref` do template". Não faça fallback silencioso para HEAD/main (o determinismo é o propósito do campo).
- **`post_setup_script` ausente no boilerplate clonado:** erro claro — "o boilerplate deveria trazer `<post_setup_script>` — crie-o no boilerplate ou remova o campo do template". Não invente os steps do script.
- **`post_setup_script` falha na execução:** pare com a saída integral do script no relatório; o conteúdo bruto do boilerplate fica no diretório clonado, e o operador decide entre corrigir o boilerplate/ref ou prosseguir manualmente.
- **rename-project.ps1 falha:** reporte erro e pare; o conteúdo bruto do boilerplate fica no diretório, e o operador pode renomear manualmente.
- **Multi-stack detectada (ex.: `*.csproj` + `angular.json` no mesmo repo):** pergunte via `ask` qual stack usar.
- **Stack sem boilerplate no registry:** reporte e encerre; não falhe (o executor faz scaffold genérico).
- **Projeto sem nome claro (ex.: repo `Candela` sem sufixo):** pergunte o nome do projeto para o rename. O nome do projeto .NET (solution name) pode ser diferente do nome do repo.
