// Package example shows an out-of-tree gateway adapter that reuses the
// in-repo OpenAI adapter and is intended to pass llmtest.RunConformance.
//
// Users place this (or a similar file) in their own module and depend on
// github.com/idristarwala/agent-factory/llm.
package example

import (
	"net/http"

	"github.com/idristarwala/agent-factory/llm"
)

// CorpGateway is a thin wrapper that points at a corporate LLM gateway
// while still speaking the OpenAI-compatible protocol.
type CorpGateway struct {
	inner *llm.OpenAIAdapter
}

// NewCorpGateway builds an adapter for a gateway that accepts OpenAI-style
// /chat/completions. Extra static headers (consumer ID, env, etc.) are set.
func NewCorpGateway(baseURL, apiKey string, extraHeaders map[string]string) *CorpGateway {
	a := llm.NewOpenAIAdapter(baseURL, apiKey, extraHeaders)
	return &CorpGateway{inner: a}
}

func (c *CorpGateway) Client() llm.Client { return c.inner }

// SetHTTPClient lets tests inject an httptest.Client.
func (c *CorpGateway) SetHTTPClient(hc *http.Client) {
	c.inner.Client = hc
}
