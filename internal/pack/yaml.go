package pack

import (
	"os"
)

// LoadYAMLOrJSON is kept for compatibility. Prefer Load which handles both formats.
func LoadYAMLOrJSON(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// SimpleYAMLToJSON is deprecated; use yaml.v3 via Load instead.
// Retained so existing tests that call it still compile while we migrate.
func SimpleYAMLToJSON(data []byte) ([]byte, error) {
	// Best-effort: round-trip through yaml.v3 then json for simple cases.
	var v any
	if err := yaml.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return json.Marshal(v)
}
