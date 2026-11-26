# Authentication

Every tool and client supports the auth methods that provider offers.
Secrets always come from the centralized loader (`SECRETS_DIR`, default `/etc/secrets`).

## BigQuery

| Method | Config | Secret file |
|--------|--------|-------------|
| Application Default Credentials | `auth: adc` | none (uses metadata / gcloud) |
| Service account JSON | `auth: service_account` | `bigquery-sa.json` |
| Workload Identity Federation | `auth: workload_identity` | `bigquery-wif.json` |
| Service account impersonation | `auth: impersonation` | none (uses ADC + `impersonate` email) |

## LLM gateways

| Provider | Methods |
|----------|---------|
| Gemini | API key (`gemini-api-key`) |
| OpenAI-compatible | API key (`llm-api-key`) or Bearer token |
| Enterprise gateways | OAuth2 client credentials, custom headers, file-rotated secrets |

## MCP

| Method | Secret |
|--------|--------|
| Bearer token | `mcp-token` |
| Custom header | configured per transport |

## Adding a new tool

1. Declare the auth methods the provider supports.
2. Read any required files or tokens via `secrets.New()`.
3. Never hard-code paths or read env vars directly for credentials.
