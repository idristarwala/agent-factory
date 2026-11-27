package tools

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// EvidenceConfig is an operator-supplied recipe for the pack selected by the caller.
type EvidenceConfig struct {
	recipe   ContextRecipe
	registry map[string]*HTTPTool
}

func LoadEvidenceConfig(path, pack string) (*EvidenceConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config struct {
		Tools map[string]struct {
			Method  string
			URL     string
			Headers map[string]string
			Query   []string
			Auth    struct {
				Type    string
				KeyFile string `yaml:"key_file"`
			}
			SideEffect string `yaml:"side_effect"`
		}
		Context ContextRecipe
		Grants  []Grant
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("expected one evidence configuration document")
	}
	if config.Context.Entity == "" && config.Context.Neighbours == "" {
		return nil, fmt.Errorf("missing entity or neighbours evidence source")
	}
	if config.Context.Media != "" && config.Context.Entity == "" {
		return nil, fmt.Errorf("media requires an entity evidence tool")
	}
	if err := config.Context.Authorize(pack, config.Grants); err != nil {
		return nil, err
	}
	registry := make(map[string]*HTTPTool, len(config.Tools))
	for name, def := range config.Tools {
		if def.Method == "" {
			def.Method = "GET"
		}
		if def.Method != "GET" || (def.SideEffect != "" && def.SideEffect != "read-only") {
			return nil, fmt.Errorf("evidence tool %q requires a read-only GET", name)
		}
		if err := validateDestination(def.URL, def.URL); err != nil {
			return nil, fmt.Errorf("invalid destination for tool %q", name)
		}
		t := NewHTTPTool(name, def.Method, def.URL)
		t.Headers, t.QueryParams = def.Headers, def.Query
		if def.Auth.Type != "" || def.Auth.KeyFile != "" {
			if def.Auth.Type != "bearer" || def.Auth.KeyFile == "" {
				return nil, fmt.Errorf("invalid bearer authentication for tool %q", name)
			}
			t.AuthType, t.KeyFile = def.Auth.Type, def.Auth.KeyFile
			if !filepath.IsAbs(t.KeyFile) {
				t.KeyFile = filepath.Join(filepath.Dir(path), t.KeyFile)
			}
		}
		registry[name] = t
	}
	for _, step := range config.Context.steps() {
		if step.name != "" && registry[step.name] == nil {
			return nil, fmt.Errorf("missing evidence tool %q", step.name)
		}
	}
	return &EvidenceConfig{recipe: config.Context, registry: registry}, nil
}

func RecordParams(id string, props map[string]any) map[string]string {
	args := map[string]string{}
	for k, v := range props {
		switch v.(type) {
		case string, bool, int, int64, float64:
			args[k] = fmt.Sprint(v)
		}
	}
	args["id"] = id
	return args
}

func (c *EvidenceConfig) Gather(ctx context.Context, id string, props map[string]any) ([]string, error) {
	record, err := c.GatherRecord(ctx, id, props)
	return record.Text, err
}

func (c *EvidenceConfig) GatherRecord(ctx context.Context, id string, props map[string]any) (RecordEvidence, error) {
	result, err := ExecuteRecipe(ctx, c.recipe, c.registry, RecordParams(id, props))
	if err != nil {
		return RecordEvidence{}, err
	}
	return c.recipe.Collect(ctx, result)
}
