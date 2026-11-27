package example

import (
	"testing"

	"github.com/idristarwala/agent-factory/llm"
	"github.com/idristarwala/agent-factory/llm/llmtest"
)

func TestCorpGatewayPassesConformance(t *testing.T) {
	h := llmtest.NewProtocolHarness(t, "openai", func(url string) llm.Client {
		return NewCorpGateway(url, "fixture", map[string]string{"X-Consumer-Id": "conformance"}).Client()
	})
	llmtest.RunConformance(t, h)
}
