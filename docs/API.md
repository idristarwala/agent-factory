# API Reference

The factory is a Go library plus CLI entry points. There is no HTTP API yet.

## CLI

| Command | Purpose |
|---------|---------|
| `cmd/goal` | Run a goal string |
| `cmd/continuous-demo` | Continuous multi-agent loop |
| `cmd/catalog-swarm` | Catalog correction demo |
| `cmd/react-demo` | ReAct worker with tools |
| `cmd/hard-catalog` | Hard catalog test |

## Extension points

| Want to add | Where |
|-------------|-------|
| New domain | `internal/domain` |
| New tool | Implement `tools.Tool`, register it |
| New LLM provider | Implement adapter, register in `internal/llm` |
| New auth method | Use `secrets.Loader` and document in `AUTH.md` |
