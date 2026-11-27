# Remaining Minor Items

These are optional polish items. The core factory is functional without them.

## go.sum

After adding `github.com/lib/pq` for the Postgres store, run:

```bash
go get github.com/lib/pq
go mod tidy
```

Then commit the updated `go.sum`.

## YAML pack loader

`LoadYAMLOrJSON` and `ParseSimpleYAML` provide a basic path. For full YAML support, add a library such as `gopkg.in/yaml.v3`.

## Provider-enforced structured output

`BuildResponseSchema` produces a schema that can be sent as `response_format` (OpenAI) or `responseSchema` (Gemini). Wire it into the request builders for stricter enforcement.

## UI and observability

Logger, metrics, and audit log exist. Wire them into the main runner loop and connect the React app to live decision endpoints when ready.

All high-impact correctness, packaging, gateway, and tool items are complete.
