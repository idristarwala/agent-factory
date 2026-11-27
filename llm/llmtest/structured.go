package llmtest

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/idristarwala/agent-factory/llm"
)

// RunStructuredConformance checks schema transmission and validates both good
// and invalid provider output. protocol is openai, anthropic or gemini.
func RunStructuredConformance(t *testing.T, protocol string, newClient func(string) llm.Client) {
	t.Helper()
	schema := map[string]any{"type": "object", "properties": map[string]any{"count": map[string]any{"type": "integer", "minimum": float64(2)}}, "required": []any{"count"}, "additionalProperties": false}
	for _, tc := range []struct {
		name, text string
		valid      bool
	}{
		{"valid", `{"count":2}`, true},
		{"wrong_type", `{"count":"2"}`, false},
		{"out_of_range", `{"count":1}`, false},
		{"extra_property", `{"count":2,"x":0}`, false},
		{"trailing_json", `{"count":2} {}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
					return
				}
				var path []string
				var reply any
				switch protocol {
				case "openai", "openai-envelope":
					path = []string{"response_format", "json_schema", "schema"}
					if protocol == "openai-envelope" {
						path = append([]string{"params"}, path...)
					}
					reply = map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": tc.text}}}}
				case "anthropic":
					path = []string{"output_config", "format", "schema"}
					reply = map[string]any{"content": []map[string]string{{"type": "text", "text": tc.text}}}
				case "gemini":
					path = []string{"generationConfig", "responseJsonSchema"}
					reply = map[string]any{"candidates": []any{map[string]any{"content": map[string]any{"parts": []map[string]string{{"text": tc.text}}}}}}
				default:
					t.Errorf("unknown fixture protocol %q", protocol)
					return
				}
				var wire any = body
				for _, key := range path {
					obj, _ := wire.(map[string]any)
					wire = obj[key]
				}
				if !reflect.DeepEqual(wire, schema) {
					t.Errorf("schema on wire: %v", wire)
				}
				json.NewEncoder(w).Encode(reply)
			}))
			defer srv.Close()
			client := newClient(srv.URL)
			response, err := client.Chat(t.Context(), llm.ChatRequest{Model: "fixture", Messages: []llm.Message{{Role: "user", Content: "count"}}, ResponseSchema: schema})
			if tc.valid {
				if err != nil || response == nil || response.Content != tc.text {
					t.Fatalf("response %v, error %v", response, err)
				}
				return
			}
			var typed *llm.Error
			if !errors.As(err, &typed) || typed.Type != "BAD_RESPONSE" || typed.Retryable {
				t.Fatalf("expected BAD_RESPONSE, got %v", err)
			}
		})
	}
}
