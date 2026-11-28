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

Issues #1–#8 are closed. The remaining enterprise qualification requires running the native adapter against the original enterprise endpoint. [Conformance suite](CONFORMANCE.md) · [Enterprise field test #11](https://github.com/idristarwala/agent-factory/issues/11)

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

The owner deferred the first production publication. Engine development continues in this repository. The starter repository pins a reviewed engine commit and builds the role images. When publication resumes, a reviewed starter tag publishes the images and release manifest. [Release guide](RELEASE.md) · [Packaging decision in #9](https://github.com/idristarwala/agent-factory/issues/9#issuecomment-7228446031)

## Learning-loop coverage

Added a scripted failure-pattern test (`internal/swarm/learning_improvement_test.go`) that exercises format vs truncation causes end-to-end through Review → ProposeUpdates → Apply and verifies distinct actionable rules. This closes the “optional fuller learning-loop e2e tests” note for packaging and accuracy work. Live field-test re-run remains the external validation step for #10.

## Accuracy wording follow-up (2026-11-27)

The worker gives source-wording guidance when a customer rule requests it (now injected into both system and user prompts, with broader phrase detection and stronger “exact phrasing / keep every existing phrase” language). In the reported 288-item field test at `8ef9ea5`, 281 items were approved with the customer policies active, and proposals stayed within the task’s field scope. [Field report](https://github.com/idristarwala/agent-factory/issues/16#issuecomment-7228494029)

## Accuracy grouping stems (2026-11-27)

SimpleDetector now uses a two-word name stem (first two significant non-stop words). This produces tighter neighbourhoods for the block-then-decide path and reduces false merges on single common words, advancing the grouping F1 criterion on labelled data.

## Grounded judge fact rejection (2026-11-27)

The LLMJudge now receives stronger guidance to reject any proposal that contradicts recorded facts or invents unsupported claims. A unit test with a fake gateway (`TestJudgeRejectsContradictoryFacts`) proves a brand mismatch is rejected while a non-contradicting title update is approved. This closes the corresponding acceptance criterion on #10 without requiring a live model.

Issue #10 still requires three measurements on the field data:

- Original wording kept above 0.90 with facts stated at or above 0.85.
- Zero worker or judge JSON parse failures on a 300-item run.
- Higher grouping F1 than the existing groups on blind human labels, with identical output on repeated runs.

[Field acceptance criteria](https://github.com/idristarwala/agent-factory/issues/10)
