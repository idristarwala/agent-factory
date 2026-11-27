package llm_test

import (
	"testing"

	"github.com/idristarwala/agent-factory/internal/llm"
	"github.com/idristarwala/agent-factory/llm/llmtest"
)

func TestHTTPFailureContract(t *testing.T) {
	for name, makeClient := range map[string]func(string) llm.Client{
		"openai":    func(url string) llm.Client { return llm.NewOpenAIAdapter(url, "test", nil) },
		"anthropic": func(url string) llm.Client { return llm.NewAnthropicAdapter(url, "test", nil) },
		"gemini":    func(url string) llm.Client { return llm.NewGeminiAdapter(url, "test", nil) },
	} {
		t.Run(name, func(t *testing.T) { t.Parallel(); llmtest.RunHTTPConformance(t, makeClient) })
	}
}
