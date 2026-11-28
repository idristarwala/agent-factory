package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAIAdapterEmitsJSONSchema(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &got)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"{\"title\":\"ok\"}"}}]}`))
	}))
	defer srv.Close()

	a := NewOpenAIAdapter(srv.URL, "k", nil)
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"title": map[string]any{"type": "string"},
		},
		"required": []string{"title"},
	}
	_, err := a.Chat(context.Background(), ChatRequest{
		Model:          "gpt-4o",
		Messages:       []Message{{Role: "user", Content: "fix title"}},
		ResponseSchema: schema,
	})
	if err != nil {
		t.Fatal(err)
	}
	rf, ok := got["response_format"].(map[string]any)
	if !ok || rf["type"] != "json_schema" {
		t.Fatalf("expected json_schema response_format, got %v", got["response_format"])
	}
}
