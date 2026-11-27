package configuredgateway

import (
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/idristarwala/agent-factory/internal/llm"
)

func TestLogicalRoutesAndRotatedCredentials(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	var calls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		expected := "first"
		if n > 1 {
			expected = "rotated"
		}
		if r.Header.Get("Authorization") != "Bearer "+expected || r.Header.Get("X-Consumer") != "factory" || r.Header.Get("x-goog-api-key") != "" || r.Header.Get("x-api-key") != "" {
			t.Error("incorrect request headers")
		}
		switch r.URL.Path {
		case "/v1/chat/completions":
			if body["model"] != "test-openai" || body["max_completion_tokens"] != float64(80) || body["max_tokens"] != nil {
				t.Errorf("unexpected OpenAI request %v", body)
			}
			fmt.Fprint(w, `{"model":"served-openai","choices":[{"message":{"content":"ok"}}]}`)
		case "/v1/models/test-gemini:generateContent":
			cfg := body["generationConfig"].(map[string]any)
			if cfg["thinkingConfig"].(map[string]any)["thinkingBudget"] != float64(0) {
				t.Error("missing zero reasoning budget")
			}
			fmt.Fprint(w, `{"modelVersion":"served-gemini","candidates":[{"content":{"parts":[{"text":"ok"}]}}]}`)
		case "/v1/messages":
			if body["model"] != "test-anthropic" || r.Header.Get("X-Route") != "review" {
				t.Error("incorrect judge route")
			}
			fmt.Fprint(w, `{"model":"served-anthropic","content":[{"type":"text","text":"ok"}]}`)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	write("ca.pem", string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})))
	write("key", "first")
	write("routing.yaml", fmt.Sprintf(`gateways:
  team:
    base_url: %s/v1
    auth: {type: bearer, key_file: key}
    tls: {ca_file: ca.pem}
    headers: {X-Consumer: factory}
    proxy: none
models:
  cheap: {gateway: team, protocol: gemini, model: test-gemini, reasoning_budget: 0}
  frontier: {gateway: team, protocol: openai, model: test-openai, token_param: max_completion_tokens}
  judge: {gateway: team, protocol: anthropic, model: test-anthropic, headers: {X-Route: review}}
`, server.URL))
	router, err := load(filepath.Join(dir, "routing.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, alias := range []string{"cheap", "frontier", "judge"} {
		req := llm.ChatRequest{Model: alias, MaxTokens: 80, Messages: []llm.Message{{Role: "user", Content: "test"}}}
		resp, err := router.Chat(t.Context(), req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.LogicalModel != alias || !strings.HasPrefix(resp.Model, "served-") {
			t.Fatalf("model mismatch %+v", resp)
		}
		write("key", "rotated")
	}
	req := llm.ChatRequest{Model: "missing"}
	_, err = router.Chat(t.Context(), req)
	var typed *llm.Error
	if !errors.As(err, &typed) || typed.Type != "BAD_REQUEST" {
		t.Fatal(err)
	}
	if _, err = router.Stream(t.Context(), req); !errors.As(err, &typed) || typed.Type != "BAD_REQUEST" {
		t.Fatal(err)
	}
	if calls.Load() != 3 {
		t.Fatalf("unexpected requests %d", calls.Load())
	}
}
