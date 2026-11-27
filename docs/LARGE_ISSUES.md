# Large Design Issues Status

## #1–#8 Pluggable LLM Gateways

Foundation complete in `internal/llm`:
- Client / ChatRequest / ChatResponse / Message / ToolSpec / ToolCall / Usage
- Adapter registry with Register/New
- Typed Error with a Retryable flag
- Router with tier policy and rate limiter
- Retry with exponential backoff
- Protocol builders and POST adapters (OpenAI, Gemini, Anthropic)
- Tool call parsing + round-trip tests
- Token refresh support with key injection
- Basic native SSE streaming with multi-protocol delta extraction
- `RunConformance` with basic chat, multi-turn, streaming optional, context cancel, and 50-way concurrency
- Concrete out-of-tree gateway example that runs the suite against an httptest server
- Schema / response_format support on OpenAI and Gemini adapters; worker sets Schema

Remaining for full close:
- Additional protocol-specific error and auth cases inside the shared suite (many already exist as adapter tests)
- Optional further streaming reassembly assertions for every protocol

## #9 Packaging as Installable Product

Present:
- `cmd/factory` CLI with import / validate / test / readiness / up / publish
- `internal/pack.Load` (JSON + full yaml.v3 for nested packs)
- Richer Pack type with RuleSpec (id/text/active) and tools alias
- Published JSON Schema at schemas/pack.schema.json
- Tool model with side-effect classes and grants check in ContinuousRunner
- Observe mode
- docker-compose, Helm chart, starter template, digest-pinned release workflow
- Multi-binary Dockerfile and optional Postgres wiring

Remaining (updated 2026-11-27):
- Live field-test re-run to confirm #10 accuracy items

The core factory loop is functional and the packaging surface is usable for local and CI/CD use. Packs can now express nested rules that the starter template already uses.

## Learning-loop coverage

Added a scripted failure-pattern test (`internal/swarm/learning_improvement_test.go`) that exercises format vs truncation causes end-to-end through Review → ProposeUpdates → Apply and verifies distinct actionable rules. This closes the “optional fuller learning-loop e2e tests” note for packaging and accuracy work. Live field-test re-run remains the external validation step for #10.

## Accuracy wording follow-up (2026-11-27)

Worker now detects "keep original wording" style rules and appends explicit preservation guidance to the system prompt. Covered by unit test. Next live re-run can measure whether original-wording-kept crosses 0.90.
