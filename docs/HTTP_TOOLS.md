# HTTP Tools and Context Recipe

Generic config-driven HTTP tools connect any REST system.

## Context recipe

```go
recipe := tools.ContextRecipe{
    Entity:     "get-product",
    Neighbours: "search-siblings",
}
runner.SetTools(registry, recipe)
```

The ContinuousRunner gathers the recipe results before the worker runs, stores them on the UnitOfWork as Evidence, injects them into the worker prompt, and also attaches them to the mutation so the Judge sees them.

This closes the accuracy gap identified in the field test: the model now sees the same evidence the worker and judge use.

The default HTTP client permits public destination addresses and rejects redirects. The tool URL must have a fixed origin. Set `FACTORY_HTTP_ALLOWED_HOSTS` to a comma-separated list of exact hostnames when operators accept tool definitions from another party. An explicitly empty list disables outbound tool destinations. Trusted integrations that need private services can inject a dedicated HTTP client in Go.
