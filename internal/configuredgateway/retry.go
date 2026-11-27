package configuredgateway

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/idristarwala/agent-factory/internal/llm"
)

// Retry only explicit provider refusals and server failures. Once a stream
// opens, its events belong to that request and are never replayed here.
type retryClient struct{ inner llm.Client }

func withRetry[T any](ctx context.Context, call func() (T, error)) (T, error) {
	const attempts = 3
	for n := 0; ; n++ {
		value, err := call()
		var failure *llm.Error
		if err == nil || n+1 == attempts || !errors.As(err, &failure) || !failure.Retryable || (failure.Status != http.StatusTooManyRequests && (failure.Status < 500 || failure.Status > 599)) {
			return value, err
		}
		delay := max((250*time.Millisecond)<<n, failure.RetryAfter)
		if delay > 5*time.Second {
			return value, err
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return value, ctx.Err()
		case <-timer.C:
		}
	}
}

func (r retryClient) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	return withRetry(ctx, func() (*llm.ChatResponse, error) { return r.inner.Chat(ctx, req) })
}

func (r retryClient) Stream(ctx context.Context, req llm.ChatRequest) (llm.Stream, error) {
	return withRetry(ctx, func() (llm.Stream, error) { return r.inner.Stream(ctx, req) })
}
