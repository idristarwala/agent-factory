package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// OpenAIAdapter implements Client for OpenAI-compatible endpoints.
// Constructs and POSTs the request body without os.Setenv.
type OpenAIAdapter struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
	Headers map[string]string
}

func NewOpenAIAdapter(baseURL, apiKey string, headers map[string]string) *OpenAIAdapter {
	return &OpenAIAdapter{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Client:  &http.Client{},
		Headers: headers,
	}
}

func (a *OpenAIAdapter) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	body, err := BuildOpenAIBody(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+a.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range a.Headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := a.Client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		typ := "GATEWAY"
		retryable := resp.StatusCode >= 500 || resp.StatusCode == 429
		if resp.StatusCode == 429 {
			typ = "RATE_LIMITED"
		} else if resp.StatusCode == 401 || resp.StatusCode == 403 {
			typ = "AUTH"
			retryable = false
		} else if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			typ = "BAD_REQUEST"
			retryable = false
		}
		return nil, &Error{Type: typ, Status: resp.StatusCode, Retryable: retryable, Message: fmt.Sprintf("status %d", resp.StatusCode)}
	}

	var out struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	content := ""
	var toolCalls []ToolCall
	if len(out.Choices) > 0 {
		content = out.Choices[0].Message.Content
		for _, tc := range out.Choices[0].Message.ToolCalls {
			toolCalls = append(toolCalls, ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			})
		}
	}
	return &ChatResponse{
		Content:   content,
		ToolCalls: toolCalls,
		Usage: Usage{
			Input:  out.Usage.PromptTokens,
			Output: out.Usage.CompletionTokens,
			Total:  out.Usage.PromptTokens + out.Usage.CompletionTokens,
		},
	}, nil
}

func (a *OpenAIAdapter) Stream(ctx context.Context, req ChatRequest) (Stream, error) {
	return nil, ErrStreamingUnsupported
}

// BuildOpenAIBody constructs the JSON body for an OpenAI-compatible request.
func BuildOpenAIBody(req ChatRequest) ([]byte, error) {
	msgs := make([]map[string]any, len(req.Messages))
	for i, m := range req.Messages {
		m := map[string]any{"role": m.Role, "content": m.Content}
		if len(m.ToolCalls) > 0 {
			// placeholder; full tool message shaping is next
		}
		msgs[i] = m
	}
	body := map[string]any{
		"model":    req.Model,
		"messages": msgs,
	}
	if req.MaxTokens > 0 {
		param := MapTokenParam("openai", req.Model)
		body[param] = req.MaxTokens
	}
	if req.Temperature != nil {
		body["temperature"] = *req.Temperature
	}
	if len(req.Tools) > 0 {
		tools := make([]map[string]any, len(req.Tools))
		for i, t := range req.Tools {
			tools[i] = map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        t.Name,
					"description": t.Description,
					"parameters":  t.Parameters,
				},
			}
		}
		body["tools"] = tools
	}
	return json.Marshal(body)
}
