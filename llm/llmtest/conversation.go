package llmtest

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/idristarwala/agent-factory/llm"
)

// Check the wire conversation as well as normalized output. A lossy adapter
// can otherwise pass a conversation test simply by returning canned text.
func checkConversation(t *testing.T, p *ProtocolHarness) {
	t.Helper()
	protocol := p.Protocol
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		if r.URL.Query().Has("key") {
			t.Error("credential in URL")
		}
		if protocol == "openai-envelope" {
			body, _ = body["params"].(map[string]any)
		}
		stream := body["stream"] == true || strings.HasSuffix(r.URL.Path, ":streamGenerateContent")
		var turns []any
		var expected []map[string]any
		var reply, frames string
		switch protocol {
		case "openai", "openai-envelope":
			turns, _ = body["messages"].([]any)
			expected = []map[string]any{{"role": "system", "content": "system prompt"}, {"role": "user", "content": "first"}, {"role": "assistant", "content": "second"}, {"role": "user", "content": "third"}}
			reply = `{"choices":[{"message":{"tool_calls":[{"id":"f","function":{"name":"lookup","arguments":"{}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":2,"completion_tokens":3,"total_tokens":5}}`
			frames = "data: " + `{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"f","function":{"name":"lookup","arguments":"{}"}}]},"finish_reason":"tool_calls"}]}` + "\n\ndata: " + `{"choices":[],"usage":{"prompt_tokens":2,"completion_tokens":3,"total_tokens":5}}` + "\n\ndata: [DONE]\n\n"
		case "anthropic":
			turns, _ = body["messages"].([]any)
			expected = []map[string]any{{"role": "user", "content": "first"}, {"role": "assistant", "content": "second"}, {"role": "user", "content": "third"}}
			sys := []any{map[string]any{"type": "text", "text": "system prompt"}}
			if !reflect.DeepEqual(body["system"], sys) {
				t.Error("missing Anthropic system turn")
			}
			reply = `{"content":[{"type":"tool_use","id":"f","name":"lookup","input":{}}],"stop_reason":"tool_use","usage":{"input_tokens":2,"output_tokens":3}}`
			frames = `data: {"type":"message_start","message":{"usage":{"input_tokens":2}}}

data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"f","name":"lookup"}}

data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{}"}}

data: {"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":3}}

`
		case "gemini":
			turns, _ = body["contents"].([]any)
			part := func(s string) []any { return []any{map[string]any{"text": s}} }
			expected = []map[string]any{{"role": "user", "parts": part("first")}, {"role": "model", "parts": part("second")}, {"role": "user", "parts": part("third")}}
			if !reflect.DeepEqual(body["systemInstruction"], map[string]any{"parts": part("system prompt")}) {
				t.Error("missing Gemini system turn")
			}
			reply = `{"candidates":[{"content":{"parts":[{"functionCall":{"name":"lookup","args":{}}}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":2,"candidatesTokenCount":3,"totalTokenCount":5}}`
			frames = "data: " + reply + "\n\n"
		}
		if len(turns) != len(expected) {
			t.Errorf("got %d turns, want %d", len(turns), len(expected))
			return
		}
		for i := range expected {
			if !reflect.DeepEqual(turns[i], expected[i]) {
				t.Errorf("incorrect turn %d: %v", i, turns[i])
			}
		}
		if stream {
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(w, frames)
		} else {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, reply)
		}
	}))
	defer server.Close()
	req := llm.ChatRequest{Model: "fixture", Messages: []llm.Message{{Role: "system", Content: "system prompt"}, {Role: "user", Content: "first"}, {Role: "assistant", Content: "second"}, {Role: "user", Content: "third"}}, Tools: []llm.ToolSpec{{Name: "lookup", Parameters: map[string]any{"type": "object"}}}}
	client := p.NewClient(server.URL)
	chat, err := client.Chat(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	if chat.Usage.Total != 5 || len(chat.ToolCalls) != 1 || chat.ToolCalls[0].Name != "lookup" || chat.FinishReason == "" {
		t.Fatalf("unexpected one-shot reply %+v", chat)
	}
	stream, err := client.Stream(t.Context(), req)
	if errors.Is(err, llm.ErrStreamingUnsupported) {
		t.Log("adapter declares streaming unsupported")
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	result, err := llm.CollectStream(stream)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result.ToolCalls, chat.ToolCalls) || result.Content != chat.Content || result.Usage != chat.Usage || result.FinishReason != chat.FinishReason {
		t.Fatalf("stream %+v differs from chat %+v", result, chat)
	}
}
