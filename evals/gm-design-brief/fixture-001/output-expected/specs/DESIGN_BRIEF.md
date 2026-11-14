---
project: MeuApp
module: design-brief
audience: [design]
status: draft
---

# DESIGN_BRIEF — MeuApp

> Contexto de produto para geração de design no Figma Make.
> Fonte: PROJECT_CHARTER.md, REQUIREMENTS.md, UI_SPEC.md — 12/11/2026

## 1. Identidade do produto

- **Nome:** MeuApp
- **Posicionamento:** aplicativo web leve para organização de tarefas
  pessoais, sem curva de aprendizado
- **Tom de voz:** amigável e direto
- **Diferencial:** simplicidade — criar uma tarefa em menos de 10 segundos

## 2. Propósito e públicos

- **Propósito:** organizar o dia a dia sem ferramentas complexas
- **Públicos/atores:** Usuário — cria, organiza e conclui suas próprias
  tarefas; Administrador — gerencia contas e configurações do ambiente

## 3. Identidade visual

- **Paleta:** a definir — sugestão: fundo claro e neutro com um único
  acento de cor para as ações principais
- **Tipografia:** a definir — sugestão: sans-serif de leitura confortável
- **Densidade/estilo geral:** espaçado, cantos arredondados, elevação
  sutil nos cartões
- **Iconografia:** a definir — sugestão: ícones de linha, traço fino

## 4. Design system / padrões visuais

- **Padrões de layout:** cabeçalho com o título "MeuApp"; barra de ações
  com o botão "Nova tarefa"; conteúdo em cartões (lista de tarefas); modal
  para criação
- **Regras de UI globais:** datas no formato DD/MM/AAAA; toasts de
  confirmação no canto inferior direito; exclusões sempre pedem confirmação
  em modal

## 5. Estado atual da UI

- **Status do produto:** greenfield — nada implementado
- **Telas já especificadas/existentes:** Login, Lista de Tarefas, Detalhe
  da Tarefa
- **Padrões já adotados:** nenhum — produto sem implementação anterior

## 6. Constraints de design

- Interface inteiramente em português
- Priorizar o fluxo de criação de tarefa com o mínimo de passos
- **Out of scope (não prototipar):** relatórios e dashboards analíticos;
  integração com calendários externos

## 7. Telas prioritárias

1. Lista de Tarefas — fluxo crítico do produto e tela principal do ator
   Usuário
2. Login — porta de entrada obrigatória antes de qualquer fluxo
3. Detalhe da Tarefa — conclusão e edição, complemento do fluxo crítico

## 8. Instruções para o Figma Make

**Faça:**
- Siga as regras de UI globais da seção 4
- Mantenha o botão "Nova tarefa" sempre visível na Lista de Tarefas

**Não faça:**
- Não invente funcionalidades de out-of-scope (relatórios analíticos,
  integração com calendários)
- Não altere a paleta sugerida sem necessidade
