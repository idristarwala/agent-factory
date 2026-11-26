package mapping

import "context"

// Row is a raw record from any source (API, BigQuery, SQL, CSV).
type Row map[string]any

// Source pulls rows. Implementations: BigQuery, REST API, SQL, file.
type Source interface {
	Name() string
	Fetch(ctx context.Context, query string, limit int) ([]Row, error)
}

// Mapping describes how source fields become Entity attributes.
// Business users never write this; Steward generates or loads it.
type Mapping struct {
	SourceName string
	Domain     string
	Type       string
	FieldMap   map[string]string
	IDField    string
}

// Apply turns one source row into an Entity-shaped map.
func (m Mapping) Apply(row Row) (id string, attrs map[string]any) {
	attrs = make(map[string]any)
	for src, dest := range m.FieldMap {
		if v, ok := row[src]; ok {
			attrs[dest] = v
		}
	}
	if idv, ok := row[m.IDField]; ok {
		id = toString(idv)
	}
	return id, attrs
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
