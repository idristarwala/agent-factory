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
		return nil, &Error{Type: "GATEWAY", Status: resp.StatusCode, Retryable: resp.StatusCode >= 500 || resp.StatusCode == 429, Message: fmt.Sprintf("status %d", resp.StatusCode)}
	}

	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
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
	if len(out.Choices) > 0 {
		content = out.Choices[0].Message.Content
	}
	return &ChatResponse{
		Content: content,
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
		msgs[i] = map[string]any{"role": m.Role, "content": m.Content}
	}
	body := map[string]any{
		"model":    req.Model,
		"messages": msgs,
	}
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	if req.Temperature != nil {
		body["temperature"] = *req.Temperature
	}
	return json.Marshal(body)
}
