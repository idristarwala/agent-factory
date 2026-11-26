package mapping

import (
	"strings"

	"github.com/idristarwala/agent-factory/internal/schema"
)

// DiscoverAttributes samples rows and builds an AttributeIndex per product type.
// Handles hundreds of PTs and thousands of attributes without a fixed list.
func DiscoverAttributes(rows []Row, ptField string) *schema.AttributeIndex {
	idx := schema.NewAttributeIndex()
	seen := map[string]map[string]struct{}{}

	for _, row := range rows {
		pt := "unknown"
		if v, ok := row[ptField]; ok {
			if s, ok := v.(string); ok && s != "" {
				pt = s
			}
		}
		if seen[pt] == nil {
			seen[pt] = map[string]struct{}{}
		}
		for col := range row {
			if isMeta(col) {
				continue
			}
			seen[pt][col] = struct{}{}
		}
	}

	for pt, cols := range seen {
		attrs := make([]string, 0, len(cols))
		for c := range cols {
			attrs = append(attrs, c)
		}
		idx.Add(pt, attrs, nil)
	}
	return idx
}

func isMeta(col string) bool {
	c := strings.ToLower(col)
	switch c {
	case "id", "sku", "product_id", "item_id", "created_at", "updated_at":
		return true
	}
	return false
}

// MapRowToEntity turns a source row into an Entity.
// Known aliases are normalized; unknown columns become raw attributes.
func MapRowToEntity(row Row, m Mapping, domain string) schema.Entity {
	id, attrs := m.Apply(row)
	mapped := map[string]struct{}{}
	for src := range m.FieldMap {
		mapped[src] = struct{}{}
	}
	for col, val := range row {
		if _, ok := mapped[col]; ok {
			continue
		}
		if isMeta(col) {
			continue
		}
		attrs[col] = val
	}
	return schema.Entity{
		ID:         id,
		Type:       m.Type,
		Domain:     domain,
		Attributes: attrs,
		Version:    1,
	}
}
