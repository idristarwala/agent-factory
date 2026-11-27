# Agent model requests

Workers, judges and stewards call the `models.Gateway` interface. A request carries the role, model tier, prompt, optional structured output schema and any inline image evidence. The production entry points resolve the gateway at startup. [Interface](gateway.go)

Set `LLM_GATEWAY_CONFIG` for logical model routing, retry policy and protocol adapters. Environment-only deployments select the provider with `LLM_PROVIDER` and can route the judge separately. See the operator reference for supported settings and defaults. [Configuration](../../docs/CONFIGURATION.md) · [Enterprise gateway](../../docs/ENTERPRISE_GATEWAY.md)
