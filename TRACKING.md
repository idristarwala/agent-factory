# Tracking

For current acceptance and outstanding field validation, see [project status](docs/STATUS.md). The entries below record implementation milestones.

## Implementation milestones

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
- Worker default output budget raised to 16000 tokens to reduce truncation
- JSON response_format support in OpenAI body builder for structured output

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
- docker-compose, Helm chart (`deploy/helm/agent-factory`), starter template
- Observe mode, release pipeline, digest notes
- Digest-pinned release CI workflow (.github/workflows/release.yml)
- go.sum committed
- Fuller factory CLI: validate loads packs via pack.Load, checks go.sum and out-of-tree example; readiness checks SECRETS_DIR and DATABASE_URL; test points at contract / conformance packages
- Multi-binary Dockerfile (goal, factory, api)
- Optional Postgres wiring in `cmd/api` via `store.NewFromEnv()` when `DATABASE_URL` is set
- Full nested YAML pack loader using gopkg.in/yaml.v3 (rules as objects with id/text/active, tools alias)
- Published JSON Schema for packs at schemas/pack.schema.json

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
- OpenAI adapter now emits response_format: {type: json_schema, ...} when Schema is set (strict: false for optional fields)
- Gemini adapter now sends responseMimeType=application/json and responseSchema when Schema is set
- Worker sets Schema on every request via BuildResponseSchema
- Continuous-swarm prefers LLMJudge when a real gateway is available

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
- `store.NewFromEnv()` helper used by `cmd/api` (SQLite by default, PostgreSQL via `DATABASE_URL`, explicit process-local memory mode)

## Outstanding work

The [issue tracker](https://github.com/idristarwala/agent-factory/issues) records remaining acceptance criteria, owner decisions and external validation.

## Conformance hardening (2026-11-27)

- `RunConformance` now fails if an adapter ignores an already-cancelled context.
- `RunConformance` concurrency subtest (20 parallel chats) added so every harness is race-checked.
- `factory test` now executes `go test` on llm, store, grouping and the out-of-tree example (exits non-zero on failure).
- Added `llm.NewFakeServer` that records request bodies/headers and supports blocking handlers for cancel and error-injection tests.
- Documented the contract and FakeServer usage in `docs/CONFORMANCE.md`.
- Factory CLI `test` command now points at the real conformance packages.

## Wiring pass (2026-11-27)

- Worker validation failures now call `LogValidationFailure` so raw reply previews appear in logs (#10).
- Continuous runner stops failed worker proposals before judge evaluation or graph writes (#10).
- Collected failure signals are attached to learning gaps via `BuildActionableRule` so structured causes influence rule text (#13/#16).

## Starter template and docs (2026-11-27)

- Expanded `examples/starter` with a concrete catalog pack.yaml so `factory validate` succeeds out of the box.
- Added `docs/GETTING_STARTED.md` covering the 15-minute path from clone to first validation.
- Learning-loop tests already cover replay-gate and same-cause deduplication; further scripted round-improvement cases can be added in internal/swarm/learning_e2e_test.go.

## Nested pack YAML (2026-11-27)

- Pack loader now uses yaml.v3 for full nested support (rules as objects, tools alias).
- Published schemas/pack.schema.json as the pack contract.
- Starter pack.yaml loads cleanly and validates the richer format.

## Accuracy and packaging follow-up (2026-11-27)

- Customer policies ingested via `rules.IngestPolicies` are now activated by default so they reach the worker and judge (addresses the "28 imported / 0 active" observation from the field re-run on #10).
- Added unit test `TestIngestPoliciesActivatesCustomerRules`.
- Example GitLab CI pipeline (`.gitlab-ci.yml`) that runs `factory validate` and `factory test` without live keys, advancing the "any CI/CD" acceptance criterion for #9.
