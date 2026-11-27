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
	msgs := make([]map[string]any, 0, len(req.Messages))
	for _, m := range req.Messages {
		if m.Role == "system" {
			continue // system goes in system field
		}
		msgs = append(msgs, map[string]any{"role": m.Role, "content": m.Content})
	}
	if len(msgs) == 0 {
		msgs = []map[string]any{{"role": "user", "content": ""}}
	}
	body := map[string]any{
		"model":      req.Model,
		"messages":   msgs,
		"max_tokens": req.MaxTokens,
	}
	if len(req.Tools) > 0 {
		tools := make([]map[string]any, len(req.Tools))
		for i, t := range req.Tools {
			tools[i] = map[string]any{
				"name":         t.Name,
				"description":  t.Description,
				"input_schema": t.Parameters,
			}
		}
		body["tools"] = tools
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
			Type  string `json:"type"`
			Text  string `json:"text"`
			ID    string `json:"id"`
			Name  string `json:"name"`
			Input any    `json:"input"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	content := ""
	var toolCalls []ToolCall
	for _, c := range out.Content {
		if c.Type == "text" || c.Text != "" {
			content += c.Text
		}
		if c.Type == "tool_use" || c.Name != "" {
			args, _ := json.Marshal(c.Input)
			toolCalls = append(toolCalls, ToolCall{
				ID:        c.ID,
				Name:      c.Name,
				Arguments: string(args),
			})
		}
	}
	return &ChatResponse{Content: content, ToolCalls: toolCalls}, nil
}

func (a *AnthropicAdapter) Stream(ctx context.Context, req ChatRequest) (Stream, error) {
	return nil, ErrStreamingUnsupported
}
