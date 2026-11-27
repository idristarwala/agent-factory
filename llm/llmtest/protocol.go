package llmtest

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/idristarwala/agent-factory/llm"
)

// ProtocolHarness runs a real adapter against local protocol fixtures. The
// OpenAI envelope fixture expects the standard parameters under "params".
type ProtocolHarness struct {
	Protocol  string
	NewClient func(baseURL string) llm.Client
	client    llm.Client
}

func NewProtocolHarness(t *testing.T, protocol string, newClient func(string) llm.Client) *ProtocolHarness {
	t.Helper()
	replies := map[string]string{
		"openai":    `{"choices":[{"message":{"content":"ok"}}],"usage":{"prompt_tokens":2,"completion_tokens":3,"total_tokens":5}}`,
		"anthropic": `{"content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":2,"output_tokens":3}}`,
		"gemini":    `{"candidates":[{"content":{"parts":[{"text":"ok"}]}}],"usageMetadata":{"promptTokenCount":2,"candidatesTokenCount":3,"totalTokenCount":5}}`,
	}
	replies["openai-envelope"] = replies["openai"]
	reply, ok := replies[protocol]
	if !ok {
		t.Fatalf("unknown fixture protocol %q", protocol)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, reply)
	}))
	t.Cleanup(server.Close)
	return &ProtocolHarness{Protocol: protocol, NewClient: newClient, client: newClient(server.URL)}
}

func (h *ProtocolHarness) Client() llm.Client { return h.client }
