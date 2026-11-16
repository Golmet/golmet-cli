---
project: MeuApp
module: arquitetura-workflow
audience: [tecnica]
status: draft
---

# ARCHITECTURE — Config do engine (fixture de eval)

## Visão geral

Este documento segue a árvore de delegação canônica do `registry.yaml`.
As seções estruturais (§1-§6) são geradas pelo gm-registry-render dentro
dos marcadores BEGIN/END_REGISTRY — edições manuais dentro do bloco são
sobrescritas; seções narrativas fora dos marcadores são preservadas.

<!-- BEGIN_REGISTRY -->
## 1. Diagrama de camadas

```mermaid
flowchart BT
    subgraph L5["5. Skills (1 skill)"]
        direction TB
        S1["gm-charter (Intake)"]
    end
    subgraph L4["4. Subagents (2)"]
        direction TB
        NEXT["@gm-next<br/>próxima ação"]
        INT["@gm-intake-orchestrator<br/>roteia o intake"]
    end
    subgraph L3["3. Primary Agents"]
        REQ["gm-req.md"]
    end
    REQ --> NEXT
    REQ --> INT
    NEXT --> S1
    INT --> S1
```

## 2. Diagrama de interações

```mermaid
flowchart LR
    User((Usuário))
    subgraph Primary["1. Primary Agents"]
        REQ["gm-req.md"]
    end
    subgraph Subagents["2. Subagents"]
        NEXT["@gm-next"]
        INT["@gm-intake-orchestrator"]
    end
    subgraph Skills["3. Skills"]
        S1["gm-charter"]
    end
    subgraph External["4. Sistemas"]
        FS["Filesystem do projeto"]
    end
    User --> REQ
    REQ --> NEXT
    REQ --> INT
    S1 -.->|"escreve"| FS
```

## 3. Primary Agents

| Primary | Descrição | Delega para | Invoca diretamente |
|---------|-----------|-------------|--------------------|
| `gm-req` | Requisitos, descoberta de negócio e intake greenfield | `@gm-next`, `@gm-intake-orchestrator` | `gm-charter` |

## 4. Subagents

| Subagent | Descrição | Invoca skills | Owner | Disponível para |
|----------|-----------|---------------|-------|-----------------|
| `@gm-next` | Próxima ação determinística pelo estado dos artefatos | `gm-charter` | `gm-req` | `gm-req` |
| `@gm-intake-orchestrator` | Orquestrador da fase de Intake | `gm-charter` | `gm-req` | `gm-req` |

## 5. Skills por domínio

### 5.1 Intake

| Skill | Descrição | Acessada via |
|-------|-----------|--------------|
| `gm-charter` | PROJECT_CHARTER.md a partir da primeira conversa | `@gm-intake-orchestrator` |

## 6. Taxonomia de domínios

| Domínio | Orquestrador padrão | Skills |
|---------|---------------------|--------|
| intake | `@gm-intake-orchestrator` | charter |
<!-- END_REGISTRY -->

## Convenções

IDs no padrão FORMAT (RN-NNN, PEND-NNN, IMP-NNN). Diagramas mermaid
regenerados a cada render — colisão de ID de nó nunca é emitida.
