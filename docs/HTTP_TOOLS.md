# Generic HTTP Tools (Issue #12)

`internal/tools/http.go` provides `HTTPTool`, a config-driven tool that can call any REST endpoint.

## Current support

- Path templating (`{id}` replacement)
- Method, headers, timeout
- Side-effect classification (read-only by default)
- Response size limit
- Basic httptest coverage

## Remaining for full acceptance

- YAML config loader that registers tools with no Go code
- Auth from key files (bearer, header, basic)
- Custom CA and proxy
- Retries on 429/5xx
- JSONPath response mapping
- Cache
- Grants enforced at call time
- Context recipe that runs tools before worker/judge
- Image fetching for multimodal
- MCP over HTTP

The skeleton is in place so evidence can be fetched without hard-coding Go.
