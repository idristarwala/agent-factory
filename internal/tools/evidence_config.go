package tools

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// EvidenceConfig is an operator-supplied recipe for the pack selected by the caller.
type EvidenceConfig struct {
	recipe   ContextRecipe
	registry map[string]*HTTPTool
	grants   []Grant
}

func LoadEvidenceConfig(path, pack string) (*EvidenceConfig, error) {
	cfg, err := loadHTTPConfig(path)
	if err != nil {
		return nil, err
	}
	if cfg.recipe.Entity == "" && cfg.recipe.Neighbours == "" {
		return nil, fmt.Errorf("missing entity or neighbours evidence source")
	}
	if cfg.recipe.Media != "" && cfg.recipe.Entity == "" {
		return nil, fmt.Errorf("media requires an entity evidence tool")
	}
	if err := cfg.recipe.Authorize(pack, cfg.grants); err != nil {
		return nil, err
	}
	return cfg, nil
}

func loadHTTPConfig(path string) (*EvidenceConfig, error) {
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
			Body    map[string]any
			Proxy   string
			TLS     struct {
				CAFile string `yaml:"ca_file"`
			}
			Limits struct {
				Retries  *int
				Timeout  *time.Duration
				CacheTTL *time.Duration `yaml:"cache_ttl"`
			}
			Auth struct {
				Type     string
				Name     string
				Username string
				KeyFile  string `yaml:"key_file"`
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
	registry := make(map[string]*HTTPTool, len(config.Tools))
	for name, def := range config.Tools {
		if def.Method == "" {
			def.Method = "GET"
		}
		readPost := def.Method == "POST" && def.SideEffect == "read-only"
		if (!readPost && def.Method != "GET") || (def.SideEffect != "" && def.SideEffect != "read-only") {
			return nil, fmt.Errorf("evidence tool %q requires GET or an explicitly read-only POST", name)
		}
		if err := validateDestination(def.URL, def.URL); err != nil {
			return nil, fmt.Errorf("invalid destination for tool %q", name)
		}
		t := NewHTTPTool(name, def.Method, def.URL)
		if def.Proxy != "" && def.Proxy != "none" {
			return nil, fmt.Errorf("unsupported proxy mode for evidence tool %q", name)
		}
		if caFile := def.TLS.CAFile; caFile != "" {
			if !filepath.IsAbs(caFile) {
				caFile = filepath.Join(filepath.Dir(path), caFile)
			}
			pem, err := os.ReadFile(caFile)
			if err != nil {
				return nil, fmt.Errorf("read CA for tool %q: %w", name, err)
			}
			roots, err := x509.SystemCertPool()
			if err != nil {
				return nil, err
			}
			if !roots.AppendCertsFromPEM(pem) {
				return nil, fmt.Errorf("invalid CA file for tool %q", name)
			}
			transport := t.Client.Transport.(*http.Transport)
			transport.TLSClientConfig = &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}
		}
		t.Headers, t.QueryParams = def.Headers, def.Query
		if def.Body != nil {
			if !readPost {
				return nil, fmt.Errorf("tool %q requires read-only POST for a JSON body", name)
			}
			body, err := json.Marshal(def.Body)
			if err != nil {
				return nil, fmt.Errorf("invalid JSON body for tool %q", name)
			}
			t.BodyTemplate = string(body)
			if t.Headers == nil {
				t.Headers = map[string]string{}
			}
			if t.Headers["Content-Type"] == "" {
				t.Headers["Content-Type"] = "application/json"
			}
		}
		if v := def.Limits.Retries; v != nil {
			if *v < 0 {
				return nil, fmt.Errorf("negative retries for tool %q", name)
			}
			t.Retries = *v
		}
		if v := def.Limits.Timeout; v != nil {
			if *v <= 0 {
				return nil, fmt.Errorf("invalid timeout for tool %q", name)
			}
			t.Client.Timeout = *v
		}
		if v := def.Limits.CacheTTL; v != nil {
			if *v < 0 {
				return nil, fmt.Errorf("negative cache TTL for tool %q", name)
			}
			t.CacheTTL = *v
		}
		if def.Auth.Type != "" || def.Auth.KeyFile != "" {
			switch def.Auth.Type {
			case "bearer":
			case "basic":
				if def.Auth.Username == "" {
					return nil, fmt.Errorf("missing basic auth username for tool %q", name)
				}
			case "header":
				if def.Auth.Name == "" {
					return nil, fmt.Errorf("missing auth header name for tool %q", name)
				}
			default:
				return nil, fmt.Errorf("unsupported authentication for tool %q", name)
			}
			if def.Auth.KeyFile == "" {
				return nil, fmt.Errorf("missing auth key file for tool %q", name)
			}
			t.AuthType, t.KeyFile, t.AuthHeader, t.Username = def.Auth.Type, def.Auth.KeyFile, def.Auth.Name, def.Auth.Username
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
	return &EvidenceConfig{recipe: config.Context, registry: registry, grants: config.Grants}, nil
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
