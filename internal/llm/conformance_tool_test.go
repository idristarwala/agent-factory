package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
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
