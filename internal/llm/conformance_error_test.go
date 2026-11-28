package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestErrorConformanceCases covers the error mappings required by issue #8:
// 400, 401, 403, 429 (Retry-After), 500, 502 (HTML), truncated body, empty choices.
func TestErrorConformanceCases(t *testing.T) {
	cases := []struct {
		name       string
		status     int
		body       string
		retryAfter string
		wantType   string
		retryable  bool
		wantAfter  bool
	}{
		{"bad_request_400", http.StatusBadRequest, `{"error":"bad"}`, "", "BAD_REQUEST", false, false},
		{"auth_401", http.StatusUnauthorized, `{"error":"unauth"}`, "", "AUTH", false, false},
		{"forbidden_403", http.StatusForbidden, `{"error":"forbidden"}`, "", "AUTH", false, false},
		{"rate_limit_429", http.StatusTooManyRequests, `{"error":"rate"}`, "5", "RATE_LIMITED", true, true},
		{"server_500", http.StatusInternalServerError, `{"error":"boom"}`, "", "GATEWAY_5XX", true, false},
		{"bad_gateway_502_html", http.StatusBadGateway, `<html>bad gateway</html>`, "", "GATEWAY_5XX", true, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.retryAfter != "" {
					w.Header().Set("Retry-After", tc.retryAfter)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			a := NewOpenAIAdapter(srv.URL, "test-key", nil)
			_, err := a.Chat(context.Background(), ChatRequest{
				Model:    "gpt",
				Messages: []Message{{Role: "user", Content: "hi"}},
			})
			if err == nil {
				t.Fatal("expected error")
			}
			e, ok := err.(*Error)
			if !ok {
				t.Fatalf("expected *Error, got %T: %v", err, err)
			}
			if e.Type != tc.wantType {
				t.Errorf("type: got %q want %q", e.Type, tc.wantType)
			}
			if e.Retryable != tc.retryable {
				t.Errorf("retryable: got %v want %v", e.Retryable, tc.retryable)
			}
			if tc.wantAfter && e.RetryAfter == 0 {
				t.Error("expected non-zero RetryAfter")
			}
		})
	}
}

func TestTruncatedAndEmptyChoices(t *testing.T) {
	t.Run("truncated_body", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"choices":[{"message":{"content":"partial`)) // truncated
		}))
		defer srv.Close()

		a := NewOpenAIAdapter(srv.URL, "key", nil)
		_, err := a.Chat(context.Background(), ChatRequest{
			Model:    "gpt",
			Messages: []Message{{Role: "user", Content: "hi"}},
		})
		if err == nil {
			t.Fatal("expected error on truncated body")
		}
		e, ok := err.(*Error)
		if !ok || e.Type != "BAD_RESPONSE" {
			t.Errorf("expected BAD_RESPONSE, got %v", err)
		}
	})

	t.Run("empty_choices", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"choices":[],"usage":{"prompt_tokens":1,"completion_tokens":0}}`))
		}))
		defer srv.Close()

		a := NewOpenAIAdapter(srv.URL, "key", nil)
		_, err := a.Chat(context.Background(), ChatRequest{
			Model:    "gpt",
			Messages: []Message{{Role: "user", Content: "hi"}},
		})
		if err == nil {
			t.Fatal("expected error on empty choices")
		}
		e, ok := err.(*Error)
		if !ok || e.Type != "BAD_RESPONSE" {
			t.Errorf("expected BAD_RESPONSE for empty choices, got %v", err)
		}
		if !strings.Contains(e.Message, "choices") {
			t.Errorf("message should mention choices, got %q", e.Message)
		}
	})
}

func TestTimeoutMapping(t *testing.T) {
	// Simulate a client that always times out via a very short deadline on a slow server.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Write([]byte(`{"choices":[{"message":{"content":"late"}}]}`))
	}))
	defer srv.Close()

	a := NewOpenAIAdapter(srv.URL, "key", nil)
	a.Client = &http.Client{Timeout: 50 * time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := a.Chat(ctx, ChatRequest{
		Model:    "gpt",
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	e, ok := err.(*Error)
	if !ok || e.Type != "TIMEOUT" || !e.Retryable {
		t.Errorf("expected retryable TIMEOUT, got %v", err)
	}
}
