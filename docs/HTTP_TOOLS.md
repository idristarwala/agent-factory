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

## Configure text evidence for catalog-swarm

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

Scalar record properties are available to path and query parameters. The graph node ID supplies `id`. Relative key paths resolve from the configuration file's directory. The loader supports read-only GET requests. With the variable unset, the command uses the seeded records alone.

For visual evidence, set `context.media` to a field name in the entity tool's JSON object. For example, `media: attachments` reads an array of image URLs from `attachments`. Use an empty array for a record with no images. Both models receive the same downloaded bytes. Mutation evidence records each image's source tool, position and SHA-256 digest.

The runner accepts up to 10 PNG, JPEG or WebP images per record and 512 KiB per image. A missing field, unsupported content, failed download or exceeded limit fails evidence collection. Image requests use the public-network and host restrictions above and carry no tool authentication headers.
