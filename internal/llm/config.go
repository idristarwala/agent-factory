package llm

import (
	"encoding/json"
	"fmt"
	"os"
)

// GatewayConfig is a declarative gateway definition.
type GatewayConfig struct {
	BaseURL string            `json:"base_url"`
	Auth    AuthConfig        `json:"auth"`
	Headers map[string]string `json:"headers"`
	TLS     TLSConfig         `json:"tls"`
	Proxy   string            `json:"proxy"` // "env" or "none"
}

type AuthConfig struct {
	Type    string `json:"type"` // "bearer"
	Key     string `json:"key,omitempty"`
	KeyFile string `json:"key_file,omitempty"`
}

type TLSConfig struct {
	CAFile string `json:"ca_file"`
}

// ModelConfig maps a logical model to a gateway and protocol.
type ModelConfig struct {
	Gateway         string `json:"gateway"`
	Protocol        string `json:"protocol"` // "openai", "gemini", "anthropic"
	Model           string `json:"model"`
	TokenParam      string `json:"token_param,omitempty"` // "max_tokens" or "max_completion_tokens"
	ReasoningBudget int    `json:"reasoning_budget,omitempty"`
}

// Config is the top-level gateway configuration.
type Config struct {
	Gateways map[string]GatewayConfig `json:"gateways"`
	Models   map[string]ModelConfig   `json:"models"`
}

// LoadConfig reads a gateway config file (JSON).
func LoadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &c, nil
}

// LoadKey reads a key from a file or returns the inline key.
func LoadKey(auth AuthConfig) (string, error) {
	if auth.KeyFile != "" {
		b, err := os.ReadFile(auth.KeyFile)
		if err != nil {
			return "", fmt.Errorf("read key file: %w", err)
		}
		return string(b), nil
	}
	return auth.Key, nil
}
