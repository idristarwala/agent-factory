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

## Remaining (requires external systems)

- [ ] Wire real cloud.google.com/go/bigquery client (interface ready)
- [ ] Wire real MCP protocol client (stdio/HTTP) (interface ready)
- [ ] Full provider-specific SSE parsers (OpenAI decoder present)
- [ ] Web UI (CLI + release binaries exist)
- [ ] Signed releases / container images

All core seams are in place. Remaining items need external credentials or a UI framework.
