# Secrets

All secrets are read from a single directory.

## Location

Default: `/etc/secrets`

Override: set `SECRETS_DIR` to any path.

## Files

| File | Used by |
|------|---------|
| `gemini-api-key` | Gemini gateway |
| `llm-api-key` | OpenAI-compatible gateway |
| `bigquery-sa.json` | BigQuery tool (service account) |
| `mcp-token` | MCP transports that need auth |

## Behavior

- Values are cached and re-read when the file mtime changes (no restart needed for rotation).
- If a file is missing, the loader falls back to an environment variable of the same name (uppercased, `-` → `_`).
- SDKs that require a file path (e.g. BigQuery) use `Loader.Path(name)`.

## Example

```bash
export SECRETS_DIR=/var/run/secrets
echo -n "$GEMINI_KEY" > $SECRETS_DIR/gemini-api-key
cp sa.json $SECRETS_DIR/bigquery-sa.json
```
