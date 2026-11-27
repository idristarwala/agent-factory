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

// AnthropicGateway uses the Messages API.
// Schema enforcement is done via a forced tool call whose input schema is the reply schema.

type AnthropicGateway struct {
	APIKey        string
	CheapModel    string
	FrontierModel string
	HTTPClient    *http.Client
}

func NewAnthropicFromEnv() (*AnthropicGateway, error) {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY must be set")
	}
	cheap := os.Getenv("ANTHROPIC_CHEAP_MODEL")
	if cheap == "" {
		cheap = "claude-3-haiku-20240307"
	}
	frontier := os.Getenv("ANTHROPIC_FRONTIER_MODEL")
	if frontier == "" {
		frontier = "claude-3-5-sonnet-20241022"
	}
	return &AnthropicGateway{
		APIKey:        key,
		CheapModel:    cheap,
		FrontierModel: frontier,
		HTTPClient:    &http.Client{Timeout: 90 * time.Second},
	}, nil
}

func (g *AnthropicGateway) Generate(req Request) (*Response, error) {
	model := g.CheapModel
	if req.Tier == TierFrontier || req.Tier == TierCouncil {
		model = g.FrontierModel
	}

	var messages []map[string]any
	if req.System != "" {
		messages = append(messages, map[string]any{
			"role":    "user",
			"content": req.System,
		})
		messages = append(messages, map[string]any{
			"role":    "assistant",
			"content": "Understood.",
		})
	}
	messages = append(messages, map[string]any{
		"role":    "user",
		"content": req.User,
	})

	body := map[string]any{
		"model":      model,
		"max_tokens": req.MaxTokens,
		"messages":   messages,
	}

	// Forced tool call for schema enforcement — uses Schema field
	if len(req.Schema) > 0 {
		body["tools"] = []map[string]any{{
			"name":         "attribute_update",
			"description":  "Return the attribute updates",
			"input_schema": req.Schema,
		}}
		body["tool_choice"] = map[string]any{
			"type": "tool",
			"name": "attribute_update",
		}
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("x-api-key", g.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
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

	var parsed struct {
		Usage struct {
			Input  int `json:"input_tokens"`
			Output int `json:"output_tokens"`
		} `json:"usage"`
		Content []struct {
			Type  string         `json:"type"`
			Text  string         `json:"text"`
			Input map[string]any `json:"input"`
		} `json:"content"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("anthropic: %s", parsed.Error.Message)
	}

	usage := map[string]int{"prompt_tokens": parsed.Usage.Input, "completion_tokens": parsed.Usage.Output, "total_tokens": parsed.Usage.Input + parsed.Usage.Output}
	for _, c := range parsed.Content {
		if c.Type == "tool_use" && c.Input != nil {
			b, _ := json.Marshal(c.Input)
			return &Response{Content: string(b), Model: model, Usage: usage}, nil
		}
		if c.Type == "text" && c.Text != "" {
			return &Response{Content: c.Text, Model: model, Usage: usage}, nil
		}
	}
	return nil, fmt.Errorf("no content in anthropic response")
}
