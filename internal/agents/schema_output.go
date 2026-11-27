package agents

import (
	"encoding/json"
)

// BuildResponseSchema creates a JSON schema from allowed and required fields.
// Sets additionalProperties to false. Addresses issue #14.
func BuildResponseSchema(allowed, required []string) map[string]any {
	props := make(map[string]any)
	for _, f := range allowed {
		props[f] = map[string]any{"type": "string"}
	}
	return map[string]any{
		"type":                 "object",
		"properties":           props,
		"required":             required,
		"additionalProperties": false,
	}
}

// SchemaJSON returns the schema as a JSON string.
func SchemaJSON(allowed, required []string) string {
	b, _ := json.Marshal(BuildResponseSchema(allowed, required))
	return string(b)
}
