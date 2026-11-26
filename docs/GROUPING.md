# Regrouping and customer policies

Both are domain-agnostic.

## Regrouping

Items (or any entities) that share a signature but sit in different groups are proposed for regrouping.

| Domain | Group field | Signature |
|--------|-------------|-----------|
| Catalog | variant_group or pt | brand + title token |
| Pricing | price_family | brand + product |
| Inventory | location_cluster | sku + region |

`SimpleDetector` finds the majority group and proposes moving outliers. The Judge still approves.

## Customer policies and decision trees

Customers can supply rules in plain text:

```
if pt == Apparel and title contains shoe then set pt = Footwear
if weight_oz > 30 and pt == Apparel then flag weight
if price < cost then reject
```

`ParseSimpleTree` turns these into Policies.  
`IngestPolicies` writes them as versioned rules.  
The Judge uses the active versions on the next pass.

## Fit together

1. Steward loads entities and discovers attributes.
2. Customer policies are ingested as versioned rules.
3. Grouping detector proposes regroupings.
4. Specialized agents propose attribute fixes.
5. Judge evaluates everything against the active rules (including customer policies).
6. Approved changes land on the graph. Customers see Decisions.
