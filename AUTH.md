# Authentication

| Connection | Runtime configuration |
| --- | --- |
| Harness or operator UI to API | Set `API_TOKEN` on the server. Send `Authorization: Bearer <token>` on API requests. `/health` is public. |
| Native model provider | Environment-only routes read the provider credentials from the process environment. Independent judges use the credential for their selected provider. |
| Configured model gateway | In `LLM_GATEWAY_CONFIG`, choose `none`, `bearer`, `google_api_key` or `anthropic_api_key`. Authenticated gateways read their configured key file. |
| Configured REST or MCP tool | Specify the auth method and file references in the tool YAML. The read-only MCP transport uses the tool’s HTTP transport configuration. |
| GitHub goal attachment | Public imports use anonymous GitHub access. With authenticated Factory API access, a repository on the configured allowlist can use the server’s GitHub credential. |

For exact keys, file lookup rules, examples and revision boundaries, see [configuration](docs/CONFIGURATION.md), [gateway configuration](docs/ENTERPRISE_GATEWAY.md), [HTTP tools](docs/HTTP_TOOLS.md) and [secrets](SECRETS.md).
