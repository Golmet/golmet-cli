---
project: MeuApp
module: figma-prompts
audience: [design]
status: draft
---

# FIGMA_PROMPTS — MeuApp

Registro append-only dos prompts gerados para o Figma Make.

## Prompt — 12/11/2026 09:00 (modo: completo)

> **Versão dos documentos:** abc1234
> **Documentos fonte:** REQUIREMENTS.md, UI_SPEC.md

Protótipo do MeuApp — aplicativo web de controle de tarefas pessoais.

### 1. Contexto

O MeuApp é um aplicativo web para organizar tarefas pessoais com o mínimo
de atrito: entrar, ver as tarefas do dia e criar novas tarefas em segundos.

### 2. Atores e permissões visuais

- **Usuário** — vê apenas as próprias listas de tarefas.
- **Administrador** — além das telas do Usuário, acessa a área administrativa
  de contas e configurações.

### 3. Lista de telas

#### Tela Login — /login

- Propósito: porta de entrada com e-mail e senha.
- Campos: e-mail (texto, obrigatório, validação de formato), senha (senha,
  obrigatório, mínimo 8 caracteres), botão "Entrar".
- Estados: loading (botão com indicador), erro de credenciais (mensagem em
  linha acima do botão).

#### Tela Lista de Tarefas — /tarefas

- Propósito: tela principal com as tarefas do usuário.
- Árvore de componentes: cabeçalho com o título "MeuApp"; barra de ações com
  o botão "Nova tarefa"; lista de cartões (título, prazo, indicador de
  concluída); modal "Nova tarefa" com o formulário.
- Campos do modal: título (texto, obrigatório, até 120 caracteres), prazo
  (data, opcional), concluída (caixa de marcar).
- Estados: loading (esqueleto no lugar dos cartões), vazio (ilustração leve
  com a mensagem "Nenhuma tarefa ainda"), erro (mensagem em linha com botão
  "Tentar de novo"), sucesso (toast "Tarefa salva").
- Regra condicional: se o prazo estiver preenchido e for anterior a hoje, o
  cartão mostra destaque de atraso.

#### Tela Detalhe da Tarefa — /tarefas/{id}

- Propósito: editar campos, concluir ou excluir a tarefa.
- Botão de conclusão "Concluir tarefa" só aparece para tarefas não concluídas.
- Estados: sucesso (toast "Tarefa salva"), exclusão sempre com confirmação
  em modal.

### 4. Fluxo de navegação

Login → Lista de Tarefas → Detalhe da Tarefa → volta para a Lista de
Tarefas. O botão "Nova tarefa" abre o modal sobre a Lista de Tarefas.

### 5. Regras de UI globais

- Datas exibidas no formato DD/MM/AAAA.
- Toasts de confirmação no canto inferior direito.
- Exclusões sempre pedem confirmação em modal.

### 6. Não incluir

Não prototipe: relatórios e dashboards analíticos e integração com
calendários externos — estão fora do escopo do produto.

### 7. Instrução de estilo

Interface limpa e leve, alinhada a um produto simples e direto: cartões com
cantos arredondados, espaçamento confortável, um único acento de cor para
as ações principais.

---

## Prompt — 12/11/2026 10:30 (modo: delta)

> **Versão dos documentos:** def5678
> **Documentos fonte:** REQUIREMENTS.md, UI_SPEC.md

Atualização do protótipo existente com base em mudanças nos requisitos.

- Na Lista de Tarefas, incluir um filtro por prioridade com as opções alta,
  média e baixa.
- No Detalhe da Tarefa, o rótulo do botão de conclusão passa a ser
  "Concluir tarefa".

---
