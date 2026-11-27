package llm

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestImageWireFormats(t *testing.T) {
	request := ChatRequest{Messages: []Message{{Role: "user", Content: "review", Images: []ImagePart{{Mime: "image/png", Base64: "YWJj"}}}}}
	for _, tc := range []struct {
		name, fragment string
		build          func(ChatRequest) ([]byte, error)
	}{
		{"openai", `"url":"data:image/png;base64,YWJj"`, buildChatBody},
		{"gemini", `"inlineData":{"data":"YWJj","mimeType":"image/png"}`, geminiBody},
		{"anthropic", `"source":{"data":"YWJj","media_type":"image/png","type":"base64"}`, func(req ChatRequest) ([]byte, error) {
			body, err := anthropicBody(req)
			if err != nil {
				return nil, err
			}
			return json.Marshal(body)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, err := tc.build(request)
			if err != nil || !strings.Contains(string(body), tc.fragment) || !strings.Contains(string(body), `"text":"review"`) {
				t.Fatalf("body %s, error %v", body, err)
			}
			bad := ChatRequest{Messages: []Message{{Role: "user", Images: []ImagePart{{URL: "https://example.com/a.png"}}}}}
			if _, err := tc.build(bad); err == nil {
				t.Fatal("accepted URL-only image")
			}
		})
	}
}

func buildChatBody(req ChatRequest) ([]byte, error) { return buildOpenAIBody(req, false) }
