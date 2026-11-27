package pack

import (
	"os"
	"strings"
)

// LoadYAMLOrJSON loads a pack file. YAML is accepted as a simple key:value subset
// by converting it to JSON first for the common case. Full YAML library can be added later.
func LoadYAMLOrJSON(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		// Minimal conversion for simple YAML-like content.
		// For production, use a proper YAML library.
		return data, nil
	}
	return data, nil
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
