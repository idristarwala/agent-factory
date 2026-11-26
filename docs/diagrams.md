# Diagrams

Editable board (two pages: **Entity relationship**, **Request flow**):

https://www.figma.com/board/KuMPrXQ5TOGtIQivDJWxOv

The diagrams below match that board. They describe the logical model behind `internal/graph`, `internal/unitofwork`, `internal/agents`, and `internal/models`.

## Logical model

Crow's foot. `PK` primary key, `FK` foreign key, `UK` unique.

Graph nodes are domain-agnostic. A retail catalog, a price, a rule, and a learning are all nodes (or edges) of one graph. Schema for a goal is loaded or evolved by the Steward.

```mermaid
erDiagram
  GOAL ||--o{ UNIT_OF_WORK : creates
  GOAL ||--o{ AGENT : swarm
  GOAL ||--o{ SESSION_EVENT : logs
  AGENT ||--o{ UNIT_OF_WORK : claims
  AGENT ||--o{ MODEL_CALL : calls
  UNIT_OF_WORK ||--o| MUTATION : proposes
  UNIT_OF_WORK }o--|| GRAPH_NODE : "input root"
  MUTATION ||--|| DECISION : judged
  MUTATION ||--o{ MUTATION_OP : ops
  DECISION }o--o| RULE : cites
  RULE |o--o| DECISION : "feedback creates"
  MUTATION_OP }o--o| GRAPH_NODE : "touches node"
  MUTATION_OP }o--o| GRAPH_EDGE : "touches edge"
  GRAPH_NODE ||--o{ GRAPH_EDGE : "from and to"
  LEARNING }o--o| DECISION : "from review"
  LEARNING }o--|| GRAPH_NODE : "resides on"

  GOAL {
    string id PK
    string statement
    string domain
    string autonomy_mode
    datetime created_at
  }
  AGENT {
    string id PK
    string goal_id FK
    string kind
    string name
    string role
    string model_tier
    string tools
  }
  UNIT_OF_WORK {
    string id PK
    string goal_id FK
    string description
    string input_node_id FK
    string expected_output
    string mode
    string status
    int budget_max_tokens
    string claimed_by FK
  }
  MUTATION {
    string id PK
    string unit_of_work_id FK "UK"
    string evidence
  }
  MUTATION_OP {
    string id PK
    string mutation_id FK
    string op
    string node_id FK
    string edge_id FK
    string payload
  }
  DECISION {
    string id PK
    string mutation_id FK "UK"
    bool approved
    string outcome
    string reason
    string feedback
    string rule_id FK
  }
  RULE {
    string id PK
    string description
    string condition
    string action
    string source
    bool enabled
    string from_decision_id FK
  }
  GRAPH_NODE {
    string id PK
    string type
    string props
  }
  GRAPH_EDGE {
    string id PK
    string from_node_id FK
    string to_node_id FK
    string type
    string props
  }
  MODEL_CALL {
    string id PK
    string agent_id FK
    string tier
    string model
    int max_tokens
    int usage_tokens
  }
  SESSION_EVENT {
    string id PK
    string goal_id FK
    string actor
    string message
    datetime at
  }
  LEARNING {
    string id PK
    string decision_id FK
    string graph_node_id FK
    string summary
    string source
  }
```

### Enums

| Field | Values |
| --- | --- |
| `agent.kind` | `steward`, `judge`, `worker`, `memory`, `rule` |
| `agent.model_tier` | `cheap`, `frontier`, `council` |
| `unit_of_work.mode` | `deterministic`, `non_deterministic` |
| `unit_of_work.status` | `proposed`, `claimed`, `in_progress`, `reviewed`, `committed`, `rejected` |
| `decision.outcome` | `approved`, `rejected`, `escalated` |
| `rule.action` | `approve`, `reject`, `escalate` |
| `rule.source` | `human`, `feedback`, `agent` |
| `mutation_op.op` | `add`, `update`, `delete`, `edge` |
| `goal.autonomy_mode` | `supervised`, `autonomous` |

`escalated` is a decision outcome. The unit stays in review until a person commits or rejects it. It is not a sixth stored status on `unit_of_work` unless a later migration adds it.

### Invariants

- Workers may only call `TierCheap`. The model gateway rejects a higher tier before any provider call.
- Steward and Judge may use `frontier` or `council`.
- A mutation is applied to the graph only after the Judge approves it, or a person overrides.
- Feedback on a decision becomes a rule (`source = feedback`). The next `Evaluate` is constrained by that rule.
- Learnings are graph-resident: a `learning` row points at a `graph_node`.

## Request flow

One goal, left to right across lanes: Human, Steward, Worker, Model gateway, Judge, Graph, Rule engine.

```mermaid
sequenceDiagram
  actor Human
  participant Steward
  participant Gateway as Model gateway
  participant Worker
  participant Graph
  participant Judge
  participant Rules as Rule engine

  Human->>Steward: 1 Submit goal or domain pack
  Steward->>Gateway: 2 InterpretGoal with graph snapshot, frontier
  Gateway-->>Steward: 3 Swarm plan
  Note over Steward: 4 CreateWorker, cheap tier only
  Steward->>Worker: 5 Units of work, status proposed
  Worker->>Worker: 6 Claim, then in progress
  Worker->>Graph: 7 Read input subgraph
  Worker->>Gateway: 8 Generate, TierCheap
  alt Worker asks above cheap
    Gateway-->>Worker: Reject before any provider call
  else Cheap tier
    Gateway-->>Worker: Draft
  end
  Worker->>Judge: 9 Mutation plus evidence, status reviewed
  Judge->>Rules: 10 Load enabled rules
  Judge->>Gateway: 11 Frontier or council
  Gateway-->>Judge: 12 Approve, reject, or escalate
  alt Autonomous and approved
    Judge->>Graph: 13a Commit adds, updates, edges, deletes
  else Supervised or escalated
    Judge->>Human: 13b Review queue
  else Rejected
    Judge->>Human: 13c Reason, no commit
  end
  Human->>Rules: 14 and 15 Feedback becomes a rule
  Rules->>Graph: 16 Learning node linked on the graph
  Note over Judge,Rules: 17 The next Evaluate is constrained by the new rule
```

```mermaid
flowchart LR
  s1["1 Submit goal or domain pack"] --> s2["2 InterpretGoal"]
  s2 --> s3["3 Frontier swarm design"]
  s3 --> s4["4 Create workers, cheap only"]
  s4 --> s5["5 Units of work proposed"]
  s5 --> s6["6 Claim, then in progress"]
  s6 --> s7["7 Read input subgraph"]
  s7 --> s8["8 TierCheap generate, or reject"]
  s8 --> s9["9 Propose mutation and evidence"]
  s9 --> s10["10 Evaluate rules and evidence"]
  s10 --> s11["11 Frontier or council"]
  s11 --> s12{"12 Decision"}
  s12 -->|"autonomous + approved"| s13a["13a Commit to graph"]
  s12 -->|"supervised or escalated"| s13b["13b Human queue"]
  s12 -->|rejected| s13c["13c No commit"]
  s13b --> s14["14 Feedback"]
  s13c --> s14
  s14 --> s15["15 Rule, source feedback"]
  s13a --> s16["16 Learning node"]
  s15 --> s16
  s16 --> s17["17 Next Evaluate constrained"]
```

Unit-of-work status moves `proposed` to `claimed` to `in_progress` to `reviewed`, then `committed` or `rejected`. An escalated decision waits on the human queue before either commit.
