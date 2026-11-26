# Running

## Local (Go)

```bash
export GEMINI_API_KEY=your-key
export LLM_PROVIDER=gemini
go run ./cmd/goal "Fix catalog titles"
```

## Continuous multi-agent

```bash
go run ./cmd/continuous-demo
```

## Docker

```bash
mkdir -p secrets
echo -n "$GEMINI_API_KEY" > secrets/gemini-api-key
docker compose up --build
```

## Kubernetes

```bash
# Edit the secret values first
kubectl apply -f deploy/k8s/deployment.yaml
kubectl apply -f deploy/k8s/job.yaml
```

## Environment

| Variable | Purpose |
|----------|---------|
| `GEMINI_API_KEY` | Gemini key (or put in secrets dir) |
| `LLM_PROVIDER` | `gemini` or `openai` |
| `SECRETS_DIR` | Override secrets location |
| `LLM_BASE_URL` | Custom OpenAI-compatible endpoint |
