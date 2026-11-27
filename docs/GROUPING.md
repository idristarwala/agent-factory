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

## Block then decide

```go
proposals := grouping.BlockThenDecide(ctx, llmClient, entities, "group")
// Then build predicted map from proposals and score:
// predicted := applyProposals(entities, proposals)
// metrics := grouping.ScorePairMetrics(entities, "group", predicted)
```

Builds candidate neighbourhoods with the cheap signature, then asks the LLM for membership decisions inside each neighbourhood. Falls back to `SimpleDetector` on error or when no client is provided.

Use this path for higher accuracy once a labelled set is available. The helper and scorer are ready; wire into a command or demo for end-to-end measurement.

## Customer policies

Plain-text decision trees become versioned rules:

```
if pt == Apparel and title contains shoe then set pt = Footwear
if weight_oz > 30 and pt == Apparel then flag weight
```

## Flow

1. Assess every entity (state + missing attributes).
2. Fix attributes that block grouping (PT, title, missing axes).
3. Propose regroupings (prefer BlockThenDecide when an LLM client is available).
4. Judge approves against active rules (including customer policies).
5. Repeat until clean.

## Compare against independent labels

Run the scoring command on the supplied example or on your own labelled fixture.

```sh
go run ./cmd/variant-demo examples/labelled_grouping.json
```

Supply the current assignments in `entities[].attributes.group`. Supply independently assigned groups in a separate `labels` array with one entry per entity, for example `{"id":"6","group":"family-a"}`. An empty group string marks an explicitly ungrouped entity. IDs must be unique in each collection. Missing, extra or incomplete labels fail the run.

The command prints pair precision, recall, F1 and pair counts for existing assignments, the deterministic detector and the nil-client fallback. Each comparison uses the same labels. Detectors receive only the source entities. Unlabelled legacy fixtures still run and report proposal counts. The bundled fixture illustrates the format. Field accuracy requires independently labelled field records.
