---
project: MeuApp
module: ui-spec
audience: [tecnica, design]
status: draft
---

# UI_SPEC — MeuApp (fixture de eval)

## 2. Telas

### 2.1 Login — /login

- Propósito: porta de entrada com e-mail e senha.

### 2.2 Lista de Tarefas — /tarefas

- Propósito: tela principal com as tarefas do usuário.

### 2.3 Detalhe da Tarefa — /tarefas/{id}

- Propósito: editar campos, concluir ou excluir a tarefa.

## 2.4 Campos de formulário

| Tela | Campo | Input | Obrigatório | Observação |
|------|-------|-------|-------------|------------|
| Login | e-mail | text | Sim | validação de formato |
| Login | senha | password | Sim | mínimo 8 caracteres |
| Lista | título da tarefa | text | Sim | até 120 caracteres |
| Lista | prazo | date | Não | exibido como DD/MM/AAAA |
| Lista | concluída | checkbox | Não | marcada = tarefa concluída |

## 2.5 Regras de exibição condicional

- Se o campo `prazo` estiver preenchido e for anterior a hoje, o cartão
  da tarefa mostra destaque de atraso.
- O botão "Concluir tarefa" só aparece para tarefas não concluídas.

## 2.6 Estados visuais

- **Loading:** esqueleto no lugar dos cartões durante o carregamento.
- **Empty:** ilustração leve + mensagem "Nenhuma tarefa ainda".
- **Error:** mensagem em linha no topo com botão "Tentar de novo".
- **Sucesso:** toast "Tarefa salva" no canto inferior direito.

## 2.7 Árvore de componentes — Lista de Tarefas

- Cabeçalho (título "MeuApp")
- Barra de ações (botão "Nova tarefa")
- Lista de cartões (cartão da tarefa: título, prazo, indicador de concluída)
- Modal "Nova tarefa" (formulário)

## 4. Fluxo de navegação

Login → Lista de Tarefas → Detalhe da Tarefa → volta para Lista de Tarefas.
"Nova tarefa" abre modal sobre a Lista de Tarefas.

## 5. Regras de UI globais

- Datas exibidas no formato DD/MM/AAAA.
- Toasts de confirmação no canto inferior direito.
- Exclusões sempre pedem confirmação em modal.
