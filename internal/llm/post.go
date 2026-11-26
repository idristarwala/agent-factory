package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Poster sends a request and returns the response body.
type Poster interface {
	Post(ctx context.Context, url string, headers map[string]string, body []byte) ([]byte, int, error)
}

// HTTPPoster is a real HTTP poster.
type HTTPPoster struct {
	Client *http.Client
}

func NewHTTPPoster(client *http.Client) *HTTPPoster {
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	return &HTTPPoster{Client: client}
}

func (p *HTTPPoster) Post(ctx context.Context, url string, headers map[string]string, body []byte) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return b, resp.StatusCode, nil
}

// ParseOpenAIResponse extracts content from an OpenAI-style response.
func ParseOpenAIResponse(body []byte) (string, Usage, error) {
	var resp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", Usage{}, err
	}
	if len(resp.Choices) == 0 {
		return "", Usage{}, fmt.Errorf("no choices")
	}
	return resp.Choices[0].Message.Content, Usage{
		Input:  resp.Usage.PromptTokens,
		Output: resp.Usage.CompletionTokens,
		Total:  resp.Usage.TotalTokens,
	}, nil
}
