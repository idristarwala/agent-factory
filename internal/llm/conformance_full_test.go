package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGeminiAdapterAgainstFake(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models/flash:generateContent" {
			http.Error(w, "bad path", http.StatusNotFound)
			return
		}
		if r.Header.Get("x-goog-api-key") != "test-key" || r.URL.Query().Has("key") {
			http.Error(w, "no key", http.StatusUnauthorized)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{
				{"content": map[string]any{"parts": []map[string]string{{"text": "gemini-ok"}}}},
			},
		})
	}))
	defer srv.Close()

	a := NewGeminiAdapter(srv.URL, "test-key", nil)
	resp, err := a.Chat(context.Background(), ChatRequest{
		Model:    "flash",
		Messages: []Message{{Role: "user", Content: "hi"}},
		MaxTokens: 10,
	})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if resp.Content != "gemini-ok" {
		t.Errorf("got %q", resp.Content)
	}
}

func TestAnthropicAdapterAgainstFake(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			http.Error(w, "bad path", http.StatusNotFound)
			return
		}
		if r.Header.Get("x-api-key") == "" {
			http.Error(w, "no key", http.StatusUnauthorized)
			return
		}
		if r.Header.Get("anthropic-version") == "" {
			http.Error(w, "no version", http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]string{{"text": "anthropic-ok"}},
		})
	}))
	defer srv.Close()

	a := NewAnthropicAdapter(srv.URL, "test-key", nil)
	resp, err := a.Chat(context.Background(), ChatRequest{
		Model:    "claude",
		Messages: []Message{{Role: "user", Content: "hi"}},
		MaxTokens: 10,
	})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if resp.Content != "anthropic-ok" {
		t.Errorf("got %q", resp.Content)
	}
}

func TestOpenAIAdapterRateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "2")
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	defer srv.Close()

	a := NewOpenAIAdapter(srv.URL, "key", nil)
	_, err := a.Chat(context.Background(), ChatRequest{
		Model:    "gpt",
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	e, ok := err.(*Error)
	if !ok || e.Type != "RATE_LIMITED" || !e.Retryable {
		t.Errorf("expected retryable RATE_LIMITED, got %v", err)
	}
}
