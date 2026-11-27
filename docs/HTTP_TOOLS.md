# HTTP Tools and Context Recipe

Generic config-driven HTTP tools connect any REST system.

## Context recipe

```go
recipe := tools.ContextRecipe{
    Entity:     "get-record",
    Neighbours: "find-related",
}
runner.SetTools(registry, recipe)
runner.SetGrants("review", []tools.Grant{{
    Pack: "review", Tools: []string{"propose", "get-record", "find-related"},
    SideEffects: []string{"read-only"},
}})
```

The ContinuousRunner gathers the recipe results before the worker runs, stores them on the UnitOfWork as Evidence, injects them into the worker prompt, and also attaches them to the mutation so the Judge sees them.

Each evidence entry names the tool that returned it. A missing grant, failed lookup, or response larger than 512 KiB fails the unit before model execution.

The default HTTP client permits public destination addresses and rejects redirects. The tool URL must have a fixed origin. Set `FACTORY_HTTP_ALLOWED_HOSTS` to a comma-separated list of exact hostnames when operators accept tool definitions from another party. An explicitly empty list disables outbound tool destinations. Trusted integrations that need private services can inject a dedicated HTTP client in Go.

## Configure an evidence source

Set `FACTORY_EVIDENCE_CONFIG` to an operator-managed YAML file before running `catalog-swarm`. The command checks grants using the active pack name. It loads the file at startup, then gathers the declared sources before each worker proposal and includes them in judge review.

```yaml
tools:
  get-record:
    method: GET
    url: https://records.example.com/entities/{id}
    auth:
      type: bearer
      key_file: secrets/records-token
  find-related:
    method: GET
    url: https://records.example.com/related
    query: [id]
context:
  entity: get-record
  neighbours: find-related
grants:
  - pack: catalog
    tools: [get-record, find-related]
    side_effects: [read-only]
```

For `continuous-swarm`, set `FACTORY_EVIDENCE_CONFIG` to the same file format and `FACTORY_PACK` to the pack that will run. Include `propose` and each source tool in that pack’s read-only grants. The command checks all grants before starting workers. For example, a pack named `review` that uses the two sources above can run with this grant and command.

```yaml
grants:
  - pack: review
    tools: [propose, get-record, find-related]
    side_effects: [read-only]
```

```sh
FACTORY_EVIDENCE_CONFIG=./evidence.yaml FACTORY_PACK=review go run ./cmd/continuous-swarm "Correct the record from its source"
```

Scalar record properties are available to path and query parameters. The graph node ID supplies `id`. Relative key paths resolve from the configuration file's directory. The loader supports GET and explicitly read-only POST requests. With the variable unset, the command uses the seeded records alone.

For visual evidence, set `context.media` to a field name in the entity tool's JSON object. For example, `media: attachments` reads an array of image URLs from `attachments`. Use an empty array for a record with no images. Both models receive identical image parts. Up to ten images pass through unchanged. Records with 11–90 images are tiled in source order into at most ten JPEG contact sheets, with a sheet, row and column cited for each original. Mutation evidence records each image's source tool, position and SHA-256 digest.

The runner accepts up to 90 PNG, JPEG or WebP sources per record and 512 KiB per source. Tiling decodes sources up to 16 megapixels and fits each into a 320-pixel square while preserving its aspect ratio. A missing field, unsupported content, failed download or exceeded limit fails evidence collection. Image requests use the public-network and host restrictions above and carry no tool authentication headers.

A search endpoint that accepts JSON can use the following tool definition. Record parameters in quoted body values preserve quotes, backslashes and line breaks across retries.

```yaml
find-related:
  method: POST
  side_effect: read-only
  url: https://records.example.com/search
  body: {query: "{label}", limit: 25}
  auth: {type: header, name: X-Records-Key, key_file: secrets/search-key}
  tls: {ca_file: certs/records-ca.pem}
  proxy: none
  limits: {timeout: 5s, retries: 2, cache_ttl: 0s}
```

For HTTP Basic authentication, set `auth.type: basic`, `auth.username` and a password in `auth.key_file`. Credentials are reread for each HTTP attempt. The timeout applies per attempt. Cancellation interrupts retry waits. Only successful GET responses without a request body are cached. A custom CA adds trust for that tool's TLS connection. Destination restrictions still apply.

## Select fields from a response

A tool can use JSONPath to select the part of a JSON response that the worker and judge should see. For example, a support system can return a request inside an envelope with attachments:

```yaml
response:
  root: $.data.request
  map:
    subject: $.summary
    attachments: $.files[*].url
```

Place `response` inside the tool definition. Paths in `map` start at the selected root. The root defaults to the entire document. Omit `map` to return the selected root directly, or omit `response` to keep the original response body, including plain text. A singular path selects one value and fails if that value is absent. A wildcard or filter returns an array, including an empty array when nothing matches. Use `context.media: attachments` to pass the selected image URLs through the existing image evidence pipeline.

## Read evidence from an MCP server

Set `mcp_tool` to a fixed tool on a Streamable HTTP endpoint. The loader requires an explicit read-only declaration. The pack grant names the local tool entry. Use the same entry in the context recipe.

```yaml
tools:
  lookup:
    url: https://requests.example.com/mcp
    mcp_tool: read_request
    side_effect: read-only
    body: {id: "{id}"}
    auth: {type: bearer, key_file: secrets/requests}
    limits: {timeout: 10s, retries: 1}
    response:
      map: {subject: $.subject}
context: {entity: lookup}
grants:
  - {pack: desk, tools: [lookup, propose], side_effects: [read-only]}
```

Each lookup opens an MCP session, calls the configured tool with the rendered body as its arguments and closes the session. The client accepts JSON and server-sent events. Structured tool results can use the JSONPath mapping above. Text results pass through directly when no mapping is configured. Tool errors fail the evidence step. Session requests use the tool’s credentials, trusted CA and destination restrictions. The timeout covers the lookup and retries bound stream reconnection attempts.

## Review external record changes

A workspace factory can route its worker’s proposed field changes to a named external tool for review. For example, configure the following three versioned definitions through `/api/definitions/{kind}/{id}`. Each row shows the `Data` in the definition edit. Send `Expected: 0` to create it, or the current head revision to edit it.

| Definition | Data |
| --- | --- |
| `tool/update-case` | `{"side_effect":"reversible","record_argument":"case_id","changes_argument":"patch"}` |
| `pack/service-desk` | `{"grants":[{"pack":"service-desk","tools":["update-case"],"side_effects":["reversible"]}]}` |
| `factory/service-desk` | `{"record_type":"request","fields":{"assigned_team":"string"},"propose_tool":"update-case"}` |

At run creation the factory checks the grant against the tool’s side-effect class and pins both definition revisions. Each worker patch becomes a proposed tool call with the record ID and validated changed fields bound to the configured argument names. The judge evaluates that exact call. The Decisions screen and API expose its arguments alongside the evidence. A human approval or rejection is stored in the run history. Approval can apply the workspace record update according to observe mode. Delivery to the external service is separate. Preparing and reviewing a proposal sends zero requests to that service.
