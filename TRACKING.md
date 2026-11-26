# Tracking

## Completed

- All high-impact correctness fixes
- OpenAI, Gemini, and Anthropic POST adapters (no os.Setenv)
- Declarative config, key-from-file, CA/proxy, headers, param mapping
- Usage recording, grants helpers, ContinuousRunner
- Factory CLI, pack loader, Helm, compose, release skeleton
- Family context and schema validation in worker
- Regression tests for core paths

## Remaining

- Full multi-protocol conformance suite against fake servers
- Digest-pinned container image release pipeline
- go.sum committed for clean builds

The core factory, gateway abstraction, and packaging foundations are complete.
