# Getting started (15 minutes)

Run [the service-desk example](RUNNING.md) from a checkout to evaluate a real record. A model gateway is required for evaluation. To inspect the sample pack and run fixture tests without provider credentials, run these commands from the repository root.

```sh
go build -o /tmp/agent-factory-cli ./cmd/factory
(cd examples/starter/packs/catalog && /tmp/agent-factory-cli validate)
go run ./cmd/factory test
```

The validator looks for files in the current directory, `packs` and `examples/starter`. It reports pack parse warnings. Its exit status alone does not certify schema conformance. The published pack schema and release guide describe the interchange format and packaged binaries. [Pack schema](../schemas/pack.schema.json) · [Releases](RELEASE.md)

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
