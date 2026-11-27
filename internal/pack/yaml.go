package pack

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// LoadYAMLOrJSON loads a pack file. Simple YAML is converted to JSON for the common case.
// Full YAML library can be added later for nested structures.
func LoadYAMLOrJSON(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		// Convert simple YAML to JSON for flat and list cases.
		jsonData, err := SimpleYAMLToJSON(data)
		if err != nil {
			return data, nil // fall back to raw if conversion fails
		}
		return jsonData, nil
	}
	return data, nil
}

// SimpleYAMLToJSON converts a limited YAML subset (maps and simple lists) to JSON.
// Sufficient for pack.yaml files with name, worker_roles, rules, tools_allowed.
func SimpleYAMLToJSON(data []byte) ([]byte, error) {
	lines := strings.Split(string(data), "\n")
	result := make(map[string]any)
	var currentKey string
	var currentList []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "- ") && currentKey != "" {
			currentList = append(currentList, strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
			continue
		}
		if strings.Contains(trimmed, ":") {
			if currentKey != "" && len(currentList) > 0 {
				result[currentKey] = currentList
				currentList = nil
			}
			parts := strings.SplitN(trimmed, ":", 2)
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			currentKey = key
			if val != "" {
				result[key] = val
				currentKey = ""
			}
		}
	}
	if currentKey != "" && len(currentList) > 0 {
		result[currentKey] = currentList
	}
	return json.Marshal(result)
}

// ParseSimpleYAML does a very basic YAML-to-map conversion for flat files.
func ParseSimpleYAML(data []byte) (map[string]string, error) {
	out := make(map[string]string)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			out[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return out, nil
}
