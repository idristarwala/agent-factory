# Agent Factory

**Domain-agnostic multi-agent execution fabric.**

Give it a goal in plain language. A Steward decomposes it into units of work, specialized agents propose changes to a central graph, a Judge approves or rejects against versioned rules, and the system continues until the outcome is achieved — improving itself from every rejection.

## Quick Start

```bash
./scripts/setup.sh
export GEMINI_API_KEY=your-key
./scripts/run-local.sh "Fix product types and weights"
```

Or with Docker:

```bash
./scripts/docker-up.sh
```

## Documentation

Full documentation lives in [`docs/`](docs/README.md):

- [Getting Started](docs/RUNNING.md)
- [Architecture & Concepts](docs/ARCHITECTURE.md)
- [Domain Packs & Data Mapping](docs/DOMAIN_PACKS.md)
- [Operations & Observability](docs/OPERATIONS.md)
- [API & UI](docs/API.md)
- [Diagrams](docs/ER_DIAGRAMS.md)

## What makes it different

- **Agent-first** — no external orchestration frameworks
- **Governed** — nothing reaches the graph without Judge approval
- **Self-improving** — rejections automatically become new rules and specialist agents
- **Domain-agnostic** — catalog is one pack; the same system works for pricing, inventory, compliance, or any Entity-based problem
- **Concurrent** — claimable work queue with parallel workers
- **Observable** — traces, metrics, and immutable audit log

## License

Proprietary. All rights reserved. See [LICENSE](LICENSE).
