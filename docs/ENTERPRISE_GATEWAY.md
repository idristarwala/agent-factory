# Enterprise Gateway Support (Issues #1–#8, #11)

Declarative config is in `internal/llm/config.go`.

Example gateway.json:
```json
{
  "name": "corp",
  "base_url": "https://llm-gateway.example.com/v1",
  "auth_type": "bearer",
  "key_file": "/run/secrets/llm_key",
  "headers": {
    "X-Consumer-Id": "my-app",
    "X-Env": "prod"
  },
  "ca_file": "/etc/ssl/corp-ca.pem",
  "proxy": "none"
}
```

## Current capabilities

- OpenAI, Gemini, and Anthropic POST adapters (no `os.Setenv`)
- Key from file (`LoadKey`)
- Custom CA and proxy control (`NewHTTPClient`)
- Typed errors (RATE_LIMITED, AUTH, BAD_REQUEST) with retryable flag
- Basic multi-protocol conformance against `httptest` servers
- Extended conformance: auth errors, concurrency, SSE reassembly
- Tool-call parsing and round-trip tests for all three protocols
- TokenRefresher with API key injection into adapters
- Basic SSE streaming for OpenAI, Gemini (`streamGenerateContent`), and Anthropic
- Multi-protocol SSE text delta extraction (`extractTextDelta`) covering OpenAI deltas, Anthropic `content_block_delta`, and Gemini `candidates[].content.parts[].text`

## Remaining for full native support

- Full parameter mapping (max_completion_tokens, reasoning_budget) wired end-to-end in all paths
- Usage recording into `internal/obs` on every call (RecordingClient exists)

See TRACKING.md for status.

## Recent advances

- Anthropic and Gemini tool-call round-trip conformance tests.
- TokenRefresher now injects the refreshed key via `SetAPIKey` on adapters.
- Basic SSE streaming implemented for OpenAI, Gemini, and Anthropic adapters.
- SSE parsers now handle Anthropic and Gemini event shapes in addition to OpenAI (tests added).

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
