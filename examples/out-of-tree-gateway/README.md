# Bring your own gateway

This example has its own Go module, `example.org/customgateway`. It uses the public `github.com/idristarwala/agent-factory/llm` package. Its local `replace` directive tests the checkout beside it. In an independent repository, require the Agent Factory module at the version you have tested.

Implement `llm.Client` for a new wire protocol or wrap `llm.NewOpenAIAdapter` for an OpenAI-compatible service. A client implements `Chat` and `Stream`. A gateway without streaming support returns `llm.ErrStreamingUnsupported` from `Stream`. Supply a constructor to `llm.Register` if the host application selects adapters by name.

The example test starts a local HTTP service, points the client at it and calls `llmtest.RunConformance(t, harness)`. Run `make conformance` from the repository root to test the in-tree adapters and the independent example under the race detector.

Call `llmtest.RunHTTPConformance(t, newClient)` with a constructor that uses the supplied fixture URL. The shared cases exercise HTTP failures for chat and streaming, malformed chat responses, request deadlines and in-flight cancellation. The fixtures require only loopback access.
