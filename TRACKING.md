# Tracking

## Completed

### Correctness (#10)
- Graph.Apply merges + regression tests
- Worker with rules, retry, family context, schema validation
- Judge with full context
- Real CreateWorker after SetRuntime
- /api/feedback
- Deterministic grouping with ungrouped items first-class
- Labelled grouping scorer (`grouping.ScorePairMetrics`)
- Block-then-decide LLM membership helper (`grouping.BlockThenDecide` + `askMembership`)
- Context recipe evidence injected into worker prompt (UnitOfWork.Evidence)
- Labelled fixture + scoring demo (`cmd/variant-demo`, `examples/labelled_grouping.json`)

### Enterprise gateway (#11)
- Declarative config loader
- Key-from-file
- Custom CA / proxy HTTP client
- Parameter mapping (max_tokens vs max_completion_tokens, thinkingBudget)
- Usage Recorder + RecordingClient
- OpenAI, Gemini, and Anthropic POST adapters
- Full multi-protocol conformance tests
- Rate limiter in router
- Token refresher for key rotation (now injects key into adapters that implement SetAPIKey)
- Tool call parsing (OpenAI, Anthropic, Gemini)
- Basic native SSE streaming for OpenAI, Gemini (streamGenerateContent), and Anthropic
- Refined multi-protocol SSE delta extraction (OpenAI, Anthropic content_block_delta, Gemini candidates/parts)

### Packaging (#9)
- Factory CLI, pack loader, validation
- Tool model with side-effect classes
- Grants check and enforcer in ContinuousRunner
- docker-compose, Helm chart, starter template
- Observe mode, release pipeline, digest notes
- Digest-pinned release CI workflow (.github/workflows/release.yml)
- go.sum committed
- Fuller factory CLI: validate loads packs via pack.Load, checks go.sum and out-of-tree example; readiness checks SECRETS_DIR and DATABASE_URL; test points at contract / conformance packages

### LLM foundation (#1–#8)
- Client types, adapter registry, typed errors
- Router with tier policy and rate limiter
- Retry with exponential backoff
- Conformance harness + multi-protocol tests
- Protocol builders and POST adapters
- Tool call parsing + round-trip tests (OpenAI, Anthropic, Gemini)
- Token refresh support (`TokenRefresher` decorator with key injection, exercised in tests)
- Basic native SSE streaming for OpenAI-compatible, Gemini, and Anthropic endpoints
- Multi-protocol SSE text delta extraction used by all stream implementations
- Concrete out-of-tree gateway example that runs `llm.RunConformance` against an httptest server (`examples/out-of-tree-gateway/adapter_test.go`)

### HTTP tools (#12)
- HTTPTool with path templating, auth, retries, cache, real key loading
- Tool config loader
- Grants in ContinuousRunner
- MCP streamable-HTTP transport
- JSONPath response mapping
- Image URL fetcher and multimodal body builders
- Context recipe fully wired: evidence gathered, attached to mutation, and injected into worker prompt

### Persistence
- MemoryStore and SQLite SQLStore
- Durable PostgresStore (`store.NewPostgresStore` / `NewPostgresStoreFromDB`) with $n placeholders, ON CONFLICT upserts, and shared contract tests against Memory + SQLite

## Remaining (minor)

- Optional: wire PostgresStore into the API/cmd entrypoints when DATABASE_URL is set (currently available as a library constructor)
- Optional: full YAML pack loader (JSON works today)

All high-impact correctness, config seams, POST adapters, packaging, HTTP tools, tool-call round-trips across protocols, auth refresh with key injection, refined streaming parsers, release automation, context-recipe prompt injection, labelled grouping demo, durable Postgres store, fuller CLI, and concrete out-of-tree conformance test are in place.
