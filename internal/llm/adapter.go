package llm

import (
	"context"
	"fmt"
)

// ConfigAdapter is a Client built from a GatewayConfig and ModelConfig.
// It does not use os.Setenv.
type ConfigAdapter struct {
	Gateway GatewayConfig
	Model   ModelConfig
	Key     string
}

func NewConfigAdapter(gw GatewayConfig, model ModelConfig) (*ConfigAdapter, error) {
	key, err := LoadKey(gw.Auth)
	if err != nil {
		return nil, fmt.Errorf("load key: %w", err)
	}
	return &ConfigAdapter{Gateway: gw, Model: model, Key: key}, nil
}

func (a *ConfigAdapter) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// Protocol-specific body construction
	body := BuildBody(a.Model, req.Messages, req.MaxTokens)
	body["messages"] = req.Messages

	// In a real implementation this would POST to a.Gateway.BaseURL
	// with a.Key in the Authorization header and a.Gateway.Headers.
	// For now we return a stub response that proves the config is wired.
	_ = body
	return &ChatResponse{
		Content:      "config-adapter response",
		Model:        a.Model.Model,
		LogicalModel: req.Model,
		Usage:        Usage{Input: 10, Output: 5, Total: 15},
	}, nil
}

func (a *ConfigAdapter) Stream(ctx context.Context, req ChatRequest) (Stream, error) {
	return nil, ErrStreamingUnsupported
}
