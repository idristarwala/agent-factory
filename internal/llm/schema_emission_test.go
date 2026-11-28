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

func TestAnthropicAdapterEmitsJSONSchema(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &got)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"content":[{"type":"text","text":"{\"title\":\"ok\"}"}]}`))
	}))
	defer srv.Close()

	a := NewAnthropicAdapter(srv.URL, "k", nil)
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"title": map[string]any{"type": "string"},
		},
		"required": []string{"title"},
	}
	_, err := a.Chat(context.Background(), ChatRequest{
		Model:          "claude-3-5-sonnet",
		Messages:       []Message{{Role: "user", Content: "fix title"}},
		ResponseSchema: schema,
	})
	if err != nil {
		t.Fatal(err)
	}
	oc, ok := got["output_config"].(map[string]any)
	if !ok {
		t.Fatalf("expected output_config, got %v", got)
	}
	fmt, ok := oc["format"].(map[string]any)
	if !ok || fmt["type"] != "json_schema" {
		t.Fatalf("expected json_schema format, got %v", oc)
	}
}

func TestGeminiAdapterEmitsJSONSchema(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &got)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"{\"title\":\"ok\"}"}]}}]}`))
	}))
	defer srv.Close()

	a := NewGeminiAdapter(srv.URL, "k", nil)
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"title": map[string]any{"type": "string"},
		},
		"required": []string{"title"},
	}
	_, err := a.Chat(context.Background(), ChatRequest{
		Model:          "gemini-1.5-flash",
		Messages:       []Message{{Role: "user", Content: "fix title"}},
		ResponseSchema: schema,
	})
	if err != nil {
		t.Fatal(err)
	}
	genCfg, ok := got["generationConfig"].(map[string]any)
	if !ok {
		t.Fatalf("expected generationConfig, got %v", got)
	}
	if genCfg["responseJsonSchema"] == nil && genCfg["responseMimeType"] != "application/json" {
		t.Fatalf("expected schema or mime type, got %v", genCfg)
	}
}
