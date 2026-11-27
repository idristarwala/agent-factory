package tools

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ExtractPath extracts a value from JSON using a simple path.
// Supports $.field and $.field.nested. Arrays use [*].
func ExtractPath(data []byte, path string) (any, error) {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	path = strings.TrimPrefix(path, "$.")
	if path == "" {
		return m, nil
	}

	parts := strings.Split(path, ".")
	var current any = m
	for _, part := range parts {
		if part == "[*]" {
			// Array wildcard — return as-is for further processing
			continue
		}
		switch v := current.(type) {
		case map[string]any:
			current = v[part]
		case []any:
			// For arrays, extract the field from each element
			var extracted []any
			for _, item := range v {
				if itemMap, ok := item.(map[string]any); ok {
					extracted = append(extracted, itemMap[part])
				}
			}
			current = extracted
		default:
			return nil, fmt.Errorf("cannot traverse %s in %T", part, current)
		}
	}
	return current, nil
}

// MapResponse extracts fields according to a ResponseMap.
func MapResponse(data []byte, rm ResponseMap) (map[string]any, error) {
	var root any
	var err error
	if rm.Root != "" {
		root, err = ExtractPath(data, rm.Root)
		if err != nil {
			return nil, err
		}
	} else {
		json.Unmarshal(data, &root)
	}

	rootBytes, _ := json.Marshal(root)
	out := map[string]any{}
	for key, path := range rm.Map {
		val, err := ExtractPath(rootBytes, path)
		if err != nil {
			continue
		}
		out[key] = val
	}
	return out, nil
}
