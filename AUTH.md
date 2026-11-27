# Authentication

| Connection | Runtime configuration |
| --- | --- |
| Harness or operator UI to API | Configure the required role tokens on the server. Send `Authorization: Bearer <token>` on API requests. `/health` is public. |
| Native model provider | Environment-only routes read the provider credentials from the process environment. Independent judges use the credential for their selected provider. |
| Configured model gateway | In `LLM_GATEWAY_CONFIG`, choose `none`, `bearer`, `google_api_key` or `anthropic_api_key`. Authenticated gateways read their configured key file. |
| Configured REST or MCP tool | Specify the auth method and file references in the tool YAML. The read-only MCP transport uses the tool’s HTTP transport configuration. |
| GitHub goal attachment | Public imports use anonymous GitHub access. With authenticated Factory API access, a repository on the configured allowlist can use the server’s GitHub credential. |

## Workspace API roles

All three roles can read workspace data, including memory, artifacts, retained browser captures, and definition history. Tokens apply across the workspace. Configure only the roles your deployment needs, using a distinct secret for each. Clients send their own role token in the bearer header.

| Role | Server variable | Additional permissions |
| --- | --- | --- |
| Viewer | `FACTORY_VIEWER_TOKEN` | GET requests. |
| Operator | `FACTORY_OPERATOR_TOKEN` | Start and cancel runs, review decisions, write memory, upload or import chat attachments, use ordinary chat, inspect sources, draft source mappings, and execute configured imports. Review feedback follows the run’s replay-gate setting. |
| Admin | `API_TOKEN` | All operations, including settings, agent and rule definitions, direct feedback, approved source mappings, and creating or approving factory definitions through chat. |

A configured token enables authentication even if the admin token is unset. Missing or invalid credentials return 401. Insufficient privileges return 403. Duplicate tokens prevent startup. With all three unset the API retains its local full-access mode. Health and CORS preflight are public.

For exact keys, file lookup rules, examples and revision boundaries, see [configuration](docs/CONFIGURATION.md), [gateway configuration](docs/ENTERPRISE_GATEWAY.md), [HTTP tools](docs/HTTP_TOOLS.md) and [secrets](SECRETS.md).
