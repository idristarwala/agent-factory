package tools

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/idristarwala/agent-factory/schemas"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

var evidenceSchema = sync.OnceValues(func() (*jsonschema.Schema, error) {
	return jsonschema.CompileString("evidence-tools.schema.json", schemas.EvidenceTools)
})

// ValidateEvidenceFile checks a draft before its grants and deployment files exist.
func ValidateEvidenceFile(path string) error {
	data, err := readEvidenceFile(path)
	if err != nil {
		return err
	}
	var value any
	if err := yaml.Unmarshal(data, &value); err != nil {
		return err
	}
	schema, err := evidenceSchema()
	if err != nil {
		return err
	}
	// Convert YAML's integers and mappings to JSON Schema's supported JSON types.
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(raw, &value); err != nil {
		return err
	}
	if err = schema.Validate(value); err != nil {
		return err
	}
	cfg, err := parseHTTPConfig(path, data, false)
	if err != nil {
		return err
	}
	if cfg.Recipe.Media != "" && cfg.Recipe.Entity == "" {
		return fmt.Errorf("media requires an entity evidence tool")
	}
	return nil
}
