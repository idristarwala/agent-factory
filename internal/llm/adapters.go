package llm

import (
	"context"
	"encoding/json"
	"time"
)

func init() {
	for _, name := range []string{"gemini", "anthropic", "openai", "groq", "openrouter", "openai-envelope"} {
		Register(name, func(cfg map[string]string) (Client, error) { return registeredAdapter(name, cfg) })
	}
}

// The registered entry point uses the same protocol implementations tested by
// the public conformance suite, with defaults for callers requesting a tier.
func registeredAdapter(name string, cfg map[string]string) (Client, error) {
	key, base := cfg["api_key"], cfg["base_url"]
	if key == "" {
		return nil, &Error{Type: "AUTH", Message: "api_key is required"}
	}
	cheap, frontier := cfg["cheap_model"], cfg["frontier_model"]
	var client Client
	switch name {
	case "gemini":
		if base == "" {
			base = "https://generativelanguage.googleapis.com/v1beta"
		}
		if cheap == "" {
			cheap = "gemini-3.5-flash-lite"
		}
		a := NewGeminiAdapter(base, key, nil)
		a.Client.Timeout = 60 * time.Second
		client = a
	case "anthropic":
		if base == "" {
			base = "https://api.anthropic.com"
		}
		a := NewAnthropicAdapter(base, key, nil)
		a.Client.Timeout = 60 * time.Second
		client = a
	default:
		if base == "" && name == "openrouter" {
			base = "https://openrouter.ai/api/v1"
		}
		if base == "" {
			return nil, &Error{Type: "BAD_REQUEST", Message: "base_url is required"}
		}
		if cheap == "" {
			cheap = "gpt-4o-mini"
			if name == "openrouter" {
				cheap = "openai/gpt-4o-mini"
			}
		}
		a := NewOpenAIAdapter(base, key, nil)
		a.Client.Timeout = 60 * time.Second
		if name == "openai-envelope" {
			var envelope OpenAIEnvelope
			if err := json.Unmarshal([]byte(cfg["envelope"]), &envelope); err != nil {
				return nil, &Error{Type: "BAD_REQUEST", Message: "invalid envelope JSON"}
			}
			if err := envelope.Validate(); err != nil {
				return nil, &Error{Type: "BAD_REQUEST", Message: err.Error()}
			}
			a.Envelope = &envelope
		}
		client = a
	}
	if frontier == "" {
		frontier = cheap
	}
	return &tierDefaults{Client: client, cheap: cheap, frontier: frontier}, nil
}

type tierDefaults struct {
	Client
	cheap, frontier string
}

func (d *tierDefaults) model(req ChatRequest) (ChatRequest, error) {
	if err := enforceTier(req); err != nil {
		return req, err
	}
	switch req.Tier {
	case "", TierCheap:
		if req.Model == "" {
			req.Model = d.cheap
		}
	case TierFrontier, TierCouncil:
		if req.Model == "" {
			req.Model = d.frontier
		}
	default:
		return req, &Error{Type: "BAD_REQUEST", Message: "unknown model tier"}
	}
	if req.Model == "" {
		return req, &Error{Type: "BAD_REQUEST", Message: "supply a model or configure a tier default"}
	}
	return req, nil
}
func (d *tierDefaults) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	wire, err := d.model(req)
	if err != nil {
		return nil, err
	}
	return d.Client.Chat(ctx, wire)
}
func (d *tierDefaults) Stream(ctx context.Context, req ChatRequest) (Stream, error) {
	wire, err := d.model(req)
	if err != nil {
		return nil, err
	}
	return d.Client.Stream(ctx, wire)
}
func (d *tierDefaults) SetAPIKey(key string) { d.Client.(APIKeySetter).SetAPIKey(key) }
