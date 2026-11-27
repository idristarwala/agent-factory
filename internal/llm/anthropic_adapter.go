package llm

import (
	"bytes"
	"context"
	"encoding/json"
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

func anthropicBody(req ChatRequest) (map[string]any, error) {
	messages := make([]map[string]any, 0, len(req.Messages))
	var system []map[string]string
	for _, msg := range req.Messages {
		images, err := msg.imageBlocks("anthropic")
		if err != nil {
			return nil, err
		}
		switch msg.Role {
		case "system":
			system = append(system, map[string]string{"type": "text", "text": msg.Content})
		case "tool":
			if msg.ToolCallID == "" {
				return nil, &Error{Type: "BAD_REQUEST", Message: "tool result requires a call ID"}
			}
			messages = append(messages, map[string]any{"role": "user", "content": []map[string]string{{"type": "tool_result", "tool_use_id": msg.ToolCallID, "content": msg.Content}}})
		default:
			var content any = msg.Content
			if len(msg.ToolCalls) > 0 || len(images) > 0 {
				var parts []map[string]any
				if msg.Content != "" {
					parts = append(parts, map[string]any{"type": "text", "text": msg.Content})
				}
				parts = append(parts, images...)
				for _, call := range msg.ToolCalls {
					var args map[string]any
					if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil || args == nil {
						return nil, &Error{Type: "BAD_REQUEST", Message: "tool arguments must be an object"}
					}
					parts = append(parts, map[string]any{"type": "tool_use", "id": call.ID, "name": call.Name, "input": args})
				}
				content = parts
			}
			messages = append(messages, map[string]any{"role": msg.Role, "content": content})
		}
	}
	if len(messages) == 0 {
		messages = append(messages, map[string]any{"role": "user", "content": ""})
	}
	body := map[string]any{"model": req.Model, "messages": messages, "max_tokens": req.MaxTokens}
	if len(system) > 0 {
		body["system"] = system
	}
	if req.Temperature != nil {
		body["temperature"] = *req.Temperature
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
	return body, nil
}

func (a *AnthropicAdapter) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	schema, err := compileOutputSchema(req.ResponseSchema)
	if err != nil {
		return nil, err
	}

	body, err := anthropicBody(req)
	if err != nil {
		return nil, err
	}
	if req.ResponseSchema != nil {
		body["output_config"] = map[string]any{"format": map[string]any{"type": "json_schema", "schema": req.ResponseSchema}}
	}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
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

	resp, err := gatewayResponse(a.Client, httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out struct {
		Model  string `json:"model"`
		Finish string `json:"stop_reason"`
		Usage  struct {
			Input      int `json:"input_tokens"`
			Output     int `json:"output_tokens"`
			CacheRead  int `json:"cache_read_input_tokens"`
			CacheWrite int `json:"cache_creation_input_tokens"`
		} `json:"usage"`
		Content []struct {
			Type  string `json:"type"`
			Text  string `json:"text"`
			ID    string `json:"id"`
			Name  string `json:"name"`
			Input any    `json:"input"`
		} `json:"content"`
	}
	if err := decodeReply(ctx, resp, &out); err != nil {
		return nil, err
	}
	if len(out.Content) == 0 {
		return nil, &Error{Type: "BAD_RESPONSE", Message: "no Anthropic content"}
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
	if err := validateOutput(schema, content); err != nil {
		return nil, err
	}
	input := out.Usage.Input + out.Usage.CacheRead + out.Usage.CacheWrite
	return &ChatResponse{Content: content, ToolCalls: toolCalls, Model: out.Model, LogicalModel: req.Model, FinishReason: out.Finish, Usage: Usage{Input: input, Output: out.Usage.Output, Total: input + out.Usage.Output}}, nil
}

func (a *AnthropicAdapter) Stream(ctx context.Context, req ChatRequest) (Stream, error) {
	if req.ResponseSchema != nil {
		return nil, ErrStreamingUnsupported
	}

	body, err := anthropicBody(req)
	if err != nil {
		return nil, err
	}
	body["stream"] = true
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.BaseURL+"/v1/messages", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("x-api-key", a.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Accept", "text/event-stream")
	for k, v := range a.Headers {
		httpReq.Header.Set(k, v)
	}
	resp, err := gatewayResponse(a.Client, httpReq)
	if err != nil {
		return nil, err
	}

	state := new(anthropicStreamState)
	return &sseStream{r: resp.Body, decode: state.decode}, nil
}

// SetAPIKey updates the API key (used by TokenRefresher).
func (a *AnthropicAdapter) SetAPIKey(key string) {
	a.APIKey = key
}
