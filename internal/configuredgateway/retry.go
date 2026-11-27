package configuredgateway

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/idristarwala/agent-factory/internal/llm"
)

// Retry only explicit provider refusals and server failures. Once a stream
// opens, its events belong to that request and are never replayed here.
type retryClient struct {
	inner  llm.Client
	policy retryPolicy
}

type retryPolicy struct {
	Attempts  int
	BaseDelay time.Duration
	MaxDelay  time.Duration
}

type retryOverrides struct {
	Attempts  *int           `yaml:"attempts"`
	BaseDelay *time.Duration `yaml:"base_delay"`
	MaxDelay  *time.Duration `yaml:"max_delay"`
}

func (v retryOverrides) resolve() (retryPolicy, error) {
	p := retryPolicy{Attempts: 3, BaseDelay: 250 * time.Millisecond, MaxDelay: 5 * time.Second}
	if v.Attempts != nil {
		p.Attempts = *v.Attempts
	}
	if v.BaseDelay != nil {
		p.BaseDelay = *v.BaseDelay
	}
	if v.MaxDelay != nil {
		p.MaxDelay = *v.MaxDelay
	}
	if p.Attempts < 1 || p.Attempts > 16 || p.BaseDelay <= 0 || p.MaxDelay < p.BaseDelay || p.MaxDelay > 24*time.Hour {
		return p, fmt.Errorf("retry requires 1–16 attempts and 0 < base_delay <= max_delay <= 24h")
	}
	return p, nil
}

func environmentRetry() (retryOverrides, error) {
	var result retryOverrides
	if raw := os.Getenv("LLM_RETRY_ATTEMPTS"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return result, fmt.Errorf("invalid LLM_RETRY_ATTEMPTS")
		}
		result.Attempts = &v
	}
	if raw := os.Getenv("LLM_RETRY_BASE_DELAY"); raw != "" {
		v, err := time.ParseDuration(raw)
		if err != nil {
			return result, fmt.Errorf("invalid LLM_RETRY_BASE_DELAY")
		}
		result.BaseDelay = &v
	}
	if raw := os.Getenv("LLM_RETRY_MAX_DELAY"); raw != "" {
		v, err := time.ParseDuration(raw)
		if err != nil {
			return result, fmt.Errorf("invalid LLM_RETRY_MAX_DELAY")
		}
		result.MaxDelay = &v
	}
	return result, nil
}

func withRetry[T any](ctx context.Context, p retryPolicy, call func() (T, error)) (T, error) {

	for n := 0; ; n++ {
		value, err := call()
		var failure *llm.Error
		if err == nil || n+1 >= p.Attempts || !errors.As(err, &failure) || !failure.Retryable || (failure.Status != http.StatusTooManyRequests && (failure.Status < 500 || failure.Status > 599)) {
			return value, err
		}
		delay := max(p.BaseDelay<<n, failure.RetryAfter)
		if delay > p.MaxDelay {
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
	return withRetry(ctx, r.policy, func() (*llm.ChatResponse, error) { return r.inner.Chat(ctx, req) })
}

func (r retryClient) Stream(ctx context.Context, req llm.ChatRequest) (llm.Stream, error) {
	return withRetry(ctx, r.policy, func() (llm.Stream, error) { return r.inner.Stream(ctx, req) })
}
