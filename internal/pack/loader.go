package pack

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Pack is the file-based domain pack format for issue #9.
// Replaces hard-coded Go packs over time.
type Pack struct {
	Name         string   `json:"name"`
	WorkerRoles  []string `json:"worker_roles"`
	Rules        []string `json:"rules"`
	ToolsAllowed []string `json:"tools_allowed"`
}

// Load reads a pack.yaml or pack.json from disk.
// YAML files are converted via SimpleYAMLToJSON before unmarshalling.
func Load(path string) (*Pack, error) {
	data, err := LoadYAMLOrJSON(path)
	if err != nil {
		return nil, err
	}
	// If it was YAML and conversion produced JSON, use it; otherwise try as-is.
	if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		// data is already converted JSON or original
	}
	var p Pack
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("pack parse: %w", err)
	}
	if p.Name == "" {
		return nil, fmt.Errorf("pack missing name")
	}
	return &p, nil
}
