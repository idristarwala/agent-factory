# Troubleshooting

## Build failures

```bash
go mod tidy
go build ./...
```

## LLM errors

| Error | Fix |
|-------|-----|
| 404 model not found | Set `GEMINI_CHEAP_MODEL` / `GEMINI_FRONTIER_MODEL` to a model your key can use |
| 429 rate limit | Wait or switch to a paid tier; Judge will fail closed |
| empty response | Increase MaxTokens or simplify the prompt |

## Docker

```bash
docker compose logs factory
docker compose down -v
```

## Kubernetes

```bash
kubectl describe pod -l app=agent-factory
kubectl logs -l app=agent-factory
```

## Tests

```bash
make test
```
