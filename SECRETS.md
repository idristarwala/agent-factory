# Supplying credentials

| Consumer | Where the value comes from | When changes take effect |
| --- | --- | --- |
| Environment-only model provider | The running process’s provider environment variables | Restart with the new environment. |
| Logical gateway route | `auth.key_file` in gateway configuration | Authenticated requests read the configured key file. |
| HTTP and MCP evidence tools | Secret files referenced by the tool configuration | New authenticated requests use the tool’s auth implementation. Cached responses can satisfy repeated reads. |
| GitHub goal imports | `factory-github-token` under `SECRETS_DIR`, falling back to `FACTORY_GITHUB_TOKEN` | New imports through an authenticated API check the credential and repository allowlist. |
| API bearer authentication | `API_TOKEN` in the API process environment | Restart with the new environment. |

The centralized secret loader defaults to `/etc/secrets`. It trims file contents and caches by modification time. When a file is unavailable, it checks the uppercase environment name with hyphens replaced by underscores. A gateway or tool using an explicit file path follows its own configuration. Keep credentials outside the repository and mount secret files read-only in containers. [Authentication](AUTH.md) · [Deployment settings](docs/CONFIGURATION.md)
