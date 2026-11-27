package llm_test

import (
	"testing"

	"github.com/idristarwala/agent-factory/internal/llm"
	"github.com/idristarwala/agent-factory/llm/llmtest"
)

func TestProtocolConformance(t *testing.T) {
	for _, name := range []string{"openai", "openai-envelope", "anthropic", "gemini"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			makeClient := func(url string) llm.Client {
				cfg := map[string]string{"base_url": url, "api_key": "test", "cheap_model": "fixture"}
				if name == "openai-envelope" {
					cfg["envelope"] = `{"params_key":"params","template":{"operation":"chat"}}`
				}
				client, err := llm.New(name, cfg)
				if err != nil {
					t.Fatal(err)
				}
				return client
			}
			llmtest.RunConformance(t, llmtest.NewProtocolHarness(t, name, makeClient))
		})
	}
}
