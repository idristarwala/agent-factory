package llm

import "testing"

func TestConformanceBasic(t *testing.T) {
	server := NewFakeServer()
	t.Cleanup(server.Close)
	RunBasicConformance(t, NewOpenAIAdapter(server.URL, "test-key", nil))
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
