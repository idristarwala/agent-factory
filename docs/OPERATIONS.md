# Operations

## Health

| Mode | How to check |
|------|--------------|
| Local API | `curl -f http://localhost:8080/health` |
| Docker | `docker compose ps` |
| Kubernetes | `kubectl get pods -l app=agent-factory` |

## Logs

Look for `Round N`, `problems:`, `applied` / `rejected`, and `agents running:`.

## Scaling

| Concern | Approach |
|---------|----------|
| Workload size | Set a run token budget and bound the source page size. |
| LLM cost | Cheap models for workers; frontier only for Judge |
| Rate limits | Retry wrapper + fail-closed Judge |
| State | Configure PostgreSQL for shared definitions and run history |

## Failure modes

| Symptom | Likely cause | Action |
|---------|--------------|--------|
| `gateway error, fail closed` | Model unavailable or rate limit | Check key, model name, quota |
| `react loop exhausted` | Worker never emitted FINAL | Raise MaxSteps or simplify goal |
| `secret not found` | Missing file in SECRETS_DIR | Add the file or set the env fallback |
| Judge always rejects | Rules too strict or missing fields | Review active rules and worker output |
