# Tracking

## Completed

- Graph.Apply merges + tests
- Worker with rules, retry, family context, schema validation
- Judge with full context
- Real CreateWorker
- /api/feedback
- Factory CLI
- Pack loader + validation
- Observe mode
- Declarative gateway config
- Key-from-file, extra headers, CA/proxy
- Parameter mapping
- Usage recording
- Grants helpers
- OpenAIAdapter that POSTs without os.Setenv
- Protocol body builders
- ContinuousRunner + WorkQueue

## Remaining

- Gemini and Anthropic adapters that POST
- Full conformance suite against fake server
- Grants fully enforced in the runner loop
- Digest-pinned image release pipeline
- go.sum committed

All high-impact correctness, config, packaging foundations, and core adapters are in place.
