# Regrouping and varianting

Domain-agnostic. Catalog variant groups, pricing families, and inventory clusters use the same machinery.

## Group states

| State | Meaning |
|-------|---------|
| ungrouped | No group; signature matches an existing one or needs a new one |
| partial | Missing attributes (or variant axes) that block proper grouping |
| wrong | In a different group than the signature majority |
| singleton | Alone in its group |
| complete | Properly grouped |

## Variant axes

Up to three axes distinguish variants (e.g. color, size, material).  
Missing axes mark an entity partial so attribute filling happens before grouping.

## Assessor

```go
assessor := &grouping.Assessor{
    GroupField:      "group",
    SignatureFields: []string{"brand", "title"},
    Axes:            grouping.VariantAxes{Axes: []string{"color", "size"}},
}
results := assessor.Assess(entities)
```

## Scoring against labels

```go
metrics := grouping.ScorePairMetrics(entities, "group", predictedGroupByID)
// metrics.Precision, Recall, F1, TruePos, FalsePos, FalseNeg
```

Pair-level precision/recall against a human-labelled set. Deterministic detector output is stable across runs (sorted entity IDs and group names).

## Customer policies

Plain-text decision trees become versioned rules:

```
if pt == Apparel and title contains shoe then set pt = Footwear
if weight_oz > 30 and pt == Apparel then flag weight
```

## Flow

1. Assess every entity (state + missing attributes).
2. Fix attributes that block grouping (PT, title, missing axes).
3. Propose regroupings.
4. Judge approves against active rules (including customer policies).
5. Repeat until clean.

## Demo

```bash
go run ./cmd/variant-demo
```
