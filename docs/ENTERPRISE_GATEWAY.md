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
