# Out-of-tree gateway example

This directory shows how a user repo can depend on the agent-factory module and supply its own gateway adapter that passes the conformance suite.

```go
import "github.com/idristarwala/agent-factory/internal/llm"

type MyHarness struct{ c *CorpGateway }

func (h MyHarness) Client() llm.Client { return h.c.Client() }

func TestMyGateway(t *testing.T) {
    // point at an httptest server that speaks OpenAI-compatible
    h := MyHarness{c: NewCorpGateway(srv.URL, "k", map[string]string{"X-Consumer-Id": "test"})}
    llm.RunConformance(t, h)
}
```

The adapter reuses `llm.OpenAIAdapter` (or Anthropic/Gemini) so path, headers, body shape, tool calls, and streaming already match the suite.
