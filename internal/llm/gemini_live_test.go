//go:build gemini_live

package llm

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLiveGeminiConversation(t *testing.T) {
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		t.Fatal("set GEMINI_API_KEY for the live test")
	}
	model := os.Getenv("GEMINI_CHEAP_MODEL")
	if model == "" {
		model = "gemini-3.5-flash-lite"
	}
	gateway := NewGeminiAdapter("https://generativelanguage.googleapis.com/v1beta", key, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	request := ChatRequest{Model: model, MaxTokens: 150, Messages: []Message{
		{Role: "system", Content: "Answer the final question with the exact identifier alone."},
		{Role: "user", Content: "Remember the identifier XQ-719."},
		{Role: "assistant", Content: "Understood."},
		{Role: "user", Content: "What identifier did I provide earlier?"},
	}}
	reply, err := gateway.Chat(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(reply.Content) != "XQ-719" || reply.Usage.Total == 0 || reply.Model == "" {
		t.Fatalf("unexpected reply %q, model %q, tokens %d", reply.Content, reply.Model, reply.Usage.Total)
	}
	stream, err := gateway.Stream(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	var result strings.Builder
	for {
		event, err := stream.Recv()
		if err != nil {
			t.Fatal(err)
		}
		if event == nil {
			break
		}
		result.WriteString(event.TextDelta)
	}
	if strings.TrimSpace(result.String()) != "XQ-719" {
		t.Fatalf("unexpected stream %q", result.String())
	}
	request.ResponseSchema = map[string]any{"type": "object", "properties": map[string]any{"identifier": map[string]string{"type": "string"}}, "required": []string{"identifier"}, "additionalProperties": false}
	request.Messages[0].Content = "Return the remembered identifier in the requested JSON object."
	reply, err = gateway.Chat(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	var structured struct{ Identifier string }
	if err := json.Unmarshal([]byte(reply.Content), &structured); err != nil || structured.Identifier != "XQ-719" {
		t.Fatalf("unexpected structured output %q", reply.Content)
	}

	request = ChatRequest{Model: model, MaxTokens: 350, Tools: []ToolSpec{{Name: "lookup", Description: "Read the observed value for a record.", Parameters: map[string]any{"type": "object", "properties": map[string]any{"record": map[string]string{"type": "string"}}, "required": []string{"record"}}}}, Messages: []Message{
		{Role: "system", Content: "Use lookup to read the record. Once the value is available, reply with the value only."},
		{Role: "user", Content: "Read record A with lookup and tell me the value."},
	}}
	first, err := gateway.Chat(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.ToolCalls) != 1 || first.ToolCalls[0].Name != "lookup" {
		t.Fatal("expected a lookup request")
	}
	request.Messages = append(request.Messages, Message{Role: "assistant", Content: first.Content, ToolCalls: first.ToolCalls}, Message{Role: "tool", ToolCallID: first.ToolCalls[0].ID, Content: "OBS-714"})
	reply, err = gateway.Chat(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(reply.Content) != "OBS-714" {
		t.Fatalf("unexpected tool result answer %q", reply.Content)
	}
}
