package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// AnthropicAdapter implements Client for Anthropic-style endpoints.
type AnthropicAdapter struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
	Headers map[string]string
}

func NewAnthropicAdapter(baseURL, apiKey string, headers map[string]string) *AnthropicAdapter {
	return &AnthropicAdapter{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Client:  &http.Client{},
		Headers: headers,
	}
}

func (a *AnthropicAdapter) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	body := map[string]any{
		"model": req.Model,
		"messages": []map[string]any{{
			"role": "user",
			"content": req.Messages[len(req.Messages)-1].Content,
		}},
		"max_tokens": req.MaxTokens,
	}
	b, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.BaseURL+"/v1/messages", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("x-api-key", a.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	for k, v := range a.Headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := a.Client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, &Error{Type: "GATEWAY", Status: resp.StatusCode, Retryable: resp.StatusCode >= 500 || resp.StatusCode == 429, Message: fmt.Sprintf("status %d", resp.StatusCode)}
	}

	var out struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	content := ""
	if len(out.Content) > 0 {
		content = out.Content[0].Text
	}
	return &ChatResponse{Content: content}, nil
}

func (a *AnthropicAdapter) Stream(ctx context.Context, req ChatRequest) (Stream, error) {
	return nil, ErrStreamingUnsupported
}
