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
	for k := range m {
		if !allow[k] {
			return nil, fmt.Errorf("response contains a field outside the write scope: %s", k)
		}
	}
	if len(m) == 0 {
		return nil, fmt.Errorf("no allowed fields in response")
	}
	return m, nil
}
