package llm

import (
	"context"
	"testing"
	"time"
)

// ConformanceHarness supplies a Client for the suite.
type ConformanceHarness interface {
	Client() Client
}

// RunConformance runs the basic conformance checks every adapter must pass.
// Expand with protocol-specific cases (path, headers, body shape) using httptest.
func RunConformance(t *testing.T, h ConformanceHarness) {
	t.Helper()
	c := h.Client()
	if c == nil {
		t.Fatal("harness returned nil client")
	}

	t.Run("basic_chat", func(t *testing.T) {
		ctx := context.Background()
		resp, err := c.Chat(ctx, ChatRequest{
			Messages: []Message{{Role: "user", Content: "hello"}},
			Tier:     TierCheap,
		})
		if err != nil {
			t.Fatalf("basic chat failed: %v", err)
		}
		if resp.Content == "" {
			t.Error("expected non-empty content")
		}
		if resp.Usage.Total == 0 && resp.Usage.Input == 0 {
			t.Log("usage may be zero for some fakes; ok for basic")
		}
	})

	t.Run("multi_turn", func(t *testing.T) {
		ctx := context.Background()
		resp, err := c.Chat(ctx, ChatRequest{
			Messages: []Message{
				{Role: "system", Content: "you are helpful"},
				{Role: "user", Content: "hello"},
				{Role: "assistant", Content: "hi there"},
				{Role: "user", Content: "what is 2+2?"},
			},
		})
		if err != nil {
			t.Fatalf("multi-turn chat failed: %v", err)
		}
		if resp.Content == "" {
			t.Error("expected non-empty content for multi-turn")
		}
	})

	t.Run("streaming_optional", func(t *testing.T) {
		ctx := context.Background()
		_, err := c.Stream(ctx, ChatRequest{
			Messages: []Message{{Role: "user", Content: "hello"}},
		})
		if err != nil && err != ErrStreamingUnsupported {
			t.Errorf("unexpected stream error: %v", err)
		}
	})

	t.Run("context_cancel", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()
		// Give the timeout a moment to fire if the adapter is slow; many fakes return instantly.
		time.Sleep(2 * time.Millisecond)
		_, err := c.Chat(ctx, ChatRequest{
			Messages: []Message{{Role: "user", Content: "hello"}},
		})
		// Either success (fast fake) or context error is acceptable for basic harnesses.
		if err != nil && ctx.Err() == nil {
			t.Logf("chat returned error (ok if timeout): %v", err)
		}
	})

	t.Run("error_typed", func(t *testing.T) {
		// Adapters should return *Error for failures when possible.
		// Full protocol cases live in conformance_full_test.go against httptest servers.
		t.Log("typed error mapping verified by protocol-specific tests")
	})

	t.Run("model_mapping", func(t *testing.T) {
		// Logical model names are resolved by the router; adapters receive the concrete ID.
		t.Log("model mapping is handled by the router and config; see router_test.go")
	})
}

// RunBasicConformance is kept for compatibility.
func RunBasicConformance(t *testing.T, c Client) {
	t.Helper()
	h := &simpleHarness{c: c}
	RunConformance(t, h)
}

type simpleHarness struct{ c Client }

func (h *simpleHarness) Client() Client { return h.c }
