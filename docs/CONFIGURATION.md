# Configuration

## Secrets

All credentials come from one directory.

| Setting | Default | Override |
|---------|---------|----------|
| Directory | `/etc/secrets` | `SECRETS_DIR` |

### Files

| File | Used by |
|------|---------|
| `gemini-api-key` | Gemini gateway |
| `llm-api-key` | OpenAI-compatible gateway |
| `bigquery-sa.json` | BigQuery service account |
| `bigquery-wif.json` | BigQuery Workload Identity |
| `mcp-token` | MCP transports |

Values are cached and re-read when the file mtime changes.

## Environment variables

| Variable | Purpose | Example |
|----------|---------|---------|
| `LLM_PROVIDER` | Which adapter to use | `gemini`, `openai` |
| `LLM_BASE_URL` | Custom OpenAI-compatible endpoint | `https://api.groq.com/openai/v1` |
| `GEMINI_CHEAP_MODEL` | Model for workers | `gemini-3.5-flash-lite` |
| `GEMINI_FRONTIER_MODEL` | Model for Judge / Steward | `gemini-3.5-flash-lite` |
| `SECRETS_DIR` | Secrets location | `/var/run/secrets` |

## Model tiers

| Tier | Who may use it | Typical model |
|------|----------------|---------------|
| cheap | Workers | Flash / lite |
| frontier | Steward, Judge | Pro or frontier |
| council | Judge (optional) | Multiple models |

Workers are blocked from frontier and council by the router.

## Domain packs

Packs live in `internal/domain`. Each pack supplies Name, NodeTypes, WorkerRoles, and DefaultRules. The Steward selects a pack by matching keywords in the goal.

## Persistence

| Mode | When |
|------|------|
| memory | Local, tests, demos |
| sql | Production (Postgres-compatible) |
