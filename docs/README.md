# Agent Factory Documentation

Welcome to the official documentation for Agent Factory — a domain-agnostic, self-improving multi-agent execution fabric.

## Getting Started

| Guide | Description |
|-------|-------------|
| [Overview](OVERVIEW.md) | What the factory is, who it is for, and core concepts |
| [Quick Start](RUNNING.md) | Install, run locally, Docker, and Kubernetes |
| [Examples](EXAMPLES.md) | Common goals and usage patterns |

## Core Concepts

| Topic | Description |
|-------|-------------|
| [Architecture](ARCHITECTURE.md) | Primitives, data flow, and continuous loop |
| [Schema](SCHEMA.md) | Unified Entity model and Decisions |
| [Rules](ARCHITECTURE.md#rules) | Versioned, immutable rule engine |
| [Learning Loop](ARCHITECTURE.md#learning) | How the Steward improves the system |

## Domain & Data

| Topic | Description |
|-------|-------------|
| [Domain Packs](DOMAIN_PACKS.md) | Catalog, pricing, inventory, merchandising |
| [Source Mapping](MAPPING.md) | Auto-map from BigQuery, API, or SQL |
| [Dynamic Attributes](ATTRIBUTES.md) | Hundreds of product-type-specific attributes |
| [Grouping & Variants](GROUPING.md) | Partial groups, ungrouped items, multi-axis varianting |

## Operations

| Topic | Description |
|-------|-------------|
| [Configuration](CONFIGURATION.md) | Secrets, model tiers, environment variables |
| [Observability](OBSERVABILITY.md) | Traces, metrics, audit log |
| [Operations](OPERATIONS.md) | Scaling, failure modes, health |
| [Security](SECURITY.md) | Auth, secrets, fail-closed Judge |
| [Troubleshooting](TROUBLESHOOTING.md) | Common issues and fixes |

## Interfaces

| Topic | Description |
|-------|-------------|
| [API Reference](API.md) | CLI, HTTP endpoints, extension points |
| [UI](UI.md) | Customer review interface and chatbot |
| [Clarification](CLARIFICATION.md) | How the Steward asks for missing information |

## Diagrams

| Topic | Description |
|-------|-------------|
| [ER & Architecture Diagrams](ER_DIAGRAMS.md) | Entity-relationship and flow diagrams |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

---

All diagrams render natively on GitHub. For production use, also see the root [README](../README.md).
