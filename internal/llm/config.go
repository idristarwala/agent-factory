package llm

import (
	"encoding/json"
	"os"
)

// GatewayConfig is the declarative form described in issue #11.
type GatewayConfig struct {
	Name     string            `json:"name"`
	BaseURL  string            `json:"base_url"`
	AuthType string            `json:"auth_type"` // bearer, none
	KeyFile  string            `json:"key_file"`
	Headers  map[string]string `json:"headers"`
	CAFile   string            `json:"ca_file"`
	Proxy    string            `json:"proxy"` // none, env
}

// LoadGatewayConfig reads a simple JSON gateway config.
func LoadGatewayConfig(path string) (*GatewayConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg GatewayConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ApplyExtraHeaders is a helper for adapters that need static headers.
func ApplyExtraHeaders(base map[string]string, extra map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}
