# Remaining Incremental Gaps

These items are tracked for completeness but do not block the core factory:

- Full query parameter support is available via `AppendQuery`.
- Image evidence summaries are available; binary ImagePart attachment to adapters is a gateway concern.
- MCP streamable HTTP has an interface and stub; full JSON-RPC session handling is incremental.
- YAML (vs JSON) tool loader can be added with a YAML library if needed.
- Native streaming for Anthropic and Gemini follows the OpenAI SSE pattern.
- PostgresStore is wired into `cmd/api` via `store.NewFromEnv()` when `DATABASE_URL` is set (falls back to memory).

The high-impact correctness, evidence path, retries/cache, config loading, durable store (now used by the API), fuller CLI, Helm chart, multi-binary Dockerfile, and out-of-tree conformance test are complete.
