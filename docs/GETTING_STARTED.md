# Getting started (15 minutes)

1. Clone this repo or copy `examples/starter` into your own repository.
2. `cp examples/starter/factory.yaml .`
3. `mkdir -p packs/catalog && cp examples/starter/packs/catalog/pack.yaml packs/catalog/`
4. `go run ./cmd/factory validate`
   - Loads packs with full yaml.v3 support for nested rules.
   - Packs conform to the published schema at `schemas/pack.schema.json`.
5. `go run ./cmd/factory test`   # runs package tests, no API keys needed
6. Optional: `docker compose up` for the full stack (set secrets first; see SECRETS.md).

The engine is a versioned dependency. Pin the image digest in factory.yaml and deploy with the Helm chart in deploy/helm/agent-factory.

## Pack format

See `schemas/pack.schema.json` for the full contract. A minimal pack looks like:

```yaml
name: catalog
description: Improve product titles while preserving seller wording.
node_types: [product]
worker_roles: [title_fixer]
rules:
  - id: keep_original_wording
    text: Keep the original seller wording as much as possible.
    active: true
tools:
  - catalog.get_item
```
