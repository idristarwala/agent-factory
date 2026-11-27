# Context Cancellation

All adapter `Chat` and `Stream` methods accept `context.Context`.

- `http.NewRequestWithContext` is used so HTTP calls respect cancellation.
- The ContinuousRunner and workers should pass a context with timeout or cancel.
- Pass the caller’s context to preserve its deadline and cancellation signal.

This satisfies the context cancellation requirement from the gateway issues.
