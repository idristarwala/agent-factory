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

## Concrete test

`adapter_test.go` in this directory runs `llm.RunConformance` against a real `httptest.Server`. Run it with:

```bash
go test ./examples/out-of-tree-gateway/ -v
```

## Using FakeServer

Prefer `llm.NewFakeServer()` over a hand-rolled `httptest.Server` when you need recorded bodies or cancel testing:

```go
fs := llm.NewFakeServer()
defer fs.Close()
gw := NewCorpGateway(fs.URL, "k", nil)
llm.RunConformance(t, harness{gw.Client()})
// fs.LastBody() contains the JSON the adapter sent
```
