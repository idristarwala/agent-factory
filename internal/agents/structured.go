package agents

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ValidateJSON ensures the model output is valid JSON object.
// Used by workers and judges to retry on parse failures.
func ValidateJSON(raw string) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	// Strip common fences
	if strings.HasPrefix(raw, "```") {
		lines := strings.Split(raw, "\n")
		if len(lines) > 1 {
			raw = strings.Join(lines[1:], "\n")
		}
		if idx := strings.LastIndex(raw, "```"); idx >= 0 {
			raw = raw[:idx]
		}
		raw = strings.TrimSpace(raw)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return m, nil
}
