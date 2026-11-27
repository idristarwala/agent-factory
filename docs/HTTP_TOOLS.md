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

For bearer, custom-header or basic authentication on HTTP and MCP tools, choose one credential reference per tool. Set `auth.key_file` to a mounted file or `auth.key_env` to a process environment variable name, for example `auth: {type: bearer, key_env: RECORDS_API_TOKEN}`. Tool validation checks the reference syntax offline. Readiness checks for a nonempty value. Each outgoing request reads the configured source again. A deployment that injects variables at process startup must restart the process to pick up an external rotation.

For HTTP Basic authentication, set `auth.type: basic`, `auth.username` and a password from the selected credential source. Credentials are reread for each HTTP attempt. The timeout applies per attempt. Cancellation interrupts retry waits. Only successful GET responses without a request body are cached. A custom CA adds trust for that tool's TLS connection. Destination restrictions still apply.

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

## Generate tool definitions from OpenAPI

Run `factory tools import openapi.yaml > draft-tools.yaml` to generate a reviewable `tools` mapping from GET operations. The CLI accepts OpenAPI 3.0 or 3.1 JSON and YAML with local references. It resolves server overrides and scalar path and query parameters. Required parameters are checked before a request. Non-GET operations are listed on stderr. An unsupported selected operation or parameter returns a failing exit code.

For an authenticated operation, select its operation ID and choose one credential source.

```sh
factory tools import -operation getTicket -key-file secrets/tickets-token openapi.yaml > draft-tools.yaml
```

To use a deployment environment variable, pass its name literally. The variable can be unset while importing or validating.

```sh
factory tools import -operation getTicket -key-env TICKETS_API_TOKEN openapi.yaml > draft-tools.yaml
```

The importer supports a single bearer or header API-key scheme for that operation. Use `-server https://tickets.example.com/v1` for a relative, templated, or ambiguous server. The generated file records the variable name or the configured file path. Relative credential paths resolve from the deployed tool file. Review the destination and parameters, add a context recipe and an explicit pack grant as shown above, then point `FACTORY_EVIDENCE_CONFIG` at the reviewed file. Import runs offline and writes only to stdout.

## Validate an evidence tool file

Run `factory validate --tools draft-tools.yaml` after importing or editing a tool. Pass multiple YAML or JSON paths to validate them together. The CLI checks the published schema, HTTP and MCP method restrictions, required authentication fields, JSONPath expressions, time limits and names referenced by the context recipe. An invalid file exits nonzero.

Draft validation works offline with an empty grant list and before deployment secrets or CA files exist. At deployment the runtime checks the destination allowlist, loads configured CAs, and authorizes the evidence recipe for the selected pack. Credential files are read at request time. [Evidence tool file schema](../schemas/evidence-tools.schema.json)

## Share tool definitions across factories

Publish a YAML or JSON file containing a single `tools` mapping in a Git repository or an OCI artifact. Vendor the reviewed file in the consuming workspace. For a Git submodule, commit the submodule revision and check it out recursively in CI. For OCI, use your existing artifact client to pull the reviewed artifact by digest before validation. The factory reads local files and can run offline after checkout.

Compute the definition file's digest with `sha256sum vendor/records/tools.yaml`. Add the following fields to the workspace's existing `factory.yaml`, replacing the placeholder with the 64 lowercase hex digits. Each package must use distinct tool names. Define the context recipe and grants in the consuming factory.

```yaml
tool_packages:
  - path: vendor/records/tools.yaml
    sha256: "<digest of the exact file bytes>"
evidence:
  context: {entity: lookup}
  grants:
    - {pack: review, tools: [lookup, propose], side_effects: [read-only]}
```

Run `factory validate factory.yaml` alongside pack validation. Set `FACTORY_EVIDENCE_CONFIG=factory.yaml` for the continuous runner or `FACTORY_SOURCE_CONFIG=factory.yaml` for API sampling and the connector. Mount the manifest and package files with the same directory layout. Runtime loading verifies the digest before interpreting definitions. Relative credential and CA paths in those definitions resolve from the manifest directory. Review destinations and credential references before granting a tool. To upgrade, commit the new vendored revision and file digest together and restart the consuming process.

## Bind proposal arguments in a reviewed pack

A file pack can specialize an allowed proposal tool with fixed JSON values. For example, two factories can use the same update tool with different tenant and dry-run settings.

```yaml
tools_allowed: [update-record]
proposal_arguments:
  update-record:
    tenant: team-a
    dry_run: true
```

Publish and approve the pack, pin its approval revision in the factory's `pack_release`, and select `update-record` as its `propose_tool`. The workspace pack must also grant the tool's side-effect class. At run creation the engine checks the fixed argument names against the tool’s record and changes argument names. A collision rejects the run. The worker supplies editable record fields. The engine binds those changes, the record ID and the reviewed constants into the proposal that the judge and human reviewer receive.

These values appear in review history. Use credential references in the transport configuration for authentication. To change the constants, approve a new pack release and pin it in a new factory revision. Already queued runs retain their reviewed values. [Pack release review](RELEASE.md)
