# Enterprise gateways

## Route factory requests by logical model

Set `LLM_GATEWAY_CONFIG` to a YAML or JSON routing file when launching the API, catalog swarm, goal command, or model smoke test. With the variable unset, these commands use their existing provider environment settings. A minimal enterprise configuration follows.

```yaml
gateways:
  corp:
    base_url: https://gateway.example.com/v1
    auth: {type: bearer, key_file: /run/secrets/gateway-token}
    headers: {X-Consumer-Id: factory}
    tls: {ca_file: /etc/ssl/corp-ca.pem}
    proxy: env
models:
  cheap: {gateway: corp, protocol: gemini, model: worker-model, reasoning_budget: 0}
  frontier: {gateway: corp, protocol: openai, model: reasoning-model, token_param: max_completion_tokens}
  judge: {gateway: corp, protocol: anthropic, model: review-model}
```

Define the model aliases used by your workload. Ordinary factory calls select their tier. Judge calls prefer the optional `judge` entry. Each model may add static headers or override a gateway header. Each gateway shares one HTTP transport across its models. Relative key and CA paths resolve beside the configuration file. Bearer, Google API key, and Anthropic API key authentication read the key file on each request so a rotated credential takes effect without restarting. Set `auth.type` to `bearer`, `google_api_key`, `anthropic_api_key`, or `none`.

### Wrapped OpenAI requests

For an endpoint that expects OpenAI parameters under a named key and returns standard OpenAI responses, configure the envelope on its model route. The same envelope applies to streaming and one-shot requests. The exact string `{model}` in a template value resolves to the configured wire model.

```yaml
models:
  frontier:
    gateway: corp
    protocol: openai-envelope
    model: reasoning-model
    envelope:
      params_key: params
      template: {operation: chat, model: "{model}", version: "2025-01-01"}
```

### Retry and usage reporting

Configured routes emit a structured `gateway call` log record for each attempt, including the requested or served model, elapsed milliseconds, status, error type, and input, output and total tokens. Streaming usage is recorded when the stream ends or closes. Prompts, response text and credentials are excluded.

An HTTP 429 or 5xx can trigger up to two retries with 250 ms and 500 ms backoff. A longer `Retry-After` takes precedence. Delays over five seconds return the typed provider error to the caller immediately. Cancellation interrupts the wait. Once streaming begins, an error ends that stream.

## Independent judge panels

For a continuous run, give each judge its own logical model route. For example, add two entries under the routing file’s `models` key. Replace the provider-specific model IDs with models available to your gateway.

```yaml
  reviewer-a: {gateway: corp, protocol: openai, model: first-model}
  reviewer-b: {gateway: corp, protocol: openai, model: second-model}
```

Set `FACTORY_JUDGE_ROUTES=reviewer-a,reviewer-b` on the CLI and set `judge_evaluations` to 2 in workspace Settings. Set `judge_approvals` to the number of approving votes required. A new workspace defaults to one evaluation and one required approval. A workspace with a saved two-vote policy keeps that threshold. Configure two distinct routes before starting it. An environment-only gateway supports one judge.

The CLI checks the route names and distinct configured model IDs before work starts. The decision journal records each judge’s verdict and identifies its model in the feedback. Run usage records each route with its returned model ID. Duplicate model identities in a panel fail the review.

### Environment and YAML routes

Both configuration forms use the same protocol adapters, typed gateway errors, bounded retries for HTTP 429 and 5xx, and metadata-only usage logs on stderr. Agent requests attach the work unit ID to each attempt's trace. CLI stdout remains available for result records.

For optional object fields, environment-selected OpenAI-compatible routes send `strict: false`. YAML routes default to strict JSON Schema and accept `schema_mode: non_strict` on OpenAI and OpenAI-envelope routes. Environment-selected Anthropic routes force the `attribute_update` tool and validate its input. YAML Anthropic routes default to native structured output and accept `schema_mode: forced_tool` for models requiring tool-based output. Both modes validate the returned content locally.
