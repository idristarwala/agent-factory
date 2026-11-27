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

## Remaining for full native support

- Full parameter mapping (max_completion_tokens, reasoning_budget) wired end-to-end in all paths
- Streaming event parsing refined for Gemini/Anthropic content blocks (OpenAI delta parsing is solid)
- Usage recording into `internal/obs` on every call (RecordingClient exists)

See TRACKING.md for status.

## Recent advances

- Anthropic and Gemini tool-call round-trip conformance tests.
- TokenRefresher now injects the refreshed key via `SetAPIKey` on adapters.
- Basic SSE streaming implemented for OpenAI, Gemini, and Anthropic adapters.
