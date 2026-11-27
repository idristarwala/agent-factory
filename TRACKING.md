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

### Packaging (#9)
- Factory CLI, pack loader, validation
- Tool model with side-effect classes
- Grants check and enforcer in ContinuousRunner
- docker-compose, Helm chart, starter template
- Observe mode, release pipeline, digest notes
- Digest-pinned release CI workflow (.github/workflows/release.yml)
- go.sum committed

### LLM foundation (#1–#8)
- Client types, adapter registry, typed errors
- Router with tier policy and rate limiter
- Retry with exponential backoff
- Conformance harness + multi-protocol tests
- Protocol builders and POST adapters
- Tool call parsing + round-trip tests (OpenAI, Anthropic, Gemini)
- Token refresh support (`TokenRefresher` decorator with key injection, exercised in tests)
- Basic native SSE streaming for OpenAI-compatible, Gemini, and Anthropic endpoints

### HTTP tools (#12)
- HTTPTool with path templating, auth, retries, cache, real key loading
- Tool config loader
- Grants in ContinuousRunner
- MCP streamable-HTTP transport
- JSONPath response mapping
- Image URL fetcher and multimodal body builders

## Remaining (minor)

- Context recipe fully wired into the runner loop (types exist; further integration pending)
- Full block-then-decide LLM membership for grouping
- Streaming delta parsing is OpenAI-shaped; Gemini/Anthropic event shapes can be refined further

All high-impact correctness, config seams, POST adapters, packaging, HTTP tools, tool-call round-trips across protocols, auth refresh with key injection, and release automation are in place.
