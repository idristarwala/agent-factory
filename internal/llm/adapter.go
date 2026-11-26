package llm

import (
	"context"
	"fmt"
)

// ConfigAdapter is a Client built from a GatewayConfig.
// It does not use os.Setenv.
type ConfigAdapter struct {
	Gateway GatewayConfig
	Key     string
}

func NewConfigAdapter(gw GatewayConfig, inlineKey string) (*ConfigAdapter, error) {
	key, err := LoadKey(gw.KeyFile, inlineKey)
	if err != nil {
		return nil, fmt.Errorf("load key: %w", err)
	}
	return &ConfigAdapter{Gateway: gw, Key: key}, nil
}

func (a *ConfigAdapter) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// Protocol-specific body construction would happen here.
	// For now we return a stub response that proves the config is wired.
	return &ChatResponse{
		Content:      "config-adapter response",
		Model:        req.Model,
		LogicalModel: req.Model,
		Usage:        Usage{Input: 10, Output: 5, Total: 15},
	}, nil
}

func (a *ConfigAdapter) Stream(ctx context.Context, req ChatRequest) (Stream, error) {
	return nil, ErrStreamingUnsupported
}
