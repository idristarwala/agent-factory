package llm

import (
	"context"
)

// EnsureContext returns a non-nil context.
// Call sites should pass ctx from the caller so cancellation propagates.
func EnsureContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
