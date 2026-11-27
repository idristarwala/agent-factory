//go:build openrouter_live

package llm

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestLiveOpenRouterToolResult(t *testing.T) {
	key := os.Getenv("OPENROUTER_API_KEY")
	if key == "" {
		t.Fatal("set OPENROUTER_API_KEY")
	}
	model := os.Getenv("OPENROUTER_MODEL")
	if model == "" {
		model = "openai/gpt-4o"
	}
	gw := NewOpenAIAdapter("https://openrouter.ai/api/v1", key, nil)
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	req := ChatRequest{Model: model, MaxTokens: 150, Messages: []Message{
		{Role: "system", Content: "Use lookup to find the value. After the tool responds, copy its exact identifier into the value field, character for character, without any surrounding words."},
		{Role: "user", Content: "Look up record A."},
	}, Tools: []ToolSpec{{Name: "lookup", Description: "Look up a record by ID.", Parameters: map[string]any{"type": "object", "properties": map[string]any{"id": map[string]string{"type": "string"}}, "required": []string{"id"}}}}}
	toolStream, err := gw.Stream(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	defer toolStream.Close()
	first, err := CollectStream(toolStream)
	if err != nil {
		t.Fatal(err)
	}
	if first.Usage.Total == 0 || first.FinishReason == "" || len(first.ToolCalls) != 1 || first.ToolCalls[0].Name != "lookup" {
		t.Fatal("expected a lookup request")
	}
	req.Messages = append(req.Messages, Message{Role: "assistant", Content: first.Content, ToolCalls: first.ToolCalls}, Message{Role: "tool", ToolCallID: first.ToolCalls[0].ID, Content: "OBS-714"})
	req.Tools = nil
	req.ResponseSchema = map[string]any{"type": "object", "properties": map[string]any{"value": map[string]string{"type": "string"}}, "required": []string{"value"}, "additionalProperties": false}
	reply, err := gw.Chat(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	var answer struct{ Value string }
	if err := json.Unmarshal([]byte(reply.Content), &answer); err != nil || answer.Value != "OBS-714" || reply.Usage.Total == 0 {
		t.Fatalf("unexpected result %q or missing usage", reply.Content)
	}
}
