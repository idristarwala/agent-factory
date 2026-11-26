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

## Remaining for full native support

- Tool-call parsing in adapters
- Auth token refresh / rotation reload
- Full parameter mapping (max_completion_tokens, reasoning_budget) wired end-to-end
- Streaming native (currently reassembly helpers + one-shot fallback)
- Usage recording into `internal/obs` on every call

See TRACKING.md for status.
