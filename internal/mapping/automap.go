package mapping

import "strings"

// Common retail field aliases. Steward uses these to auto-map.
var aliases = map[string][]string{
	"title":       {"title", "name", "product_name", "item_name", "product_title"},
	"description": {"description", "desc", "long_description", "product_description"},
	"brand":       {"brand", "brand_name", "manufacturer"},
	"pt":          {"pt", "product_type", "category", "product_category", "type"},
	"price":       {"price", "current_price", "list_price", "sale_price"},
	"weight_oz":   {"weight_oz", "weight", "item_weight", "shipping_weight"},
	"color":       {"color", "colour", "primary_color"},
	"size":        {"size", "product_size", "variant_size"},
	"sku":         {"sku", "item_id", "product_id", "id", "asin"},
	"image":       {"image", "image_url", "primary_image", "thumbnail"},
}

// InferMapping looks at source columns and builds a Mapping.
// Business users never do this; Steward calls it.
func InferMapping(sourceName, domain string, columns []string) Mapping {
	fieldMap := map[string]string{}
	idField := ""

	lower := map[string]string{}
	for _, c := range columns {
		lower[strings.ToLower(c)] = c
	}

	if domain != "catalog" && domain != "pricing" && domain != "inventory" && domain != "merchandising" {
		typ := domain
		if typ == "" {
			typ = "entity"
		}
		for _, col := range columns {
			fieldMap[col] = col
		}
		idField = lower["id"]
		if idField == "" {
			for _, col := range columns {
				if strings.HasSuffix(strings.ToLower(col), "_id") {
					if idField != "" {
						idField = ""
						break
					}
					idField = col
				}
			}
		}
		return Mapping{SourceName: sourceName, Domain: domain, Type: typ, IDField: idField, FieldMap: fieldMap}
	}
	for dest, candidates := range aliases {
		for _, cand := range candidates {
			if orig, ok := lower[cand]; ok {
				fieldMap[orig] = dest
				if dest == "sku" && idField == "" {
					idField = orig
				}
				break
			}
		}
	}
	if idField == "" && len(columns) > 0 {
		idField = columns[0]
	}

	typ := "product"
	if domain == "pricing" {
		typ = "price"
	} else if domain == "inventory" {
		typ = "stock"
	}

	return Mapping{
		SourceName: sourceName,
		Domain:     domain,
		Type:       typ,
		FieldMap:   fieldMap,
		IDField:    idField,
	}
}
