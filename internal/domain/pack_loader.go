package domain

import (
	"encoding/json"
	"os"
)

// LoadPackFromJSON loads a domain pack from a JSON file.
// YAML can be converted to JSON externally or via a future library.
func LoadPackFromJSON(path string) (Pack, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Pack{}, err
	}
	var p Pack
	if err := json.Unmarshal(data, &p); err != nil {
		return Pack{}, err
	}
	return p, nil
}

// StructuredOutputHint returns a note for models that support response_format.
func StructuredOutputHint() string {
	return "Respond with valid JSON only. Do not include markdown fences or explanations."
}
