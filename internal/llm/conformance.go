package llm

import (
	"context"
	"sync"
	"testing"
)

// ConformanceHarness supplies a Client for the suite.
type ConformanceHarness interface {
	Client() Client
}

// RunConformance runs the basic conformance checks every adapter must pass.
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
		stream, err := c.Stream(ctx, ChatRequest{
			Messages: []Message{{Role: "user", Content: "hello"}},
		})
		if stream != nil {
			defer stream.Close()
		}
		if err != nil && err != ErrStreamingUnsupported {
			t.Errorf("unexpected stream error: %v", err)
		}
	})

	t.Run("context_cancel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // already cancelled
		_, err := c.Chat(ctx, ChatRequest{
			Messages: []Message{{Role: "user", Content: "hello"}},
		})
		// A correct adapter must observe the cancelled context.
		// Fast fakes that ignore ctx and return success fail this check.
		if err == nil {
			t.Error("expected error on already-cancelled context; adapter must check ctx.Err()")
		}
	})

	t.Run("concurrency", func(t *testing.T) {
		const n = 50
		var wg sync.WaitGroup
		errs := make(chan error, n)
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx := context.Background()
				_, err := c.Chat(ctx, ChatRequest{
					Messages: []Message{{Role: "user", Content: "parallel"}},
					Tier:     TierCheap,
				})
				if err != nil {
					errs <- err
				}
			}()
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			t.Errorf("parallel chat failed: %v", err)
		}
	})

}
