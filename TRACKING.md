# Tracking

## Implemented and verified

- [x] Multi-step ReAct workers with tool calling — tested live with Gemini
- [x] Tool layer: interface + registry
- [x] MCP tool adapter + MCP transport interface
- [x] SQL connector + RealBigQueryTool (injectable client)
- [x] Versioned rules engine
- [x] Agent versioning + registry
- [x] Dual-mode store (memory + SQL)
- [x] Domain packs: catalog, pricing, inventory, merchandising
- [x] Steward learning loop
- [x] Judge fails closed
- [x] LLM Client interface, registry, router, config, auth, codecs, retry
- [x] Streaming fallback (CollectStream + ChatAsStream)
- [x] SSE decoder for OpenAI-style streams
- [x] Conformance suite entry point + basic cases
- [x] GitHub Actions CI
- [x] Makefile (test, build, release binaries)
- [x] Full build + tests pass
- [x] WorkQueue with claim/lease (MemoryQueue + SQLQueue stub)
- [x] ContinuousRunner with parallel workers and closed LearningLoop
- [x] Generalized LLMWorker (arbitrary attributes)
- [x] Feedback API endpoint that creates versioned rules
- [x] Graph Snapshot + Apply
- [x] Generic problem detector interface
- [x] Continuous-swarm demo command

## Remaining (requires external systems)

- [ ] Wire real cloud.google.com/go/bigquery client (interface ready)
- [ ] Wire real MCP protocol client (stdio/HTTP) (interface ready)
- [ ] Full provider-specific SSE parsers (OpenAI decoder present)
- [ ] Web UI fully connected to live Decisions (feedback endpoint exists)
- [ ] Signed releases / container images
- [ ] Domain-pack specific Detect implementations
- [ ] Full persistence wiring so ContinuousRunner resumes after restart

Core concurrent self-learning swarm is in place and demonstrated.
