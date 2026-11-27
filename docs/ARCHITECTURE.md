# Architecture

## Core primitives

### Unit of work
A claimable task with a description, expected output, and mode (deterministic or non-deterministic).

### Central graph
The single source of truth. Agents propose mutations; only approved mutations are applied.

### Agents
- **Steward** — interprets goals, creates units of work, spawns specialized agents.
- **Worker** — executes a unit of work (cheap model). Can be specialized (title-fixer, weight-fixer, etc.).
- **Judge** — evaluates proposals against versioned rules (frontier or cheap). Fails closed.

### Continuous loop
Detect remaining problems → spawn specialized agents → propose → judge → apply → repeat until clean.

## Data flow

```
goal text
   ↓
Steward.InterpretGoal()
   ↓
[]UnitOfWork
   ↓
specialized Workers
   ↓
graph.Mutation proposals
   ↓
Judge.Evaluate()
   ↓
approved → graph update
rejected → learning loop may create new rules
   ↓
repeat until DetectProblems() returns empty
```

## Persistence

Workspace definitions and run history use SQLite by default. Set `DATABASE_URL` for PostgreSQL or `FACTORY_STORE_MODE=memory` for process-local storage. The continuous CLI uses a process-local work graph and queue and publishes its terminal run to the workspace store. [Storage configuration](CONFIGURATION.md)

## Domain packs

Pluggable definitions (catalog, pricing, inventory, merchandising) that supply node types, worker roles, and default rules.
