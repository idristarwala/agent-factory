# Tracking

## Completed

- Graph.Apply merges + regression tests
- Worker with rules, retry, family note
- Judge with full context
- Real CreateWorker after SetRuntime
- /api/feedback
- Factory CLI skeleton
- Pack loader and validation
- Observe mode skeleton
- Declarative gateway config
- Key-from-file
- Extra headers
- Custom CA / proxy HTTP client
- Parameter mapping helpers
- Usage Recorder + RecordingClient
- Grants check and enforcer
- ContinuousRunner and WorkQueue
- Protocol adapters no longer use os.Setenv (explicit constructors)
- Expanded basic conformance suite (RunConformance)
- go.sum committed

## Remaining incremental

- Wire RecordingClient into production call paths
- Enforce grants inside the ContinuousRunner loop
- Full protocol adapters (Anthropic native, richer Gemini reasoning budget)
- Complete conformance suite cases against httptest fake gateways (path/headers/body)
- Release pipeline and full Helm templates
- Family/sibling batching in worker prompts
- Schema-validated structured output (json schema + retry)
- Durable store implementations

High-impact correctness and major config/packaging seams are closed. Remaining work is wiring, richer adapters, and completeness.
