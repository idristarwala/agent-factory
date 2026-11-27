# Tracking

## Completed

- All high-impact correctness fixes
- OpenAI, Gemini, and Anthropic POST adapters (context-aware HTTP)
- Multi-protocol conformance tests against fake servers
- Declarative config, key-from-file, CA/proxy, headers, param mapping
- Usage recording, grants, ContinuousRunner
- Factory CLI, pack loader, Helm, compose, release pipeline
- Family context and schema validation
- Context cancellation support documented and helpers added
- Regression tests
- Token param mapping (max_completion_tokens for reasoning models) wired into OpenAI adapter
- Reasoning budget support for Gemini via metadata
- Starter factory.yaml template for packaging
- Expanded RunConformance with multi-turn and context-cancel cases

## Remaining

- Full digest-pinned release automation in CI (docker build + inspect)
- Tool-call parsing in all adapters
- Auth token refresh / rotation reload
- Native streaming (currently reassembly helpers + one-shot fallback)
- Labelled grouping scorer and full block-then-decide LLM membership

The Agent Factory continues to advance toward production packaging and full gateway conformance.
