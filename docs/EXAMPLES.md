# Examples

## Fix a single bad title

```bash
go run ./cmd/catalog-demo
```

## Continuous runner on its built-in fixture

```bash
go run ./cmd/continuous-swarm
```

## ReAct worker with tools

```bash
go run ./cmd/react-demo
```

For a first run against your own source records, use the [run guide](RUNNING.md). For container configuration, see the [packaging guide](../PACKAGING.md).

## Kubernetes Job

```bash
kubectl apply -f deploy/k8s/job.yaml
kubectl logs job/agent-factory-run
```
