package llm

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"
)

// TokenRefresher wraps a Client and refreshes an API key before calls when needed.
// Used for key rotation / auth refresh hooks (issues #3 / #8).
// When the inner client is an adapter that exposes SetAPIKey, the new key is injected.
type TokenRefresher struct {
	Inner    Client
	Refresh  func(ctx context.Context) (string, error)
	Interval time.Duration

	mu         sync.RWMutex
	generation uint64
	expiresAt  time.Time
}

// APIKeySetter is implemented by adapters that can accept a rotated key.
type APIKeySetter interface {
	SetAPIKey(key string)
}

// NewTokenRefresher creates a refresher that calls Refresh when the key is stale.
func NewTokenRefresher(inner Client, refresh func(ctx context.Context) (string, error), interval time.Duration) *TokenRefresher {
	return &TokenRefresher{
		Inner:    inner,
		Refresh:  refresh,
		Interval: interval,
	}
}

// acquire holds a read lock while the adapter builds and sends the request.
// A rotation waits for existing calls before mutating a shared adapter key.
func (t *TokenRefresher) acquire(ctx context.Context, rejected uint64) (uint64, error) {
	t.mu.RLock()
	if t.generation != rejected && time.Now().Before(t.expiresAt) {
		return t.generation, nil
	}
	t.mu.RUnlock()
	t.mu.Lock()
	if t.generation == rejected || !time.Now().Before(t.expiresAt) {
		key, err := t.Refresh(ctx)
		if err != nil {
			t.mu.Unlock()
			return 0, err
		}
		if setter, ok := t.Inner.(APIKeySetter); ok {
			setter.SetAPIKey(key)
		}
		t.generation++
		t.expiresAt = time.Now().Add(t.Interval)
	}
	t.mu.Unlock()
	t.mu.RLock()
	return t.generation, nil
}

func withRefresh[T any](ctx context.Context, t *TokenRefresher, call func() (T, error)) (T, error) {
	var zero T
	var rejected uint64
	for attempt := 0; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return zero, err
		}
		generation, err := t.acquire(ctx, rejected)
		if err != nil {
			return zero, err
		}
		value, err := func() (T, error) {
			defer t.mu.RUnlock()
			return call()
		}()
		var failure *Error
		if attempt != 0 || !errors.As(err, &failure) || failure.Status != http.StatusUnauthorized {
			return value, err
		}
		rejected = generation
	}
}

func (t *TokenRefresher) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	return withRefresh(ctx, t, func() (*ChatResponse, error) { return t.Inner.Chat(ctx, req) })
}

func (t *TokenRefresher) Stream(ctx context.Context, req ChatRequest) (Stream, error) {
	return withRefresh(ctx, t, func() (Stream, error) { return t.Inner.Stream(ctx, req) })
}
