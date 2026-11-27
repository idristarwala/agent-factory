# Record attributes

Factory records carry an `attributes` object. A factory definition selects the fields the worker may edit and their JSON types. A source mapping can copy differently named or nested source values into those fields. Additional source attributes can supply context without becoming editable. [Example records](../examples/service-desk/records.json) · [Source mapping](MAPPING.md) · [Worker schema](SCHEMA.md)

## Catalog example helpers

A catalog can have hundreds of product types, each with dozens or hundreds of attributes. The schema does not hard-code them.

## How it works

1. Supply sample rows to the catalog attribute-discovery helper.
2. `DiscoverAttributes` groups columns by product type.
3. Each PT gets an `AttributeSet` (list of attribute names).
4. Entity values are stored sparsely — only non-empty attributes are kept.
5. Known aliases (`title`, `brand`, `pt`, …) are normalized; everything else stays as a raw attribute.

## Storage

| Piece | Where |
|-------|-------|
| Attribute names per PT | `AttributeIndex` (small, in memory or cached) |
| Attribute values | `Entity.Attributes` (sparse map) |
| Missing required | `SparseAttributes.Missing(required)` |
