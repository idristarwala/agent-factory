# Getting started (15 minutes)

1. Clone this repo or copy `examples/starter` into your own repository.
2. `cp examples/starter/factory.yaml .`
3. `mkdir -p packs/catalog && cp examples/starter/packs/catalog/pack.yaml packs/catalog/`
4. `go run ./cmd/factory validate`
5. `go run ./cmd/factory test`   # runs package tests, no API keys needed
6. Optional: `docker compose up` for the full stack (set secrets first; see SECRETS.md).

The engine is a versioned dependency. Pin the image digest in factory.yaml and deploy with the Helm chart in deploy/helm/agent-factory.
