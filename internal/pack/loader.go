package pack

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// RuleSpec is a versioned rule inside a domain pack.
type RuleSpec struct {
	ID     string `json:"id" yaml:"id"`
	Text   string `json:"text" yaml:"text"`
	Active bool   `json:"active" yaml:"active"`
}

// Pack is the file-based domain pack format for issue #9.
// Replaces hard-coded Go packs over time.
type Pack struct {
	Name         string     `json:"name" yaml:"name"`
	Description  string     `json:"description,omitempty" yaml:"description,omitempty"`
	NodeTypes    []string   `json:"node_types,omitempty" yaml:"node_types,omitempty"`
	WorkerRoles  []string   `json:"worker_roles" yaml:"worker_roles"`
	Rules        []RuleSpec `json:"rules" yaml:"rules"`
	ToolsAllowed []string   `json:"tools_allowed,omitempty" yaml:"tools_allowed,omitempty"`
	Tools        []string   `json:"tools,omitempty" yaml:"tools,omitempty"`
}

// Load reads a pack.yaml or pack.json from disk.
// YAML is unmarshalled with yaml.v3 so nested maps and lists are fully supported.
func Load(path string) (*Pack, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p Pack
	if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		if err := yaml.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("pack yaml parse: %w", err)
		}
	} else {
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("pack json parse: %w", err)
		}
	}
	if p.Name == "" {
		return nil, fmt.Errorf("pack missing name")
	}
	// Normalise tools alias
	if len(p.ToolsAllowed) == 0 && len(p.Tools) > 0 {
		p.ToolsAllowed = p.Tools
	}
	return &p, nil
}
