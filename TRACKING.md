# Tracking

## Implemented and verified

- [x] Multi-step ReAct workers with tool calling (`internal/agents/react_worker.go`) — tested live with Gemini
- [x] Tool layer: interface + registry (`internal/tools/tool.go`)
- [x] MCP tool adapter seam (`internal/tools/mcp.go`)
- [x] SQL + BigQuery connectors (`internal/tools/db.go`)
- [x] Versioned rules engine (`internal/rules/versioned.go`)
- [x] Agent versioning + registry (`internal/agents/version.go`)
- [x] Dual-mode store (memory + SQL) (`internal/store/`)
- [x] Domain packs (`internal/domain/`)
- [x] Steward learning loop (`internal/learning/loop.go`)
- [x] Judge fails closed
- [x] LLM Client interface, registry, router, config, auth, codecs, retry
- [x] Full build + tests pass

## Still open / next

- [ ] Full SSE streaming implementation
- [ ] Real BigQuery client (currently stub)
- [ ] Real MCP client transport (currently CallFn injection)
- [ ] Full conformance suite cases
- [ ] No-code UI / operator surface beyond CLI
- [ ] CI/CD packaging
- [ ] More domain packs (pricing, inventory, merchandising)

## Existing GitHub issues (foundations in place)

1. Provider/gateway abstraction and adapter registry
2. Declarative gateway configuration
3. Auth header hooks and token refresh
4. Logical model names and per-gateway model mapping
5. Request/response shape adapters
6. Streaming responses
7. Typed errors, retries, timeouts and rate limits
8. Conformance test suite
9. Ship as a package
