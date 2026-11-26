# Dynamic attributes

A catalog can have hundreds of product types, each with dozens or hundreds of attributes. The schema does not hard-code them.

## How it works

1. Steward samples rows from the source.
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

## Scale

- 500 PTs × 200 attributes = 100k attribute definitions (names only).
- An item with 30 filled attributes stores 30 keys, not 200.
- Agents only load the AttributeSet for the PT they are working on.
