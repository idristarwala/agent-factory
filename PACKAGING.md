# Packaging

## Local Docker

```bash
mkdir -p secrets
echo -n "$GEMINI_API_KEY" > secrets/gemini-api-key
docker build -t agent-factory .
docker run --rm -v $(pwd)/secrets:/etc/secrets:ro agent-factory "Fix catalog titles"
```

## Docker Compose

```bash
mkdir -p secrets
echo -n "$GEMINI_API_KEY" > secrets/gemini-api-key
docker compose up --build
```

Brings up the factory + Postgres. Secrets mounted at `/etc/secrets`.

## Kubernetes

```bash
kubectl apply -f deploy/k8s/deployment.yaml
kubectl apply -f deploy/k8s/job.yaml
```

Replace the placeholder secret values before applying.

## Secrets

All environments use the same contract: `SECRETS_DIR` (default `/etc/secrets`).
See `SECRETS.md` and `AUTH.md`.

## CI and Release

`.github/workflows/ci.yml` runs `go vet`, `go test -race`, and `go build` on every PR.
`go.sum` is committed so clean clones build.

`.github/workflows/release.yml` builds binaries, pushes a digest-pinned image to GHCR, writes a release manifest containing the digest, and creates a GitHub release on `v*` tags.

See `docs/RELEASE.md`.

## Durable Postgres store

```go
import (
    "github.com/idristarwala/agent-factory/internal/store"
    _ "github.com/lib/pq" // or your preferred driver
)

s, err := store.NewPostgresStore("postgres", os.Getenv("DATABASE_URL"))
// use s as the Store for Graph / units of work
```

Shared contract tests live in `internal/store/contract_test.go` and run against Memory and SQLite. Point the same tests at a live Postgres DSN when available.

## Out-of-tree gateway conformance

```bash
go test ./examples/out-of-tree-gateway/ -v
```

The example adapter reuses the in-repo OpenAI adapter and passes `llm.RunConformance`.

## Helm

A starter Helm chart lives at `deploy/helm/agent-factory`.

```bash
helm install agent-factory ./deploy/helm/agent-factory
# or with a Postgres DSN:
helm install agent-factory ./deploy/helm/agent-factory --set env.DATABASE_URL=postgres://...
```

The chart deploys the `api` binary. Set `DATABASE_URL` to enable the durable Postgres store.

## API with durable store

```bash
export DATABASE_URL=postgres://factory:factory@localhost:5432/factory?sslmode=disable
go run ./cmd/api
# GET /health reports {"status":"ok","store":"postgres"}
```

Requires the `github.com/lib/pq` driver (blank-imported in `cmd/api`). Run `go mod tidy` after clone if go.sum is incomplete.
