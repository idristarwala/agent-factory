package llm

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestToolHistoryWire(t *testing.T) {
	req := ChatRequest{Model: "model", Messages: []Message{
		{Role: "system", Content: "Read the source."},
		{Role: "user", Content: "Find A."},
		{Role: "assistant", ToolCalls: []ToolCall{{ID: "call1", Name: "lookup", Arguments: `{"id":"A"}`}}},
		{Role: "tool", ToolCallID: "call1", Content: "found A"},
	}}
	data, err := buildOpenAIBody(req, false)
	if err != nil {
		t.Fatal(err)
	}
	var openAI struct{ Messages []map[string]any }
	if err := json.Unmarshal(data, &openAI); err != nil {
		t.Fatal(err)
	}
	expectedCalls := []any{map[string]any{"id": "call1", "type": "function", "function": map[string]any{"name": "lookup", "arguments": `{"id":"A"}`}}}
	if !reflect.DeepEqual(openAI.Messages[2]["tool_calls"], expectedCalls) || openAI.Messages[3]["tool_call_id"] != "call1" {
		t.Fatal(openAI.Messages)
	}
	body, err := anthropicBody(req)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := json.Marshal(body)
	var wire struct {
		System   []struct{ Text string }
		Messages []struct {
			Role    string
			Content json.RawMessage
		}
	}
	if err := json.Unmarshal(got, &wire); err != nil {
		t.Fatal(err)
	}
	if len(wire.System) != 1 || wire.System[0].Text != req.Messages[0].Content || len(wire.Messages) != 3 {
		t.Fatalf("unexpected conversation %s", got)
	}
	var use, result []map[string]any
	if err := json.Unmarshal(wire.Messages[1].Content, &use); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(wire.Messages[2].Content, &result); err != nil {
		t.Fatal(err)
	}
	if wire.Messages[1].Role != "assistant" || wire.Messages[2].Role != "user" || !reflect.DeepEqual(use, []map[string]any{{"type": "tool_use", "id": "call1", "name": "lookup", "input": map[string]any{"id": "A"}}}) || !reflect.DeepEqual(result, []map[string]any{{"type": "tool_result", "tool_use_id": "call1", "content": "found A"}}) {
		t.Fatalf("unexpected tool history %s", got)
	}
}
