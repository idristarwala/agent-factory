# LLM Gateway Conformance

Every adapter must pass `llmtest.RunConformance`.

## Running the suite

```bash
go test ./internal/llm/ -count=1
(cd examples/out-of-tree-gateway && go test -race ./...)
```

## Protocol harness

Construct a `llmtest.NewProtocolHarness` with the protocol and an adapter factory that accepts a server URL. Pass the harness to `llmtest.RunConformance`. The harness runs a local HTTP server, exercises the adapter on the wire, and releases the server when the test finishes. [External adapter test](../examples/out-of-tree-gateway/adapter_test.go)

## Contract

- Basic chat and multi-turn return non-empty content.
- Streaming is optional; return `ErrStreamingUnsupported` if not implemented.
- An already-cancelled context must produce an error (adapters must observe `ctx`).
- 429 maps to typed `RATE_LIMITED` (retryable); 401 maps to `AUTH` (not retryable).
- Logical model names are resolved by the router; adapters receive the concrete ID.

Protocol-specific body shape, tool-call round-trips and SSE delta extraction live in the `conformance_*_test.go` files.

## Out-of-tree adapters

See `examples/out-of-tree-gateway/` for a complete pattern that calls `llmtest.RunConformance` against an httptest server. Copy that module into your own repository.

## Shared suite cases (RunConformance)

Every adapter that calls `llmtest.RunConformance` now exercises:

1. Basic chat (non-empty content)
2. Multi-turn conversation
3. Streaming optional (clean fallback allowed)
4. Already-cancelled context must error
5. Concurrent chats (50 parallel, race-safe)
6. Documented typed error contract (full mapping in protocol tests)

Protocol-specific tool-call round-trips, JSON-schema structured output, SSE reassembly, 429/401 mapping, and auth refresh live in `conformance_*_test.go` and are run by CI.
