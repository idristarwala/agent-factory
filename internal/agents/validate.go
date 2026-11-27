package agents

import (
	"fmt"
)

// ParseAndValidate parses model output as JSON and restricts keys to the allow-list.
func ParseAndValidate(raw string, allowed []string) (map[string]any, error) {
	m, err := ValidateJSON(raw)
	if err != nil {
		return nil, err
	}
	allow := map[string]bool{}
	for _, k := range allowed {
		allow[k] = true
	}
	out := map[string]any{}
	for k, v := range m {
		if allow[k] {
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no allowed fields in response")
	}
	return out, nil
}
