package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
	"net/http/httptest"
	"testing"
)

// TestOpenAIAdapterToolCallRoundTrip verifies that tools are serialized on the
// request and tool_calls are parsed from the response.
func TestOpenAIAdapterToolCallRoundTrip(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			http.Error(w, "bad path", http.StatusNotFound)
			return
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{
				"message": map[string]any{
					"role":    "assistant",
					"content": "",
					"tool_calls": []map[string]any{{
						"id":   "call_1",
						"type": "function",
						"function": map[string]any{
							"name":      "lookup",
							"arguments": `{"q":"widget"}`,
						},
					}},
				},
			}},
			"usage": map[string]int{"prompt_tokens": 10, "completion_tokens": 5},
		})
	}))
	defer srv.Close()

	a := NewOpenAIAdapter(srv.URL, "test-key", nil)
	resp, err := a.Chat(context.Background(), ChatRequest{
		Model: "gpt-4o",
		Messages: []Message{{
			Role:    "user",
			Content: "find the widget",
		}},
		Tools: []ToolSpec{{
			Name:        "lookup",
			Description: "Look up an item",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"q": map[string]any{"type": "string"},
				},
			},
		}},
		MaxTokens: 50,
	})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}
	tc := resp.ToolCalls[0]
	if tc.ID != "call_1" || tc.Name != "lookup" || tc.Arguments != `{"q":"widget"}` {
		t.Errorf("unexpected tool call: %+v", tc)
	}

	// Request side: tools must be present
	tools, ok := gotBody["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("expected tools in request body, got %#v", gotBody["tools"])
	}
	fn, _ := tools[0].(map[string]any)["function"].(map[string]any)
	if fn["name"] != "lookup" {
		t.Errorf("tool name not serialized: %v", fn)
	}
}

// TestAnthropicAdapterToolCallRoundTrip verifies tools are serialized and tool_use is parsed.
func TestAnthropicAdapterToolCallRoundTrip(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			http.Error(w, "bad path", http.StatusNotFound)
			return
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{
				{
					"type":  "tool_use",
					"id":    "toolu_1",
					"name":  "lookup",
					"input": map[string]any{"q": "widget"},
				},
			},
		})
	}))
	defer srv.Close()

	a := NewAnthropicAdapter(srv.URL, "test-key", nil)
	resp, err := a.Chat(context.Background(), ChatRequest{
		Model: "claude-3",
		Messages: []Message{{
			Role:    "user",
			Content: "find the widget",
		}},
		Tools: []ToolSpec{{
			Name:        "lookup",
			Description: "Look up an item",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"q": map[string]any{"type": "string"},
				},
			},
		}},
		MaxTokens: 50,
	})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}
	tc := resp.ToolCalls[0]
	if tc.ID != "toolu_1" || tc.Name != "lookup" {
		t.Errorf("unexpected tool call: %+v", tc)
	}

	tools, ok := gotBody["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("expected tools in request body, got %#v", gotBody["tools"])
	}
	if tools[0].(map[string]any)["name"] != "lookup" {
		t.Errorf("tool name not serialized: %v", tools[0])
	}
}

// TestGeminiAdapterToolCallRoundTrip verifies functionDeclarations and functionCall parsing.
func TestGeminiAdapterToolCallRoundTrip(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, ":generateContent") {
			http.Error(w, "bad path", http.StatusNotFound)
			return
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{{
				"content": map[string]any{
					"parts": []map[string]any{{
						"functionCall": map[string]any{
							"name": "lookup",
							"args": map[string]any{"q": "widget"},
						},
					}},
				},
			}},
		})
	}))
	defer srv.Close()

	a := NewGeminiAdapter(srv.URL, "test-key", nil)
	resp, err := a.Chat(context.Background(), ChatRequest{
		Model: "gemini-flash",
		Messages: []Message{{
			Role:    "user",
			Content: "find the widget",
		}},
		Tools: []ToolSpec{{
			Name:        "lookup",
			Description: "Look up an item",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"q": map[string]any{"type": "string"},
				},
			},
		}},
		MaxTokens: 50,
	})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}
	tc := resp.ToolCalls[0]
	if tc.Name != "lookup" {
		t.Errorf("unexpected tool call: %+v", tc)
	}

	tools, ok := gotBody["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("expected tools in request body, got %#v", gotBody["tools"])
	}
	decls := tools[0].(map[string]any)["functionDeclarations"].([]any)
	if decls[0].(map[string]any)["name"] != "lookup" {
		t.Errorf("function declaration not serialized: %v", decls[0])
	}
}

// TestTokenRefresherExercised verifies the auth-refresh decorator runs before Chat.
func TestTokenRefresherExercised(t *testing.T) {
	refreshed := 0
	inner := &recordingClient{resp: &ChatResponse{Content: "ok"}}
	r := NewTokenRefresher(inner, func(ctx context.Context) (string, error) {
		refreshed++
		return "fresh-key", nil
	}, time.Hour)

	_, err := r.Chat(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if refreshed != 1 {
		t.Errorf("expected refresh to be called once, got %d", refreshed)
	}
	// Second call within interval should not refresh again
	_, err = r.Chat(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "hi again"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if refreshed != 1 {
		t.Errorf("expected no second refresh, got %d", refreshed)
	}
}

type recordingClient struct {
	resp *ChatResponse
}

func (c *recordingClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	return c.resp, nil
}
func (c *recordingClient) Stream(ctx context.Context, req ChatRequest) (Stream, error) {
	return nil, ErrStreamingUnsupported
}
