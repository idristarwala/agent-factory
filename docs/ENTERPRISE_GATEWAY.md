# Enterprise Gateway Support (Issue #11)

The declarative config skeleton is in `internal/llm/config.go`.

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

Adapters can call `ApplyExtraHeaders` to merge static headers.

Remaining for full native support:
- Per-model protocol routing
- Key reload from file
- Custom CA and proxy in http.Client
- Parameter mapping (max_completion_tokens, reasoning_budget)
- Usage recording into obs
- Typed retries on 429/5xx

These continue as incremental work.
