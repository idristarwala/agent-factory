package llm

import (
	"context"
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

	mu        sync.Mutex
	key       string
	expiresAt time.Time
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

func (t *TokenRefresher) ensure(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.key != "" && time.Now().Before(t.expiresAt) {
		return nil
	}
	key, err := t.Refresh(ctx)
	if err != nil {
		return err
	}
	t.key = key
	t.expiresAt = time.Now().Add(t.Interval)
	if setter, ok := t.Inner.(APIKeySetter); ok {
		setter.SetAPIKey(key)
	}
	return nil
}

func (t *TokenRefresher) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if err := t.ensure(ctx); err != nil {
		return nil, err
	}
	return t.Inner.Chat(ctx, req)
}

func (t *TokenRefresher) Stream(ctx context.Context, req ChatRequest) (Stream, error) {
	if err := t.ensure(ctx); err != nil {
		return nil, err
	}
	return t.Inner.Stream(ctx, req)
}
