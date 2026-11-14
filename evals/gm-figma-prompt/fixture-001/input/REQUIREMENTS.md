# REQUIREMENTS — MeuApp (fixture de eval)

> **Propósito:** insumo neutro para o eval de prompt do Figma Make.

## 1. Escopo

Aplicativo web de controle de tarefas pessoais: autenticação simples,
lista de tarefas, criação/edição e conclusão de tarefas.

## 2. Atores

- **Usuário** — pessoa que organiza suas próprias tarefas; vê apenas as
  próprias listas.
- **Administrador** — gerencia contas de usuários e configurações do
  ambiente; além das telas do Usuário, acessa a área administrativa.

## 7. Casos de Uso

- UC-01 — Entrar com e-mail e senha (tela Login)
- UC-02 — Ver a lista de tarefas do dia (tela Lista de Tarefas)
- UC-03 — Criar uma tarefa nova (botão "Nova tarefa")
- UC-04 — Concluir ou editar uma tarefa (tela Detalhe da Tarefa)

## 8. Saídas e Rótulos

- Botão principal da lista: "Nova tarefa"
- Ação de conclusão: "Concluir tarefa"
- Lista vazia: "Nenhuma tarefa ainda"
- Toast de sucesso: "Tarefa salva"

## Decisions and Out of Scope

- Relatórios e dashboards analíticos — fora do escopo do produto
- Integração com calendários externos — fora do escopo do produto
