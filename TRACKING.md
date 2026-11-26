# Tracking

## Implemented and verified

- [x] Multi-step ReAct workers with tool calling — tested live with Gemini
- [x] Tool layer: interface + registry
- [x] MCP tool adapter seam
- [x] SQL + BigQuery connectors (BigQuery is a stub ready for real client)
- [x] Versioned rules engine
- [x] Agent versioning + registry
- [x] Dual-mode store (memory + SQL)
- [x] Domain packs: catalog, pricing, inventory, merchandising
- [x] Steward learning loop
- [x] Judge fails closed
- [x] LLM Client interface, registry, router, config, auth, codecs, retry
- [x] Streaming fallback (CollectStream + ChatAsStream)
- [x] Conformance suite entry point + basic cases
- [x] GitHub Actions CI (vet, test -race, build)
- [x] Full build + tests pass

## Remaining (production hardening)

- [ ] Real BigQuery client (replace stub with cloud.google.com/go/bigquery)
- [ ] Real MCP client transport (replace CallFn injection with actual MCP protocol)
- [ ] Full SSE decoding for OpenAI / Anthropic / Gemini
- [ ] No-code web UI (CLI exists)
- [ ] Production packaging / release process

## Existing GitHub issues (foundations in place)

1–9 still open; core abstractions delivered.
