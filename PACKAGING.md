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

## CI

`.github/workflows/ci.yml` runs `go vet` and `go test -race` on every PR.
`go.sum` is committed so clean clones build.
