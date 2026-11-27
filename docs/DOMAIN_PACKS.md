# Domain packs

A pack defines how a problem domain is handled.

## Example pack files

| Pack | Focus |
|------|-------|
| [catalog](../packs/catalog/pack.yaml) | Titles, descriptions, attributes, variants |
| [pricing](../packs/pricing/pack.yaml) | Prices, margins, promotions |
| [inventory](../packs/inventory/pack.yaml) | Stock, replenishment, allocation |
| [merchandising](../packs/merchandising/pack.yaml) | Assortment, placement |

## Configure your own factory

For a runtime factory, define the record type and editable fields and import the JSON definition. For a file-based pack, supply a YAML or JSON document with worker roles, rules and allowed tool names. The service-desk example and pack schema show the two contracts. [Run guide](RUNNING.md) · [Pack schema](../schemas/pack.schema.json)

The continuous CLI selects its external evidence grant with `FACTORY_PACK`. Workspace goals explicitly select a factory and can pin its revision.

Validate a pack with `factory validate path/to/pack.yaml`. In a Go integration, load it with `pack.Load(path)` and set `steward.PlanningPack` before planning without a graph. The planner creates one unit per declared worker role. With no selected pack, it creates one general worker unit. Graph-backed planning continues to create work per pending record. The catalog demo embeds its pack file so a compiled binary can run outside the checkout.
