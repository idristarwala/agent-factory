package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// OpenAICompatGateway talks to any OpenAI-compatible endpoint.

type OpenAICompatGateway struct {
	BaseURL       string
	APIKey        string
	CheapModel    string
	FrontierModel string
	HTTPClient    *http.Client
}

func NewOpenAICompatFromEnv() (*OpenAICompatGateway, error) {
	base := os.Getenv("LLM_BASE_URL")
	key := os.Getenv("LLM_API_KEY")
	if base == "" || key == "" {
		return nil, fmt.Errorf("LLM_BASE_URL and LLM_API_KEY must be set")
	}
	cheap := os.Getenv("LLM_CHEAP_MODEL")
	if cheap == "" {
		cheap = "llama-3.1-8b-instant"
	}
	frontier := os.Getenv("LLM_FRONTIER_MODEL")
	if frontier == "" {
		frontier = cheap
	}
	return &OpenAICompatGateway{
		BaseURL:       base,
		APIKey:        key,
		CheapModel:    cheap,
		FrontierModel: frontier,
		HTTPClient:    &http.Client{Timeout: 60 * time.Second},
	}, nil
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model          string         `json:"model"`
	Messages       []chatMessage  `json:"messages"`
	Temperature    float64        `json:"temperature,omitempty"`
	MaxTokens      int            `json:"max_tokens,omitempty"`
	ResponseFormat map[string]any `json:"response_format,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (g *OpenAICompatGateway) Generate(req Request) (*Response, error) {
	model := g.CheapModel
	switch req.Tier {
	case TierFrontier, TierCouncil:
		model = g.FrontierModel
	case TierCheap, "":
		model = g.CheapModel
	default:
		return nil, fmt.Errorf("unknown tier: %s", req.Tier)
	}

	body := chatRequest{
		Model:       model,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}
	if req.System != "" {
		body.Messages = append(body.Messages, chatMessage{Role: "system", Content: req.System})
	}
	body.Messages = append(body.Messages, chatMessage{Role: "user", Content: req.User})

	// Use the Schema field that exists on Request
	if req.Schema != nil {
		body.ResponseFormat = map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name":   "attribute_update",
				"schema": req.Schema,
				"strict": false, // false so optional fields (not in required) are allowed; true requires every property in required
			},
		}
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := g.BaseURL + "/chat/completions"
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+g.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var parsed chatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("decode error: %w\nbody: %s", err, string(respBody))
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("provider error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response: %s", string(respBody))
	}

	return &Response{
		Content: parsed.Choices[0].Message.Content,
		Model:   parsed.Model,
		Usage: map[string]int{
			"prompt_tokens":     parsed.Usage.PromptTokens,
			"completion_tokens": parsed.Usage.CompletionTokens,
			"total_tokens":      parsed.Usage.TotalTokens,
		},
	}, nil
}

func NewOpenAICompat(baseURL, apiKey, cheapModel, frontierModel string) *OpenAICompatGateway {
	if cheapModel == "" {
		cheapModel = "gpt-4o-mini"
	}
	if frontierModel == "" {
		frontierModel = cheapModel
	}
	return &OpenAICompatGateway{
		BaseURL:       baseURL,
		APIKey:        apiKey,
		CheapModel:    cheapModel,
		FrontierModel: frontierModel,
		HTTPClient:    &http.Client{Timeout: 60 * time.Second},
	}
}
