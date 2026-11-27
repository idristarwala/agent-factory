package tools

import (
	"encoding/json"
	"strings"
)

// SimpleMap extracts fields using simple $.field or $.nested.field paths.
// Full JSONPath is a future increment; this covers the common cases from issue #12.
func SimpleMap(body string, mapping map[string]string) (map[string]any, error) {
	var data any
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return nil, err
	}
	out := make(map[string]any)
	for key, path := range mapping {
		out[key] = extract(data, strings.TrimPrefix(path, "$."))
	}
	return out, nil
}

func extract(data any, path string) any {
	parts := strings.Split(path, ".")
	cur := data
	for _, p := range parts {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = m[p]
	}
	return cur
}
