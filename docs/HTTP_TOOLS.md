# Generic HTTP Tools (Issue #12)

## Current support

- HTTPTool with path templating, headers, timeout, side-effect
- SimpleMap for $.field extraction
- FetchImages for multimodal evidence
- ContextRecipe + ExecuteRecipe that runs tools before the worker
- Basic grants check helpers
- MCP HTTP stub
- Tests against httptest

## Remaining

- Full YAML loader that registers tools with zero Go code
- Complete auth (header, basic, oauth)
- Retries with backoff
- Cache with TTL
- Full JSONPath
- Image parts wired into gateway adapters
- Grants enforced on every Call
- MCP streamable HTTP fully implemented

The core evidence path is now available. Config-driven connection of REST systems is progressing.
