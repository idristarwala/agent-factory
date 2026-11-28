package agents

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ValidateJSON ensures the model output is valid JSON object.
// Used by workers and judges to retry on parse failures.
// Recovers common model mistakes: markdown fences, surrounding prose, and trailing commas.
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

	// Prefer the first balanced object if the model wrapped the JSON in prose.
	if extracted := extractFirstJSONObject(raw); extracted != "" {
		raw = extracted
	}

	// Strip trailing commas before closing braces/brackets (common LLM mistake).
	raw = stripTrailingCommas(raw)

	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return m, nil
}

// extractFirstJSONObject returns the first balanced {...} substring, or empty.
func extractFirstJSONObject(s string) string {
	start := strings.Index(s, "{")
	if start < 0 {
		return ""
	}
	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}

// stripTrailingCommas removes commas immediately before } or ].
func stripTrailingCommas(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			// Look ahead for optional whitespace then } or ].
			j := i + 1
			for j < len(s) && (s[j] == ' ' || s[j] == '\n' || s[j] == '\t' || s[j] == '\r') {
				j++
			}
			if j < len(s) && (s[j] == '}' || s[j] == ']') {
				continue // skip the comma
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
