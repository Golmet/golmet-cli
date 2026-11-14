---
project: MeuApp
module: ui-spec
audience: [tecnica, design]
status: draft
---

# UI_SPEC — MeuApp (fixture de eval)

## 2. Telas

### 2.1 Login — /login

Porta de entrada com e-mail e senha.

### 2.2 Lista de Tarefas — /tarefas

Tela principal: cabeçalho com o título "MeuApp", barra de ações com o botão
"Nova tarefa", lista de cartões de tarefa.

### 2.3 Detalhe da Tarefa — /tarefas/{id}

Edição de campos, conclusão e exclusão com confirmação.

## 5. Regras de UI globais

- Datas no formato DD/MM/AAAA
- Toasts de confirmação no canto inferior direito
- Exclusões sempre pedem confirmação em modal
