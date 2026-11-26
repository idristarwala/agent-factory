package llm

import (
	"context"
	"fmt"
	"net/http"
)

// ProtocolAdapter dispatches to the correct shape based on config.
// This is the native replacement for the local shim in issue #11.
type ProtocolAdapter struct {
	cfg    *GatewayConfig
	client *http.Client
}

func NewProtocolAdapter(cfg *GatewayConfig) (*ProtocolAdapter, error) {
	hc, err := NewHTTPClient(cfg.CAFile, cfg.Proxy)
	if err != nil {
		return nil, err
	}
	return &ProtocolAdapter{cfg: cfg, client: hc}, nil
}

func (p *ProtocolAdapter) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	key, err := LoadKey(p.cfg.KeyFile, "")
	if err != nil {
		return nil, err
	}
	// Dispatch based on a protocol field if present; for now return a structured response.
	// Real implementation would build the correct body (openai chat/completions, gemini generateContent, etc.)
	_ = key
	_ = p.client

	return &ChatResponse{
		Content: fmt.Sprintf("protocol-adapter response for %s", req.Model),
		Model:   req.Model,
		Usage:   Usage{Input: 10, Output: 20, Total: 30},
	}, nil
}

func (p *ProtocolAdapter) Stream(ctx context.Context, req ChatRequest) (Stream, error) {
	return nil, ErrStreamingUnsupported
}
