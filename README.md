# Agent Factory

A domain-agnostic swarm of agents that works on a goal until the outcome is achieved.

## Quick start

```bash
# 1. Setup
./scripts/setup.sh

# 2. Run a goal
export GEMINI_API_KEY=your-key
./scripts/run-local.sh "Fix titles and product types"

# 3. Or with Docker
./scripts/docker-up.sh
```

## What it does

1. You give a **goal** (plain text).
2. The **Steward** turns it into units of work and spawns specialized agents.
3. Agents propose changes to a **central graph**.
4. The **Judge** approves or rejects against versioned rules.
5. The loop continues until the outcome is clean.

## Layout

| Path | Purpose |
|------|---------|
| `cmd/` | Entry points (goal, continuous-demo, catalog-swarm, …) |
| `internal/` | Core (graph, agents, rules, tools, secrets, swarm) |
| `deploy/k8s/` | Kubernetes manifests |
| `scripts/` | Setup and run helpers |
| `docs/` | Full documentation |

## Documentation

| Doc | Contents |
|-----|----------|
| [Overview](docs/OVERVIEW.md) | Concepts, principles, flow |
| [Architecture](docs/ARCHITECTURE.md) | Primitives, data flow, persistence |
| [Schema](docs/SCHEMA.md) | Unified Entity and Decision |
| [Mapping](docs/MAPPING.md) | Source auto-mapping (BQ, API, SQL) |
| [Attributes](docs/ATTRIBUTES.md) | Dynamic PT attributes |
| [Grouping](docs/GROUPING.md) | Variant states, axes, policies |
| [Clarification](docs/CLARIFICATION.md) | Steward asks when goal or source is missing |
| [Running](docs/RUNNING.md) | Local, Docker, Kubernetes |
| [Configuration](docs/CONFIGURATION.md) | Secrets, env vars, tiers, packs |
| [Operations](docs/OPERATIONS.md) | Health, logs, scaling, failure modes |
| [Examples](docs/EXAMPLES.md) | Goal, continuous, ReAct, compose |
| [API Reference](docs/API.md) | CLI, types, extension points |
| [UI](docs/UI.md) | React review app + chatbot |
| [Security](docs/SECURITY.md) | Secrets, auth, fail-closed |
| [Troubleshooting](docs/TROUBLESHOOTING.md) | Build, LLM, Docker, k8s |
| [Domain packs](docs/DOMAIN_PACKS.md) | Built-in packs |
| [Secrets](SECRETS.md) / [Auth](AUTH.md) | Secret layout and auth methods |
| [Packaging](PACKAGING.md) | Docker, compose, Kubernetes |
| [Contributing](docs/CONTRIBUTING.md) | How to propose changes |

## License

See [LICENSE](LICENSE). All rights reserved. No copying without permission.
