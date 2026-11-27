# Source mapping

Business users point at where their data lives (BigQuery table, REST API, SQL).  
The Steward maps it to the unified Entity schema. Users never write mappings.

## Flow

1. User supplies a source (BQ dataset, API URL, or SQL connection) and a goal.
2. Steward samples columns.
3. `InferMapping` matches common retail names (`title`, `product_name`, `brand`, `pt`, `price`, `sku`, …).
4. Rows are pulled and turned into Entities.
5. Steward chooses memory vs DB based on volume.
6. Agents work on Entities. Customers see Decisions.

## Auto-mapped fields

| Entity attribute | Source aliases |
|------------------|----------------|
| title | title, name, product_name, item_name |
| description | description, desc, long_description |
| brand | brand, brand_name, manufacturer |
| pt | pt, product_type, category |
| price | price, current_price, list_price |
| weight_oz | weight_oz, weight, item_weight |
| sku | sku, item_id, product_id, id |
| image | image, image_url, primary_image |

## Sources

| Source | Implementation |
|--------|----------------|
| BigQuery | `BigQuerySource` (injectable client) |
| REST API | `APISource` (injectable client) |
| SQL | Use existing `SQLTool` + same mapping |

## Inspect a source API response

Send a JSON response sample as the body of `POST /api/sources/profile`. The endpoint accepts up to 1 MiB and uses the same API authentication as factory runs. It returns candidate record locations as JSON Pointers. An empty pointer denotes the root. Each candidate reports the record count and profiles the first 100 records, including nested field paths, observed types, missing counts, and whether a string or integer field has distinct nonempty values across the sample. A single observation cannot establish uniqueness.

The endpoint returns structural statistics only. It does not save the sample, select an identity or start a run. Arrays inside candidate records remain array fields. For example, in `{"data":{"records":[{"key":"A"},{"key":"B"}]}}`, the pointer `/data/records` identifies the two-record collection and `/key` is unique in that sample.

Post a sample and a mapping to `/api/sources/preview`. The following mapping selects an array, uses two fields for identity and copies a nested value into a factory attribute.

```json
{"collection":"/items","record_type":"meter","identity":["/tenant","/key"],"fields":{"reading":"/info/reading"}}
```

The preview body has `sample` and `mapping` keys. It returns indexed rows with normalized records or identity errors. Missing optional attributes are omitted. To save a valid mapping, POST the same body and an `expected` revision (zero for creation) to `/api/sources/mappings/{id}`. GET that path for immutable history. Only the mapping is saved.

To run it, POST a goal with `factory`, `description` and `source: {"mapping":"meters","revision":1,"payload":…}`. The server applies that exact mapping, validates record identities again and queues the run. Its record snapshot and mapping reference are retained for review.

## Sample a remote source

Set `FACTORY_SOURCE_CONFIG` on the API server to a YAML file defining available read-only HTTP endpoints and the factories granted access. For example:

```yaml
tools:
  case-feed:
    url: https://records.example.com/cases
    query: [cursor]
    auth: {type: bearer, key_file: secrets/records-token}
grants:
  - {pack: case-review, tools: [case-feed], side_effects: [read-only]}
```

POST `{}` to `/api/sources/case-review/case-feed/sample` to receive the collection and field profiles for one fresh response. Supply `{"query":{"cursor":"next-page"}}` to sample a particular page. Only configured query names are accepted. The HTTP limits, destination policy, TLS and credential-file options are shared with evidence tools.

To use the sampled bytes for mapping preview or a pinned goal, request `{"include_sample":true}` and pass the returned `sample` as the mapping sample or goal source payload. The endpoint returns structural statistics by default and does not persist the raw sample.
