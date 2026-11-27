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

## Ask the steward for a mapping proposal

POST `sample`, `goal`, `record_type` and `expected` to `/api/sources/proposals/{id}`. Start with expected draft revision zero. If the sample has multiple candidate collections, select the JSON Pointer returned in `collections` and resend it as `collection`. If identity is ambiguous, resend with `identity`, an ordered list of one or more field pointers. The server checks identity collisions before calling the model.

The steward sees the goal, selected identity and field statistics. The server checks proposed paths against the sample and returns the normalized preview and immutable steward draft. GET the same route for draft history. To approve a draft, POST `sample`, `expected` (the current approved mapping revision) and `proposal_revision` to `/api/sources/mappings/{id}`. The approved mapping records the human approval and its draft revision. A run still selects an explicit approved mapping revision.

In Goal builder, select **Source API response** as the input format. Paste a sample or fetch a configured source by name. Enter an outcome and a saved factory ID, then choose **Propose mapping**. When prompted, select the record collection and identity fields (selection order determines composite-key order). Review the proposed field paths and normalized records before choosing **Approve mapping for this run**. Starting the goal from the form or chat uses the approved revision and sampled payload.

Approved mappings record the observed JSON kinds at the selected identity and field paths. Missing values and explicit nulls are distinct. Imports using that revision check those paths against the approved sample; unrelated new fields are ignored. Earlier mapping revisions without observations keep their original behavior.

By default, a rejected row prevents a run from starting. To process valid records from a mixed payload, set `source.allow_partial` to true or choose **Process valid rows and report rejected rows** under **Import another payload with this mapping**. The run records each rejected row's zero-based index and reason in `import_problems`. Both sides of a duplicate identity are rejected. A payload with no valid records cannot start a run.

## Import cursor-paginated sources

POST the following to `/api/imports/{factory}/{import-id}`. The feed must grant this factory read access and declare the cursor query parameter. Each response must include the next-token JSON Pointer. A null or empty string marks the final page.

```json
{"expected":0,"max_pages":3,"spec":{"mapping":"meters","revision":1,"feed":"meter-feed","cursor_param":"after","next_path":"/pagination/next","allow_partial":true}}
```

The steward can propose a next-token JSON Pointer from observed string or null envelope fields. Goal builder shows the pointer before approval. When the pinned mapping contains an approved pointer, the import can omit `next_path`. An explicit conflicting pointer is rejected. Supply `cursor_param` from the upstream request contract.

The server fetches up to ten pages per request. It atomically saves each page's normalized records, row-level rejections and next token as one immutable checkpoint. GET the same URL to inspect committed pages. Resume with the same specification and the number of committed pages as `expected`. A stale checkpoint returns 409. On a later-page failure, earlier commits remain available. Repeated cursors stop the import. Later occurrences of an already committed record ID are rejected; partial mode preserves other records.

To review a committed page, POST a goal with `factory`, `description` and `"import":{"id":"batch","page":1}`. The run snapshots that page and pins the approved mapping revision. PostgreSQL and SQLite retain checkpoints across restarts; in-process storage retains them for the life of the process.

### Compare a declared OpenAPI response

In Goal builder, open **Compare with an OpenAPI response** and paste an OpenAPI 3.0 or 3.1 document. Select its operation path, method and response status. The proposal API accepts the same selection as `openapi: {document, path, method, status}`. The document can be a JSON object or JSON/YAML text. The defaults are GET and status 200.

After choosing a collection, compare the declared types and required properties with the observed fields. Required markers refer to the containing object. The steward receives field summaries. Examples, descriptions and server credentials in the document are excluded. Only sampled paths are eligible for a mapping. This structural comparison supports application/json responses, object properties, array items, nullable fields and local references. It does not perform full OpenAPI validation or expand composed schemas.

### Trace a reviewed record

New mapped runs and decisions expose source lineage for each accepted record. The lineage includes the pinned mapping name and revision, its original JSON Pointer and the SHA-256 of the source row, including unmapped fields. To reproduce the digest, decode the source JSON using exact number tokens, then hash the compact bytes returned by Go's `encoding/json.Marshal` for the row. Rejected rows keep their place in the source indexing. For committed pages, lineage also identifies the configured feed, import ID and page number.

A remote sample returned with `include_sample` includes a `sample_receipt`. Pass that receipt with the unchanged payload in the goal's `source` to retain verified feed attribution. The server stores the factory, feed and payload digest, and checks them at import. Goal builder does this automatically. Pasted or edited payloads retain their row fingerprints and mapping revision and are labeled as submitted payloads. Older decisions can be reviewed without a lineage field.

### Control which imported fields a worker may edit

Goal builder sends the factory's editable field names as `editable_candidates`. The steward selects mapped destination fields for the goal. Review the checkboxes before approving. Approval stores the human selection and its steward draft revision. Identity paths and their enclosing objects stay read-only.

A run started from a payload or an import page uses the intersection of the pinned factory and mapping permissions. If that intersection is empty or excludes a required factory field, the editing run cannot start. An empty `editable_fields` list allows read-only ingestion. Earlier mapping revisions that omit the property use the factory's existing field permissions. These local edit permissions grant no upstream write access.

### Map relationships between records

Select **Propose relationships between records** or set `include_relationships` on a proposal request. Review each proposed source pointer and normalized target. Edit the relationship name in Goal builder before approval. The human-approved mapping retains a reference to the original steward draft. An explicit mapping can supply `"relationships":[{"name":"assigned-to","path":"/assignee/id"}]`. Each pointer may select a string or integer ID, or a list of IDs. Missing and null references produce no link. Invalid targets surface as row problems. Relationship values use the existing entity relation format and accompany the record as evidence for workers and judges. Target IDs are references supplied by the source and may refer to records outside the current page.
