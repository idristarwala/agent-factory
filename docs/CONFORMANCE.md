# LLM Gateway Conformance

Every adapter must pass `llm.RunConformance`.

## Running the suite

```bash
go test ./internal/llm/ -count=1
go test ./examples/out-of-tree-gateway/ -v
```

## FakeServer

`llm.NewFakeServer` starts an `httptest.Server` that:

- records method, path, headers and body of every request;
- returns a minimal OpenAI-compatible success by default;
- can be given a custom `Handler` or a `BlockFor` duration for cancel tests.

Point your adapter at `fs.URL` and pass it to `RunConformance`.

## Contract

- Basic chat and multi-turn return non-empty content.
- Streaming is optional; return `ErrStreamingUnsupported` if not implemented.
- An already-cancelled context must produce an error (adapters must observe `ctx`).
- 429 maps to typed `RATE_LIMITED` (retryable); 401 maps to `AUTH` (not retryable).
- Logical model names are resolved by the router; adapters receive the concrete ID.

Protocol-specific body shape, tool-call round-trips and SSE delta extraction live in the `conformance_*_test.go` files.

## Out-of-tree adapters

See `examples/out-of-tree-gateway/` for a complete pattern that calls `llm.RunConformance` against an httptest server. Copy that module into your own repository.
