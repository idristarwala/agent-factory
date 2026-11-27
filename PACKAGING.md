# Packaging

## Run the container

From the repository root, build the image and create a volume for a disposable example workspace. Export `GEMINI_API_KEY` in the launching shell. These commands import a definition and run its first revision against a mounted fixture.

```sh
docker build -t agent-factory .
state=$(docker volume create)
docker run --rm -v "$state:/state" -v "$PWD/examples/service-desk:/input:ro" -e FACTORY_DB_PATH=/state/factory.sqlite --entrypoint factory agent-factory import factory desk 0 /input/factory.json
docker run --rm -v "$state:/state" -v "$PWD/examples/service-desk:/input:ro" -e FACTORY_DB_PATH=/state/factory.sqlite -e LLM_PROVIDER=gemini -e GEMINI_API_KEY agent-factory -factory desk -revision 1 -records /input/records.json 'Assign the support team from the routing guide'
```

Compose starts the API and PostgreSQL. Supply a model route in the API service environment, either by forwarding its provider credential or by mounting a gateway config and setting `LLM_GATEWAY_CONFIG` in the service. The supplied Compose file mounts the secrets directory. The environment-only Gemini route reads `GEMINI_API_KEY` from the process environment. [Configuration](docs/CONFIGURATION.md)

The Kubernetes Job expects a pre-imported definition, `factory-run` and `factory-records` ConfigMaps, database credentials and a configured model route. Inspect the manifest before deploying it with site-specific resources. [Job manifest](deploy/k8s/job.yaml)

## CI and Release

`.github/workflows/ci.yml` runs `go vet`, `go test -race`, and `go build` on every PR.
`go.sum` is committed so clean clones build.

`.github/workflows/release.yml` builds binaries, pushes the engine, worker and connector images to GHCR, writes a release manifest containing their digests, and creates a GitHub release on `v*` tags.

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
(cd examples/out-of-tree-gateway && go test -v ./...)
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

Requires the `github.com/lib/pq` driver (blank-imported in `cmd/api`). The module checksums are committed.

## Factory CLI

`factory test` now runs the critical package tests when `go.mod` is present:

```bash
factory test

```

Exits non-zero if any package fails, so it is usable in CI.

