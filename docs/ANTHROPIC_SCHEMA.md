# Anthropic structured responses

The environment-only Anthropic client requests a forced tool call whose input schema is the worker’s reply schema. It extracts the returned tool input as structured content. [Environment gateway](../internal/configuredgateway/environment.go)

A logical route configured with `LLM_GATEWAY_CONFIG` and `protocol: anthropic` uses the shared Messages adapter. It sends `output_config.format` with a JSON Schema and validates the returned text against that schema. Select an endpoint and model that support the configured request format. [Configured adapter](../internal/llm/anthropic_adapter.go) · [Gateway configuration](ENTERPRISE_GATEWAY.md)
