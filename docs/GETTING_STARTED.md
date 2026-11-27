# Getting started (15 minutes)

Run [the service-desk example](RUNNING.md) from a checkout to evaluate a real record. A model gateway is required for evaluation. To inspect the sample pack and run fixture tests without provider credentials, run these commands from the repository root.

```sh
go run ./cmd/factory validate examples/starter/factory.yaml
go run ./cmd/factory test
```

Pass pack files, pack directories or a `factory.yaml` manifest. Manifest pack paths are relative to that manifest. A missing file, malformed pack or schema violation returns a nonzero exit code. With no arguments the CLI discovers local manifests and pack files, falling back to the repository starter. Validation covers the listed packs. Use the corresponding runtime checks for gateway and tool configuration. [Pack schema](../schemas/pack.schema.json) · [Releases](RELEASE.md)

## CI examples

GitHub Actions and GitLab CI examples that invoke `factory validate` and `factory test` live at the repository root (`.github/workflows/` and `.gitlab-ci.yml`). Both exit non-zero on failure and need no provider credentials.

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
