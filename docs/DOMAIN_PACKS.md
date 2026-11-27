# Domain packs

A pack defines how a problem domain is handled.

## Example Go packs

| Pack | Focus |
|------|-------|
| catalog | Titles, descriptions, attributes, variants |
| pricing | Prices, margins, promotions |
| inventory | Stock, replenishment, allocation |
| merchandising | Assortment, placement |

## Configure your own factory

For a runtime factory, define the record type and editable fields and import the JSON definition. For a file-based pack, supply a YAML or JSON document with worker roles, rules and allowed tool names. The service-desk example and pack schema show the two contracts. [Run guide](RUNNING.md) · [Pack schema](../schemas/pack.schema.json)

The continuous CLI selects its external evidence grant with `FACTORY_PACK`. Workspace goals explicitly select a factory and can pin its revision.
