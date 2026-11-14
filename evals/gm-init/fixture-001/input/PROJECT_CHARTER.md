---
status: confirmed
repos: [MeuApp-API]
legado: false
---

# PROJECT CHARTER — MeuApp (Controle de Ordens de Serviço)

## Visão Geral

Sistema de gestão de ordens de serviço para oficinas mecânicas de pequeno
porte: abertura de OS, acompanhamento do status do serviço e histórico por
veículo. Substitui o controle por bloco de notas e planilha solta.

## 1. Repos

| Repo | Caminho | Branch | HEAD congelado |
|---|---|---|---|
| MeuApp-API | ../MeuApp-API | main | a1b2c3d |

## Escopo (primeira entrega)

- Abertura de ordem de serviço (cliente, veículo, descrição do problema)
- Acompanhamento do status da OS (aberta, em execução, aguardando peça, concluída)
- Histórico de serviços por veículo

## Fora de escopo (por enquanto)

- Emissão de nota fiscal
- Controle de estoque de peças
- Aplicativo do cliente final (motorista)

## Stakeholders

- Proprietário da oficina (PO — aprova escopo)
- Mecânicos executores (registram andamento)
- Atendente (abre e consulta OS)
- Patrocinador: proprietário

## Decisões de abertura

- Sem sistema legado — greenfield (legado: false)
- Repositório único de backend: MeuApp-API
