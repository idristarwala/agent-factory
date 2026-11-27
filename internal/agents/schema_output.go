package agents

import (
	"encoding/json"
)

// BuildResponseSchema creates a simple JSON schema object from allowed fields.
// Can be attached to provider response_format / responseSchema for stricter output.
func BuildResponseSchema(allowed []string) map[string]any {
	props := make(map[string]any)
	for _, f := range allowed {
		props[f] = map[string]any{"type": "string"}
	}
	return map[string]any{
		"type":       "object",
		"properties": props,
		"required":   allowed,
	}
}

// SchemaJSON returns the schema as a JSON string for logging or request bodies.
func SchemaJSON(allowed []string) string {
	b, _ := json.Marshal(BuildResponseSchema(allowed))
	return string(b)
}
