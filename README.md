# Agent Factory

Agent-first platform for any problem in retail (and beyond).

## Core Ideas

- **Central Graph**: Single source of truth. Every agent reads from and proposes mutations to the graph. Universal for any domain (catalog, pricing, inventory, merchandising, compliance, etc.).
- **Unit of Work**: Discrete, claimable pieces of work. Agents claim, process, and propose graph changes.
- **Steward**: Oversees the factory. Interprets goals, creates specialized agents, assigns units of work, manages autonomy.
- **Judge (Arbiter)**: Evaluates agent proposals against rules and evidence. Approves or rejects.
- **Workers**: Specialized agents that do the work. Always on cheapest viable models.
- **Memory & Learnings**: Shared, graph-resident or graph-linked.
- **Rule Engine**: Hybrid (human-authored + agent-proposed). Human feedback on results becomes rules.

## Governance Loop

1. Agents propose graph mutations (from units of work).
2. Judge evaluates and approves/rejects.
3. Humans review results and provide feedback.
4. Feedback is turned into rules.
5. Future proposals and judgments are constrained by the updated rules.

## Language & Runtime

- Core fabric and agent runtime: **Go** (concurrency, deployment simplicity, scale).
- No-code UI / control plane: TypeScript (later).
- Custom execution fabric. No external orchestration frameworks.

## Domains

The same fabric applies to all retail facets:
- Catalog / PIM
- Pricing & Promotions
- Inventory & Supply
- Merchandising & Assortment
- Content & Marketing
- Customer & Personalization
- Order & Fulfillment
- Returns & Service
- Compliance & Governance
- Vendor Management

## Status

Early scaffold. Architecture first.

## Build & Test

Primary development happens in this repository. Clone it locally to build and test the Go core. The no-code UI will live in the same repo.

```bash
git clone https://github.com/idristarwala/agent-factory.git
cd agent-factory
go build ./...
go test ./...
```
