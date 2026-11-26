# Observability

Structured JSON logs with end-to-end trace IDs.

## Trace

Every goal gets a `trace_id`. It flows:

```
goal → steward → worker → judge → graph
```

## Span

```json
{"ts":"...","level":"info","event":"span","trace":"trace-001","span":"...","name":"worker.execute","dur_ms":120,"status":"ok","attrs":{"role":"title-fixer","entity":"sku-1"}}
```

## Usage

```go
ctx := obs.WithTrace(ctx, traceID)
log.Info(ctx, "goal received", map[string]any{"goal": goal})
sp := log.Span(ctx, "worker.execute")
defer sp.End(err)
```

## Demo

```bash
go run ./cmd/trace-demo
```

## Next

- OpenTelemetry export
- Metrics (decisions/hour, reject rate, LLM latency)
- Trace UI
