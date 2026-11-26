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

## Remaining incremental

- Wire RecordingClient into production call paths
- Enforce grants inside the ContinuousRunner loop
- Full protocol adapters without os.Setenv
- Complete conformance suite cases
- Release pipeline and full Helm templates
- Family/sibling batching
- Schema-validated structured output
- go.sum committed

High-impact correctness and major config/packaging seams are closed. Remaining work is wiring and completeness.
