# Entity-Relationship and Architecture Diagrams

This document contains detailed diagrams of the Agent Factory control plane and data model.

## 1. Core Data Model (ER Diagram)

```mermaid
erDiagram
    GOAL ||--o{ UNIT_OF_WORK : "creates"
    STEWARD ||--o{ UNIT_OF_WORK : "interprets into"
    UNIT_OF_WORK ||--o{ MUTATION : "produces"
    WORKER ||--o{ MUTATION : "proposes"
    JUDGE ||--o{ DECISION : "evaluates"
    MUTATION ||--|| DECISION : "results in"
    DECISION ||--o{ RULE : "may create via learning"
    GRAPH ||--o{ NODE : "contains"
    GRAPH ||--o{ EDGE : "contains"
    NODE ||--o{ ATTRIBUTE : "has"
    RULE ||--o{ RULE_VERSION : "has versions"
    LEARNING_LOOP ||--o{ RULE : "proposes"
    LEARNING_LOOP ||--o{ AGENT_VERSION : "registers"
    WORK_QUEUE ||--o{ UNIT_OF_WORK : "holds"
    AGENT_REGISTRY ||--o{ AGENT_VERSION : "tracks"

    GOAL {
        string text
        string domain
        string success_criteria
    }
    UNIT_OF_WORK {
        string id PK
        string description
        string status
        string input_subgraph
        timestamp claimed_at
        string claimed_by
    }
    MUTATION {
        string unit_of_work_id FK
        json updates
        json adds
        string[] evidence
    }
    DECISION {
        bool approved
        string reason
        string feedback
        timestamp decided_at
    }
    RULE {
        string id PK
        int version
        string description
        string source
        bool active
    }
    NODE {
        string id PK
        string type
        json props
    }
    WORK_QUEUE {
        string uow_id FK
        timestamp lease_expires
        string worker_id
    }
```

## 2. Continuous Execution Flow

```mermaid
flowchart TD
    A[Goal Text] --> B[Steward.InterpretGoal]
    B --> C{Problems remaining?}
    C -->|No| D[Done]
    C -->|Yes| E[Create Units of Work]
    E --> F[Enqueue to WorkQueue]
    F --> G[Parallel Workers Claim]
    G --> H[Worker.Execute → Mutation]
    H --> I[Judge.Evaluate against active Rules]
    I --> J{Approved?}
    J -->|Yes| K[Graph.Apply Mutation]
    J -->|No| L[Record Rejection]
    K --> M[Complete UoW]
    L --> M
    M --> N[LearningLoop.Review Decisions]
    N --> O[Propose new Rules / Agents]
    O --> P[Apply Updates]
    P --> B
```

## 3. Self-Learning Loop

```mermaid
sequenceDiagram
    participant Runner as ContinuousRunner
    participant Judge
    participant Learning as LearningLoop
    participant Rules as RuleEngine
    participant Registry as AgentRegistry

    Runner->>Judge: Evaluate(mutation)
    Judge-->>Runner: Decision (approved/rejected)
    Runner->>Learning: Review(decisions)
    Learning->>Learning: Detect gaps from reject patterns
    Learning->>Rules: Propose new VersionedRule
    Learning->>Registry: Register specialist agent
    Rules-->>Learning: New active version
    Registry-->>Learning: Agent registered
    Learning-->>Runner: Applied updates
    Runner->>Runner: Next round uses new rules/agents
```

## 4. Source Mapping to Entity

```mermaid
flowchart LR
    A[Source payload] --> B[ProfileSample]
    B --> C[Review identifiers and field paths]
    C --> D[SourceMapping.Preview]
    D --> E[Approved mapping revision]
    E --> F[Import mapped records]
    F --> G[Workers + Judge]
    G --> H[Decisions visible to customer]
```

## 5. Customer Feedback Path

```mermaid
flowchart TD
    A[React UI] -->|POST /api/feedback| B[API]
    B --> C[RuleEngine.Propose]
    C --> D[New VersionedRule Source=feedback]
    D --> E[Active Rules]
    E --> F[Next ContinuousRunner round]
    F --> G[Judge uses new rule]
```

## 6. Persistence Modes

```mermaid
flowchart LR
    A[Steward] --> B[MemoryGraph + MemoryQueue]
    B --> D[ContinuousRunner]
    D --> E[Run history with audit entries and model usage]
    E --> F[Configured workspace store]
```

## Notes

- All mutations are proposed and only applied after Judge approval.
- Rules are immutable and versioned; only one version is active.
- LearningLoop closes the loop so rejects improve the next round automatically.
- Domain packs supply detectors, attribute sets, and default rules without changing the core.
- The same model works for catalog, pricing, inventory, merchandising, or any Entity-based problem.
