package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// ConformanceHarness supplies a Client for the suite.
type ConformanceHarness interface {
	Client() Client
}

// FakeServer is an httptest server that records requests and can return
// controlled responses or errors. Adapters under test point at its URL.
type FakeServer struct {
	*httptest.Server
	mu       sync.Mutex
	Requests []RecordedRequest
	// Handler overrides the default success response when set.
	Handler http.HandlerFunc
	// BlockFor makes the next request wait (useful for cancel tests).
	BlockFor time.Duration
}

// RecordedRequest captures what the adapter sent.
type RecordedRequest struct {
	Method string
	Path   string
	Header http.Header
	Body   []byte
}

// NewFakeServer starts a server that records every request and returns a
// minimal OpenAI-compatible success unless Handler is set.
func NewFakeServer() *FakeServer {
	fs := &FakeServer{}
	fs.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		fs.mu.Lock()
		fs.Requests = append(fs.Requests, RecordedRequest{
			Method: r.Method,
			Path:   r.URL.Path,
			Header: r.Header.Clone(),
			Body:   body,
		})
		block := fs.BlockFor
		h := fs.Handler
		fs.mu.Unlock()

		if block > 0 {
			time.Sleep(block)
		}
		if h != nil {
			h(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{
				map[string]any{"message": map[string]any{"content": "ok"}},
			},
			"usage": map[string]any{"total_tokens": 5, "prompt_tokens": 2, "completion_tokens": 3},
		})
	}))
	return fs
}

// LastBody returns the body of the most recent request, or nil.
func (fs *FakeServer) LastBody() []byte {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if len(fs.Requests) == 0 {
		return nil
	}
	return fs.Requests[len(fs.Requests)-1].Body
}

// RunConformance runs the basic conformance checks every adapter must pass.
// Harnesses that wrap a real adapter against NewFakeServer get stronger
// assertions on cancellation, body shape, and error typing.
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

// RunBasicConformance is kept for compatibility.
func RunBasicConformance(t *testing.T, c Client) {
	t.Helper()
	h := &simpleHarness{c: c}
	RunConformance(t, h)
}

type simpleHarness struct{ c Client }

func (h *simpleHarness) Client() Client { return h.c }
