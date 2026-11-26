# Context Cancellation

All adapter `Chat` and `Stream` methods accept `context.Context`.

- `http.NewRequestWithContext` is used so HTTP calls respect cancellation.
- The ContinuousRunner and workers should pass a context with timeout or cancel.
- `EnsureContext` in `internal/llm/context.go` provides a safe default.

This satisfies the context cancellation requirement from the gateway issues.
