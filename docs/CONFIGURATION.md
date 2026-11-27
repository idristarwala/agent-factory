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

### Independent judge model

Set `LLM_PROVIDER` for the worker and other ordinary calls. To route only judge calls to a different model, set both `LLM_JUDGE_PROVIDER` (gemini or openrouter) and `LLM_JUDGE_MODEL`. The judge uses its provider’s dedicated runtime credential, `GEMINI_API_KEY` or `OPENROUTER_API_KEY`. Run history records the role, served model and tokens for each worker and judge call.

For a Gemini worker with a GPT-4o judge, use `LLM_PROVIDER=gemini`, `LLM_JUDGE_PROVIDER=openrouter`, and `LLM_JUDGE_MODEL=openai/gpt-4o`. For the reverse, use `LLM_PROVIDER=openrouter`, `LLM_CHEAP_MODEL=openai/gpt-4o`, `LLM_JUDGE_PROVIDER=gemini`, and `LLM_JUDGE_MODEL=gemini-3.5-flash-lite`. With both credentials available, `make test-mixed-judges` executes both configurations through human approval.
