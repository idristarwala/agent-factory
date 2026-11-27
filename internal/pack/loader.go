package pack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/idristarwala/agent-factory/schemas"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"io"
	"os"
	"slices"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// RuleSpec is a versioned rule inside a domain pack.
type RuleSpec struct {
	ID     string `json:"id" yaml:"id"`
	Text   string `json:"text" yaml:"text"`
	Active bool   `json:"active" yaml:"active"`
}

// UnmarshalJSON applies the file schema default while retaining explicit false.
func (r *RuleSpec) UnmarshalJSON(data []byte) error {
	type wireRule RuleSpec
	decoded := wireRule{Active: true}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*r = RuleSpec(decoded)
	return nil
}

// Pack is the file-based domain pack format for issue #9.
type Pack struct {
	ProposalArguments map[string]map[string]json.RawMessage `json:"proposal_arguments,omitempty" yaml:"proposal_arguments,omitempty"`
	Name              string                                `json:"name" yaml:"name"`
	Description       string                                `json:"description,omitempty" yaml:"description,omitempty"`
	NodeTypes         []string                              `json:"node_types,omitempty" yaml:"node_types,omitempty"`
	WorkerRoles       []string                              `json:"worker_roles" yaml:"worker_roles"`
	Rules             []RuleSpec                            `json:"rules" yaml:"rules"`
	ToolsAllowed      []string                              `json:"tools_allowed,omitempty" yaml:"tools_allowed,omitempty"`
	Tools             []string                              `json:"tools,omitempty" yaml:"tools,omitempty"`
}

const schemaBase = "https://raw.githubusercontent.com/idristarwala/agent-factory/main/schemas/"

var packSchema = sync.OnceValues(func() (*jsonschema.Schema, error) {
	c := jsonschema.NewCompiler()
	if err := c.AddResource(schemaBase+"rule.schema.json", strings.NewReader(schemas.Rule)); err != nil {
		return nil, err
	}
	if err := c.AddResource(schemaBase+"pack.schema.json", strings.NewReader(schemas.Pack)); err != nil {
		return nil, err
	}
	return c.Compile(schemaBase + "pack.schema.json")
})

var ruleSchema = sync.OnceValues(func() (*jsonschema.Schema, error) {
	return jsonschema.CompileString(schemaBase+"rule.schema.json", schemas.Rule)
})

// Load reads a pack.yaml or pack.json from disk.
// YAML is unmarshalled with yaml.v3 so nested maps and lists are fully supported.
func Load(path string) (*Pack, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(path, data)
}

// Parse validates and decodes a single pack from captured file bytes.
func Parse(path string, data []byte) (*Pack, error) {
	var p Pack
	if err := decodeFile(path, data, packSchema, &p); err != nil {
		return nil, err
	}
	// Normalise tools alias
	if len(p.ToolsAllowed) == 0 && len(p.Tools) > 0 {
		p.ToolsAllowed = p.Tools
	}
	for tool := range p.ProposalArguments {
		if !slices.Contains(p.ToolsAllowed, tool) {
			return nil, fmt.Errorf("proposal_arguments names tool %q outside tools_allowed", tool)
		}
	}
	return &p, nil
}

// LoadRule validates a standalone YAML or JSON rule using the inline pack contract.
func LoadRule(path string) (RuleSpec, error) {
	var rule RuleSpec
	data, err := os.ReadFile(path)
	if err == nil {
		err = decodeFile(path, data, ruleSchema, &rule)
	}
	return rule, err
}

func decodeFile(path string, data []byte, contract func() (*jsonschema.Schema, error), dest any) error {
	var err error
	var input any
	if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		decoder := yaml.NewDecoder(bytes.NewReader(data))
		if err := decoder.Decode(&input); err != nil {
			return fmt.Errorf("YAML parse: %w", err)
		}
		if decoder.Decode(new(any)) != io.EOF {
			return fmt.Errorf("expected one YAML document")
		}
		data, err = json.Marshal(input)
		if err != nil {
			return err
		}
	}
	if err := json.Unmarshal(data, &input); err != nil {
		return fmt.Errorf("JSON: %w", err)
	}
	schema, err := contract()
	if err != nil {
		return err
	}
	if err := schema.Validate(input); err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}
