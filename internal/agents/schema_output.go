package agents

// BuildResponseSchema creates a JSON schema from allowed and required fields.
// Sets additionalProperties to false. Addresses issue #14.
func BuildResponseSchema(allowed, required []string) map[string]any {
	if required == nil {
		required = []string{}
	}
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
