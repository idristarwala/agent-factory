package pack

import (
	"encoding/json"
	"os"
	"strings"
)

// LoadYAMLOrJSON loads a pack file. Simple YAML is converted to JSON for the common case.
func LoadYAMLOrJSON(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		jsonData, err := SimpleYAMLToJSON(data)
		if err != nil {
			return data, nil
		}
		return jsonData, nil
	}
	return data, nil
}

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
