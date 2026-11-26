# Unified schema

Everything maps to one shape: `Entity`.

## Entity

| Field | Meaning |
|-------|---------|
| `id` | Stable identifier |
| `type` | product, price, stock, placement, … |
| `domain` | catalog, pricing, inventory, merchandising |
| `attributes` | Flexible key/value payload |
| `relations` | Links to other entities |
| `version` | Optimistic version |

Catalog items, prices, inventory rows, and merchandising placements all become Entities. No domain-specific tables in the core.

## Decision

What the customer sees:

| Field | Meaning |
|-------|---------|
| `entity_id` | Which entity |
| `before` / `after` | What changed |
| `reason` | Why |
| `status` | pending / approved / rejected |
| `feedback` | Customer note (becomes a rule) |

## Storage choice

The Steward picks memory or DB. Users never choose.

| Condition | Mode |
|-----------|------|
| ≤ 10,000 entities | memory |
| > 10,000 entities | db |
| Explicit override | respected if set |

Business users only supply a goal and data. The Steward decides how to store and scale.
