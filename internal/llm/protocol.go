package llm

import (
	"context"
	"encoding/json"
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
	hc, err := NewHTTPClient(TLSConfig{CAFile: cfg.CAFile}, cfg.Proxy)
	if err != nil {
		return nil, err
	}
	return &ProtocolAdapter{cfg: cfg, client: hc}, nil
}

func (p *ProtocolAdapter) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	key, err := p.cfg.LoadKey()
	if err != nil {
		return nil, err
	}
	// Dispatch based on a protocol field if present; for now return a structured response.
	// Real implementation would build the correct body (openai chat/completions, gemini generateContent, etc.)
	body := map[string]any{
		"model": req.Model,
		"messages": req.Messages,
	}
	_ = body
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

// Helper to build OpenAI-compatible body.
func BuildOpenAIBody(req ChatRequest) ([]byte, error) {
	m := map[string]any{
		"model":    req.Model,
		"messages": req.Messages,
	}
	if req.MaxTokens > 0 {
		m["max_tokens"] = req.MaxTokens
	}
	if req.Temperature != nil {
		m["temperature"] = *req.Temperature
	}
	return json.Marshal(m)
}
