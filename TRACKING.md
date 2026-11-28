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
- Digest-pinned release CI in the separate [starter repository](https://github.com/idristarwala/agent-factory-starter)
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
- `RunConformance` concurrency subtest (50 parallel chats) added so every harness is race-checked.
- `factory test` now executes `go test` on llm, store, grouping and the out-of-tree example (exits non-zero on failure).
- The external adapter uses the shared protocol harness documented in `docs/CONFORMANCE.md`.
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
- The [standalone starter GitLab pipeline](https://github.com/idristarwala/agent-factory-starter/blob/main/.gitlab-ci.yml) validates consumer packs and retains the tested commit receipt. The root GitLab pipeline exercises the engine checkout.

## Accuracy grouping refinement (2026-11-27)

- SimpleDetector signature now uses a stop-word-aware name stem (skips "the", "a", etc.) so titles like "The Super Widget" and "Super Widget" share a bucket.
- Added TestUngroupedItemsPlaced to prove ungrouped items with a matching signature are proposed into the majority group (#10).
- Deterministic majority and stable proposal ordering preserved.

## Learning-loop scripted improvement (2026-11-27)

- Added `TestScriptedFailurePatternsProduceDistinctActionableRules` in `internal/swarm/learning_improvement_test.go`.
- The test feeds format then truncation failure signals, asserts rules are proposed and applied, and confirms distinct actionable guidance for each cause.
- Advances the remaining fuller learning-loop e2e coverage noted for packaging (#9) and accuracy (#10).

## Wording-preservation emphasis (2026-11-27)

- Worker system prompt is strengthened when any active rule mentions preserving original wording / not rewriting (addresses the 0.89 vs 0.90 original-wording-kept gap from the #10 field re-run).
- Added `TestPreserveOriginalWordingRuleStrengthensPrompt` which verifies the system prompt and that the rule text reaches the user prompt.
- Advances the customer-rule influence criterion without requiring a live gateway.

## Stronger wording-preservation prompt (2026-11-27)

- Expanded detection of preserve-wording rules (added "preserve original", "keep original", "seller wording").
- When triggered, the worker now injects a strong "IMPORTANT" note into the user prompt in addition to the strengthened system prompt.
- Updated TestPreserveOriginalWordingRuleStrengthensPrompt to assert the new user-prompt note.
- Further advances the original-wording-kept criterion for the #10 field re-run without requiring a live gateway.

## Two-word grouping stems (2026-11-27)

- SimpleDetector `nameStem` now keeps the first two significant non-stop words (e.g. "super widget" instead of only "super"). This produces tighter, more accurate neighbourhoods for the block-then-decide path and reduces false merges on single common words.
- Added `TestTwoWordStem` and updated the signature assertion. Determinism and ungrouped placement tests continue to pass.
- Advances the grouping F1 criterion on the #10 labelled set without a live model.

## Even stronger wording-preservation (2026-11-27)

- Detection now also matches "do not change the wording".
- System prompt insists on exact phrasing, sentence structure and vocabulary; user prompt says "Keep every existing phrase exactly as written".
- Updated `TestPreserveOriginalWordingRuleStrengthensPrompt` to assert the new language.
- Further advances the original-wording-kept > 0.90 gap reported in the field re-run.

## Grounded judge fact rejection (2026-11-27)

- LLMJudge system and user prompts now explicitly require rejection of any value that contradicts the original entity's recorded facts or invents unsupported claims.
- Added `TestJudgeRejectsContradictoryFacts` (and a prompt-inclusion check) using a fake gateway that detects a brand mismatch in the original vs proposed JSON.
- Covers the #10 acceptance criterion "The Judge rejects a proposal that states a value contradicting the item's facts (unit test with a fake gateway)".
- Live field re-run remains the external step for the original-wording-kept, zero-parse-failure and grouping-F1 measurements.
