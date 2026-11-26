# Architecture

## Primitives

### Central Graph
- Nodes and edges represent entities, attributes, relations, evidence, decisions, rules, and learnings.
- Domain-agnostic. Schema is loaded or evolved by Steward for the current goal.
- All durable state lives here (or is linked from here).

### Unit of Work
- Claimable, trackable task.
- Has input subgraph, expected output, budget, success criteria, and execution mode (deterministic or non-deterministic).
- Lifecycle: proposed → claimed → in-progress → reviewed by Judge → committed or rejected.

### Agents
- **Steward**: Goal interpretation, swarm design, unit-of-work creation, autonomy mode, cost control.
- **Judge / Arbiter**: Proposal evaluation against rules + evidence. Approve / reject / escalate.
- **Workers**: Specialized, model-tiered (cheap models only). Can be deterministic or reasoning agents.
- Memory and Rule agents as needed.

## Execution Fabric (custom)
- Message-based communication.
- Event-sourced agent sessions for durability and replay.
- Partitioned workers for scale (catalog-sized graphs).
- Deterministic mode: constrained agents (rules + pure functions + cheap model calls).
- Non-deterministic mode: full reasoning loops, tool use, sub-agent spawning.
- Steward routes work between modes.

## Rule Engine
- Policy-as-code + decision tables (deterministic core).
- Agent-proposed rules.
- Human feedback on results → new or updated rules.
- Approval gates (human or Steward in autonomous mode).

## Model Tiering
- Workers: cheapest viable models (Gemini Flash class, etc.).
- Steward and Judge: frontier models or council.

## No-Code Surface
- Goal statement or domain pack selection.
- Steward proposes swarm + rules.
- Monitoring, review queues, feedback capture, rule editor.
- All mediated through the same graph and agents.
