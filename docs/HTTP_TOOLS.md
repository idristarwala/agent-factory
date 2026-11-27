# Generic HTTP Tools (Issue #12)

## Completed

- HTTPTool with path templating, headers, timeout, side-effect, size limit
- SimpleMap for field extraction
- FetchImages for media
- ContextRecipe + ExecuteRecipe
- Wired into ContinuousRunner via GatherEvidence
- Grants helpers
- ImagePart type for multimodal
- MCP HTTP stub
- Tests

## Remaining (minor)

- Full YAML zero-code registration of tools
- Complete retries + cache with TTL
- Grants checked on every Call inside the runner
- Image parts fully wired into gateway request bodies
- MCP streamable HTTP complete implementation

The evidence path is usable. A REST system can be connected and its data gathered before the worker and judge run.
