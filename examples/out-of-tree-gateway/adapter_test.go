package example

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/idristarwala/agent-factory/llm"
	"github.com/idristarwala/agent-factory/llm/llmtest"
)

// TestCorpGatewayPassesConformance is a concrete out-of-tree test that
// exercises llmtest.RunConformance against a real httptest server speaking
// OpenAI-compatible responses. This is the pattern users copy into their
// own module.
func TestCorpGatewayPassesConformance(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{
				map[string]any{
					"message": map[string]any{"content": "ok"},
				},
			},
			"usage": map[string]any{"total_tokens": 5, "prompt_tokens": 2, "completion_tokens": 3},
		})
	}))
	defer srv.Close()

	gw := NewCorpGateway(srv.URL, "test-key", map[string]string{"X-Consumer-Id": "conformance"})
	gw.SetHTTPClient(srv.Client())

	h := conformanceHarness{c: gw.Client()}
	llmtest.RunConformance(t, h)
}

type conformanceHarness struct{ c llm.Client }

func (h conformanceHarness) Client() llm.Client { return h.c }

func TestCorpGatewayHTTPFailures(t *testing.T) {
	llmtest.RunHTTPConformance(t, func(url string) llm.Client {
		return NewCorpGateway(url, "fixture", nil).Client()
	})
}

func TestCorpGatewayStructuredOutput(t *testing.T) {
	llmtest.RunStructuredConformance(t, "openai", func(url string) llm.Client { return NewCorpGateway(url, "fixture", nil).Client() })
}
