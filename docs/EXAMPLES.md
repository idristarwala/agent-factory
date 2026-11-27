# Examples

## Fix a single bad title

```bash
go run ./cmd/catalog-demo
```

## Continuous multi-agent on a hard catalog

```bash
go run ./cmd/continuous-demo
```

## ReAct worker with tools

```bash
go run ./cmd/react-demo
```

## Docker Compose

```bash
mkdir -p secrets
echo -n "$GEMINI_API_KEY" > secrets/gemini-api-key
docker compose up --build
```

## Kubernetes Job

```bash
kubectl apply -f deploy/k8s/job.yaml
kubectl logs job/agent-factory-run
```
