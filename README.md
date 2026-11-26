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

## Docs

- [Architecture](docs/ARCHITECTURE.md)
- [Running](docs/RUNNING.md)
- [Secrets & Auth](SECRETS.md) / [AUTH.md](AUTH.md)
- [Packaging](PACKAGING.md)
- [Domain packs](docs/DOMAIN_PACKS.md)
- [Contributing](docs/CONTRIBUTING.md)

## License

See [LICENSE](LICENSE). All rights reserved. No copying without permission.
