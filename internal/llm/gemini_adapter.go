package llm

import (
	"context"
	"encoding/json"
	"fmt"
)

// GeminiAdapter posts to a Gemini-style endpoint.
type GeminiAdapter struct {
	BaseURL string
	Key     string
	Headers map[string]string
	Poster  Poster
	Model   string
}

func NewGeminiAdapter(baseURL, key, model string, headers map[string]string, poster Poster) *GeminiAdapter {
	if poster == nil {
		poster = NewHTTPPoster(nil)
	}
	return &GeminiAdapter{
		BaseURL: baseURL,
		Key:     key,
		Headers: headers,
		Poster:  poster,
		Model:   model,
	}
}

func (a *GeminiAdapter) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	body, err := BuildGeminiBody(a.Model, req.Messages, req.MaxTokens, 0)
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Content-Type": "application/json",
	}
	for k, v := range a.Headers {
		headers[k] = v
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", a.BaseURL, a.Model, a.Key)
	respBody, status, err := a.Poster.Post(ctx, url, headers, body)
	if err != nil {
		return nil, &Error{Type: "TRANSPORT", Message: err.Error(), Retryable: true, Cause: err}
	}
	if status >= 400 {
		retryable := status == 429 || status >= 500
		return nil, &Error{Type: statusType(status), Status: status, Retryable: retryable, Message: string(respBody)}
	}

	var resp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, &Error{Type: "BAD_RESPONSE", Message: err.Error(), Cause: err}
	}
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, &Error{Type: "BAD_RESPONSE", Message: "no candidates"}
	}

	return &ChatResponse{
		Content:      resp.Candidates[0].Content.Parts[0].Text,
		Model:        a.Model,
		LogicalModel: req.Model,
	}, nil
}

func (a *GeminiAdapter) Stream(ctx context.Context, req ChatRequest) (Stream, error) {
	return nil, ErrStreamingUnsupported
}

func init() {
	Register("gemini-post", func(cfg map[string]string) (Client, error) {
		return NewGeminiAdapter(cfg["base_url"], cfg["api_key"], cfg["model"], nil, nil), nil
	})
}

var _ Client = (*GeminiAdapter)(nil)
