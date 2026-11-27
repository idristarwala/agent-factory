package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMultiProtocolConformance(t *testing.T) {
	t.Helper()

	// Fake OpenAI
	openAI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`))
	}))
	defer openAI.Close()

	adapter := NewOpenAIAdapter(openAI.URL, "test-key", nil)
	resp, err := adapter.Chat(context.Background(), ChatRequest{
		Model:    "gpt-test",
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("openai adapter: %v", err)
	}
	if resp.Content != "ok" {
		t.Errorf("expected ok, got %q", resp.Content)
	}

	// Fake Gemini
	gemini := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"ok"}]}}]}`))
	}))
	defer gemini.Close()

	gAdapter := NewGeminiAdapter(gemini.URL, "test-key", nil)
	resp, err = gAdapter.Chat(context.Background(), ChatRequest{
		Model:    "gemini-test",
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("gemini adapter: %v", err)
	}
	if resp.Content != "ok" {
		t.Errorf("expected ok, got %q", resp.Content)
	}

	// Fake Anthropic
	anthropic := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"content":[{"text":"ok"}]}`))
	}))
	defer anthropic.Close()

	aAdapter := NewAnthropicAdapter(anthropic.URL, "test-key", nil)
	resp, err = aAdapter.Chat(context.Background(), ChatRequest{
		Model:    "claude-test",
		Messages: []Message{{Role: "user", Content: "hi"}},
		MaxTokens: 100,
	})
	if err != nil {
		t.Fatalf("anthropic adapter: %v", err)
	}
	if resp.Content != "ok" {
		t.Errorf("expected ok, got %q", resp.Content)
	}
}
