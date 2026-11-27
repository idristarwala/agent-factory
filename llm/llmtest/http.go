package llmtest

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/idristarwala/agent-factory/llm"
)

// RunHTTPConformance exercises failure responses from a local gateway.
// newClient must point every request at the supplied base URL.
func RunHTTPConformance(t *testing.T, newClient func(baseURL string) llm.Client) {
	t.Helper()
	req := llm.ChatRequest{Model: "fixture", Messages: []llm.Message{{Role: "user", Content: "hello"}}}
	for _, tc := range []struct {
		name  string
		code  int
		kind  string
		retry bool
		body  string
	}{
		{"bad_request", 400, "BAD_REQUEST", false, "bad input"},
		{"unauthorized", 401, "AUTH", false, "unauthorized"},
		{"forbidden", 403, "AUTH", false, "forbidden"},
		{"rate_limit", 429, "RATE_LIMITED", true, "rate limit"},
		{"server_error", 500, "GATEWAY_5XX", true, "server failure"},
		{"proxy_html", 502, "GATEWAY_5XX", true, "<html>Bad gateway</html>"},
		{"truncated", 200, "BAD_RESPONSE", false, `{"unfinished":`},
		{"empty_result", 200, "BAD_RESPONSE", false, `{"choices":[],"content":[],"candidates":[]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.code == 429 {
					w.Header().Set("Retry-After", "7")
				}
				w.WriteHeader(tc.code)
				io.WriteString(w, tc.body)
			}))
			defer srv.Close()
			client := newClient(srv.URL)
			check := func(err error) {
				t.Helper()
				var e *llm.Error
				if !errors.As(err, &e) || e.Type != tc.kind || e.Retryable != tc.retry {
					t.Fatalf("expected %s (retryable %v), got %v", tc.kind, tc.retry, err)
				}
				if tc.code >= 400 && e.Status != tc.code {
					t.Fatalf("status %d, want %d", e.Status, tc.code)
				}
				if tc.code == 429 && e.RetryAfter != 7*time.Second {
					t.Fatalf("retry delay %v", e.RetryAfter)
				}
			}
			_, err := client.Chat(t.Context(), req)
			check(err)
			if tc.code >= 400 {
				stream, err := client.Stream(t.Context(), req)
				if stream != nil {
					defer stream.Close()
				}
				if !errors.Is(err, llm.ErrStreamingUnsupported) {
					check(err)
				}
			}
		})
	}
	for _, timeout := range []bool{false, true} {
		name := "cancel_in_flight"
		if timeout {
			name = "timeout"
		}
		t.Run(name, func(t *testing.T) {
			started := make(chan struct{})
			release := make(chan struct{})
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				close(started)
				select {
				case <-r.Context().Done():
				case <-release:
				}
			}))
			defer srv.Close()
			defer close(release)
			client := newClient(srv.URL)
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			result := make(chan error, 1)
			go func() { _, err := client.Chat(ctx, req); result <- err }()
			select {
			case <-started:
			case <-ctx.Done():
				t.Fatal("request did not reach gateway")
			}
			if !timeout {
				cancel()
			}
			select {
			case err := <-result:
				if timeout {
					var e *llm.Error
					if !errors.As(err, &e) || e.Type != "TIMEOUT" || !e.Retryable {
						t.Fatalf("expected retryable timeout, got %v", err)
					}
				} else if !errors.Is(err, context.Canceled) {
					t.Fatalf("expected cancellation, got %v", err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("request did not return after cancellation or timeout")
			}
		})
	}
}
