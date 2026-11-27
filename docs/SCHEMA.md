# Worker output schema

A factory definition specifies a record type, editable fields and their JSON types. Its optional `required` list identifies fields that must appear in a proposal. The worker constructs a JSON Schema for each unit, attaches it to the model request, then parses and validates the returned values. Properties outside the editable set fail validation. [Example factory definition](../examples/service-desk/factory.json)

OpenAI-compatible gateways receive a JSON Schema response format. Gemini receives a structured JSON response configuration. For Anthropic, the request format depends on the selected adapter. [Anthropic schema handling](ANTHROPIC_SCHEMA.md)

The shared store uses versioned records for factory definitions, rules, agents, runs and memory. Adding an application field to a factory’s record type changes its definition revision. [Storage contract](CONFIGURATION.md#storage-contract-for-api-changes)
