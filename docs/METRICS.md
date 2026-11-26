# Metrics and audit

## Metrics

| Metric | Meaning |
|--------|---------|
| decisions | Total Judge decisions |
| approved / rejected | Counts |
| reject_rate | rejected / decisions |
| llm_calls / llm_errors | Gateway usage |
| llm_avg_ms | Average latency |

## Audit log

Append-only record of every approved or rejected mutation. Query by entity.

## Demo

```bash
go run ./cmd/obs-demo
```
