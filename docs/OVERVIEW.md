# Agent Factory — Overview

Agent Factory is a domain-agnostic execution fabric for multi-agent work. It takes a natural-language goal and drives a swarm of specialized agents until the outcome is achieved.

## Who it is for

- Retail operators fixing catalog, pricing, inventory, or merchandising at scale
- Platform teams that need a governed agent loop (propose → judge → apply)
- Anyone who wants agents that keep working until a measurable goal is met

## Core concepts

| Concept | Description |
|---------|-------------|
| Goal | Plain-text statement of what should be true when finished |
| Unit of work | A claimable, discrete task created by the Steward |
| Central graph | Single source of truth; all changes are mutations |
| Steward | Interprets goals, creates work, spawns specialized agents |
| Worker | Executes a unit of work on the cheapest viable model |
| Judge | Approves or rejects proposals against versioned rules |
| Rule | Versioned, immutable constraint; only one version active |
| Learning loop | Steward reviews outcomes and creates new rules or agents |

## Design principles

1. **Agent-first** — no external orchestration frameworks
2. **Domain-agnostic** — catalog is one pack; pricing, inventory, merchandising are others
3. **Governed** — nothing reaches the graph without Judge approval
4. **Continuous** — the loop runs until problems are gone or a round limit is hit
5. **Observable** — agent versions, rule versions, and decisions are recorded

## High-level flow

```
goal text → Steward → Units of work → Specialized Workers → Judge → Central Graph → repeat until clean
```
