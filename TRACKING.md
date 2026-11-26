# Tracking

## Completed

- All high-impact correctness fixes
- OpenAI, Gemini, and Anthropic POST adapters (explicit constructors, no os.Setenv)
- Multi-protocol conformance tests against fake servers (basic chat, rate-limit error typing, path/header checks)
- Extended conformance: auth errors, concurrency under race, SSE streaming reassembly, key-from-file path
- Declarative config, key-from-file, CA/proxy, headers, param mapping
- Usage recording, grants, ContinuousRunner
- Factory CLI, pack loader, Helm, compose, release script
- Family context and schema validation
- Grouping detector is deterministic; ungrouped items are first-class; signature includes type
- Regression tests
- go.sum committed; clean clones build
- Compile conflicts in llm package resolved; core packages pass `go test`

## Remaining

- Fully automated digest-pinned CI release pipeline (current release.sh and CI skeleton still need docker digest pinning + manifest publish)
- Demo `cmd/*` mains need API updates to match current Worker/Judge/Rules signatures
- Full 10-case conformance suite from issue #8 still needs tool-call parsing in adapters, auth refresh, and more protocol body-shape checks
- Durable store contract tests for both memory and SQL backends

The factory is functionally complete for the core loop and gateway adapters. Remaining items are release hygiene, tool-call completeness, and demo polish.
