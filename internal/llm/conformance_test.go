package llm

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type clientHarness struct{ client Client }

func (h clientHarness) Client() Client { return h.client }

func TestConformanceBasic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[{"message":{"content":"ok"}}]}`)
	}))
	t.Cleanup(server.Close)
	RunConformance(t, clientHarness{NewOpenAIAdapter(server.URL, "test-key", nil)})
}

func TestAdapterNoSetenv(t *testing.T) {
	// Ensure adapters can be created from config without relying on env mutation.
	cfg := map[string]string{
		"base_url": "http://localhost:9999",
		"api_key":  "test-key",
	}
	c, err := New("openai", cfg)
	if err != nil {
		t.Fatalf("openai adapter from config: %v", err)
	}
	if c == nil {
		t.Fatal("nil client")
	}
}
