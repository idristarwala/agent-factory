package pack

import (
	"encoding/json"
	"fmt"
	"os"
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
func Load(path string) (*Pack, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
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
