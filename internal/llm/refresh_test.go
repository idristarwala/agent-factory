package llm

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

type keyHolder struct {
	APIKey string
}

func (k *keyHolder) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	return &ChatResponse{Content: k.APIKey}, nil
}
func (k *keyHolder) Stream(ctx context.Context, req ChatRequest) (Stream, error) {
	return nil, ErrStreamingUnsupported
}
func (k *keyHolder) SetAPIKey(key string) { k.APIKey = key }

func TestTokenRefresherInjectsKey(t *testing.T) {
	holder := &keyHolder{APIKey: "old"}
	calls := 0
	ref := NewTokenRefresher(holder, func(ctx context.Context) (string, error) {
		calls++
		return "fresh-key", nil
	}, time.Hour)

	resp, err := ref.Chat(context.Background(), ChatRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "fresh-key" {
		t.Errorf("expected injected key, got %q", resp.Content)
	}
	if calls != 1 {
		t.Errorf("expected 1 refresh, got %d", calls)
	}

	// Second call should not refresh
	_, err = ref.Chat(context.Background(), ChatRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Errorf("expected still 1 refresh, got %d", calls)
	}
}

func TestUnauthorizedRefresh(t *testing.T) {
	for _, tc := range []struct {
		name    string
		status  int
		recover bool
		calls   int
	}{
		{"rotated", 401, true, 2}, {"repeated_401", 401, false, 2}, {"forbidden", 403, false, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if tc.recover && r.Header.Get("Authorization") == "Bearer fresh" {
					io.WriteString(w, `{"choices":[{"message":{"content":"ok"}}]}`)
					return
				}
				w.WriteHeader(tc.status)
			}))
			defer server.Close()
			refreshes := 0
			ref := NewTokenRefresher(NewOpenAIAdapter(server.URL, "", nil), func(context.Context) (string, error) {
				refreshes++
				if refreshes == 1 {
					return "stale", nil
				}
				return "fresh", nil
			}, time.Hour)
			_, err := ref.Chat(t.Context(), ChatRequest{})
			if (err == nil) != tc.recover || int(calls.Load()) != tc.calls || refreshes != tc.calls {
				t.Fatalf("calls %d refreshes %d error %v", calls.Load(), refreshes, err)
			}
		})
	}
}

func TestConcurrentUnauthorizedRefresh(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer fresh" {
			w.WriteHeader(401)
			return
		}
		io.WriteString(w, `{"choices":[{"message":{"content":"ok"}}]}`)
	}))
	defer server.Close()
	var refreshes atomic.Int32
	ref := NewTokenRefresher(NewOpenAIAdapter(server.URL, "", nil), func(context.Context) (string, error) {
		if refreshes.Add(1) == 1 {
			return "stale", nil
		}
		return "fresh", nil
	}, time.Hour)
	results := make(chan error, 50)
	for i := range 50 {
		go func() {
			if i%2 == 0 {
				stream, err := ref.Stream(t.Context(), ChatRequest{})
				if stream != nil {
					stream.Close()
				}
				results <- err
			} else {
				_, err := ref.Chat(t.Context(), ChatRequest{})
				results <- err
			}
		}()
	}
	for range 50 {
		if err := <-results; err != nil {
			t.Error(err)
		}
	}
	if refreshes.Load() != 2 {
		t.Fatalf("refreshed %d times", refreshes.Load())
	}
}
