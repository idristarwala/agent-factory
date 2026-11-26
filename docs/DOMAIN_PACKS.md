# Domain packs

A pack defines how a problem domain is handled.

## Built-in

| Pack | Focus |
|------|-------|
| catalog | Titles, descriptions, attributes, variants |
| pricing | Prices, margins, promotions |
| inventory | Stock, replenishment, allocation |
| merchandising | Assortment, placement |

## Adding a pack

Implement a function that returns a `domain.Pack` and register it in `All()`.

```go
func MyPack() Pack {
    return Pack{
        Name: "my-domain",
        WorkerRoles: []string{"specialist"},
        DefaultRules: []string{"Rule one"},
    }
}
```

The Steward selects a pack based on the goal text.
