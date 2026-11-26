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
