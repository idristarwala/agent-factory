# Agent Factory Documentation

A domain-agnostic agent swarm that takes a goal and works until the outcome is achieved.

## Getting started

| Guide | Description |
|-------|-------------|
| [Overview](OVERVIEW.md) | What it is, core concepts, design principles |
| [Quick start](RUNNING.md) | Run locally, with Docker, or on Kubernetes |
| [Examples](EXAMPLES.md) | Goal runs, continuous swarm, ReAct workers |
| [Configuration](CONFIGURATION.md) | Secrets, environment variables, model tiers |

## Architecture

| Guide | Description |
|-------|-------------|
| [Architecture](ARCHITECTURE.md) | Primitives, data flow, continuous loop |
| [ER Diagrams](ER_DIAGRAMS.md) | Entity relationships and system flows |
| [Schema](SCHEMA.md) | Unified Entity and Decision model |
| [Domain Packs](DOMAIN_PACKS.md) | Catalog, pricing, inventory, merchandising |

## Capabilities

| Guide | Description |
|-------|-------------|
| [Mapping](MAPPING.md) | Source auto-mapping (BigQuery, SQL, API) |
| [Attributes](ATTRIBUTES.md) | Dynamic attribute discovery and fill |
| [Grouping](GROUPING.md) | Variant grouping and policies |
| [Clarification](CLARIFICATION.md) | Steward asks when goals or sources are incomplete |
| [Rules](../internal/rules/) | Versioned rule engine |

## Operations

| Guide | Description |
|-------|-------------|
| [Operations](OPERATIONS.md) | Health, logs, scaling, failure modes |
| [Observability](OBSERVABILITY.md) | Traces, metrics, audit log |
| [Security](SECURITY.md) | Secrets, auth, fail-closed behavior |
| [Troubleshooting](TROUBLESHOOTING.md) | Common issues and fixes |

## Interfaces

| Guide | Description |
|-------|-------------|
| [API](API.md) | HTTP endpoints and extension points |
| [UI](UI.md) | Review interface and chatbot |
| [Contributing](CONTRIBUTING.md) | How to propose changes |

## Related

- [Secrets](../SECRETS.md)
- [Auth](../AUTH.md)
- [Packaging](../PACKAGING.md)
- [License](../LICENSE)
